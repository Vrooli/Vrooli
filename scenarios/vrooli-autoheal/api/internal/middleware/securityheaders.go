package middleware

import "net/http"

// NewSecurityHeadersMiddleware stamps the baseline browser-hardening headers
// at the API router boundary so REST, health, and dev-routing responses share
// one policy.
func NewSecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := w.Header()
			header.Set("X-Content-Type-Options", "nosniff")
			header.Set("X-Frame-Options", "DENY")
			header.Set("X-XSS-Protection", "0")
			header.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			next.ServeHTTP(w, r)
		})
	}
}
