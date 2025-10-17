package main

import (
	"encoding/json"
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int           // requests per interval
	interval time.Duration // time interval
}

type visitor struct {
	tokens    int
	lastVisit time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		interval: interval,
	}

	// Clean up old visitors every minute
	go rl.cleanupVisitors()

	return rl
}

// cleanupVisitors removes old entries to prevent memory leak
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastVisit) > rl.interval*2 {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

// getVisitor retrieves or creates a visitor entry
func (rl *RateLimiter) getVisitor(ip string) *visitor {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	if !exists {
		v = &visitor{
			tokens:    rl.rate,
			lastVisit: time.Now(),
		}
		rl.visitors[ip] = v
	}

	// Refill tokens based on time passed
	timePassed := time.Since(v.lastVisit)
	tokensToAdd := int(timePassed / rl.interval * time.Duration(rl.rate))
	v.tokens = min(rl.rate, v.tokens+tokensToAdd)
	v.lastVisit = time.Now()

	return v
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(ip string) bool {
	v := rl.getVisitor(ip)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// Middleware creates an HTTP middleware for rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract IP address
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		// Check X-Forwarded-For header for proxy scenarios
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			ip = xff
		}

		// Check rate limit
		if !rl.Allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-RateLimit-Limit", string(rune(rl.rate)))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", time.Now().Add(rl.interval).Format(time.RFC3339))
			w.WriteHeader(http.StatusTooManyRequests)

			errorResponse := map[string]interface{}{
				"error": APIError{
					Code:      ErrorCodeRateLimit,
					Message:   "Rate limit exceeded. Please try again later.",
					Timestamp: time.Now().Format(time.RFC3339),
					Path:      r.URL.Path,
				},
			}
			json.NewEncoder(w).Encode(errorResponse)
			return
		}

		next.ServeHTTP(w, r)
	})
}
