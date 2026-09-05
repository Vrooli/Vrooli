package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
)

// Recovery returns middleware that catches panics and writes a 500 JSON error.
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := debug.Stack()
					log.Error("panic recovered",
						"panic", fmt.Sprintf("%v", rec),
						"stack", string(stack),
						"path", r.URL.Path,
						"method", r.Method,
						"request_id", r.Header.Get("X-Request-ID"),
					)
					httputil.WriteError(w, nil, r, http.StatusInternalServerError, "internal", "An internal error occurred", "")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
