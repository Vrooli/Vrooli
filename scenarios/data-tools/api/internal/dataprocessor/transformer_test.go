package dataprocessor

import (
	"math"
	"testing"
)

func TestCompareOrdered(t *testing.T) {
	tests := []struct {
		name string
		a, b interface{}
		want int
	}{
		{name: "integers less", a: int(-1), b: int(2), want: -1},
		{name: "integers equal", a: int(2), b: int(2), want: 0},
		{name: "integers greater", a: int(3), b: int(2), want: 1},
		{name: "float less", a: -1.5, b: 2.5, want: -1},
		{name: "float equal", a: 2.5, b: 2.5, want: 0},
		{name: "float greater", a: 3.5, b: 2.5, want: 1},
		{name: "signed zero", a: math.Copysign(0, -1), b: 0.0, want: 0},
		{name: "NaN left", a: math.NaN(), b: 1.0, want: 0},
		{name: "NaN right", a: 1.0, b: math.NaN(), want: 0},
		{name: "negative infinity", a: math.Inf(-1), b: math.Inf(1), want: -1},
		{name: "positive infinity", a: math.Inf(1), b: math.MaxFloat64, want: 1},
	}

	transformer := NewTransformer()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int
			switch a := tt.a.(type) {
			case int:
				got = compareOrdered(a, tt.b.(int))
			case float64:
				got = compareOrdered(a, tt.b.(float64))
			}
			if got != tt.want {
				t.Fatalf("compareOrdered(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}

	if got := transformer.compare(1, 2); got != -1 {
		t.Errorf("compare(int, int) = %d, want -1", got)
	}
	if got := transformer.compare(math.NaN(), 1.0); got != 0 {
		t.Errorf("compare(NaN, 1) = %d, want 0", got)
	}
	if got := transformer.compareValues(math.Inf(1), "-Inf"); got != 1 {
		t.Errorf("compareValues(+Inf, -Inf) = %d, want 1", got)
	}
}
