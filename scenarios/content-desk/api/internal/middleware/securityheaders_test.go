package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"content-desk/internal/middleware"

	"github.com/stretchr/testify/require"
)

// TestSecurityHeadersMiddleware_SetsBaselineHeaders proves every response that
// passes through the router carries the baseline OWASP security headers,
// regardless of what the inner handler writes (or whether it writes at all).
func TestSecurityHeadersMiddleware_SetsBaselineHeaders(t *testing.T) {
	mw := middleware.NewSecurityHeadersMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	require.Equal(t, "0", rec.Header().Get("X-XSS-Protection"))
	require.Contains(t, rec.Header().Get("Strict-Transport-Security"), "max-age=31536000")
}

// TestSecurityHeadersMiddleware_SetsHeadersEvenWhenHandlerDoesNotWrite proves
// the headers are stamped before the inner handler runs, so even a handler that
// never writes a body still emits them.
func TestSecurityHeadersMiddleware_SetsHeadersEvenWhenHandlerDoesNotWrite(t *testing.T) {
	mw := middleware.NewSecurityHeadersMiddleware()
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	handler.ServeHTTP(rec, req)

	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
}
