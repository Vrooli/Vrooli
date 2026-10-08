package localexchange

import (
	"strings"
	"sync"
	"time"
)

// RateLimiter is a small process-local limiter for the local exchange RPC.
// The socket is intentionally available to every local user, so a principal
// must not be able to turn it into an unbounded token-minting oracle.
type RateLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	entries map[string]bucket
}

type bucket struct {
	started time.Time
	count   int
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	if max <= 0 {
		max = 20
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{max: max, window: window, entries: map[string]bucket{}}
}

func (r *RateLimiter) Allow(principal string, now time.Time) bool {
	principal = strings.TrimSpace(principal)
	if principal == "" || now.IsZero() {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	b := r.entries[principal]
	if b.started.IsZero() || now.Sub(b.started) >= r.window {
		b = bucket{started: now, count: 0}
	}
	if b.count >= r.max {
		r.entries[principal] = b
		return false
	}
	b.count++
	r.entries[principal] = b
	return true
}
