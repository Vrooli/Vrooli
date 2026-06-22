package graphsweeper

import (
	"sync"
	"time"
)

// Breaker states.
const (
	BreakerClosed   = "closed"
	BreakerOpen     = "open"
	BreakerHalfOpen = "half-open"
)

// Clock is the time seam, keeping the breaker and sweeper deterministic in tests.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// breaker is a small circuit breaker over the upstream ingest pipeline. It opens
// after threshold consecutive failures, stays open for cooldown, then admits a
// single half-open probe before closing on success or re-opening on failure.
type breaker struct {
	mu        sync.Mutex
	threshold int
	cooldown  time.Duration
	clock     Clock

	failures int
	state    string
	openedAt time.Time
}

func newBreaker(threshold int, cooldown time.Duration, clock Clock) *breaker {
	if threshold < 1 {
		threshold = 1
	}
	if clock == nil {
		clock = systemClock{}
	}
	return &breaker{threshold: threshold, cooldown: cooldown, clock: clock, state: BreakerClosed}
}

// Allow reports whether work may proceed, transitioning open→half-open once the
// cooldown has elapsed.
func (b *breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == BreakerOpen {
		if b.clock.Now().Sub(b.openedAt) >= b.cooldown {
			b.state = BreakerHalfOpen
			return true
		}
		return false
	}
	return true
}

// Success resets the breaker to closed.
func (b *breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.state = BreakerClosed
}

// Failure records a failure, opening the breaker once the threshold is reached
// (or immediately when probing in half-open).
func (b *breaker) Failure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.state == BreakerHalfOpen || b.failures >= b.threshold {
		b.state = BreakerOpen
		b.openedAt = b.clock.Now()
	}
}

// State returns the current breaker state.
func (b *breaker) State() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
