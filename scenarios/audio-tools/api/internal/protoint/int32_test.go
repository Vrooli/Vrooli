package protoint

import (
	"math"
	"testing"
)

func TestFromIntBounds(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input int
		want  int32
	}{
		{"normal", 42, 42},
		{"high", int(math.MaxInt32) + 1, math.MaxInt32},
		{"low", int(math.MinInt32) - 1, math.MinInt32},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := FromInt(tc.input); got != tc.want {
				t.Fatalf("FromInt(%d) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}
