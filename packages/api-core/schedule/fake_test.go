package schedule

import (
	"testing"
	"time"
)

func TestFakeNowAdvanceAndSleep(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fake := NewFake(start)
	if got := fake.Now(); !got.Equal(start) {
		t.Fatalf("Now() = %v, want %v", got, start)
	}
	fake.Advance(2 * time.Hour)
	fake.Sleep(30 * time.Minute)
	if got, want := fake.Now(), start.Add(150*time.Minute); !got.Equal(want) {
		t.Fatalf("Now() = %v, want %v", got, want)
	}
}

func TestFakeTickDeliversExactlyOneValuePerCall(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fake := NewFake(start)
	ticker := fake.NewTicker(time.Second)
	fake.Tick()
	if got := <-ticker.C(); !got.Equal(start) {
		t.Fatalf("tick = %v, want %v", got, start)
	}
	select {
	case <-ticker.C():
		t.Fatal("tick delivered more than once")
	default:
	}
	fake.Advance(time.Minute)
	fake.Tick()
	if got := <-ticker.C(); !got.Equal(start.Add(time.Minute)) {
		t.Fatalf("tick = %v, want advanced time", got)
	}
}

func TestFakeTickerStopPreventsDelivery(t *testing.T) {
	fake := NewFake(time.Unix(0, 0))
	ticker := fake.NewTicker(time.Second)
	ticker.Stop()
	fake.Tick()
	select {
	case <-ticker.C():
		t.Fatal("stopped ticker delivered a value")
	default:
	}
}

func TestFakeTimerFiresOnAdvanceAndCanReset(t *testing.T) {
	start := time.Unix(0, 0)
	fake := NewFake(start)
	timer := fake.NewTimer(time.Minute)
	fake.Advance(59 * time.Second)
	select {
	case <-timer.C():
		t.Fatal("timer fired early")
	default:
	}
	fake.Advance(time.Second)
	if got := <-timer.C(); !got.Equal(start.Add(time.Minute)) {
		t.Fatalf("timer = %v", got)
	}
	timer.Reset(time.Minute)
	timer.Stop()
	fake.Advance(time.Minute)
	select {
	case <-timer.C():
		t.Fatal("stopped timer delivered a value")
	default:
	}
}
