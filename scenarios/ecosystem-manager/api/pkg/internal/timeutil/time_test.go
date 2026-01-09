package timeutil

import (
	"testing"
	"time"
)

func TestNowRFC3339(t *testing.T) {
	// Test that the function returns a valid RFC3339 timestamp
	result := NowRFC3339()

	// Verify it's not empty
	if result == "" {
		t.Error("NowRFC3339() returned empty string")
	}

	// Verify it can be parsed as RFC3339
	parsed, err := time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("NowRFC3339() returned invalid RFC3339 format: %v, error: %v", result, err)
	}

	// Verify the timestamp is recent (within last second)
	now := time.Now()
	diff := now.Sub(parsed)
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Second {
		t.Errorf("NowRFC3339() returned timestamp too far from current time: %v (diff: %v)", result, diff)
	}
}

func TestNowRFC3339Format(t *testing.T) {
	// Test multiple calls to ensure consistent format
	results := make([]string, 5)
	for i := range results {
		results[i] = NowRFC3339()
		time.Sleep(time.Millisecond) // Small delay to ensure different timestamps
	}

	// Verify all results are parseable
	for i, result := range results {
		_, err := time.Parse(time.RFC3339, result)
		if err != nil {
			t.Errorf("Call %d: NowRFC3339() returned invalid RFC3339 format: %v", i, result)
		}
	}

	// Verify results are different (timestamps progressed)
	for i := 1; i < len(results); i++ {
		if results[i] == results[i-1] {
			// This is okay if calls were too fast, but log it
			t.Logf("Consecutive calls returned same timestamp: %v", results[i])
		}
	}
}
