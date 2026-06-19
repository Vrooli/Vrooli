package safety

import (
	"sync"
	"time"
)

// nowFunc is the time source (injectable for deterministic tests).
type nowFunc func() time.Time

// RateLimiter is a simple sliding-window abuse throttle for AI submits on the
// public tier. It is process-local (the gate is a soft abuse cap, not a
// distributed quota) and thread-safe. A limit of 0 means unlimited.
type RateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	events []time.Time
	now    nowFunc
}

// NewRateLimiter builds a limiter allowing limitPerMin events per rolling minute.
// limitPerMin <= 0 disables limiting (every Allow returns true).
func NewRateLimiter(limitPerMin int) *RateLimiter {
	return &RateLimiter{
		limit:  limitPerMin,
		window: time.Minute,
		now:    time.Now,
	}
}

// withClock overrides the time source (tests).
func (r *RateLimiter) withClock(fn nowFunc) *RateLimiter {
	r.now = fn
	return r
}

// Allow reports whether one more event fits within the window, recording it when
// it does. When the limiter is disabled (limit <= 0) it always allows.
func (r *RateLimiter) Allow() bool {
	if r == nil || r.limit <= 0 {
		return true
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := r.now().Add(-r.window)
	kept := r.events[:0]
	for _, t := range r.events {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	r.events = kept

	if len(r.events) >= r.limit {
		return false
	}
	r.events = append(r.events, r.now())
	return true
}
