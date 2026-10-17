package middleware

import "net/http"

// NewSecurityHeadersMiddleware applies the baseline response policy to every
// API route, including error responses emitted by downstream handlers.
// Strict-Transport-Security is set here because production deployments
// terminate TLS before forwarding to this service.
func NewSecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "0")
			next.ServeHTTP(w, r)
		})
	}
}
