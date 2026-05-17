// Package clock is the canonical seam for wall-clock-dependent code.
//
// Production wires clock.System{} once in main.go and threads it through
// every constructor that reads time. Tests use testutil/mocks.FakeClock
// to assert exact times and durations without sleeping.
//
// The interface starts minimal — Now() only — because the template's
// only time-dependent code today is request-duration logging. Scenarios
// that need Sleep, NewTicker, or Since extend this interface; doing so
// is non-breaking as long as System and FakeClock grow the new methods
// together.
package clock

import "time"

// seam: Clock is the wall-clock seam (SEAMS.md row "Clock"). Production
// wires clock.System{}; tests wire mocks.FakeClock.
//
// Clock abstracts wall-clock primitives.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
}

// System is the production Clock; methods delegate to the equivalent
// time.* primitives. Constructed once in main.go.
type System struct{}

// Now reports the current local time.
func (System) Now() time.Time { return time.Now() }

// Compile-time guarantee that System satisfies Clock.
var _ Clock = System{}
