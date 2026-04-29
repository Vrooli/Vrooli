// Package clock is the canonical seam for wall-clock-dependent code in
// workspace-sandbox.
//
// Every production-side caller that previously reached for `time.Now()`,
// `time.Since`, `time.Sleep`, or `time.NewTicker` now consumes a Clock
// argument instead. The Round 4 plan calls this out explicitly: the
// goal is that every time-dependent behavior (idle TTLs, exponential
// backoff, daemon-reaper grace windows, log rotation, manual-review
// expiry) is reachable from `go test ./...` in a deterministic way,
// not by sleeping in a test or hoping the wall clock cooperated.
//
// # Why an interface, not a function
//
// A common alternative is to expose a `now func() time.Time` field. We
// rejected it because:
//
//  1. Tickers — `time.NewTicker` cannot be mocked through a `now`
//     callback. The reconciler runner and any future periodic loop need
//     a fake-driven ticker, which forces a method on the Clock anyway.
//  2. Sleep is a concrete operation many tracker/reaper paths call;
//     making it an interface method keeps deterministic-time code from
//     real-walltime sleeps in tests.
//  3. A single `Clock` type travels through constructors more cleanly
//     than three function fields (`now`, `sleep`, `tick`).
//
// # Production wiring
//
// `main.go` constructs `clock.System{}` once and threads it through
// every constructor that needs time. There is no default fallback in
// any constructor: passing nil panics in the production wiring path
// (Round 4 greenfield rule — no "if nil { default }" hidden seams).
//
// # Test wiring
//
// Tests use `testutil/mocks.FakeClock`, which lives next to the other
// hand-written fakes. `FakeClock.Advance(d)` moves time forward and
// fires any tickers whose next deadline has passed. `FakeClock.Sleep`
// is implemented as Advance, so production code that polls in a
// `for time.Now().Before(deadline) { time.Sleep(...) }` loop terminates
// after one iteration in tests instead of spinning on a stuck clock.
package clock

import "time"

// Clock abstracts wall-clock primitives. It is the dependency every
// time-dependent component in workspace-sandbox declares; the
// production implementation is `System{}` and the test implementation
// is `testutil/mocks.FakeClock`.
type Clock interface {
	// Now returns the current time.
	Now() time.Time

	// Since returns the elapsed time since t (`Now().Sub(t)`).
	Since(t time.Time) time.Duration

	// Sleep blocks for at least d. Production sleeps the real wall
	// clock; FakeClock.Sleep advances the fake clock instead so test
	// loops terminate.
	Sleep(d time.Duration)

	// NewTicker returns a Ticker that fires at the given interval. The
	// returned channel is buffered to one slot, matching the semantics
	// of time.NewTicker.
	NewTicker(d time.Duration) Ticker
}

// Ticker is the return shape of Clock.NewTicker. It mirrors the
// receive-channel + Stop method of time.Ticker but is interface-typed
// so a fake implementation can drive ticks from Advance instead of
// real time.
type Ticker interface {
	// C returns the channel on which ticks are delivered.
	C() <-chan time.Time

	// Stop halts the ticker. Idempotent: calling Stop twice is safe.
	Stop()
}

// System is the production Clock. Methods delegate to the equivalent
// time.* primitives. Constructed once in main.go and threaded
// throughout the dependency graph.
type System struct{}

// Now reports the current local time.
func (System) Now() time.Time { return time.Now() }

// Since is shorthand for time.Since(t).
func (System) Since(t time.Time) time.Duration { return time.Since(t) }

// Sleep delegates to time.Sleep.
func (System) Sleep(d time.Duration) { time.Sleep(d) }

// NewTicker returns a Ticker backed by a real time.Ticker. The caller
// must Stop the returned Ticker; failing to do so leaks the underlying
// goroutine inside time.Ticker.
func (System) NewTicker(d time.Duration) Ticker {
	return systemTicker{t: time.NewTicker(d)}
}

// systemTicker adapts *time.Ticker to the Ticker interface.
type systemTicker struct{ t *time.Ticker }

func (s systemTicker) C() <-chan time.Time { return s.t.C }
func (s systemTicker) Stop()               { s.t.Stop() }
