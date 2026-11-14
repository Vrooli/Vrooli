package apihttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersIncludeErrorResponses(t *testing.T) {
	handler := SecurityHeaders(http.NotFoundHandler())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d", response.Code)
	}
	for name, value := range map[string]string{
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains", "X-Content-Type-Options": "nosniff",
		"X-Frame-Options": "DENY", "X-XSS-Protection": "0",
	} {
		if got := response.Header().Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
}
