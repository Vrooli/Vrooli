package httpmw

import (
	"net/http"
	"time"

	"agent-manager/internal/orchestration/obs"
)

// Logging records concise request timing independently of route composition.
func Logging(next http.Handler) http.Handler {
	httpLog := obs.Component("http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		httpLog.Debug("request", "method", r.Method, "uri", r.RequestURI, obs.KeyDuration, time.Since(start).Milliseconds())
	})
}
