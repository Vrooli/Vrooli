package middleware

import "net/http"

// NewSecurityHeadersMiddleware stamps the API-wide browser security headers at
// the router boundary. Search Hub serves JSON and RPC responses rather than
// framed UI, so DENY is the appropriate framing policy. CORS remains a
// deliberate per-scenario decision and is not inferred here.
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
