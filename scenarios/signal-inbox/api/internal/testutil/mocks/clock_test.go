package mocks

import (
	"testing"
	"time"
)

// TestFakeClock_NowDoesNotAdvanceOnItsOwn proves the central guarantee
// of FakeClock: wall-clock drift never enters tests using it. Production
// timing bugs are then attributable to the code under test, not flakey
// CPU schedules.
func TestFakeClock_NowDoesNotAdvanceOnItsOwn(t *testing.T) {
	c := NewFakeClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	a := c.Now()
	time.Sleep(2 * time.Millisecond)
	b := c.Now()
	if !a.Equal(b) {
		t.Fatalf("FakeClock advanced on its own: a=%s b=%s", a, b)
	}
}

func TestFakeClock_AdvanceMovesNow(t *testing.T) {
	c := NewFakeClock(time.Time{})
	start := c.Now()
	c.Advance(5 * time.Minute)
	got := c.Now().Sub(start)
	if got != 5*time.Minute {
		t.Fatalf("expected 5m elapsed, got %v", got)
	}
}

func TestFakeClock_SetNowOverrides(t *testing.T) {
	c := NewFakeClock(time.Time{})
	target := time.Date(2030, 6, 15, 12, 0, 0, 0, time.UTC)
	c.SetNow(target)
	if !c.Now().Equal(target) {
		t.Fatalf("SetNow ignored: got %s, want %s", c.Now(), target)
	}
}

// TestNewFakeClock_ZeroDefaultsToStableEpoch documents the magic-default
// contract: callers passing time.Time{} get a stable, reproducible
// timestamp. If this constant ever drifts, every test that relies on
// snapshot-style assertions on times will silently change baselines.
func TestNewFakeClock_ZeroDefaultsToStableEpoch(t *testing.T) {
	c := NewFakeClock(time.Time{})
	want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !c.Now().Equal(want) {
		t.Fatalf("zero default = %s, want %s", c.Now(), want)
	}
}
