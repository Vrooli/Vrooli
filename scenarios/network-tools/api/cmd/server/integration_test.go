package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestSSLValidationHandler tests SSL validation endpoint
func TestSSLValidationHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("SuccessHTTPSValidation", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body: map[string]interface{}{
				"url": "https://google.com",
				"options": map[string]interface{}{
					"check_expiry":   true,
					"check_chain":    true,
					"check_hostname": true,
				},
			},
		})

		resp := assertSuccessResponse(t, w, http.StatusOK)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if _, ok := data["valid"]; !ok {
				t.Error("Expected 'valid' field in SSL validation response")
			}
		}
	})

	t.Run("ErrorMissingURL", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body:   map[string]interface{}{},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "URL")
	})

	t.Run("ErrorInvalidURL", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body: map[string]interface{}{
				"url": "not-a-url",
			},
		})

		// Should fail with bad request or internal error
		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 400 or 500, got %d", w.Code)
		}
	})

	t.Run("ErrorHTTPNotHTTPS", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body: map[string]interface{}{
				"url": "http://example.com",
			},
		})

		// Should handle HTTP URLs gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 200 or 400, got %d", w.Code)
		}
	})
}

// TestAPITestHandler tests API testing endpoint
func TestAPITestHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("SuccessBasicAPITest", func(t *testing.T) {
		mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})

		w, _ := makeHTTPRequest(server.handleAPITest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/api/test",
			Body: map[string]interface{}{
				"base_url": mockServer.URL,
				"test_suite": []map[string]interface{}{
					{
						"endpoint": "/test",
						"method":   "GET",
						"test_cases": []map[string]interface{}{
							{
								"name":            "basic test",
								"expected_status": 200,
							},
						},
					},
				},
			},
		})

		resp := assertSuccessResponse(t, w, http.StatusOK)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if _, ok := data["test_results"]; !ok {
				t.Error("Expected 'test_results' field in response")
			}
		}
	})

	t.Run("ErrorMissingBaseURL", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleAPITest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/api/test",
			Body: map[string]interface{}{
				"test_suite": []map[string]interface{}{},
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestListTargetsHandler tests target listing endpoint
func TestListTargetsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("SuccessEmptyList", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleListTargets, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/network/targets",
		})

		// Without database, should return error or empty list
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestCreateTargetHandler tests target creation endpoint
func TestCreateTargetHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("ErrorNoDatabase", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleCreateTarget, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/targets",
			Body: map[string]interface{}{
				"name":    "test-target",
				"address": "example.com",
				"port":    443,
			},
		})

		// Without database, should fail
		if w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
			t.Errorf("Expected error status, got %d", w.Code)
		}
	})
}

// TestListAlertsHandler tests alert listing endpoint
func TestListAlertsHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("ErrorNoDatabase", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleListAlerts, HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/network/alerts",
		})

		// Without database, should fail
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestHelperGetServiceName tests service name resolution
func TestHelperGetServiceName(t *testing.T) {
	tests := []struct {
		port     int
		expected string
	}{
		{80, "http"},
		{443, "https"},
		{22, "ssh"},
		{21, "ftp"},
		{25, "smtp"},
		{3306, "mysql"},
		{5432, "postgresql"},
		{9999, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := getServiceName(tt.port)
			if result != tt.expected {
				t.Errorf("Expected %s for port %d, got %s", tt.expected, tt.port, result)
			}
		})
	}
}

// TestHelperMapToJSON tests JSON helper function
func TestHelperMapToJSON(t *testing.T) {
	t.Run("ValidMap", func(t *testing.T) {
		testMap := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		result := mapToJSON(testMap)
		if result == "" {
			t.Error("Expected non-empty JSON string")
		}

		// Validate it's valid JSON
		var decoded map[string]string
		if err := json.Unmarshal([]byte(result), &decoded); err != nil {
			t.Errorf("Result is not valid JSON: %v", err)
		}
	})

	t.Run("EmptyMap", func(t *testing.T) {
		testMap := map[string]string{}
		result := mapToJSON(testMap)
		if result != "{}" {
			t.Errorf("Expected '{}', got '%s'", result)
		}
	})
}

// TestCORSMiddleware tests CORS handling
func TestCORSMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("OptionsRequest", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := corsMiddleware(handler)

		req := httptest.NewRequest("OPTIONS", "/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")

		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			t.Error("Expected CORS headers to be set")
		}
	})

	t.Run("DevelopmentMode", func(t *testing.T) {
		oldEnv := os.Getenv("VROOLI_ENV")
		os.Setenv("VROOLI_ENV", "development")
		defer func() {
			if oldEnv == "" {
				os.Unsetenv("VROOLI_ENV")
			} else {
				os.Setenv("VROOLI_ENV", oldEnv)
			}
		}()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := corsMiddleware(handler)

		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Origin", "http://localhost:4000")

		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			t.Error("Expected CORS origin header in development mode")
		}
	})
}

// TestAuthMiddleware tests authentication handling
func TestAuthMiddleware(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("HealthEndpointNoAuth", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for health endpoint, got %d", w.Code)
		}
	})

	t.Run("OptionsRequestNoAuth", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := authMiddleware(handler)

		req := httptest.NewRequest("OPTIONS", "/api/test", nil)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 for OPTIONS, got %d", w.Code)
		}
	})

	t.Run("DevelopmentMode", func(t *testing.T) {
		oldEnv := os.Getenv("AUTH_MODE")
		os.Setenv("AUTH_MODE", "development")
		defer func() {
			if oldEnv == "" {
				os.Unsetenv("AUTH_MODE")
			} else {
				os.Setenv("AUTH_MODE", oldEnv)
			}
		}()

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrapped := authMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		wrapped.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200 in development mode, got %d", w.Code)
		}
	})
}
