package middleware

import (
	"net/http"
	"os"
	"strings"
)

// SecurityHeaders wraps the API handler and sets the response security
// headers on every response, in one place, so individual handlers never
// have to remember them. It is applied as the outermost wrap in main.go.
//
// Threat model (see PRD.md): the server is owner-trusted; the goal is to keep
// untrusted parties who merely share the network/tunnel out, and to harden the
// browser surface. TLS termination is handled upstream (app-monitor tunnel /
// reverse proxy), so HSTS is advisory here but still emitted so downstream
// proxies and scanners see a consistent policy.
//
// CORS: the API authenticates with explicit request headers (owner bearer JWT
// and X-Device-Token), never ambient cookies, so it does NOT enable
// Access-Control-Allow-Credentials. The allowed origin is the configured UI
// origin (CORS_ALLOW_ORIGIN) when set, otherwise the request's own Origin is
// reflected — never a wildcard combined with credentials. Preflight OPTIONS
// requests short-circuit with the headers and a 204.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ApplySecurityHeaders(w)
		ApplyCORSHeaders(w, r)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// ApplySecurityHeaders sets the browser-hardening response headers. It is the
// one place these values are defined; the SecurityHeaders middleware calls it
// for every response, and the raw byte/stream/error response paths
// (transfer download, the SSE stream, the JSON error writer) call it directly
// so the headers are present even on those non-Connect surfaces.
func ApplySecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
}

// ApplyCORSHeaders reflects the configured/known origin onto the response. The
// API authenticates with explicit request headers (owner bearer JWT and
// X-Device-Token), never ambient cookies, so it deliberately does NOT enable
// Access-Control-Allow-Credentials and never pairs a wildcard origin with
// credentials. The allowed origin is CORS_ALLOW_ORIGIN when set, otherwise the
// request's own Origin is reflected.
func ApplyCORSHeaders(w http.ResponseWriter, r *http.Request) {
	origin := strings.TrimSpace(os.Getenv("CORS_ALLOW_ORIGIN"))
	if origin == "" {
		origin = strings.TrimSpace(r.Header.Get("Origin"))
	}
	if origin == "" {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Add("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Device-Token")
}
