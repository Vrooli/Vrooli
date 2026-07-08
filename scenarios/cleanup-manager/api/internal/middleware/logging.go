// Package middleware holds the production middleware stack that wraps
// the API router. Every middleware accepts its time/log dependencies as
// parameters so tests can inject fakes.
package middleware

import (
	"io"
	"log"
	"net/http"

	"cleanup-manager/internal/clock"
)

// NewLoggingMiddleware returns a middleware that logs each request's
// method, URI, and elapsed duration. Time is read from the injected
// Clock so tests using mocks.FakeClock can assert exact durations
// without depending on the wall clock. Logger defaults to a discard logger
// when nil; tests inject a buffer-backed *log.Logger to capture output.
func NewLoggingMiddleware(clk clock.Clock, logger *log.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := clk.Now()
			next.ServeHTTP(w, r)
			logger.Printf("[%s] %s %s", r.Method, r.RequestURI, clk.Now().Sub(start))
		})
	}
}
