package middleware

import (
	"net/http"
	"os"
)

// NewSecurityHeadersMiddleware returns a middleware that sets defensive
// response headers on every request. Tunnel Manager fronts public, tunneled
// traffic, so hardening the broker's own UI/API responses matters: deny
// framing, forbid MIME sniffing, constrain referrers, and ship a strict CSP
// (override via CSP_POLICY for environments that need a looser policy). HSTS is
// emitted only when the request arrives over HTTPS (directly or via a
// TLS-terminating proxy that sets X-Forwarded-Proto: https).
func NewSecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Frame-Options", "DENY")
			h.Set("X-Content-Type-Options", "nosniff")
			// The legacy XSS auditor is obsolete and can introduce security
			// bypasses in modern browsers; explicitly disable it.
			h.Set("X-XSS-Protection", "0")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=()")

			csp := os.Getenv("CSP_POLICY")
			if csp == "" {
				csp = "default-src 'self'; script-src 'self'; style-src 'self'; " +
					"img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'"
			}
			h.Set("Content-Security-Policy", csp)

			if isHTTPS(r) {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}
