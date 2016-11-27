// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Stratex node: This file was copied from database/sql and slightly modified.

package sqltypes

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"
	"time"
)

var someTime = time.Unix(123, 0)
var answer int64 = 42

type userDefined float64

type userDefinedSlice []int

type conversionTest struct {
	s, d interface{} // source and destination

	// following are used if they're non-zero
	wantint    int64
	wantuint   uint64
	wantstr    string
	wantbytes  []byte
	wantraw    sql.RawBytes
	wantf32    float32
	wantf64    float64
	wanttime   time.Time
	wantbool   bool // used if d is of type *bool
	wanterr    string
	wantiface  interface{}
	wantptr    *int64 // if non-nil, *d's pointed value must be equal to *wantptr
	wantnil    bool   // if true, *d must be *int64(nil)
	wantusrdef userDefined
}

// Target variables for scanning into.
var (
	scanstr    string
	scanbytes  []byte
	scanraw    sql.RawBytes
	scanint    int
	scanint8   int8
	scanint16  int16
	scanint32  int32
	scanuint8  uint8
	scanuint16 uint16
	scanbool   bool
	scanf32    float32
	scanf64    float64
	scantime   time.Time
	scanptr    *int64
	scaniface  interface{}
)

var conversionTests = []conversionTest{
	// Exact conversions (destination pointer type matches source type)
	{s: "foo", d: &scanstr, wantstr: "foo"},
	{s: 123, d: &scanint, wantint: 123},
	{s: someTime, d: &scantime, wanttime: someTime},

	// To strings
	{s: "string", d: &scanstr, wantstr: "string"},
	{s: []byte("byteslice"), d: &scanstr, wantstr: "byteslice"},
	{s: 123, d: &scanstr, wantstr: "123"},
	{s: int8(123), d: &scanstr, wantstr: "123"},
	{s: int64(123), d: &scanstr, wantstr: "123"},
	{s: uint8(123), d: &scanstr, wantstr: "123"},
	{s: uint16(123), d: &scanstr, wantstr: "123"},
	{s: uint32(123), d: &scanstr, wantstr: "123"},
	{s: uint64(123), d: &scanstr, wantstr: "123"},
	{s: 1.5, d: &scanstr, wantstr: "1.5"},

	// From time.Time:
	{s: time.Unix(1, 0).UTC(), d: &scanstr, wantstr: "1970-01-01T00:00:01Z"},
	{s: time.Unix(1453874597, 0).In(time.FixedZone("here", -3600*8)), d: &scanstr, wantstr: "2016-01-26T22:03:17-08:00"},
	{s: time.Unix(1, 2).UTC(), d: &scanstr, wantstr: "1970-01-01T00:00:01.000000002Z"},
	{s: time.Time{}, d: &scanstr, wantstr: "0001-01-01T00:00:00Z"},
	{s: time.Unix(1, 2).UTC(), d: &scanbytes, wantbytes: []byte("1970-01-01T00:00:01.000000002Z")},
	{s: time.Unix(1, 2).UTC(), d: &scaniface, wantiface: time.Unix(1, 2).UTC()},

	// To []byte
	{s: nil, d: &scanbytes, wantbytes: nil},
	{s: "string", d: &scanbytes, wantbytes: []byte("string")},
	{s: []byte("byteslice"), d: &scanbytes, wantbytes: []byte("byteslice")},
	{s: 123, d: &scanbytes, wantbytes: []byte("123")},
	{s: int8(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: int64(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: uint8(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: uint16(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: uint32(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: uint64(123), d: &scanbytes, wantbytes: []byte("123")},
	{s: 1.5, d: &scanbytes, wantbytes: []byte("1.5")},

	// To sql.RawBytes
	{s: nil, d: &scanraw, wantraw: nil},
	{s: []byte("byteslice"), d: &scanraw, wantraw: sql.RawBytes("byteslice")},
	{s: 123, d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: int8(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: int64(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: uint8(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: uint16(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: uint32(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: uint64(123), d: &scanraw, wantraw: sql.RawBytes("123")},
	{s: 1.5, d: &scanraw, wantraw: sql.RawBytes("1.5")},

	// Strings to integers
	{s: "255", d: &scanuint8, wantuint: 255},
	{s: "256", d: &scanuint8, wanterr: "converting driver.Value type string (\"256\") to a uint8: value out of range"},
	{s: "256", d: &scanuint16, wantuint: 256},
	{s: "-1", d: &scanint, wantint: -1},
	{s: "foo", d: &scanint, wanterr: "converting driver.Value type string (\"foo\") to a int: invalid syntax"},

	// int64 to smaller integers
	{s: int64(5), d: &scanuint8, wantuint: 5},
	{s: int64(256), d: &scanuint8, wanterr: "converting driver.Value type int64 (\"256\") to a uint8: value out of range"},
	{s: int64(256), d: &scanuint16, wantuint: 256},
	{s: int64(65536), d: &scanuint16, wanterr: "converting driver.Value type int64 (\"65536\") to a uint16: value out of range"},

	// True bools
	{s: true, d: &scanbool, wantbool: true},
	{s: "True", d: &scanbool, wantbool: true},
	{s: "TRUE", d: &scanbool, wantbool: true},
	{s: "1", d: &scanbool, wantbool: true},
	{s: 1, d: &scanbool, wantbool: true},
	{s: int64(1), d: &scanbool, wantbool: true},
	{s: uint16(1), d: &scanbool, wantbool: true},

	// False bools
	{s: false, d: &scanbool, wantbool: false},
	{s: "false", d: &scanbool, wantbool: false},
	{s: "FALSE", d: &scanbool, wantbool: false},
	{s: "0", d: &scanbool, wantbool: false},
	{s: 0, d: &scanbool, wantbool: false},
	{s: int64(0), d: &scanbool, wantbool: false},
	{s: uint16(0), d: &scanbool, wantbool: false},

	// Not bools
	{s: "yup", d: &scanbool, wanterr: `sql/driver: couldn't convert "yup" into type bool`},
	{s: 2, d: &scanbool, wanterr: `sql/driver: couldn't convert 2 into type bool`},

	// Floats
	{s: float64(1.5), d: &scanf64, wantf64: float64(1.5)},
	{s: int64(1), d: &scanf64, wantf64: float64(1)},
	{s: float64(1.5), d: &scanf32, wantf32: float32(1.5)},
	{s: "1.5", d: &scanf32, wantf32: float32(1.5)},
	{s: "1.5", d: &scanf64, wantf64: float64(1.5)},

	// Pointers
	{s: interface{}(nil), d: &scanptr, wantnil: true},
	{s: int64(42), d: &scanptr, wantptr: &answer},

	// To interface{}
	{s: float64(1.5), d: &scaniface, wantiface: float64(1.5)},
	{s: int64(1), d: &scaniface, wantiface: int64(1)},
	{s: "str", d: &scaniface, wantiface: "str"},
	{s: []byte("byteslice"), d: &scaniface, wantiface: []byte("byteslice")},
	{s: true, d: &scaniface, wantiface: true},
	{s: nil, d: &scaniface},
	{s: []byte(nil), d: &scaniface, wantiface: []byte(nil)},

	// To a user-defined type
	{s: 1.5, d: new(userDefined), wantusrdef: 1.5},
	{s: int64(123), d: new(userDefined), wantusrdef: 123},
	{s: "1.5", d: new(userDefined), wantusrdef: 1.5},
	{s: []byte{1, 2, 3}, d: new(userDefinedSlice), wanterr: `unsupported Scan, storing driver.Value type []uint8 into type *sql.userDefinedSlice`},

	// Other errors
	{s: complex(1, 2), d: &scanstr, wanterr: `unsupported Scan, storing driver.Value type complex128 into type *string`},
}

func intPtrValue(intptr interface{}) interface{} {
	return reflect.Indirect(reflect.Indirect(reflect.ValueOf(intptr))).Int()
}

func intValue(intptr interface{}) int64 {
	return reflect.Indirect(reflect.ValueOf(intptr)).Int()
}

func uintValue(intptr interface{}) uint64 {
	return reflect.Indirect(reflect.ValueOf(intptr)).Uint()
}

func float64Value(ptr interface{}) float64 {
	return *(ptr.(*float64))
}

func float32Value(ptr interface{}) float32 {
	return *(ptr.(*float32))
}

func timeValue(ptr interface{}) time.Time {
	return *(ptr.(*time.Time))
}

func TestConversions(t *testing.T) {
	for n, ct := range conversionTests {
		err := ConvertAssign(ct.d, ct.s)
		errstr := ""
		if err != nil {
			errstr = err.Error()
		}
		errf := func(format string, args ...interface{}) {
			base := fmt.Sprintf("ConvertAssign #%d: for %v (%T) -> %T, ", n, ct.s, ct.s, ct.d)
			t.Errorf(base+format, args...)
		}
		if errstr != ct.wanterr {
			errf("got error %q, want error %q", errstr, ct.wanterr)
		}
		if ct.wantstr != "" && ct.wantstr != scanstr {
			errf("want string %q, got %q", ct.wantstr, scanstr)
		}
		if ct.wantint != 0 && ct.wantint != intValue(ct.d) {
			errf("want int %d, got %d", ct.wantint, intValue(ct.d))
		}
		if ct.wantuint != 0 && ct.wantuint != uintValue(ct.d) {
			errf("want uint %d, got %d", ct.wantuint, uintValue(ct.d))
		}
		if ct.wantf32 != 0 && ct.wantf32 != float32Value(ct.d) {
			errf("want float32 %v, got %v", ct.wantf32, float32Value(ct.d))
		}
		if ct.wantf64 != 0 && ct.wantf64 != float64Value(ct.d) {
			errf("want float32 %v, got %v", ct.wantf64, float64Value(ct.d))
		}
		if bp, boolTest := ct.d.(*bool); boolTest && *bp != ct.wantbool && ct.wanterr == "" {
			errf("want bool %v, got %v", ct.wantbool, *bp)
		}
		if !ct.wanttime.IsZero() && !ct.wanttime.Equal(timeValue(ct.d)) {
			errf("want time %v, got %v", ct.wanttime, timeValue(ct.d))
		}
		if ct.wantnil && *ct.d.(**int64) != nil {
			errf("want nil, got %v", intPtrValue(ct.d))
		}
		if ct.wantptr != nil {
			if *ct.d.(**int64) == nil {
				errf("want pointer to %v, got nil", *ct.wantptr)
			} else if *ct.wantptr != intPtrValue(ct.d) {
				errf("want pointer to %v, got %v", *ct.wantptr, intPtrValue(ct.d))
			}
		}
		if ifptr, ok := ct.d.(*interface{}); ok {
			if !reflect.DeepEqual(ct.wantiface, scaniface) {
				errf("want interface %#v, got %#v", ct.wantiface, scaniface)
				continue
			}
			if srcBytes, ok := ct.s.([]byte); ok {
				dstBytes := (*ifptr).([]byte)
				if len(srcBytes) > 0 && &dstBytes[0] == &srcBytes[0] {
					errf("copy into interface{} didn't copy []byte data")
				}
			}
		}
		if ct.wantusrdef != 0 && ct.wantusrdef != *ct.d.(*userDefined) {
			errf("want userDefined %f, got %f", ct.wantusrdef, *ct.d.(*userDefined))
		}
	}
}
