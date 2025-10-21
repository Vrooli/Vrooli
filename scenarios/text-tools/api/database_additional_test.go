package main

import (
	"math"
	"math/rand"
	"testing"
	"time"
)

func TestComputeBackoffDelay(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	tests := []struct {
		name    string
		attempt int
		seed    int64
	}{
		{
			name:    "InitialBackoff",
			attempt: 1,
			seed:    7,
		},
		{
			name:    "DoublingBackoff",
			attempt: 2,
			seed:    11,
		},
		{
			name:    "MaxBackoffCapped",
			attempt: 10,
			seed:    19,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(tt.seed))
			result := sampleBackoff(tt.attempt, rng)

			expectedBase := time.Duration(math.Min(float64(initialBackoff)*math.Pow(backoffFactor, float64(tt.attempt-1)), float64(maxBackoff)))
			expectedRand := rand.New(rand.NewSource(tt.seed))
			expected := expectedBase + time.Duration(expectedRand.Float64()*float64(expectedBase)*0.25)

			if result != expected {
				t.Errorf("sampleBackoff(attempt=%d) = %v, want %v", tt.attempt, result, expected)
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
