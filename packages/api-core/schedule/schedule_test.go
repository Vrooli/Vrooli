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

func TestFakeTimerRegistryTracksOnlyActiveTimers(t *testing.T) {
	clock := NewFake(time.Unix(0, 0))
	timer := clock.NewTimer(time.Second)
	if got := len(clock.timers); got != 1 {
		t.Fatalf("registered timers = %d, want 1", got)
	}
	if !timer.Stop() {
		t.Fatal("first Stop() = false, want true")
	}
	if got := len(clock.timers); got != 0 {
		t.Fatalf("registered timers after Stop = %d, want 0", got)
	}

	timer.Reset(time.Second)
	if got := len(clock.timers); got != 1 {
		t.Fatalf("registered timers after Reset = %d, want 1", got)
	}
	clock.Advance(time.Second)
	select {
	case <-timer.C():
	default:
		t.Fatal("reset timer did not fire")
	}
	if got := len(clock.timers); got != 0 {
		t.Fatalf("registered timers after firing = %d, want 0", got)
	}
}
