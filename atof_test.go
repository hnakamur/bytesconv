// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytesconv_test

import (
	"bytes"
	"math"
	"math/rand"
	"reflect"
	"sync"
	"testing"
	"time"

	. "github.com/hnakamur/bytesconv"
)

type atofTest struct {
	in  []byte
	out []byte
	err error
}

var atoftests = []atofTest{
	{[]byte(""), []byte("0"), ErrSyntax},
	{[]byte("1"), []byte("1"), nil},
	{[]byte("+1"), []byte("1"), nil},
	{[]byte("1x"), []byte("0"), ErrSyntax},
	{[]byte("1.1."), []byte("0"), ErrSyntax},
	{[]byte("1e23"), []byte("1e+23"), nil},
	{[]byte("1E23"), []byte("1e+23"), nil},
	{[]byte("100000000000000000000000"), []byte("1e+23"), nil},
	{[]byte("1e-100"), []byte("1e-100"), nil},
	{[]byte("123456700"), []byte("1.234567e+08"), nil},
	{[]byte("99999999999999974834176"), []byte("9.999999999999997e+22"), nil},
	{[]byte("100000000000000000000001"), []byte("1.0000000000000001e+23"), nil},
	{[]byte("100000000000000008388608"), []byte("1.0000000000000001e+23"), nil},
	{[]byte("100000000000000016777215"), []byte("1.0000000000000001e+23"), nil},
	{[]byte("100000000000000016777216"), []byte("1.0000000000000003e+23"), nil},
	{[]byte("-1"), []byte("-1"), nil},
	{[]byte("-0.1"), []byte("-0.1"), nil},
	{[]byte("-0"), []byte("-0"), nil},
	{[]byte("1e-20"), []byte("1e-20"), nil},
	{[]byte("625e-3"), []byte("0.625"), nil},

	// zeros
	{[]byte("0"), []byte("0"), nil},
	{[]byte("0e0"), []byte("0"), nil},
	{[]byte("-0e0"), []byte("-0"), nil},
	{[]byte("+0e0"), []byte("0"), nil},
	{[]byte("0e-0"), []byte("0"), nil},
	{[]byte("-0e-0"), []byte("-0"), nil},
	{[]byte("+0e-0"), []byte("0"), nil},
	{[]byte("0e+0"), []byte("0"), nil},
	{[]byte("-0e+0"), []byte("-0"), nil},
	{[]byte("+0e+0"), []byte("0"), nil},
	{[]byte("0e+01234567890123456789"), []byte("0"), nil},
	{[]byte("0.00e-01234567890123456789"), []byte("0"), nil},
	{[]byte("-0e+01234567890123456789"), []byte("-0"), nil},
	{[]byte("-0.00e-01234567890123456789"), []byte("-0"), nil},
	{[]byte("0e291"), []byte("0"), nil}, // issue 15364
	{[]byte("0e292"), []byte("0"), nil}, // issue 15364
	{[]byte("0e347"), []byte("0"), nil}, // issue 15364
	{[]byte("0e348"), []byte("0"), nil}, // issue 15364
	{[]byte("-0e291"), []byte("-0"), nil},
	{[]byte("-0e292"), []byte("-0"), nil},
	{[]byte("-0e347"), []byte("-0"), nil},
	{[]byte("-0e348"), []byte("-0"), nil},

	// NaNs
	{[]byte("nan"), []byte("NaN"), nil},
	{[]byte("NaN"), []byte("NaN"), nil},
	{[]byte("NAN"), []byte("NaN"), nil},

	// Infs
	{[]byte("inf"), []byte("+Inf"), nil},
	{[]byte("-Inf"), []byte("-Inf"), nil},
	{[]byte("+INF"), []byte("+Inf"), nil},
	{[]byte("-Infinity"), []byte("-Inf"), nil},
	{[]byte("+INFINITY"), []byte("+Inf"), nil},
	{[]byte("Infinity"), []byte("+Inf"), nil},

	// largest float64
	{[]byte("1.7976931348623157e308"), []byte("1.7976931348623157e+308"), nil},
	{[]byte("-1.7976931348623157e308"), []byte("-1.7976931348623157e+308"), nil},
	// next float64 - too large
	{[]byte("1.7976931348623159e308"), []byte("+Inf"), ErrRange},
	{[]byte("-1.7976931348623159e308"), []byte("-Inf"), ErrRange},
	// the border is ...158079
	// borderline - okay
	{[]byte("1.7976931348623158e308"), []byte("1.7976931348623157e+308"), nil},
	{[]byte("-1.7976931348623158e308"), []byte("-1.7976931348623157e+308"), nil},
	// borderline - too large
	{[]byte("1.797693134862315808e308"), []byte("+Inf"), ErrRange},
	{[]byte("-1.797693134862315808e308"), []byte("-Inf"), ErrRange},

	// a little too large
	{[]byte("1e308"), []byte("1e+308"), nil},
	{[]byte("2e308"), []byte("+Inf"), ErrRange},
	{[]byte("1e309"), []byte("+Inf"), ErrRange},

	// way too large
	{[]byte("1e310"), []byte("+Inf"), ErrRange},
	{[]byte("-1e310"), []byte("-Inf"), ErrRange},
	{[]byte("1e400"), []byte("+Inf"), ErrRange},
	{[]byte("-1e400"), []byte("-Inf"), ErrRange},
	{[]byte("1e400000"), []byte("+Inf"), ErrRange},
	{[]byte("-1e400000"), []byte("-Inf"), ErrRange},

	// denormalized
	{[]byte("1e-305"), []byte("1e-305"), nil},
	{[]byte("1e-306"), []byte("1e-306"), nil},
	{[]byte("1e-307"), []byte("1e-307"), nil},
	{[]byte("1e-308"), []byte("1e-308"), nil},
	{[]byte("1e-309"), []byte("1e-309"), nil},
	{[]byte("1e-310"), []byte("1e-310"), nil},
	{[]byte("1e-322"), []byte("1e-322"), nil},
	// smallest denormal
	{[]byte("5e-324"), []byte("5e-324"), nil},
	{[]byte("4e-324"), []byte("5e-324"), nil},
	{[]byte("3e-324"), []byte("5e-324"), nil},
	// too small
	{[]byte("2e-324"), []byte("0"), nil},
	// way too small
	{[]byte("1e-350"), []byte("0"), nil},
	{[]byte("1e-400000"), []byte("0"), nil},

	// try to overflow exponent
	{[]byte("1e-4294967296"), []byte("0"), nil},
	{[]byte("1e+4294967296"), []byte("+Inf"), ErrRange},
	{[]byte("1e-18446744073709551616"), []byte("0"), nil},
	{[]byte("1e+18446744073709551616"), []byte("+Inf"), ErrRange},

	// Parse errors
	{[]byte("1e"), []byte("0"), ErrSyntax},
	{[]byte("1e-"), []byte("0"), ErrSyntax},
	{[]byte(".e-1"), []byte("0"), ErrSyntax},
	{[]byte("1\x00.2"), []byte("0"), ErrSyntax},

	// http://www.exploringbinary.com/java-hangs-when-converting-2-2250738585072012e-308/
	{[]byte("2.2250738585072012e-308"), []byte("2.2250738585072014e-308"), nil},
	// http://www.exploringbinary.com/php-hangs-on-numeric-value-2-2250738585072011e-308/
	{[]byte("2.2250738585072011e-308"), []byte("2.225073858507201e-308"), nil},

	// A very large number (initially wrongly parsed by the fast algorithm).
	{[]byte("4.630813248087435e+307"), []byte("4.630813248087435e+307"), nil},

	// A different kind of very large number.
	{[]byte("22.222222222222222"), []byte("22.22222222222222"), nil},
	{append(append([]byte("2."), bytes.Repeat([]byte("2"), 4000)...), "e+1"...), []byte("22.22222222222222"), nil},

	// Exactly halfway between 1 and math.Nextafter(1, 2).
	// Round to even (down).
	{[]byte("1.00000000000000011102230246251565404236316680908203125"), []byte("1"), nil},
	// Slightly lower; still round down.
	{[]byte("1.00000000000000011102230246251565404236316680908203124"), []byte("1"), nil},
	// Slightly higher; round up.
	{[]byte("1.00000000000000011102230246251565404236316680908203126"), []byte("1.0000000000000002"), nil},
	// Slightly higher, but you have to read all the way to the end.
	{append(append([]byte("1.00000000000000011102230246251565404236316680908203125"), bytes.Repeat([]byte("0"), 10000)...), "1"...), []byte("1.0000000000000002"), nil},
}

