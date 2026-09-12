package middleware

import "net/http"

// NewSecurityHeadersMiddleware returns a middleware that stamps the baseline
// OWASP security response headers onto every response the API emits — REST
// helpers, Connect-RPC handlers, and the health endpoint alike. Setting them
// once here, at the router boundary, is deliberate: it is the single place
// that covers all response paths, so individual handlers and the shared httpx
// response writers never have to remember to set them.
//
// The four headers below are the always-safe, app-agnostic defaults:
//
//   - X-Content-Type-Options: nosniff — stop MIME sniffing of responses.
//   - X-Frame-Options: DENY — refuse framing (clickjacking defense). The API
//     serves JSON, not framed UI, so DENY is correct here; the UI surface owns
//     its own framing policy.
//   - X-XSS-Protection: 0 — explicitly disable the legacy, buggy XSS auditor.
//     Modern guidance (and the rule's intent of "the header is considered")
//     is to send it set to 0 rather than rely on the removed heuristic.
//   - Strict-Transport-Security — force HTTPS for a year once seen over TLS.
//
// CORS is intentionally NOT set here: a cross-origin policy is a deliberate
// per-scenario decision (which origins, which methods, credentials or not),
// not a blanket template default. Add a dedicated CORS middleware when the
// scenario actually serves browsers from another origin.
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
