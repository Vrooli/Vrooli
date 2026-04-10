package clock

import (
	"testing"
	"time"
)

// [REQ:LD-TEMPORAL-FLOW] Tests for clock abstraction

func TestReal_Now(t *testing.T) {
	c := Real()
	before := time.Now().UTC()
	now := c.Now()
	after := time.Now().UTC()

	if now.Before(before) || now.After(after) {
		t.Errorf("Real().Now() = %v, expected between %v and %v", now, before, after)
	}
}

func TestReal_Today(t *testing.T) {
	c := Real()
	expected := time.Now().UTC().Format("2006-01-02")
	got := c.Today()

	if got != expected {
		t.Errorf("Real().Today() = %q, expected %q", got, expected)
	}
}

func TestReal_Yesterday(t *testing.T) {
	c := Real()
	expected := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	got := c.Yesterday()

	if got != expected {
		t.Errorf("Real().Yesterday() = %q, expected %q", got, expected)
	}
}

func TestReal_DaysAgo(t *testing.T) {
	c := Real()
	for _, days := range []int{0, 1, 7, 30} {
		expected := time.Now().UTC().AddDate(0, 0, -days).Format("2006-01-02")
		got := c.DaysAgo(days)

		if got != expected {
			t.Errorf("Real().DaysAgo(%d) = %q, expected %q", days, got, expected)
		}
	}
}

func TestFixed_Now(t *testing.T) {
	fixedTime := time.Date(2026, 3, 10, 12, 30, 0, 0, time.UTC)
	c := Fixed(fixedTime)

	got := c.Now()
	if !got.Equal(fixedTime) {
		t.Errorf("Fixed().Now() = %v, expected %v", got, fixedTime)
	}

	// Call again to verify it's always the same
	got2 := c.Now()
	if !got2.Equal(fixedTime) {
		t.Errorf("Fixed().Now() second call = %v, expected %v", got2, fixedTime)
	}
}

func TestFixed_Today(t *testing.T) {
	fixedTime := time.Date(2026, 3, 10, 12, 30, 0, 0, time.UTC)
	c := Fixed(fixedTime)

	expected := "2026-03-10"
	got := c.Today()

	if got != expected {
		t.Errorf("Fixed().Today() = %q, expected %q", got, expected)
	}
}

func TestFixed_Yesterday(t *testing.T) {
	fixedTime := time.Date(2026, 3, 10, 12, 30, 0, 0, time.UTC)
	c := Fixed(fixedTime)

	expected := "2026-03-09"
	got := c.Yesterday()

	if got != expected {
		t.Errorf("Fixed().Yesterday() = %q, expected %q", got, expected)
	}
}

func TestFixed_DaysAgo(t *testing.T) {
	fixedTime := time.Date(2026, 3, 10, 12, 30, 0, 0, time.UTC)
	c := Fixed(fixedTime)

	tests := []struct {
		days     int
		expected string
	}{
		{0, "2026-03-10"},
		{1, "2026-03-09"},
		{7, "2026-03-03"},
		{30, "2026-02-08"},
	}

	for _, tt := range tests {
		got := c.DaysAgo(tt.days)
		if got != tt.expected {
			t.Errorf("Fixed().DaysAgo(%d) = %q, expected %q", tt.days, got, tt.expected)
		}
	}
}

func TestFixed_LocalTimeConvertsToUTC(t *testing.T) {
	// Test that local times are converted to UTC
	loc, _ := time.LoadLocation("America/New_York")
	localTime := time.Date(2026, 3, 10, 12, 30, 0, 0, loc)
	c := Fixed(localTime)

	got := c.Now()
	if got.Location() != time.UTC {
		t.Errorf("Fixed() should return UTC times, got location %v", got.Location())
	}
}
