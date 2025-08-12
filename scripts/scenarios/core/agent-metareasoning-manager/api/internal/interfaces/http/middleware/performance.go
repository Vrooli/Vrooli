package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// PerformanceMiddleware adds timing headers to responses
func PerformanceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap the response writer to capture status
		wrapped := &ResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		durationMs := float64(duration.Nanoseconds()) / 1e6
		
		// Add performance headers
		wrapped.Header().Set("X-Response-Time", fmt.Sprintf("%.2fms", durationMs))
		wrapped.Header().Set("X-Timestamp", strconv.FormatInt(start.Unix(), 10))
		
		// Log slow requests
		if durationMs > 100 {
			log.Printf("SLOW REQUEST: %s %s took %.2fms (status: %d)", 
				r.Method, r.URL.Path, durationMs, wrapped.statusCode)
		}
	})
}

// ResponseWriter wraps http.ResponseWriter to capture status code
type ResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}