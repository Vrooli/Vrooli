package stats

import (
	"math"
	"testing"
)

func TestNearestRankIndexP90AgainstIndependentRank(t *testing.T) {
	for n := 1; n <= 60; n++ {
		// Keep this computation independent from NearestRankIndex so the table
		// test catches a shared implementation mistake.
		expected := int(math.Ceil(0.9*float64(n))) - 1
		if expected < 0 {
			expected = 0
		}
		if expected >= n {
			expected = n - 1
		}
		if got := NearestRankIndex(n, 0.9); got != expected {
			t.Fatalf("n=%d: index=%d, want %d", n, got, expected)
		}
	}
}

func TestPercentileEmpty(t *testing.T) {
	if _, ok := Percentile([]int(nil), 0.9); ok {
		t.Fatal("empty input returned a percentile")
	}
}