var atof32tests = []atofTest{
	// Exactly halfway between 1 and the next float32.
	// Round to even (down).
	{[]byte("1.000000059604644775390625"), []byte("1"), nil},
	// Slightly lower.
	{[]byte("1.000000059604644775390624"), []byte("1"), nil},
	// Slightly higher.
	{[]byte("1.000000059604644775390626"), []byte("1.0000001"), nil},
	// Slightly higher, but you have to read all the way to the end.
	{append(append([]byte("1.000000059604644775390625"), bytes.Repeat([]byte("0"), 10000)...), "1"...), []byte("1.0000001"), nil},

	// largest float32: (1<<128) * (1 - 2^-24)
	{[]byte("340282346638528859811704183484516925440"), []byte("3.4028235e+38"), nil},
	{[]byte("-340282346638528859811704183484516925440"), []byte("-3.4028235e+38"), nil},
	// next float32 - too large
	{[]byte("3.4028236e38"), []byte("+Inf"), ErrRange},
	{[]byte("-3.4028236e38"), []byte("-Inf"), ErrRange},
	// the border is 3.40282356779...e+38
	// borderline - okay
	{[]byte("3.402823567e38"), []byte("3.4028235e+38"), nil},
	{[]byte("-3.402823567e38"), []byte("-3.4028235e+38"), nil},
	// borderline - too large
	{[]byte("3.4028235678e38"), []byte("+Inf"), ErrRange},
	{[]byte("-3.4028235678e38"), []byte("-Inf"), ErrRange},

	// Denormals: less than 2^-126
	{[]byte("1e-38"), []byte("1e-38"), nil},
	{[]byte("1e-39"), []byte("1e-39"), nil},
	{[]byte("1e-40"), []byte("1e-40"), nil},
	{[]byte("1e-41"), []byte("1e-41"), nil},
	{[]byte("1e-42"), []byte("1e-42"), nil},
	{[]byte("1e-43"), []byte("1e-43"), nil},
	{[]byte("1e-44"), []byte("1e-44"), nil},
	{[]byte("6e-45"), []byte("6e-45"), nil}, // 4p-149 = 5.6e-45
	{[]byte("5e-45"), []byte("6e-45"), nil},
	// Smallest denormal
	{[]byte("1e-45"), []byte("1e-45"), nil}, // 1p-149 = 1.4e-45
	{[]byte("2e-45"), []byte("1e-45"), nil},

	// 2^92 = 8388608p+69 = 4951760157141521099596496896 (4.9517602e27)
	// is an exact power of two that needs 8 decimal digits to be correctly
	// parsed back.
	// The float32 before is 16777215p+68 = 4.95175986e+27
	// The halfway is 4.951760009. A bad algorithm that thinks the previous
	// float32 is 8388607p+69 will shorten incorrectly to 4.95176e+27.
	{[]byte("4951760157141521099596496896"), []byte("4.9517602e+27"), nil},
}

