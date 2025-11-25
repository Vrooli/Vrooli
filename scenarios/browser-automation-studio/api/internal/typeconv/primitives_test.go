package typeconv

import (
	"encoding/json"
	"testing"
	"time"
)

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"string", "hello", "hello"},
		{"empty string", "", ""},
		{"byte slice", []byte("world"), "world"},
		{"int", 42, ""},
		{"nil", nil, ""},
		{"float", 3.14, ""},
		{"bool", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			if result != tt.expected {
				t.Errorf("ToString(%v) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
	}{
		{"int", 42, 42},
		{"int32", int32(100), 100},
		{"int64", int64(200), 200},
		{"float64", 3.14, 3},
		{"json.Number int", json.Number("42"), 42},
		{"string int", "123", 123},
		{"string invalid", "abc", 0},
		{"nil", nil, 0},
		{"empty string", "", 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToInt(tt.input)
			if result != tt.expected {
				t.Errorf("ToInt(%v) = %d, expected %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{"bool true", true, true},
		{"bool false", false, false},
		{"string true", "true", true},
		{"string false", "false", false},
		{"string 1", "1", true},
		{"string 0", "0", false},
		{"string invalid", "abc", false},
		{"int", 1, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToBool(tt.input)
			if result != tt.expected {
				t.Errorf("ToBool(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToFloat(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected float64
	}{
		{"float64", 3.14, 3.14},
		{"float32", float32(2.5), 2.5},
		{"int", 42, 42.0},
		{"int32", int32(100), 100.0},
		{"int64", int64(200), 200.0},
		{"json.Number float", json.Number("3.14"), 3.14},
		{"string float", "2.5", 2.5},
		{"string invalid", "abc", 0.0},
		{"nil", nil, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToFloat(tt.input)
			if result != tt.expected {
				t.Errorf("ToFloat(%v) = %f, expected %f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToTimePtr(t *testing.T) {
	now := time.Now()
	rfc3339Time := "2024-01-15T10:30:00Z"
	rfc3339NanoTime := "2024-01-15T10:30:00.123456789Z"

	tests := []struct {
		name      string
		input     any
		expectNil bool
		validate  func(*time.Time) bool
	}{
		{
			name:      "time.Time",
			input:     now,
			expectNil: false,
			validate:  func(t *time.Time) bool { return t.Equal(now) },
		},
		{
			name:      "pointer to time.Time",
			input:     &now,
			expectNil: false,
			validate:  func(t *time.Time) bool { return t.Equal(now) },
		},
		{
			name:      "RFC3339 string",
			input:     rfc3339Time,
			expectNil: false,
			validate:  func(t *time.Time) bool { return !t.IsZero() },
		},
		{
			name:      "RFC3339Nano string",
			input:     rfc3339NanoTime,
			expectNil: false,
			validate:  func(t *time.Time) bool { return !t.IsZero() },
		},
		{
			name:      "empty string",
			input:     "",
			expectNil: true,
		},
		{
			name:      "invalid string",
			input:     "not a time",
			expectNil: true,
		},
		{
			name:      "nil",
			input:     nil,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToTimePtr(tt.input)
			if tt.expectNil {
				if result != nil {
					t.Errorf("ToTimePtr(%v) should return nil, got %v", tt.input, result)
				}
			} else {
				if result == nil {
					t.Errorf("ToTimePtr(%v) should not return nil", tt.input)
				} else if tt.validate != nil && !tt.validate(result) {
					t.Errorf("ToTimePtr(%v) validation failed", tt.input)
				}
			}
		})
	}
}

func TestToStringSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected []string
	}{
		{"[]string", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"[]any with strings", []any{"x", "y", "z"}, []string{"x", "y", "z"}},
		{"[]any with mixed", []any{"a", 42, "b"}, []string{"a", "b"}},
		{"single string", "hello", []string{"hello"}},
		{"empty string", "", []string{}},
		{"nil", nil, []string{}},
		{"int", 42, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToStringSlice(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ToStringSlice(%v) length = %d, expected %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("ToStringSlice(%v)[%d] = %q, expected %q", tt.input, i, result[i], tt.expected[i])
				}
			}
		})
	}
}
