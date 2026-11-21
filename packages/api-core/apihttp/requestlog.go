package apihttp

import (
	"log"
	"net/http"

	"github.com/vrooli/api-core/schedule"
)

type requestLogger interface {
	Printf(string, ...any)
}

// RequestLoggingMiddleware logs the method, request URI, and elapsed time for
// each request. The clock and logger are injectable so callers can verify
// timing without sleeping in tests. A nil logger uses the standard logger.
func RequestLoggingMiddleware(clock schedule.Clock, logger requestLogger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = log.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := clock.Now()
			next.ServeHTTP(w, r)
			logger.Printf("[%s] %s %s", r.Method, r.RequestURI, clock.Now().Sub(start))
		})
	}
}
