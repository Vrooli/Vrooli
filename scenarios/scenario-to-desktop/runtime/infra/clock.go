// Package infra provides infrastructure abstractions for testing and external integration.
package infra

import "time"

// Clock abstracts time operations for testing.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// Sleep pauses execution for the given duration.
	Sleep(d time.Duration)
	// After returns a channel that receives the current time after duration d.
	After(d time.Duration) <-chan time.Time
	// NewTicker returns a new Ticker that sends the current time on its channel
	// after each tick of duration d.
	NewTicker(d time.Duration) Ticker
}

// Ticker abstracts time.Ticker for testing.
type Ticker interface {
	// C returns the channel on which ticks are delivered.
	C() <-chan time.Time
	// Stop stops the ticker.
	Stop()
}

// RealClock implements Clock using the standard time package.
type RealClock struct{}

// Now returns the current time.
func (RealClock) Now() time.Time {
	return time.Now()
}

// Sleep pauses execution for the given duration.
func (RealClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// After returns a channel that receives the current time after duration d.
func (RealClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

// NewTicker returns a new Ticker that sends the current time on its channel.
func (RealClock) NewTicker(d time.Duration) Ticker {
	return &realTicker{ticker: time.NewTicker(d)}
}

// realTicker wraps time.Ticker to implement Ticker interface.
type realTicker struct {
	ticker *time.Ticker
}

func (t *realTicker) C() <-chan time.Time {
	return t.ticker.C
}

func (t *realTicker) Stop() {
	t.ticker.Stop()
}

// Ensure RealClock implements Clock.
var _ Clock = RealClock{}
