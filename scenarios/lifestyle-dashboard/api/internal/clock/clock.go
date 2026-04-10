// Package clock provides a time abstraction for testable time-dependent code.
// This implements the Time Provider Seam documented in docs/internal/SEAMS.md.
//
// Usage:
//
//	// Production code
//	clock := clock.Real()
//	now := clock.Now()
//
//	// Test code
//	clock := clock.Fixed(time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC))
//	now := clock.Now()  // Always returns the fixed time
package clock

import "time"

// Clock provides time operations that can be controlled in tests.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// Today returns today's date in YYYY-MM-DD format.
	Today() string

	// Yesterday returns yesterday's date in YYYY-MM-DD format.
	Yesterday() string

	// DaysAgo returns the date N days ago in YYYY-MM-DD format.
	DaysAgo(n int) string
}

// realClock implements Clock using system time.
type realClock struct{}

// Real returns a Clock that uses system time.
func Real() Clock {
	return realClock{}
}

func (realClock) Now() time.Time {
	return time.Now().UTC()
}

func (realClock) Today() string {
	return time.Now().UTC().Format("2006-01-02")
}

func (realClock) Yesterday() string {
	return time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
}

func (realClock) DaysAgo(n int) string {
	return time.Now().UTC().AddDate(0, 0, -n).Format("2006-01-02")
}

// fixedClock implements Clock with a fixed time for testing.
type fixedClock struct {
	t time.Time
}

// Fixed returns a Clock that always returns the given time.
func Fixed(t time.Time) Clock {
	return fixedClock{t: t.UTC()}
}

func (f fixedClock) Now() time.Time {
	return f.t
}

func (f fixedClock) Today() string {
	return f.t.Format("2006-01-02")
}

func (f fixedClock) Yesterday() string {
	return f.t.AddDate(0, 0, -1).Format("2006-01-02")
}

func (f fixedClock) DaysAgo(n int) string {
	return f.t.AddDate(0, 0, -n).Format("2006-01-02")
}
