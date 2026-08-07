// Package mocks holds the canonical hand-written fakes for every seam
// the production code declares. One file per interface; tests arrange via
// struct-field mutation rather than chained builder methods.
package mocks

import (
	"sync"
	"time"

	"vrooli-memory/internal/clock"
)

// FakeClock is the testutil counterpart to clock.System. It implements
// the same Clock interface but advances only when the test calls
// Advance or SetNow — production wall-clock drift never enters tests
// using it.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	tickers []*FakeTicker
}

// FakeTicker is manually driven by tests; it never sleeps or advances itself.
type FakeTicker struct{ ch chan time.Time }

func (t *FakeTicker) C() <-chan time.Time { return t.ch }
func (t *FakeTicker) Stop()               {}
func (t *FakeTicker) Fire(at time.Time)   { t.ch <- at }

// NewFakeClock constructs a FakeClock at start. If start is the zero
// value, a stable default (2026-01-01T00:00:00Z) is used so tests that
// don't care about the absolute time don't have to set one.
func NewFakeClock(start time.Time) *FakeClock {
	if start.IsZero() {
		start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return &FakeClock{now: start}
}

// Now returns the current fake time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the fake clock forward by d.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
}

// SetNow forces the fake clock to t.
func (c *FakeClock) SetNow(t time.Time) {
	c.mu.Lock()
	c.now = t
	c.mu.Unlock()
}

// NewTicker returns a manually-driven ticker and records it for tests that
// need to fire the scheduled loop without waiting on wall time.
func (c *FakeClock) NewTicker(_ time.Duration) clock.Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &FakeTicker{ch: make(chan time.Time, 1)}
	c.tickers = append(c.tickers, t)
	return t
}

// LastTicker returns the most recently created ticker.
func (c *FakeClock) LastTicker() *FakeTicker {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.tickers) == 0 {
		return nil
	}
	return c.tickers[len(c.tickers)-1]
}

// Compile-time guarantee that FakeClock satisfies clock.Clock.
var _ clock.Clock = (*FakeClock)(nil)
