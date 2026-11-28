package server

import (
	"log/slog"
	"net/http"
	"os"
	"time"
)

// Logger is the narrow logging seam used by the HTTP server. Keeping logging
// injectable makes middleware tests deterministic and prevents production
// handlers from reaching for a process-global logger.
type Logger interface {
	Info(string, ...any)
}

func NewProcessLogger() Logger {
	return slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{}))
}

type discardLogger struct{}

func (discardLogger) Info(string, ...any) {}

// LoggingMiddleware logs HTTP requests in structured format.
func LoggingMiddleware(next http.Handler) http.Handler {
	return LoggingMiddlewareWithLogger(NewProcessLogger(), next)
}

func LoggingMiddlewareWithLogger(logger Logger, next http.Handler) http.Handler {
	if logger == nil {
		logger = discardLogger{}
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logStructuredWith(logger, "request", map[string]interface{}{
			"method":   r.Method,
			"path":     r.RequestURI,
			"duration": time.Since(start).String(),
		})
	})
}

// SecurityHeadersMiddleware applies the baseline browser-facing policy to
// every API and Connect response. The UI is served by its own origin, so the
// API can keep a restrictive default policy while still allowing same-origin
// embedding when the operator shell requires it.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'self'")
		next.ServeHTTP(w, r)
	})
}

// LogStructured outputs logs in a structured JSON-like format for better observability.
func LogStructured(msg string, fields map[string]interface{}) {
	logStructuredWith(NewProcessLogger(), msg, fields)
}

func LogStructuredWith(logger Logger, msg string, fields map[string]interface{}) {
	logStructuredWith(logger, msg, fields)
}

func logStructuredWith(logger Logger, msg string, fields map[string]interface{}) {
	if logger == nil {
		logger = discardLogger{}
	}
	logger.Info(msg, "fields", fields)
}
