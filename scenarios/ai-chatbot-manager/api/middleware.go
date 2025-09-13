package main

import (
	"net/http"
	"time"
)

// LoggingMiddleware logs all incoming requests
func LoggingMiddleware(logger *Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a response writer wrapper to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Log request details
		duration := time.Since(start)
		logger.Printf("%s %s %s %d %v",
			r.RemoteAddr,
			r.Method,
			r.RequestURI,
			wrapped.statusCode,
			duration,
		)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// CORSMiddleware handles CORS headers
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization, X-CSRF-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware recovers from panics and returns 500 error
func RecoveryMiddleware(logger *Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RateLimitMiddleware implements basic rate limiting
type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Use IP address as key
		key := r.RemoteAddr

		now := time.Now()
		// Clean old requests
		if timestamps, exists := rl.requests[key]; exists {
			var valid []time.Time
			for _, t := range timestamps {
				if now.Sub(t) < rl.window {
					valid = append(valid, t)
				}
			}
			rl.requests[key] = valid
		}

		// Check rate limit
		if len(rl.requests[key]) >= rl.limit {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Add current request
		rl.requests[key] = append(rl.requests[key], now)

		next.ServeHTTP(w, r)
	})
}

// ContentTypeMiddleware ensures JSON content type for API endpoints
func ContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set default content type for API responses
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}