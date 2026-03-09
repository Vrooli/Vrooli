package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
)

// TestHandleHealth tests the health check endpoint
func TestHandleHealth(t *testing.T) {
	// Build the same handler used in main.go (no DB in unit tests)
	handler := health.New().Version("1.0.0").Handler()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health handler status = %d, want %d", w.Code, http.StatusOK)
	}

	var response health.Response
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
		return
	}

	// Without DB check, status should be healthy
	if response.Status != "healthy" {
		t.Errorf("health handler status = %q, want %q", response.Status, "healthy")
	}

	if response.Version != "1.0.0" {
		t.Errorf("health handler version = %q, want %q", response.Version, "1.0.0")
	}
}

// TestCORSMiddleware tests CORS middleware functionality
func TestCORSMiddleware(t *testing.T) {
	// Create a test handler that CORS middleware will wrap
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wrap with CORS middleware
	router := mux.NewRouter()
	router.Use(corsMiddleware)
	router.HandleFunc("/test", testHandler)

	tests := []struct {
		name          string
		method        string
		origin        string
		expectAllowed bool
	}{
		{
			name:          "localhost origin allowed",
			method:        "GET",
			origin:        "http://localhost:36300",
			expectAllowed: true,
		},
		{
			name:          "127.0.0.1 origin allowed",
			method:        "GET",
			origin:        "http://127.0.0.1:36300",
			expectAllowed: true,
		},
		{
			name:          "preflight request",
			method:        "OPTIONS",
			origin:        "http://localhost:36300",
			expectAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/test", nil)
			req.Header.Set("Origin", tt.origin)

			if tt.method == "OPTIONS" {
				req.Header.Set("Access-Control-Request-Method", "POST")
			}

			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Check CORS headers
			allowOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tt.expectAllowed {
				if allowOrigin != tt.origin && allowOrigin == "" {
					t.Errorf("Expected Access-Control-Allow-Origin to be set for %s, got %q", tt.origin, allowOrigin)
				}
			}

			// Preflight requests should return 200
			if tt.method == "OPTIONS" && w.Code != http.StatusOK {
				t.Errorf("Preflight request status = %d, want %d", w.Code, http.StatusOK)
			}
		})
	}
}
