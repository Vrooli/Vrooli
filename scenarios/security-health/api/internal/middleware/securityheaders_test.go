package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"security-health/internal/middleware"
)

// [REQ:REQ-P0-020]
func TestSecurityHeadersMiddlewareSetsBaselineHeaders(t *testing.T) {
	handler := middleware.NewSecurityHeadersMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/probe", nil))

	want := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "0",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}
	for header, value := range want {
		if got := rec.Header().Get(header); got != value {
			t.Fatalf("%s = %q, want %q", header, got, value)
		}
	}
}