type atofSimpleTest struct {
	x float64
	s []byte
}

var (
	atofOnce               sync.Once
	atofRandomTests        []atofSimpleTest
	benchmarksRandomBits   [1024][]byte
	benchmarksRandomNormal [1024][]byte
)

func initAtof() {
	atofOnce.Do(initAtofOnce)
}

func initAtofOnce() {
	// The atof routines return NumErrors wrapping
	// the error and the string. Convert the table above.
	for i := range atoftests {
		test := &atoftests[i]
		if test.err != nil {
			test.err = &NumError{"ParseFloat", test.in, test.err}
		}
	}
	for i := range atof32tests {
		test := &atof32tests[i]
		if test.err != nil {
			test.err = &NumError{"ParseFloat", test.in, test.err}
		}
	}

	// Generate random inputs for tests and benchmarks
	rand.Seed(time.Now().UnixNano())
	if testing.Short() {
		atofRandomTests = make([]atofSimpleTest, 100)
	} else {
		atofRandomTests = make([]atofSimpleTest, 10000)
	}
	for i := range atofRandomTests {
		n := uint64(rand.Uint32())<<32 | uint64(rand.Uint32())
		x := math.Float64frombits(n)
		s := FormatFloat(x, 'g', -1, 64)
		atofRandomTests[i] = atofSimpleTest{x, s}
	}

	for i := range benchmarksRandomBits {
		bits := uint64(rand.Uint32())<<32 | uint64(rand.Uint32())
		x := math.Float64frombits(bits)
		benchmarksRandomBits[i] = FormatFloat(x, 'g', -1, 64)
	}

	for i := range benchmarksRandomNormal {
		x := rand.NormFloat64()
		benchmarksRandomNormal[i] = FormatFloat(x, 'g', -1, 64)
	}
}

