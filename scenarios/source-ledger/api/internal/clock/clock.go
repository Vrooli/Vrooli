// Package clock is the canonical seam for wall-clock-dependent code.
//
// Production wires clock.System{} once in main.go and threads it through
// every constructor that reads time. Tests use testutil/mocks.FakeClock
// to assert exact times and durations without sleeping.
package clock

import "time"

// Clock abstracts wall-clock primitives.
type Clock interface {
	Now() time.Time
	NewTicker(time.Duration) Ticker
}

type Ticker interface {
	C() <-chan time.Time
	Stop()
}

// System is the production Clock; methods delegate to the equivalent
// time.* primitives. Constructed once in main.go.
type System struct{}

// Now reports the current local time.
func (System) Now() time.Time { return time.Now() }

type systemTicker struct{ *time.Ticker }

func (t systemTicker) C() <-chan time.Time { return t.Ticker.C }

func (System) NewTicker(d time.Duration) Ticker { return systemTicker{time.NewTicker(d)} }

// Compile-time guarantee that System satisfies Clock.
var _ Clock = System{}
