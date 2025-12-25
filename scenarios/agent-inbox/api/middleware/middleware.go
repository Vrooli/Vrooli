// Package middleware provides HTTP middleware for the Agent Inbox API.
//
// Middleware Design:
//   - Request IDs enable end-to-end request tracing
//   - Structured logging provides machine-readable logs
//   - CORS is whitelist-based for security
package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Context keys for request metadata.
type contextKey string

const (
	// RequestIDKey is the context key for the request ID.
	RequestIDKey contextKey = "request_id"

	// RequestIDHeader is the HTTP header for request ID propagation.
	RequestIDHeader = "X-Request-ID"
)

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.statusCode = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

// RequestID generates or propagates request IDs for tracing.
// If the client provides X-Request-ID, it is reused; otherwise, a new UUID is generated.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for existing request ID from client
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()[:8] // Short ID for readability
		}

		// Add to context and response header
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)
		w.Header().Set(RequestIDHeader, requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id := ctx.Value(RequestIDKey); id != nil {
		return id.(string)
	}
	return ""
}

// Logging returns middleware that logs each HTTP request with structured fields.
// Log format: [request_id] METHOD /path status_code duration
//
// Signal Design:
//   - All requests are logged with timing information
//   - Error responses (4xx, 5xx) are clearly distinguishable
//   - Request IDs enable log correlation with client errors
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Get request ID
		requestID := GetRequestID(r.Context())

		// Determine log level based on status code
		logLevel := "INFO"
		if wrapped.statusCode >= 500 {
			logLevel = "ERROR"
		} else if wrapped.statusCode >= 400 {
			logLevel = "WARN"
		}

		// Structured log output
		// Format: [level] [request_id] METHOD path status duration
		log.Printf("[%s] [%s] %s %s %d %s",
			logLevel,
			requestID,
			r.Method,
			r.RequestURI,
			wrapped.statusCode,
			duration,
		)
	})
}

// CORS returns middleware that handles Cross-Origin Resource Sharing.
// It allows the UI to make requests to the API from a different origin.
//
// Security Design:
//   - Whitelist-based origin validation (not wildcard)
//   - Configurable via CORS_ALLOWED_ORIGINS environment variable
//   - Defaults to localhost for development safety
func CORS(next http.Handler) http.Handler {
	// Get allowed origins from environment or use localhost defaults
	allowedOriginsStr := os.Getenv("CORS_ALLOWED_ORIGINS")
	if allowedOriginsStr == "" {
		uiPort := os.Getenv("UI_PORT")
		if uiPort == "" {
			uiPort = "35000"
		}
		allowedOriginsStr = fmt.Sprintf("http://localhost:%s,http://127.0.0.1:%s", uiPort, uiPort)
	}

	allowedOrigins := strings.Split(allowedOriginsStr, ",")
	allowedOriginSet := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowedOriginSet[strings.TrimSpace(origin)] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowedOriginSet[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
