package main

import (
	"testing"
	"time"
)

func TestCalculateBackoff(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name     string
		current  time.Duration
		expected time.Duration
	}{
		{
			name:     "InitialBackoff",
			current:  1 * time.Second,
			expected: 2 * time.Second,
		},
		{
			name:     "DoubleBackoff",
			current:  2 * time.Second,
			expected: 4 * time.Second,
		},
		{
			name:     "MaxBackoffReached",
			current:  40 * time.Second,
			expected: 60 * time.Second, // Should cap at maxBackoff
		},
		{
			name:     "ExceedsMaxBackoff",
			current:  100 * time.Second,
			expected: 60 * time.Second, // Should return maxBackoff
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBackoff(tt.current)
			if result != tt.expected {
				t.Errorf("calculateBackoff(%v) = %v, want %v", tt.current, result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("SubstringFound", func(t *testing.T) {
		str := "apple banana cherry"
		if !contains(str, "banana") {
			t.Error("Expected contains() to return true for 'banana'")
		}
	})

	t.Run("SubstringNotFound", func(t *testing.T) {
		str := "apple banana cherry"
		if contains(str, "grape") {
			t.Error("Expected contains() to return false for 'grape'")
		}
	})

	t.Run("EmptyString", func(t *testing.T) {
		str := ""
		if contains(str, "any") {
			t.Error("Expected contains() to return false for empty string")
		}
	})

	t.Run("EmptySubstring", func(t *testing.T) {
		str := "test string"
		if !contains(str, "") {
			t.Error("Expected contains() to return true for empty substring")
		}
	})

	t.Run("ExactMatch", func(t *testing.T) {
		str := "exact"
		if !contains(str, "exact") {
			t.Error("Expected contains() to return true for exact match")
		}
	})
}
