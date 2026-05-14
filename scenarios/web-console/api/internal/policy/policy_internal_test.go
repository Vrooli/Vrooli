package policy

import (
	"testing"
	"time"
)

// TestCustomDurationConstants verifies custom duration bounds are internally consistent.
func TestCustomDurationConstants(t *testing.T) {
	if customDurationMin >= customDurationMax {
		t.Errorf("min (%s) should be less than max (%s)", customDurationMin, customDurationMax)
	}
	if customDurationMin != time.Minute {
		t.Errorf("customDurationMin should be 1m, got %s", customDurationMin)
	}
	if customDurationMax != 7*24*time.Hour {
		t.Errorf("customDurationMax should be 168h, got %s", customDurationMax)
	}
}
