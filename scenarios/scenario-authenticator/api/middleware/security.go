package middleware

import (
	"net/http"
	"os"
)

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent clickjacking attacks
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Enable XSS protection (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Content Security Policy - strict by default
		// Note: In production, remove 'unsafe-inline' and use nonces or hashes
		// For development, 'unsafe-inline' allows quick prototyping
		csp := os.Getenv("CSP_POLICY")
		if csp == "" {
			// Default development CSP with inline scripts/styles for prototyping
			// WARNING: Remove 'unsafe-inline' in production for better XSS protection
			csp = "default-src 'self'; " +
				"script-src 'self' 'unsafe-inline'; " +
				"style-src 'self' 'unsafe-inline'; " +
				"img-src 'self' data:; " +
				"font-src 'self'; " +
				"connect-src 'self'; " +
				"frame-ancestors 'none'"
		}
		w.Header().Set("Content-Security-Policy", csp)

		// Referrer Policy - don't leak information in referer header
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy - disable unnecessary browser features
		w.Header().Set("Permissions-Policy",
			"geolocation=(), "+
				"microphone=(), "+
				"camera=(), "+
				"payment=(), "+
				"usb=(), "+
				"magnetometer=()")

		// Note: HSTS (Strict-Transport-Security) should only be set when using HTTPS
		// Uncomment in production with HTTPS:
		// w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

		next.ServeHTTP(w, r)
	})
}
