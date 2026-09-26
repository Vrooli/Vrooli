package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp    time.Time `json:"timestamp"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	StatusCode   int       `json:"status_code"`
	Duration     string    `json:"duration"`
	ClientIP     string    `json:"client_ip"`
	UserAgent    string    `json:"user_agent"`
	RequestID    string    `json:"request_id,omitempty"`
	ErrorMessage string    `json:"error,omitempty"`
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

// NewResponseWriter creates a new response writer wrapper
func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader captures the status code and, as the universal response-writer
// wrapper applied to every route, stamps a secure header floor on the way out.
// This is the catch-all enforcement point so even handlers that write directly
// to the ResponseWriter still emit OWASP security headers.
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		w := rw.ResponseWriter
		if w.Header().Get("X-Frame-Options") == "" {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
		rw.statusCode = code
		w.WriteHeader(code)
		rw.written = true
	}
}

// Write writes the response body
func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// Logging middleware logs HTTP requests and responses
func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create wrapped response writer
			rw := NewResponseWriter(w)

			// Get client IP
			clientIP := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = forwarded
			}

			// Get request ID if present
			requestID := r.Header.Get("X-Request-ID")

			// Log incoming request
			logger.Info("incoming request",
				"method", r.Method,
				"path", r.URL.Path,
				"client_ip", clientIP,
				"request_id", requestID,
			)

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Log based on status code
			if rw.statusCode >= 400 {
				logger.Warn("request completed",
					"method", r.Method,
					"path", r.URL.Path,
					"status", rw.statusCode,
					"duration", duration,
					"client_ip", clientIP,
					"request_id", requestID,
				)
			} else {
				logger.Info("request completed",
					"method", r.Method,
					"path", r.URL.Path,
					"status", rw.statusCode,
					"duration", duration,
					"client_ip", clientIP,
					"request_id", requestID,
				)
			}
		})
	}
}

// DetailedLogging provides more comprehensive logging
func DetailedLogging(logger *slog.Logger, logStore *LogStore) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create wrapped response writer
			rw := NewResponseWriter(w)

			// Get client IP
			clientIP := r.RemoteAddr
			if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
				clientIP = forwarded
			}

			// Get request ID (set by RequestID middleware)
			requestID := r.Header.Get("X-Request-ID")

			// Set request ID in response
			rw.Header().Set("X-Request-ID", requestID)

			// Log incoming request with headers
			logger.Info("incoming request",
				"request_id", requestID,
				"method", r.Method,
				"path", r.URL.Path,
				"client_ip", clientIP,
				"user_agent", r.UserAgent(),
			)

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Create log entry
			entry := LogEntry{
				Timestamp:  start,
				Method:     r.Method,
				Path:       r.URL.Path,
				StatusCode: rw.statusCode,
				Duration:   duration.String(),
				ClientIP:   clientIP,
				UserAgent:  r.UserAgent(),
				RequestID:  requestID,
			}

			// Store log entry if log store is provided
			if logStore != nil {
				logStore.Add(entry)
			}

			// Log response
			if rw.statusCode >= 400 {
				entry.ErrorMessage = fmt.Sprintf("Request failed with status %d", rw.statusCode)
				logger.Warn("request completed",
					"request_id", requestID,
					"method", r.Method,
					"path", r.URL.Path,
					"status", rw.statusCode,
					"duration", duration,
				)
			} else {
				logger.Info("request completed",
					"request_id", requestID,
					"method", r.Method,
					"path", r.URL.Path,
					"status", rw.statusCode,
					"duration", duration,
				)
			}
		})
	}
}

// LogStore stores log entries in memory
type LogStore struct {
	mu      sync.Mutex
	entries []LogEntry
	maxSize int
}

// NewLogStore creates a new log store
func NewLogStore(maxSize int) *LogStore {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &LogStore{
		entries: make([]LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a log entry to the store
func (ls *LogStore) Add(entry LogEntry) {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if len(ls.entries) >= ls.maxSize {
		// Remove oldest entry
		ls.entries = ls.entries[1:]
	}
	ls.entries = append(ls.entries, entry)
}

// GetAll returns all stored log entries
func (ls *LogStore) GetAll() []LogEntry {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	result := make([]LogEntry, len(ls.entries))
	copy(result, ls.entries)
	return result
}

// GetRecent returns the most recent n log entries
func (ls *LogStore) GetRecent(n int) []LogEntry {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	if n > len(ls.entries) {
		n = len(ls.entries)
	}
	result := make([]LogEntry, n)
	copy(result, ls.entries[len(ls.entries)-n:])
	return result
}

// Clear removes all log entries
func (ls *LogStore) Clear() {
	ls.mu.Lock()
	defer ls.mu.Unlock()
	ls.entries = ls.entries[:0]
}
