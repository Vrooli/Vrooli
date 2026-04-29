// Package mocks — FakeClock and FakeTicker.
//
// FakeClock is the testutil counterpart to clock.System. It implements
// the same Clock interface but advances only when the test calls
// Advance or SetNow. Sleep is implemented as Advance, so production
// code that polls in `for clock.Now().Before(deadline) { clock.Sleep(...) }`
// loops terminates after one iteration in a test instead of spinning
// against a stuck clock.
//
// Tickers returned from FakeClock.NewTicker fire on Advance: every
// time the fake clock advances past the ticker's next deadline, one
// tick is delivered (multiple deadlines passed in a single Advance
// produce multiple back-to-back ticks).
package mocks

import (
	"sync"
	"time"

	"workspace-sandbox/internal/clock"
)

// FakeClock is a Clock whose Now() advances only when the test
// explicitly calls Advance or SetNow.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	tickers []*FakeTicker
}

// NewFakeClock constructs a FakeClock at start. If start is the zero
// value, a stable default (2026-01-01T00:00:00Z) is used so tests
// that don't care about the absolute time don't have to set one.
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

// Since returns the difference between Now() and t.
func (c *FakeClock) Since(t time.Time) time.Duration {
	return c.Now().Sub(t)
}

// Sleep advances the fake clock by d. Production sleep blocks; the
// fake variant only "blocks" the conceptual deadline. Production code
// that polls (`for now < deadline { clock.Sleep(step) }`) terminates
// after one iteration here instead of stalling on a real wall clock.
func (c *FakeClock) Sleep(d time.Duration) {
	c.Advance(d)
}

// SetNow forces the fake clock to t. Tickers are not retro-fired; if
// SetNow jumps backwards, ticker next-deadlines stay where they were.
func (c *FakeClock) SetNow(t time.Time) {
	c.mu.Lock()
	c.now = t
	tickers := append([]*FakeTicker(nil), c.tickers...)
	c.mu.Unlock()
	for _, ft := range tickers {
		ft.fireUpTo(t)
	}
}

// Advance moves the fake clock forward by d and fires any tickers
// whose next deadline now lies in the past.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	now := c.now
	tickers := append([]*FakeTicker(nil), c.tickers...)
	c.mu.Unlock()
	for _, ft := range tickers {
		ft.fireUpTo(now)
	}
}

// NewTicker returns a FakeTicker that fires on Advance.
func (c *FakeClock) NewTicker(d time.Duration) clock.Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()
	ft := &FakeTicker{
		interval: d,
		next:     c.now.Add(d),
		ch:       make(chan time.Time, 1),
	}
	c.tickers = append(c.tickers, ft)
	return ft
}

// Compile-time guarantee that FakeClock satisfies clock.Clock.
var _ clock.Clock = (*FakeClock)(nil)

// FakeTicker is the test counterpart to time.Ticker. Ticks are
// delivered when the owning FakeClock advances past the ticker's next
// deadline. The channel is buffered to one slot, matching production
// semantics — back-pressure drops ticks rather than blocking Advance.
type FakeTicker struct {
	mu       sync.Mutex
	interval time.Duration
	next     time.Time
	ch       chan time.Time
	stopped  bool
}

// C returns the receive channel.
func (ft *FakeTicker) C() <-chan time.Time { return ft.ch }

// Stop halts the ticker. Idempotent.
func (ft *FakeTicker) Stop() {
	ft.mu.Lock()
	ft.stopped = true
	ft.mu.Unlock()
}

// fireUpTo emits ticks for every deadline at or before now. Drops
// ticks when the buffer is full (matches time.Ticker behavior — the
// runtime delivers at most one queued tick).
func (ft *FakeTicker) fireUpTo(now time.Time) {
	ft.mu.Lock()
	if ft.stopped || ft.interval <= 0 {
		ft.mu.Unlock()
		return
	}
	var firings []time.Time
	for !ft.next.After(now) {
		firings = append(firings, ft.next)
		ft.next = ft.next.Add(ft.interval)
	}
	ft.mu.Unlock()
	for _, t := range firings {
		select {
		case ft.ch <- t:
		default:
			// Drop tick — buffer full.
		}
	}
}
