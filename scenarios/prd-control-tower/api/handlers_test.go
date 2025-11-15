package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// TestHandleHealth tests the health check endpoint
func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleHealth() status = %d, want %d", w.Code, http.StatusOK)
	}

	var response HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode health response: %v", err)
		return
	}

	if response.Service != "prd-control-tower-api" {
		t.Errorf("handleHealth() service = %q, want %q", response.Service, "prd-control-tower-api")
	}

	// Status may be "degraded" when db is nil, which is expected in unit tests
	if response.Status != "healthy" && response.Status != "degraded" {
		t.Errorf("handleHealth() status = %q, want %q or %q", response.Status, "healthy", "degraded")
	}

	// Verify response structure contains dependencies
	if response.Dependencies == nil {
		t.Errorf("handleHealth() dependencies should not be nil")
	}
}

// TestCORSMiddleware tests CORS middleware functionality
func TestCORSMiddleware(t *testing.T) {
	// Create a test handler that CORS middleware will wrap
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
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
