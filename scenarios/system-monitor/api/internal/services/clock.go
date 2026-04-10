package services

import "time"

// Clock abstracts time operations for testability.
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
}

// RealClock uses the real system clock.
type RealClock struct{}

func (RealClock) Now() time.Time                  { return time.Now() }
func (RealClock) Since(t time.Time) time.Duration { return time.Since(t) }

// StubClock is a test helper that returns a controllable time.
type StubClock struct {
	current time.Time
}

// NewStubClock creates a StubClock fixed at the given time.
func NewStubClock(t time.Time) *StubClock {
	return &StubClock{current: t}
}

func (c *StubClock) Now() time.Time                  { return c.current }
func (c *StubClock) Since(t time.Time) time.Duration { return c.current.Sub(t) }

// Advance moves the clock forward by d.
func (c *StubClock) Advance(d time.Duration) { c.current = c.current.Add(d) }

// Set sets the clock to t.
func (c *StubClock) Set(t time.Time) { c.current = t }
