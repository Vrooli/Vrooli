package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddleware_SetsDefensiveHeaders(t *testing.T) {
	mw := NewSecurityHeadersMiddleware()
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	want := map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
		"X-XSS-Protection":       "0",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for k, v := range want {
		if got := rr.Header().Get(k); got != v {
			t.Errorf("header %s = %q, want %q", k, got, v)
		}
	}
	if rr.Header().Get("Content-Security-Policy") == "" {
		t.Error("Content-Security-Policy header must be set")
	}
}

func TestSecurityHeadersMiddleware_HSTSOnlyOverHTTPS(t *testing.T) {
	mw := NewSecurityHeadersMiddleware()
	h := mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	// Plain HTTP: no HSTS.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS must not be set on plain HTTP")
	}

	// Forwarded HTTPS: HSTS present.
	rr = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS must be set when X-Forwarded-Proto is https")
	}
}
