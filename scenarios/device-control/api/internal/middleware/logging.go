// Package middleware holds the production middleware stack that wraps
// the API router. Every middleware accepts its time/log dependencies as
// parameters so tests can inject fakes.
package middleware

import (
	"log"
	"net/http"
	"time"
)

// Clock is the smallest time dependency this middleware needs. The richer
// api-core/schedule.Clock remains assignable to it at the composition root.
type Clock interface {
	Now() time.Time
}

// NewLoggingMiddleware returns a middleware that logs each request's
// method, path, and elapsed duration without query strings or bodies. Time and logging are supplied by the
// composition root so tests can control both dependencies.
func NewLoggingMiddleware(clk Clock, logger *log.Logger) func(http.Handler) http.Handler {
	if clk == nil {
		panic("middleware.NewLoggingMiddleware requires Clock")
	}
	if logger == nil {
		panic("middleware.NewLoggingMiddleware requires Logger")
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := clk.Now()
			next.ServeHTTP(w, r)
			// Query strings can carry credentials or media/control material in
			// integrations. Log the route path only; request bodies are untouched.
			logger.Printf("[%s] %s %s", r.Method, r.URL.Path, clk.Now().Sub(start))
		})
	}
}
