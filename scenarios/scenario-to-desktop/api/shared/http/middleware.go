package http

import (
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/handlers"
)

// CORSConfig holds configuration for CORS middleware.
type CORSConfig struct {
	// AllowedOrigin is the explicit origin to allow (e.g., "https://example.com").
	// If empty, falls back to UIPort-based origins in development.
	AllowedOrigin string
	// UIPort is the development UI port to generate localhost origins.
	// Used only if AllowedOrigin is empty.
	UIPort string
	// Logger is used for warnings. If nil, uses default slog logger.
	Logger *slog.Logger
}

// CORSMiddleware creates a CORS middleware handler with the given configuration.
// It handles preflight OPTIONS requests and sets appropriate CORS headers.
//
// Security note: This middleware is designed for development and controlled production
// environments. The AllowedOrigin or UIPort must be explicitly configured.
func CORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowedOrigin := config.AllowedOrigin

			if allowedOrigin == "" {
				// SECURITY: In development, UI_PORT must be provided by lifecycle system
				// In production, ALLOWED_ORIGIN should be explicitly configured
				if config.UIPort != "" {
					// UIPort is set - use it to build allowed origins
					origin := r.Header.Get("Origin")
					allowedOrigins := []string{
						"http://localhost:" + config.UIPort,
						"http://127.0.0.1:" + config.UIPort,
					}

					// Check if origin matches any allowed origin
					for _, allowed := range allowedOrigins {
						if origin == allowed {
							allowedOrigin = origin
							break
						}
					}

					// If no match found, use primary UI port
					if allowedOrigin == "" {
						allowedOrigin = "http://localhost:" + config.UIPort
					}
				} else {
					// SECURITY: Neither ALLOWED_ORIGIN nor UI_PORT is set
					// This is a configuration error - log and use restrictive CORS
					logger.Warn("CORS configuration missing",
						"message", "Neither ALLOWED_ORIGIN nor UI_PORT is set",
						"action", "using restrictive localhost-only CORS for security")
					// Set to request origin only if it's localhost
					origin := r.Header.Get("Origin")
					if origin != "" && (origin == "http://localhost" || origin == "http://127.0.0.1" ||
						strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:")) {
						allowedOrigin = origin
					} else {
						allowedOrigin = "http://localhost" // Minimal fallback
					}
				}
			}

			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Build-ID")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSMiddlewareFromEnv creates a CORS middleware using environment variables.
// It reads ALLOWED_ORIGIN and UI_PORT from the environment.
func CORSMiddlewareFromEnv(logger *slog.Logger) func(http.Handler) http.Handler {
	return CORSMiddleware(CORSConfig{
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),
		UIPort:        os.Getenv("UI_PORT"),
		Logger:        logger,
	})
}

// LoggingMiddleware creates a request logging middleware that writes Apache-style
// combined logs to the provided writer.
func LoggingMiddleware(w io.Writer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return handlers.LoggingHandler(w, next)
	}
}

// LoggingMiddlewareStdout creates a logging middleware that writes to stdout.
func LoggingMiddlewareStdout() func(http.Handler) http.Handler {
	return LoggingMiddleware(os.Stdout)
}

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	size        int
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.status = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

// Unwrap returns the underlying ResponseWriter for http.ResponseController.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// StructuredLoggingMiddleware creates a middleware that logs requests using slog.
// This is an alternative to LoggingMiddleware that integrates with structured logging.
func StructuredLoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapped := &responseWriter{
				ResponseWriter: w,
				status:         http.StatusOK,
			}

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			// Log at different levels based on status
			logFn := logger.Info
			if wrapped.status >= 500 {
				logFn = logger.Error
			} else if wrapped.status >= 400 {
				logFn = logger.Warn
			}

			logFn("HTTP request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.status,
				"size", wrapped.size,
				"duration_ms", duration.Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
			)
		})
	}
}

// RecoveryMiddleware creates a middleware that recovers from panics and logs them.
// It returns a 500 Internal Server Error to the client.
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						"panic", rec,
						"method", r.Method,
						"path", r.URL.Path,
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// Chain combines multiple middleware into a single middleware.
// Middleware are applied in the order they are passed.
func Chain(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(final http.Handler) http.Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// RequestIDMiddleware creates a middleware that generates and propagates request IDs.
// The ID is added to the X-Request-ID response header.
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
			}

			w.Header().Set("X-Request-ID", requestID)

			// Add to context for downstream handlers
			// Note: In a full implementation, you'd use context.WithValue here
			next.ServeHTTP(w, r)
		})
	}
}

// generateRequestID generates a simple request ID.
// For production, consider using UUID or a more robust ID generator.
func generateRequestID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// TimeoutMiddleware creates a middleware that enforces a request timeout.
// If the handler takes longer than the timeout, the request is cancelled.
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "request timed out")
	}
}
