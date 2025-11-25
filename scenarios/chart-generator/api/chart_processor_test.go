package main

import (
	"testing"
)

// Helper test functions - utility tests for common functions

func Test_toFloat64(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected float64
	}{
		{"Float64", float64(42.5), 42.5},
		{"Float32", float32(10.2), 10.199999809265137}, // Float32 precision
		{"Int", 100, 100.0},
		{"Int32", int32(50), 50.0},
		{"Int64", int64(75), 75.0},
		{"String", "123.45", 123.45},
		{"InvalidString", "abc", 0.0},
		{"Nil", nil, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toFloat64(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func Test_contains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{"Found", []string{"a", "b", "c"}, "b", true},
		{"NotFound", []string{"a", "b", "c"}, "d", false},
		{"EmptySlice", []string{}, "a", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && len(substr) > 0 && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
