package main

import (
	"net/http"
	"sync"
	"time"
)

// timeProvider is a function that returns the current time.
// Used for testing to control time progression.
type timeProvider func() time.Time

// RateLimiter implements a sliding window rate limiter.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*rateBucket
	rate    int           // Max requests per window
	window  time.Duration // Time window
	now     timeProvider  // Injectable time function for testing
}

// rateBucket tracks requests for a single key.
type rateBucket struct {
	timestamps []time.Time
	lastClean  time.Time
}

// NewRateLimiter creates a new rate limiter.
// rate is the maximum number of requests allowed per window.
func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*rateBucket),
		rate:    rate,
		window:  window,
		now:     time.Now,
	}
}

// UseTimeProvider sets a custom time provider for testing.
// This follows the Use*() injection pattern for test seams.
func (rl *RateLimiter) UseTimeProvider(provider timeProvider) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if provider == nil {
		rl.now = time.Now
		return
	}
	rl.now = provider
}

// Allow checks if a request for the given key should be allowed.
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	bucket, exists := rl.buckets[key]
	if !exists {
		bucket = &rateBucket{
			timestamps: make([]time.Time, 0, rl.rate),
			lastClean:  now,
		}
		rl.buckets[key] = bucket
	}

	// Clean old timestamps
	cutoff := now.Add(-rl.window)
	validTimestamps := make([]time.Time, 0, len(bucket.timestamps))
	for _, ts := range bucket.timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}
	bucket.timestamps = validTimestamps

	// Check if under rate limit
	if len(bucket.timestamps) >= rl.rate {
		return false
	}

	// Allow and record this request
	bucket.timestamps = append(bucket.timestamps, now)
	return true
}

// Remaining returns the number of requests remaining for the given key.
func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, exists := rl.buckets[key]
	if !exists {
		return rl.rate
	}

	// Clean old timestamps
	now := rl.now()
	cutoff := now.Add(-rl.window)
	count := 0
	for _, ts := range bucket.timestamps {
		if ts.After(cutoff) {
			count++
		}
	}

	remaining := rl.rate - count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// RateLimitKeyFunc extracts a rate limiting key from a request.
type RateLimitKeyFunc func(*http.Request) string

// Middleware returns HTTP middleware that enforces the rate limit.
// keyFunc extracts the key to rate limit by (e.g., IP address, email, etc.).
func (rl *RateLimiter) Middleware(keyFunc RateLimitKeyFunc) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if key == "" {
				// No key to rate limit by, allow request
				next(w, r)
				return
			}

			if !rl.Allow(key) {
				remaining := rl.Remaining(key)
				logStructured("rate_limit_exceeded", map[string]interface{}{
					"level":     "warn",
					"key":       key,
					"remaining": remaining,
					"path":      r.URL.Path,
				})
				writeJSONError(w, http.StatusTooManyRequests,
					"Too many requests. Please try again later.",
					ApiErrorTypeRateLimited)
				return
			}

			next(w, r)
		}
	}
}

// IPKeyFunc returns a key function that uses the client's IP address.
func IPKeyFunc() RateLimitKeyFunc {
	return func(r *http.Request) string {
		// Check X-Forwarded-For header first (for proxied requests)
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first IP in the chain
			for i := 0; i < len(xff); i++ {
				if xff[i] == ',' {
					return xff[:i]
				}
			}
			return xff
		}

		// Check X-Real-IP header
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}

		// Fall back to RemoteAddr
		addr := r.RemoteAddr
		// Strip port if present
		for i := len(addr) - 1; i >= 0; i-- {
			if addr[i] == ':' {
				return addr[:i]
			}
		}
		return addr
	}
}

// CleanupOldBuckets removes buckets that haven't been accessed recently.
// Call this periodically to prevent memory leaks.
func (rl *RateLimiter) CleanupOldBuckets(maxAge time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	cutoff := now.Add(-maxAge)

	for key, bucket := range rl.buckets {
		// If no recent timestamps and last clean was before cutoff, remove
		if len(bucket.timestamps) == 0 && bucket.lastClean.Before(cutoff) {
			delete(rl.buckets, key)
		}
	}
}
