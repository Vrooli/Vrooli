package middleware

import (
	"log/slog"
	"net/http"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/httputil"
)

// MaxBodySize returns middleware that limits request body size.
// If the Content-Length header exceeds maxBytes, a structured JSON 413 error is
// returned immediately. Otherwise, the body is wrapped with MaxBytesReader as a
// safety net for chunked or missing Content-Length requests.
func MaxBodySize(maxBytes int64, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				httputil.WriteError(w, log, r, http.StatusRequestEntityTooLarge,
					"validation", "Request body too large", "content-length exceeds limit")
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}
