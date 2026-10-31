package clock

import (
	"testing"
	"time"
)

func TestFakeSetAdvanceAndUTC(t *testing.T) {
	start := time.Date(2026, 9, 6, 1, 2, 3, 0, time.FixedZone("test", -5*60*60))
	fake := NewFake(start)
	if got := fake.Now(); !got.Equal(start.UTC()) || got.Location() != time.UTC {
		t.Fatalf("initial clock = %v (%s), want UTC %v", got, got.Location(), start.UTC())
	}
	fake.Advance(90 * time.Second)
	want := start.UTC().Add(90 * time.Second)
	if got := fake.Now(); !got.Equal(want) {
		t.Fatalf("advanced clock = %v, want %v", got, want)
	}
	fake.Set(start.Add(time.Hour))
	if got := fake.Now(); !got.Equal(start.Add(time.Hour).UTC()) {
		t.Fatalf("set clock = %v", got)
	}
}
