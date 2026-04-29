package clock

import (
	"testing"
	"time"
)

func TestSystemNow_Monotonic(t *testing.T) {
	c := System{}
	a := c.Now()
	time.Sleep(2 * time.Millisecond)
	b := c.Now()
	if !b.After(a) {
		t.Fatalf("System.Now() did not advance: a=%s b=%s", a, b)
	}
}

func TestSystemSince_RoughlyAccurate(t *testing.T) {
	c := System{}
	start := c.Now()
	time.Sleep(5 * time.Millisecond)
	got := c.Since(start)
	if got < 5*time.Millisecond {
		t.Fatalf("System.Since() too small: %v", got)
	}
}

func TestSystemTicker_FiresAndStops(t *testing.T) {
	c := System{}
	tk := c.NewTicker(2 * time.Millisecond)
	defer tk.Stop()
	select {
	case <-tk.C():
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("ticker did not fire within 100ms")
	}
}
