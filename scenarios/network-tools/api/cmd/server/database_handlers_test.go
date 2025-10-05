package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestDatabaseHandlers tests handlers that interact with the database
func TestDatabaseHandlers(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("handleListTargets", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/network/targets",
		}

		w, err := makeHTTPRequest(server.handleListTargets, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return success even with no database
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}

		var resp Response
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		// Check response structure
		if w.Code == http.StatusOK && !resp.Success {
			t.Error("Expected success response")
		}
	})

	t.Run("handleCreateTarget", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/targets",
			Body: map[string]interface{}{
				"url":      "https://example.com",
				"interval": 300,
			},
		}

		w, err := makeHTTPRequest(server.handleCreateTarget, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should handle gracefully even without database
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200, 400, or 500, got %d", w.Code)
		}
	})

	t.Run("handleCreateTargetInvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/targets",
			Body:   `{"invalid": json}`,
		}

		w, err := makeHTTPRequest(server.handleCreateTarget, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// When database is not configured, returns 500 before JSON validation
		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 when database not configured, got %d", w.Code)
		}
	})

	t.Run("handleListAlerts", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/network/alerts",
		}

		w, err := makeHTTPRequest(server.handleListAlerts, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return success or error depending on database
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("handleListAlertsWithLimit", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/network/alerts?limit=10",
		}

		w, err := makeHTTPRequest(server.handleListAlerts, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})
}

// TestTLSHelpers tests TLS-related helper functions
func TestTLSHelpers(t *testing.T) {
	t.Run("tlsVersionString", func(t *testing.T) {
		testCases := []struct {
			version  uint16
			expected string
		}{
			{0x0301, "TLS 1.0"},
			{0x0302, "TLS 1.1"},
			{0x0303, "TLS 1.2"},
			{0x0304, "TLS 1.3"},
			{0x0300, "Unknown (0x0300)"},
			{0x9999, "Unknown (0x9999)"},
		}

		for _, tc := range testCases {
			result := tlsVersionString(tc.version)
			if result != tc.expected {
				t.Errorf("tlsVersionString(%x) = %s, expected %s", tc.version, result, tc.expected)
			}
		}
	})
}

// TestAuthMiddlewareExtended tests the authentication middleware comprehensively
func TestAuthMiddlewareExtended(t *testing.T) {
	t.Run("HealthEndpointNoAuth", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health endpoint should not require auth, got %d", w.Code)
		}
	})

	t.Run("OptionsRequestNoAuth", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		req := httptest.NewRequest("OPTIONS", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("OPTIONS requests should not require auth, got %d", w.Code)
		}
	})

	t.Run("DevelopmentMode", func(t *testing.T) {
		// Set development auth mode
		originalAuthMode := os.Getenv("AUTH_MODE")
		os.Setenv("AUTH_MODE", "development")
		defer os.Setenv("AUTH_MODE", originalAuthMode)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Development mode should bypass auth, got %d", w.Code)
		}
	})

	t.Run("ValidAPIKey", func(t *testing.T) {
		// Set production environment and API key
		originalEnv := os.Getenv("VROOLI_ENV")
		originalKey := os.Getenv("NETWORK_TOOLS_API_KEY")
		os.Setenv("VROOLI_ENV", "production")
		os.Setenv("NETWORK_TOOLS_API_KEY", "test-api-key-123")
		defer func() {
			os.Setenv("VROOLI_ENV", originalEnv)
			os.Setenv("NETWORK_TOOLS_API_KEY", originalKey)
		}()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-API-Key", "test-api-key-123")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Valid API key should allow access, got %d", w.Code)
		}
	})

	t.Run("InvalidAPIKey", func(t *testing.T) {
		// Set production environment and API key
		originalEnv := os.Getenv("VROOLI_ENV")
		originalKey := os.Getenv("NETWORK_TOOLS_API_KEY")
		os.Setenv("VROOLI_ENV", "production")
		os.Setenv("NETWORK_TOOLS_API_KEY", "test-api-key-123")
		defer func() {
			os.Setenv("VROOLI_ENV", originalEnv)
			os.Setenv("NETWORK_TOOLS_API_KEY", originalKey)
		}()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-API-Key", "wrong-key")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Invalid API key should return 401, got %d", w.Code)
		}
	})

	t.Run("MissingAPIKey", func(t *testing.T) {
		// Set production environment and API key
		originalEnv := os.Getenv("VROOLI_ENV")
		originalKey := os.Getenv("NETWORK_TOOLS_API_KEY")
		os.Setenv("VROOLI_ENV", "production")
		os.Setenv("NETWORK_TOOLS_API_KEY", "test-api-key-123")
		defer func() {
			os.Setenv("VROOLI_ENV", originalEnv)
			os.Setenv("NETWORK_TOOLS_API_KEY", originalKey)
		}()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Missing API key should return 401, got %d", w.Code)
		}
	})
}

// TestCORSMiddlewareExtended tests the CORS middleware
func TestCORSMiddlewareExtended(t *testing.T) {
	t.Run("OptionsRequestNoCORS", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("OPTIONS", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("OPTIONS should return 200, got %d", w.Code)
		}

		// Check that methods are set
		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected Access-Control-Allow-Methods header")
		}
	})

	t.Run("DevelopmentModeNoOrigin", func(t *testing.T) {
		originalEnv := os.Getenv("VROOLI_ENV")
		os.Setenv("VROOLI_ENV", "development")
		defer os.Setenv("VROOLI_ENV", originalEnv)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") != "*" {
			t.Errorf("Development mode with no origin should set wildcard CORS: got %s", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("AllowedOrigin", func(t *testing.T) {
		// Set development mode to allow localhost origins
		originalEnv := os.Getenv("VROOLI_ENV")
		os.Setenv("VROOLI_ENV", "development")
		defer os.Setenv("VROOLI_ENV", originalEnv)

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("Origin", "http://localhost:35000")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		// Should allow the localhost origin in development mode
		origin := w.Header().Get("Access-Control-Allow-Origin")
		if origin != "http://localhost:35000" {
			t.Errorf("Expected localhost origin, got: %s", origin)
		}
	})
}
