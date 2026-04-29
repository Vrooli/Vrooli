package mocks

import (
	"testing"
	"time"
)

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
	if got := c.Since(start); got != 5*time.Minute {
		t.Fatalf("expected 5m elapsed, got %v", got)
	}
}

func TestFakeClock_SleepEqualsAdvance(t *testing.T) {
	c := NewFakeClock(time.Time{})
	start := c.Now()
	c.Sleep(7 * time.Second)
	if got := c.Now().Sub(start); got != 7*time.Second {
		t.Fatalf("expected 7s elapsed after Sleep, got %v", got)
	}
}

func TestFakeClock_TickerFiresOnAdvance(t *testing.T) {
	c := NewFakeClock(time.Time{})
	tk := c.NewTicker(time.Second)
	defer tk.Stop()

	select {
	case <-tk.C():
		t.Fatal("ticker fired before any advance")
	default:
	}

	c.Advance(time.Second)
	select {
	case <-tk.C():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ticker did not fire after advance")
	}
}

func TestFakeClock_TickerCoalescesWhenBufferFull(t *testing.T) {
	c := NewFakeClock(time.Time{})
	tk := c.NewTicker(time.Second)
	defer tk.Stop()

	// Three ticks worth without draining; channel is buffered to 1.
	c.Advance(3 * time.Second)

	select {
	case <-tk.C():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ticker did not deliver any tick")
	}
	// Subsequent reads should not produce ticks because we coalesced.
	select {
	case <-tk.C():
		t.Fatal("expected coalescing — only one queued tick allowed")
	default:
	}
}

func TestFakeClock_StopHaltsFiring(t *testing.T) {
	c := NewFakeClock(time.Time{})
	tk := c.NewTicker(time.Second)
	tk.Stop()
	c.Advance(10 * time.Second)
	select {
	case <-tk.C():
		t.Fatal("stopped ticker still fired")
	default:
	}
}

func TestFakeClock_SetNowAdvancesTickers(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFakeClock(start)
	tk := c.NewTicker(time.Minute)
	defer tk.Stop()

	c.SetNow(start.Add(2 * time.Minute))
	select {
	case <-tk.C():
		// expected — first tick fired.
	case <-time.After(100 * time.Millisecond):
		t.Fatal("SetNow forward did not fire ticker")
	}
}
