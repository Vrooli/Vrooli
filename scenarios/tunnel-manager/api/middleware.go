package main

import (
	"log/slog"
	"net/http"
	"time"
)

// loggingMiddleware logs each request with structured fields.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request", "method", r.Method, "path", r.RequestURI, "duration_ms", time.Since(start).Milliseconds())
	})
}
