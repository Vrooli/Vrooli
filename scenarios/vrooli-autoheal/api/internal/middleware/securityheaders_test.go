package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/middleware"
)

func TestSecurityHeadersMiddlewareSetsBaselineHeaders(t *testing.T) {
	handler := middleware.NewSecurityHeadersMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	for header, want := range map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "0",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	} {
		if got := recorder.Header().Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}
