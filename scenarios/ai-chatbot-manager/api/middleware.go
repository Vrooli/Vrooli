package main

import (
	"bufio"
	"fmt"
	"net"
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

// Hijack implements the http.Hijacker interface for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, fmt.Errorf("responseWriter doesn't support hijacking")
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

// AuthenticationMiddleware provides API key authentication for protected endpoints
type AuthenticationMiddleware struct {
	apiKeys map[string]bool // Simple API key store
	logger  *Logger
}

// NewAuthenticationMiddleware creates a new authentication middleware
func NewAuthenticationMiddleware(logger *Logger) *AuthenticationMiddleware {
	// In production, these would come from environment variables or a database
	apiKeys := make(map[string]bool)
	// Default API key for development - should be changed in production
	apiKeys["dev-api-key-change-in-production"] = true
	
	return &AuthenticationMiddleware{
		apiKeys: apiKeys,
		logger:  logger,
	}
}

// Middleware is the authentication middleware function
func (am *AuthenticationMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for certain endpoints
		path := r.URL.Path
		if path == "/health" || path == "/api/v1/widget.js" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check for API key in header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			// Check for API key in query parameter (for browser testing)
			apiKey = r.URL.Query().Get("api_key")
		}
		
		// Validate API key
		if apiKey == "" {
			am.logger.Printf("Authentication failed: No API key provided for %s", path)
			http.Error(w, `{"error":"API key required"}`, http.StatusUnauthorized)
			return
		}
		
		if !am.apiKeys[apiKey] {
			am.logger.Printf("Authentication failed: Invalid API key for %s", path)
			http.Error(w, `{"error":"Invalid API key"}`, http.StatusUnauthorized)
			return
		}
		
		// API key is valid, proceed
		next.ServeHTTP(w, r)
	})
}