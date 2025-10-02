package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with security headers middleware
	handler := SecurityHeadersMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify response status
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify security headers are set
	testCases := []struct {
		header   string
		expected string
	}{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tc := range testCases {
		t.Run(tc.header, func(t *testing.T) {
			actual := w.Header().Get(tc.header)
			if actual != tc.expected {
				t.Errorf("Expected %s header to be %q, got %q", tc.header, tc.expected, actual)
			}
		})
	}
}

func TestSecurityHeadersMiddleware_CSP(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with security headers middleware
	handler := SecurityHeadersMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify CSP header exists and contains key directives
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Expected Content-Security-Policy header to be set")
	}

	// Check for key CSP directives
	requiredDirectives := []string{
		"default-src 'self'",
		"script-src 'self'",
		"style-src 'self'",
		"frame-ancestors 'none'",
	}

	for _, directive := range requiredDirectives {
		if !contains(csp, directive) {
			t.Errorf("Expected CSP to contain %q, but it didn't. CSP: %s", directive, csp)
		}
	}
}

func TestSecurityHeadersMiddleware_PermissionsPolicy(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with security headers middleware
	handler := SecurityHeadersMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify Permissions-Policy header exists and restricts features
	permPolicy := w.Header().Get("Permissions-Policy")
	if permPolicy == "" {
		t.Error("Expected Permissions-Policy header to be set")
	}

	// Check for restricted features
	restrictedFeatures := []string{
		"geolocation=()",
		"microphone=()",
		"camera=()",
		"payment=()",
	}

	for _, feature := range restrictedFeatures {
		if !contains(permPolicy, feature) {
			t.Errorf("Expected Permissions-Policy to contain %q, but it didn't. Policy: %s", feature, permPolicy)
		}
	}
}

func TestSecurityHeadersMiddleware_NoHSTS(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with security headers middleware
	handler := SecurityHeadersMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify HSTS is NOT set (since we're in development without HTTPS)
	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("Expected Strict-Transport-Security to not be set in development, got %q", hsts)
	}
}

func TestSecurityHeadersMiddleware_PreservesHandlerResponse(t *testing.T) {
	// Create a test handler that sets custom headers
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("Custom Response"))
	})

	// Wrap with security headers middleware
	handler := SecurityHeadersMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Verify handler response is preserved
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	customHeader := w.Header().Get("X-Custom-Header")
	if customHeader != "custom-value" {
		t.Errorf("Expected X-Custom-Header to be 'custom-value', got %q", customHeader)
	}

	body := w.Body.String()
	if body != "Custom Response" {
		t.Errorf("Expected body 'Custom Response', got %q", body)
	}

	// Security headers should still be present
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("Security headers should still be set")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