func testAtof(t *testing.T, opt bool) {
	initAtof()
	oldopt := SetOptimize(opt)
	for i := 0; i < len(atoftests); i++ {
		test := &atoftests[i]
		out, err := ParseFloat(test.in, 64)
		outs := FormatFloat(out, 'g', -1, 64)
		if !bytes.Equal(outs, test.out) || !reflect.DeepEqual(err, test.err) {
			t.Errorf("ParseFloat(%v, 64) = %v, %v want %v, %v",
				test.in, out, err, test.out, test.err)
		}

		if float64(float32(out)) == out {
			out, err := ParseFloat(test.in, 32)
			out32 := float32(out)
			if float64(out32) != out {
				t.Errorf("ParseFloat(%v, 32) = %v, not a float32 (closest is %v)", test.in, out, float64(out32))
				continue
			}
			outs := FormatFloat(float64(out32), 'g', -1, 32)
			if !bytes.Equal(outs, test.out) || !reflect.DeepEqual(err, test.err) {
				t.Errorf("ParseFloat(%v, 32) = %v, %v want %v, %v  # %v",
					test.in, out32, err, test.out, test.err, out)
			}
		}
	}
	for _, test := range atof32tests {
		out, err := ParseFloat(test.in, 32)
		out32 := float32(out)
		if float64(out32) != out {
			t.Errorf("ParseFloat(%v, 32) = %v, not a float32 (closest is %v)", test.in, out, float64(out32))
			continue
		}
		outs := FormatFloat(float64(out32), 'g', -1, 32)
		if !bytes.Equal(outs, test.out) || !reflect.DeepEqual(err, test.err) {
			t.Errorf("ParseFloat(%v, 32) = %v, %v want %v, %v  # %v",
				test.in, out32, err, test.out, test.err, out)
		}
	}
	SetOptimize(oldopt)
}

func TestAtof(t *testing.T) { testAtof(t, true) }

func TestAtofSlow(t *testing.T) { testAtof(t, false) }

func TestAtofRandom(t *testing.T) {
	initAtof()
	for _, test := range atofRandomTests {
		x, _ := ParseFloat(test.s, 64)
		switch {
		default:
			t.Errorf("number %s badly parsed as %b (expected %b)", test.s, x, test.x)
		case x == test.x:
		case math.IsNaN(test.x) && math.IsNaN(x):
		}
	}
	t.Logf("tested %d random numbers", len(atofRandomTests))
}

