package cliutil

import (
	"math"
	"strconv"
	"testing"
)

func TestParseInt32(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int32
	}{
		{"empty is unset", "", 0},
		{"garbage is unset", "abc", 0},
		{"in-range value preserved", "25", 25},
		{"whitespace trimmed", "  7 ", 7},
		{"max int32 preserved", strconv.Itoa(math.MaxInt32), math.MaxInt32},
		{"overflow clamps instead of wrapping", strconv.Itoa(math.MaxInt32 + 1), math.MaxInt32},
		{"huge overflow clamps", "9999999999999", math.MaxInt32},
		{"negative clamps to zero", "-5", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseInt32(tc.in); got != tc.want {
				t.Fatalf("ParseInt32(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}
