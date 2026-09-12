package middleware

import "net/http"

// NewSecurityHeadersMiddleware stamps baseline browser security headers on
// every API response. These headers are intentionally centralized at the
// router boundary so REST handlers, Connect handlers, and health probes share
// the same default posture.
//
// CORS is deliberately not set here. Allowed origins, methods, and credential
// behavior are scenario-specific policy and must be handled by a dedicated CORS
// middleware when a scenario actually exposes cross-origin browser traffic.
func NewSecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-XSS-Protection", "0")
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			next.ServeHTTP(w, r)
		})
	}
}
