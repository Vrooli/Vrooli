package middleware

import (
	"net/http"
	"os"
)

// SecurityHeaders sets defensive response headers on every request. Ported from
// the old middleware/security.go with two hardenings (plan §5): the default CSP
// drops 'unsafe-inline' (the old default allowed it "for prototyping"), and HSTS
// is emitted when the request arrives over HTTPS (or behind a TLS-terminating
// proxy that sets X-Forwarded-Proto: https) rather than being commented out.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-XSS-Protection", "1; mode=block")
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

func isHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return r.Header.Get("X-Forwarded-Proto") == "https"
}
