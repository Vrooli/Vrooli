package middleware

import (
	"fmt"
	"log"
	"net/http"
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

// responseWriter wraps http.ResponseWriter to capture status code
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

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.ResponseWriter.WriteHeader(code)
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
func Logging(logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
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
			logger.Printf(">>> REQUEST: %s %s from %s", r.Method, r.URL.Path, clientIP)
			
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
			
			// Log based on status code
			if rw.statusCode >= 400 {
				entry.ErrorMessage = fmt.Sprintf("Request failed with status %d", rw.statusCode)
				logger.Printf("<<< ERROR RESPONSE: %s %s - Status: %d - Duration: %v",
					r.Method, r.URL.Path, rw.statusCode, duration)
			} else {
				logger.Printf("<<< RESPONSE: %s %s - Status: %d - Duration: %v",
					r.Method, r.URL.Path, rw.statusCode, duration)
			}
		})
	}
}

// DetailedLogging provides more comprehensive logging
func DetailedLogging(logger *log.Logger, logStore *LogStore) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
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
			
			// Get or generate request ID
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
				r.Header.Set("X-Request-ID", requestID)
			}
			
			// Set request ID in response
			rw.Header().Set("X-Request-ID", requestID)
			
			// Log incoming request with headers
			logger.Printf(">>> INCOMING REQUEST: [%s] %s %s from %s", requestID, r.Method, r.URL.Path, clientIP)
			logger.Printf("    Headers: %+v", r.Header)
			logger.Printf("    User-Agent: %s", r.UserAgent())
			
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
			logLevel := "INFO"
			if rw.statusCode >= 400 {
				logLevel = "ERROR"
				entry.ErrorMessage = fmt.Sprintf("Request failed with status %d", rw.statusCode)
			}
			
			logger.Printf("<<< RESPONSE: [%s] %s %s %s - Status: %d - Duration: %v",
				requestID, logLevel, r.Method, r.URL.Path, rw.statusCode, duration)
		})
	}
}

// LogStore stores log entries in memory
type LogStore struct {
	entries  []LogEntry
	maxSize  int
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
	if len(ls.entries) >= ls.maxSize {
		// Remove oldest entry
		ls.entries = ls.entries[1:]
	}
	ls.entries = append(ls.entries, entry)
}

// GetAll returns all stored log entries
func (ls *LogStore) GetAll() []LogEntry {
	return ls.entries
}

// GetRecent returns the most recent n log entries
func (ls *LogStore) GetRecent(n int) []LogEntry {
	if n > len(ls.entries) {
		return ls.entries
	}
	return ls.entries[len(ls.entries)-n:]
}

// Clear removes all log entries
func (ls *LogStore) Clear() {
	ls.entries = ls.entries[:0]
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().Unix(), time.Now().Nanosecond())
}