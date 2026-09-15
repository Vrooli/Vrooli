package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"search-hub/internal/middleware"

	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersMiddlewareSetsBaselineHeaders(t *testing.T) {
	handler := middleware.NewSecurityHeadersMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	require.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	require.Equal(t, "0", rec.Header().Get("X-XSS-Protection"))
	require.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
}
