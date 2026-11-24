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
	if got, err := transformer.compareValues(math.Inf(1), "-Inf"); err != nil || got != 1 {
		t.Errorf("compareValues(+Inf, -Inf) = %d, want 1", got)
	}
}

func TestFilterRejectsInvalidConditionsInsteadOfReturningMisleadingRows(t *testing.T) {
	transformer := NewTransformer()
	data := []map[string]interface{}{{"age": 30, "name": "A"}}
	tests := []struct {
		name      string
		condition string
	}{
		{name: "nonnumeric equality literal", condition: "age = not-a-number"},
		{name: "numeric literal with trailing junk", condition: "age > 20junk"},
		{name: "missing field", condition: "salary = 10"},
		{name: "unsupported operator", condition: "age ~~ 20"},
		{name: "malformed condition", condition: "age >"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := transformer.Transform(data, []Transformation{{
				Type:       TransformFilter,
				Parameters: map[string]interface{}{"condition": tt.condition},
			}})
			if err == nil {
				t.Fatalf("Transform accepted invalid condition %q", tt.condition)
			}
		})
	}

	t.Run("malformed condition is rejected for empty input", func(t *testing.T) {
		_, err := transformer.Transform(nil, []Transformation{{
			Type:       TransformFilter,
			Parameters: map[string]interface{}{"condition": "age >"},
		}})
		if err == nil {
			t.Fatal("Transform accepted malformed condition for empty input")
		}
	})

	t.Run("boolean fields reject ordering operators", func(t *testing.T) {
		_, err := transformer.Transform([]map[string]interface{}{{"active": true}}, []Transformation{{
			Type:       TransformFilter,
			Parameters: map[string]interface{}{"condition": "active > false"},
		}})
		if err == nil {
			t.Fatal("Transform accepted ordering comparison on a boolean field")
		}
	})
}
