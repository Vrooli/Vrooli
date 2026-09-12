package scheduletest

import (
	"testing"
	"time"

	"github.com/vrooli/api-core/schedule"
)

func TestFakeClockTimerAndTickerAdvanceDeterministically(t *testing.T) {
	clock := New(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	timer := clock.NewTimer(5 * time.Second)
	ticker := clock.NewTicker(2 * time.Second)

	clock.Advance(4 * time.Second)
	select {
	case <-timer.C():
		t.Fatal("timer fired early")
	default:
	}
	select {
	case got := <-ticker.C():
		if got.Second() != 2 || got.Nanosecond() != 0 {
			t.Fatalf("first tick = %v", got)
		}
	default:
		t.Fatal("ticker did not fire")
	}

	clock.Advance(time.Second)
	select {
	case <-timer.C():
	default:
		t.Fatal("timer did not fire at deadline")
	}
}

func TestSystemElapsedIgnoresWallClockAdjustment(t *testing.T) {
	start := time.Now()
	wallAdjusted := start.Add(-time.Hour)
	if elapsed := schedule.Since(start); elapsed < 0 {
		t.Fatalf("elapsed time was negative: %v", elapsed)
	}
	if time.Since(wallAdjusted) < time.Hour {
		t.Fatal("wall-clock adjustment changed monotonic elapsed measurement")
	}
}
