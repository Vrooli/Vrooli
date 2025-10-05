package main

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

// TestAPIHandlerEdgeCases tests additional edge cases for API handlers
func TestAPIHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("handleHealthWithDatabase", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(server.handleHealth, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if status, ok := resp["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got %v", resp["status"])
		}
	})

	t.Run("handleHTTPRequestWithHeaders", func(t *testing.T) {
		mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Custom-Header") != "test-value" {
				t.Error("Custom header not received")
			}
			w.WriteHeader(http.StatusOK)
		})

		req := createTestHTTPRequest(mockServer.URL, "GET", map[string]string{
			"X-Custom-Header": "test-value",
		}, nil)

		w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http",
			Body:   req,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("handleHTTPRequestPOSTWithBody", func(t *testing.T) {
		mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST, got %s", r.Method)
			}
			w.WriteHeader(http.StatusCreated)
		})

		req := createTestHTTPRequest(mockServer.URL, "POST", nil, map[string]interface{}{
			"test": "data",
		})

		w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http",
			Body:   req,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		resp := assertSuccessResponse(t, w, http.StatusOK)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if statusCode, ok := data["status_code"].(float64); ok {
				if int(statusCode) != http.StatusCreated {
					t.Errorf("Expected status code 201, got %v", statusCode)
				}
			}
		}
	})

	t.Run("handleDNSQueryAAAARecord", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body: map[string]interface{}{
				"query":       "google.com",
				"record_type": "AAAA",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should succeed, fail gracefully, or reject unsupported types
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200, 400, or 500, got %d", w.Code)
		}
	})

	t.Run("handleDNSQueryNSRecord", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body: map[string]interface{}{
				"query":       "google.com",
				"record_type": "NS",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200, 400, or 500, got %d", w.Code)
		}
	})

	t.Run("handleConnectivityTestTraceroute", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/test/connectivity",
			Body: map[string]interface{}{
				"target":    "127.0.0.1",
				"test_type": "traceroute",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Traceroute may fail but should handle gracefully
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("handleNetworkScanMultiplePorts", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleNetworkScan, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/scan",
			Body: map[string]interface{}{
				"target": "127.0.0.1",
				"ports":  []int{80, 443, 8080},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("handleAPITestWithEndpoints", func(t *testing.T) {
		mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		})

		w, err := makeHTTPRequest(server.handleAPITest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/api/test",
			Body: map[string]interface{}{
				"base_url": mockServer.URL,
				"endpoints": []map[string]interface{}{
					{
						"path":   "/test",
						"method": "GET",
					},
				},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		assertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("handleSSLValidationExpiredCert", func(t *testing.T) {
		// Test with a URL that has SSL but may have issues
		w, err := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body: map[string]interface{}{
				"url": "https://expired.badssl.com/",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should handle gracefully with or without error
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})
}

// TestServerConfigurationEdgeCases tests server configuration edge cases
func TestServerConfigurationEdgeCases(t *testing.T) {
	t.Run("NewServerWithDatabaseURL", func(t *testing.T) {
		// This test documents the database URL patterns
		// Actual test is skipped if no DB configured
		dbURL := os.Getenv("DATABASE_URL")
		if dbURL == "" {
			t.Skip("No DATABASE_URL configured")
		}

		server, err := NewServer()
		if err != nil {
			t.Logf("NewServer failed: %v", err)
			return
		}
		defer server.db.Close()

		if server == nil {
			t.Error("Expected server to be created")
		}
	})
}

// TestHelperFunctionsComprehensive tests helper functions for full coverage
func TestHelperFunctionsComprehensive(t *testing.T) {
	t.Run("getEnvWithDefault", func(t *testing.T) {
		result := getEnv("NONEXISTENT_VAR_12345", "default_value")
		if result != "default_value" {
			t.Errorf("Expected 'default_value', got '%s'", result)
		}
	})

	t.Run("getEnvWithValue", func(t *testing.T) {
		os.Setenv("TEST_VAR_12345", "test_value")
		defer os.Unsetenv("TEST_VAR_12345")

		result := getEnv("TEST_VAR_12345", "default_value")
		if result != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", result)
		}
	})

	t.Run("mapToJSONWithComplexData", func(t *testing.T) {
		data := map[string]string{
			"string": "value",
			"key":    "value2",
		}

		result := mapToJSON(data)
		if result == "" {
			t.Error("Expected non-empty JSON string")
		}

		// Verify it's valid JSON
		var decoded map[string]string
		if err := json.Unmarshal([]byte(result), &decoded); err != nil {
			t.Errorf("Result is not valid JSON: %v", err)
		}
	})

	t.Run("getServiceNameAllPorts", func(t *testing.T) {
		testCases := []struct {
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
			{8080, "http-proxy"},
			{8443, "https-alt"},
			{9999, "unknown"},
		}

		for _, tc := range testCases {
			result := getServiceName(tc.port)
			if result != tc.expected {
				t.Errorf("getServiceName(%d) = %s, expected %s", tc.port, result, tc.expected)
			}
		}
	})
}
