package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouterStampsSecurityHeaders(t *testing.T) {
	srv := newTestServer()
	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	srv.Router().ServeHTTP(resp, req)

	want := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "0",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}
	for name, value := range want {
		if got := resp.Header().Get(name); got != value {
			t.Fatalf("header %s = %q, want %q", name, got, value)
		}
	}
}
