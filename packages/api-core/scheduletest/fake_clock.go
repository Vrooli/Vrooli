// Package scheduletest provides deterministic implementations of
// api-core/schedule for package tests.
package scheduletest

import (
	"sync"
	"time"

	"github.com/vrooli/api-core/schedule"
)

var _ schedule.Clock = (*FakeClock)(nil)

// FakeClock advances only when Advance is called. Timers and tickers are
// delivered synchronously from Advance, making tests deterministic and fast.
type FakeClock struct {
	mu        sync.Mutex
	now       time.Time
	timers    map[*fakeTimer]struct{}
	tickers   map[*fakeTicker]struct{}
	immediate bool
	onTimer   func(time.Duration)
}

func New(start time.Time) *FakeClock {
	if start.IsZero() {
		start = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	return &FakeClock{now: start, timers: make(map[*fakeTimer]struct{}), tickers: make(map[*fakeTicker]struct{})}
}

// NewImmediate creates a fake clock whose timers fire as soon as they are
// created. It is useful for retry tests: the callback observes the requested
// delay while the operation remains deterministic and does not need a second
// goroutine to advance the clock.
func NewImmediate(start time.Time, onTimer func(time.Duration)) *FakeClock {
	c := New(start)
	c.immediate = true
	c.onTimer = onTimer
	return c
}

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	now := c.now
	for timer := range c.timers {
		if !timer.stopped && !now.Before(timer.deadline) {
			timer.stopped = true
			timer.ch <- now
			delete(c.timers, timer)
		}
	}
	for ticker := range c.tickers {
		if ticker.stopped || ticker.interval <= 0 {
			continue
		}
		for !now.Before(ticker.next) {
			ticker.ch <- ticker.next
			ticker.next = ticker.next.Add(ticker.interval)
		}
	}
	c.mu.Unlock()
}

// SetNow replaces the current fake time without advancing scheduled work.
// Tests that need to model a wall-clock correction should use this method;
// elapsed-time code should continue to use Advance so monotonic progression
// remains explicit.
func (c *FakeClock) SetNow(now time.Time) {
	c.mu.Lock()
	c.now = now
	c.mu.Unlock()
}

func (c *FakeClock) Sleep(d time.Duration) { c.Advance(d) }

func (c *FakeClock) NewTimer(d time.Duration) schedule.Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{clock: c, ch: make(chan time.Time, 1), deadline: c.now.Add(d)}
	if c.immediate {
		if c.onTimer != nil {
			c.onTimer(d)
		}
		t.stopped = true
		t.ch <- c.now
		return t
	}
	c.timers[t] = struct{}{}
	return t
}

func (c *FakeClock) NewTicker(d time.Duration) schedule.Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTicker{clock: c, ch: make(chan time.Time, 32), interval: d, next: c.now.Add(d)}
	c.tickers[t] = struct{}{}
	return t
}

type fakeTimer struct {
	clock    *FakeClock
	ch       chan time.Time
	deadline time.Time
	stopped  bool
}

func (t *fakeTimer) C() <-chan time.Time { return t.ch }

func (t *fakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	wasActive := !t.stopped
	t.stopped = true
	delete(t.clock.timers, t)
	return wasActive
}

func (t *fakeTimer) Reset(d time.Duration) bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	wasActive := !t.stopped
	t.stopped = false
	t.deadline = t.clock.now.Add(d)
	t.clock.timers[t] = struct{}{}
	return wasActive
}

type fakeTicker struct {
	clock    *FakeClock
	ch       chan time.Time
	interval time.Duration
	next     time.Time
	stopped  bool
}

func (t *fakeTicker) C() <-chan time.Time { return t.ch }

func (t *fakeTicker) Stop() {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	t.stopped = true
	delete(t.clock.tickers, t)
}

func (t *fakeTicker) Reset(d time.Duration) {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	t.interval = d
	t.next = t.clock.now.Add(d)
	t.stopped = false
	t.clock.tickers[t] = struct{}{}
}

// FakeTicker is the deterministic ticker returned by FakeClock. It is
// exported so tests that need a manually-triggered maintenance tick can use
// LastTicker and Fire without depending on implementation details.
type FakeTicker struct{ *fakeTicker }

func (c *FakeClock) LastTicker() *FakeTicker {
	c.mu.Lock()
	defer c.mu.Unlock()
	for ticker := range c.tickers {
		return &FakeTicker{ticker}
	}
	return nil
}

func (t *FakeTicker) Fire(at time.Time) { t.ch <- at }
