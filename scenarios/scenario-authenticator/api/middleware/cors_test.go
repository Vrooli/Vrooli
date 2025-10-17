package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCORSMiddleware_AllowedOrigin(t *testing.T) {
	// Set allowed origins
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:5173")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with CORS middleware
	handler := CORSMiddleware(testHandler)

	// Create test request with allowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify CORS headers
	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Expected Access-Control-Allow-Origin to be 'http://localhost:3000', got %q",
			w.Header().Get("Access-Control-Allow-Origin"))
	}

	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("Expected Access-Control-Allow-Credentials to be 'true'")
	}

	// Verify handler was called
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestCORSMiddleware_DisallowedOrigin(t *testing.T) {
	// Set allowed origins
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with CORS middleware
	handler := CORSMiddleware(testHandler)

	// Create test request with disallowed origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://evil.com")
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify CORS origin header is NOT set for disallowed origin
	if w.Header().Get("Access-Control-Allow-Origin") == "http://evil.com" {
		t.Error("Disallowed origin should not be reflected in Access-Control-Allow-Origin")
	}

	// Methods and Headers should still be set
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods should be set")
	}
}

func TestCORSMiddleware_DefaultOrigins(t *testing.T) {
	// Clear any existing CORS env var
	os.Unsetenv("CORS_ALLOWED_ORIGINS")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CORS middleware
	handler := CORSMiddleware(testHandler)

	// Test default allowed origins
	defaultOrigins := []string{
		"http://localhost:3000",
		"http://localhost:5173",
		"http://localhost:8080",
	}

	for _, origin := range defaultOrigins {
		t.Run(origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", origin)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if w.Header().Get("Access-Control-Allow-Origin") != origin {
				t.Errorf("Expected default origin %q to be allowed, but got %q",
					origin, w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	// Set allowed origins
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called for OPTIONS preflight request")
	})

	// Wrap with CORS middleware
	handler := CORSMiddleware(testHandler)

	// Create OPTIONS preflight request
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify preflight response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
	}

	// Verify CORS headers are set
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods should be set for preflight")
	}

	if w.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Access-Control-Allow-Headers should be set for preflight")
	}
}

func TestCORSMiddleware_NoOriginHeader(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CORS middleware
	handler := CORSMiddleware(testHandler)

	// Create test request without Origin header (e.g., same-origin request)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify handler was called
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// CORS origin header should not be set
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Access-Control-Allow-Origin should not be set when no Origin header present")
	}

	// But methods and headers should still be set
	if w.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Access-Control-Allow-Methods should be set")
	}
}

func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	// Set wildcard origin (for testing purposes only, not recommended for production)
	os.Setenv("CORS_ALLOWED_ORIGINS", "*")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CORS middleware
	handler := CORSMiddleware(testHandler)

	// Create test request with any origin
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://any-domain.com")
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify wildcard allows any origin
	if w.Header().Get("Access-Control-Allow-Origin") != "http://any-domain.com" {
		t.Error("Wildcard should allow any origin")
	}
}

func TestCORSMiddleware_MultipleOrigins(t *testing.T) {
	// Set multiple allowed origins with spaces
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, http://example.com, https://app.example.com")
	defer os.Unsetenv("CORS_ALLOWED_ORIGINS")

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with CORS middleware
	handler := CORSMiddleware(testHandler)

	testCases := []struct {
		origin  string
		allowed bool
	}{
		{"http://localhost:3000", true},
		{"http://example.com", true},
		{"https://app.example.com", true},
		{"http://notallowed.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Origin", tc.origin)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			allowedOrigin := w.Header().Get("Access-Control-Allow-Origin")
			if tc.allowed && allowedOrigin != tc.origin {
				t.Errorf("Expected origin %q to be allowed, but got %q", tc.origin, allowedOrigin)
			}
			if !tc.allowed && allowedOrigin == tc.origin {
				t.Errorf("Expected origin %q to be blocked, but it was allowed", tc.origin)
			}
		})
	}
}
