package services

import (
	"sync"
	"time"
)

// Clock abstracts time operations for testability.
//
// After is what makes a scheduled loop testable. A loop that calls
// time.After directly can only be exercised by sleeping for its real
// interval, which is why the schedulers in this package take a Clock and wait
// through it instead.
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	After(d time.Duration) <-chan time.Time
}

// RealClock uses the real system clock.
type RealClock struct{}

func (RealClock) Now() time.Time                         { return time.Now() }
func (RealClock) Since(t time.Time) time.Duration        { return time.Since(t) }
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// StubClock is a test helper that returns a controllable time.
//
// It is safe for concurrent use: a scheduler goroutine waits on After while
// the test goroutine calls Advance.
type StubClock struct {
	mu      sync.Mutex
	current time.Time
	waiters []stubWaiter

	// armed is signalled every time a caller starts waiting through After.
	// Without it a test could Advance before the loop under test has armed its
	// timer, and the advance would be lost — the loop would then wait forever
	// on a deadline that is already in the past.
	armed chan struct{}
}

type stubWaiter struct {
	deadline time.Time
	ch       chan time.Time
}

// NewStubClock creates a StubClock fixed at the given time.
func NewStubClock(t time.Time) *StubClock {
	return &StubClock{current: t, armed: make(chan struct{}, 64)}
}

func (c *StubClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.current
}

func (c *StubClock) Since(t time.Time) time.Duration {
	return c.Now().Sub(t)
}

// After returns a channel that fires once Advance moves the clock to or past
// the deadline. A non-positive duration fires immediately.
func (c *StubClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	ch := make(chan time.Time, 1)
	deadline := c.current.Add(d)
	if d <= 0 {
		ch <- c.current
		return ch
	}
	c.waiters = append(c.waiters, stubWaiter{deadline: deadline, ch: ch})
	c.notifyArmed()
	return ch
}

// notifyArmed records that a waiter is now registered. The caller holds c.mu.
// The send is non-blocking so a clock nobody is synchronising against cannot
// stall.
func (c *StubClock) notifyArmed() {
	select {
	case c.armed <- struct{}{}:
	default:
	}
}

// WaitUntilArmed blocks until a caller is waiting through After, or the
// timeout expires. Tests call this before Advance so the advance cannot race
// ahead of the loop it is meant to drive.
func (c *StubClock) WaitUntilArmed(timeout time.Duration) bool {
	select {
	case <-c.armed:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Advance moves the clock forward by d and fires every waiter whose deadline
// has now passed.
func (c *StubClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.current = c.current.Add(d)
	now := c.current

	remaining := c.waiters[:0]
	fired := make([]stubWaiter, 0, len(c.waiters))
	for _, w := range c.waiters {
		if !w.deadline.After(now) {
			fired = append(fired, w)
			continue
		}
		remaining = append(remaining, w)
	}
	c.waiters = remaining
	c.mu.Unlock()

	for _, w := range fired {
		w.ch <- now
	}
}

// Set sets the clock to t without firing waiters.
func (c *StubClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.current = t
}
