package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestCORSDefaultsAllowLocalhost validates default CORS behavior. [REQ:KO-API-003]
func TestCORSDefaultsAllowLocalhost(t *testing.T) {
	os.Unsetenv("CORS_ALLOWED_ORIGINS")
	srv := newTestServer()
	srv.setupRoutes()

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	srv.handler().ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected allow-origin header, got %q", got)
	}
}

// TestCORSAllowlistEnforced validates allowlist-based CORS. [REQ:KO-API-003]
func TestCORSAllowlistEnforced(t *testing.T) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "https://allowed.example")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	srv := newTestServer()
	srv.setupRoutes()

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	req.Header.Set("Origin", "https://allowed.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()

	srv.handler().ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.example" {
		t.Fatalf("expected allow-origin header, got %q", got)
	}
}
