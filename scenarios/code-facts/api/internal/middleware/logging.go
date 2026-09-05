// Package middleware holds the production middleware stack that wraps
// the API router. Every middleware accepts its time/log dependencies as
// parameters so tests can inject fakes.
package middleware

import (
	"net/http"

	"code-facts/internal/logging"

	"github.com/vrooli/api-core/schedule"
)

// NewLoggingMiddleware returns a middleware that logs each request's
// method, URI, and elapsed duration. Time is read from the injected
// Clock so tests using scheduletest.FakeClock can assert exact durations
// without depending on the wall schedule. Logger is required; tests inject a
// buffer-backed implementation to capture output.
func NewLoggingMiddleware(clk schedule.Clock, logger logging.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		panic("middleware.NewLoggingMiddleware requires a logger")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := clk.Now()
			next.ServeHTTP(w, r)
			logger.Printf("[%s] %s %s", r.Method, r.RequestURI, clk.Now().Sub(start))
		})
	}
}
