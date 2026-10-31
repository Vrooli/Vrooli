package schedule

import (
	"testing"
	"time"
)

func TestSinceUsesMonotonicReading(t *testing.T) {
	t.Helper()

	start := time.Now()
	wallClockAdjustedStart := start.Add(-time.Hour)

	if elapsed := Since(wallClockAdjustedStart); elapsed < time.Hour {
		t.Fatalf("Since() = %s, want at least one hour after a wall-clock adjustment", elapsed)
	}
}
