package main

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	requests map[string]*bucket
	mu       sync.Mutex
	rate     int           // requests per window
	window   time.Duration // time window
}

type bucket struct {
	tokens     int
	lastRefill time.Time
}

// NewRateLimiter creates a new rate limiter
// rate: max requests per window
// window: time window duration
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*bucket),
		rate:     rate,
		window:   window,
	}

	// Cleanup old entries every 5 minutes
	go rl.cleanup()

	return rl
}

// Allow checks if a request from the given identifier should be allowed
func (rl *RateLimiter) Allow(identifier string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.requests[identifier]

	if !exists {
		// First request from this identifier
		rl.requests[identifier] = &bucket{
			tokens:     rl.rate - 1,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(b.lastRefill)
	refillAmount := int(elapsed / rl.window * time.Duration(rl.rate))

	if refillAmount > 0 {
		b.tokens = min(rl.rate, b.tokens+refillAmount)
		b.lastRefill = now
	}

	// Check if we have tokens available
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanup removes stale entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for id, b := range rl.requests {
			if now.Sub(b.lastRefill) > rl.window*2 {
				delete(rl.requests, id)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware creates HTTP middleware for rate limiting
func (rl *RateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use IP address as identifier (in production, use user ID or API key)
		identifier := r.RemoteAddr

		if !rl.Allow(identifier) {
			http.Error(w, `{"error":"rate_limit_exceeded","message":"Too many requests. Please try again later."}`, http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Helper function (Go 1.21+)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
