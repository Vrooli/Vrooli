package schedule

import (
	"sync"
	"time"
)

// Fake is a deterministic Clock for tests. Time moves only when the test
// explicitly calls Advance, Sleep, or Tick; no goroutine is started.
type Fake struct {
	mu      sync.Mutex
	now     time.Time
	timers  map[*fakeTimer]struct{}
	tickers map[*fakeTicker]struct{}
}

// NewFake returns a controllable clock starting at start.
func NewFake(start time.Time) *Fake {
	return &Fake{now: start, timers: make(map[*fakeTimer]struct{}), tickers: make(map[*fakeTicker]struct{})}
}

func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

func (f *Fake) Advance(d time.Duration) {
	f.mu.Lock()
	f.now = f.now.Add(d)
	f.fireTimersLocked()
	f.mu.Unlock()
}

func (f *Fake) Sleep(d time.Duration) { f.Advance(d) }

// Tick delivers one timestamp to every live ticker. Its channels are buffered
// so a test may tick before the scheduler receives.
func (f *Fake) Tick() {
	f.mu.Lock()
	defer f.mu.Unlock()
	for ticker := range f.tickers {
		if !ticker.stopped {
			select {
			case ticker.ch <- f.now:
			default:
			}
		}
	}
}

func (f *Fake) NewTicker(_ time.Duration) Ticker {
	f.mu.Lock()
	defer f.mu.Unlock()
	ticker := &fakeTicker{owner: f, ch: make(chan time.Time, 1)}
	f.tickers[ticker] = struct{}{}
	return ticker
}

func (f *Fake) NewTimer(d time.Duration) Timer {
	f.mu.Lock()
	defer f.mu.Unlock()
	timer := &fakeTimer{owner: f, ch: make(chan time.Time, 1), deadline: f.now.Add(d)}
	f.timers[timer] = struct{}{}
	if d <= 0 {
		timer.fireLocked(f.now)
	}
	return timer
}

func (f *Fake) fireTimersLocked() {
	for timer := range f.timers {
		if !timer.stopped && !timer.fired && !timer.deadline.After(f.now) {
			timer.fireLocked(f.now)
		}
	}
}

type fakeTimer struct {
	owner    *Fake
	ch       chan time.Time
	deadline time.Time
	stopped  bool
	fired    bool
}

func (t *fakeTimer) C() <-chan time.Time { return t.ch }

func (t *fakeTimer) Stop() bool {
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	wasActive := !t.stopped && !t.fired
	t.stopped = true
	return wasActive
}

func (t *fakeTimer) Reset(d time.Duration) bool {
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	wasActive := !t.stopped && !t.fired
	t.stopped = false
	t.fired = false
	t.deadline = t.owner.now.Add(d)
	if d <= 0 {
		t.fireLocked(t.owner.now)
	}
	return wasActive
}

func (t *fakeTimer) fireLocked(at time.Time) {
	t.fired = true
	select {
	case t.ch <- at:
	default:
	}
}

type fakeTicker struct {
	owner   *Fake
	ch      chan time.Time
	stopped bool
}

func (t *fakeTicker) C() <-chan time.Time { return t.ch }

func (t *fakeTicker) Stop() {
	t.owner.mu.Lock()
	t.stopped = true
	delete(t.owner.tickers, t)
	t.owner.mu.Unlock()
}

func (t *fakeTicker) Reset(_ time.Duration) {
	t.owner.mu.Lock()
	t.stopped = false
	t.owner.tickers[t] = struct{}{}
	t.owner.mu.Unlock()
}

var _ Clock = (*Fake)(nil)
