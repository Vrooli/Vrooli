// Package clock provides time abstraction for testable code.
// It allows injecting controlled time sources in tests while using
// real time in production.
package clock

import "time"

// Clock provides time operations. Use RealClock for production
// and MockClock for testing.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// RealClock implements Clock using the system clock.
type RealClock struct{}

// Now returns the current system time.
func (RealClock) Now() time.Time {
	return time.Now()
}

// New returns a RealClock instance for production use.
func New() Clock {
	return RealClock{}
}