var roundTripCases = []struct {
	f float64
	s []byte
}{
	// Issue 2917.
	// This test will break the optimized conversion if the
	// FPU is using 80-bit registers instead of 64-bit registers,
	// usually because the operating system initialized the
	// thread with 80-bit precision and the Go runtime didn't
	// fix the FP control word.
	{8865794286000691 << 39, []byte("4.87402195346389e+27")},
	{8865794286000692 << 39, []byte("4.8740219534638903e+27")},
}

func TestRoundTrip(t *testing.T) {
	for _, tt := range roundTripCases {
		old := SetOptimize(false)
		s := FormatFloat(tt.f, 'g', -1, 64)
		if !bytes.Equal(s, tt.s) {
			t.Errorf("no-opt FormatFloat(%b) = %s, want %s", tt.f, s, tt.s)
		}
		f, err := ParseFloat(tt.s, 64)
		if f != tt.f || err != nil {
			t.Errorf("no-opt ParseFloat(%s) = %b, %v want %b, nil", tt.s, f, err, tt.f)
		}
		SetOptimize(true)
		s = FormatFloat(tt.f, 'g', -1, 64)
		if !bytes.Equal(s, tt.s) {
			t.Errorf("opt FormatFloat(%b) = %s, want %s", tt.f, s, tt.s)
		}
		f, err = ParseFloat(tt.s, 64)
		if f != tt.f || err != nil {
			t.Errorf("opt ParseFloat(%s) = %b, %v want %b, nil", tt.s, f, err, tt.f)
		}
		SetOptimize(old)
	}
}

// TestRoundTrip32 tries a fraction of all finite positive float32 values.
func TestRoundTrip32(t *testing.T) {
	step := uint32(997)
	if testing.Short() {
		step = 99991
	}
	count := 0
	for i := uint32(0); i < 0xff<<23; i += step {
		f := math.Float32frombits(i)
		if i&1 == 1 {
			f = -f // negative
		}
		s := FormatFloat(float64(f), 'g', -1, 32)

		parsed, err := ParseFloat(s, 32)
		parsed32 := float32(parsed)
		switch {
		case err != nil:
			t.Errorf("ParseFloat(%q, 32) gave error %s", s, err)
		case float64(parsed32) != parsed:
			t.Errorf("ParseFloat(%q, 32) = %v, not a float32 (nearest is %v)", s, parsed, parsed32)
		case parsed32 != f:
			t.Errorf("ParseFloat(%q, 32) = %b (expected %b)", s, parsed32, f)
		}
		count++
	}
	t.Logf("tested %d float32's", count)
}

func BenchmarkAtof64Decimal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseFloat([]byte("33909"), 64)
	}
}

func BenchmarkAtof64Float(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseFloat([]byte("339.7784"), 64)
	}
}

func BenchmarkAtof64FloatExp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseFloat([]byte("-5.09e75"), 64)
	}
}

func BenchmarkAtof64Big(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseFloat([]byte("123456789123456789123456789"), 64)
	}
}

func BenchmarkAtof64RandomBits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseFloat(benchmarksRandomBits[i%1024], 64)
	}
}

func BenchmarkAtof64RandomFloats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseFloat(benchmarksRandomNormal[i%1024], 64)
	}
}

func BenchmarkAtof32Decimal(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseFloat([]byte("33909"), 32)
	}
}

func BenchmarkAtof32Float(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseFloat([]byte("339.778"), 32)
	}
}

func BenchmarkAtof32FloatExp(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ParseFloat([]byte("12.3456e32"), 32)
	}
}

var float32strings [4096][]byte

func BenchmarkAtof32Random(b *testing.B) {
	n := uint32(997)
	for i := range float32strings {
		n = (99991*n + 42) % (0xff << 23)
		float32strings[i] = FormatFloat(float64(math.Float32frombits(n)), 'g', -1, 32)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseFloat(float32strings[i%4096], 32)
	}
}
