package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestDatabaseHandlersWithMockDB tests database handlers with comprehensive scenarios
func TestDatabaseHandlersWithMockDB(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("handleListTargetsWithoutDB", func(t *testing.T) {
		server := env.Server
		server.db = nil // Ensure no database

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/network/targets",
		}

		w, err := makeHTTPRequest(server.handleListTargets, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 without database, got %d", w.Code)
		}

		var resp Response
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Success {
			t.Error("Expected success=false for database error")
		}
	})

	t.Run("handleCreateTargetWithoutDB", func(t *testing.T) {
		server := env.Server
		server.db = nil

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/targets",
			Body: map[string]interface{}{
				"name":        "Test Target",
				"target_type": "http",
				"address":     "example.com",
				"port":        80,
			},
		}

		w, err := makeHTTPRequest(server.handleCreateTarget, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 without database, got %d", w.Code)
		}
	})

	t.Run("handleCreateTargetInvalidBody", func(t *testing.T) {
		server := env.Server

		req := httptest.NewRequest("POST", "/api/v1/network/targets", bytes.NewBufferString("{invalid json"))
		w := httptest.NewRecorder()

		server.handleCreateTarget(w, req)

		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 400 or 500 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("handleListAlertsWithoutDB", func(t *testing.T) {
		server := env.Server
		server.db = nil

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/v1/network/alerts",
		}

		w, err := makeHTTPRequest(server.handleListAlerts, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 500 without database, got %d", w.Code)
		}
	})
}

// TestHelperFunctionsCoverage tests uncovered helper functions
func TestHelperFunctionsCoverage(t *testing.T) {
	t.Run("sendSuccess", func(t *testing.T) {
		w := httptest.NewRecorder()
		testData := map[string]string{"test": "data"}

		sendSuccess(w, testData)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp Response
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if !resp.Success {
			t.Error("Expected success=true")
		}
	})

	t.Run("sendError", func(t *testing.T) {
		w := httptest.NewRecorder()

		sendError(w, "test error", http.StatusBadRequest)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400, got %d", w.Code)
		}

		var resp Response
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Success {
			t.Error("Expected success=false")
		}

		if resp.Error != "test error" {
			t.Errorf("Expected error message 'test error', got '%s'", resp.Error)
		}
	})

	t.Run("getEnv", func(t *testing.T) {
		// Test with existing env var
		os.Setenv("TEST_VAR_GET_ENV", "test_value")
		defer os.Unsetenv("TEST_VAR_GET_ENV")

		result := getEnv("TEST_VAR_GET_ENV", "default")
		if result != "test_value" {
			t.Errorf("Expected 'test_value', got '%s'", result)
		}

		// Test with default value
		result = getEnv("NON_EXISTENT_VAR", "default_value")
		if result != "default_value" {
			t.Errorf("Expected 'default_value', got '%s'", result)
		}
	})

	t.Run("mapToJSON", func(t *testing.T) {
		testMap := map[string]string{
			"key1": "value1",
			"key2": "value2",
		}

		result := mapToJSON(testMap)

		var decoded map[string]string
		if err := json.Unmarshal([]byte(result), &decoded); err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}

		if decoded["key1"] != "value1" || decoded["key2"] != "value2" {
			t.Error("JSON encoding/decoding mismatch")
		}
	})

	t.Run("getServiceName", func(t *testing.T) {
		tests := []struct {
			port     int
			expected string
		}{
			{21, "ftp"},
			{22, "ssh"},
			{23, "telnet"},
			{25, "smtp"},
			{80, "http"},
			{443, "https"},
			{3306, "mysql"},
			{5432, "postgresql"},
			{8080, "http-proxy"},
			{8443, "https-alt"},
			{9999, "unknown"},
		}

		for _, tt := range tests {
			result := getServiceName(tt.port)
			if result != tt.expected {
				t.Errorf("getServiceName(%d) = %s, expected %s", tt.port, result, tt.expected)
			}
		}
	})
}

// TestAPITestHandlerEdgeCases tests additional edge cases
func TestAPITestHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("APITestInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/network/api/test", bytes.NewBufferString("{invalid"))
		w := httptest.NewRecorder()

		server.handleAPITest(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("APITestWithHeaders", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleAPITest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/api/test",
			Body: map[string]interface{}{
				"base_url": "https://httpbin.org",
				"test_suite": []map[string]interface{}{
					{
						"endpoint": "/get",
						"method":   "GET",
						"headers": map[string]string{
							"User-Agent": "NetworkTools/1.0",
						},
					},
				},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("APITestWithBody", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleAPITest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/api/test",
			Body: map[string]interface{}{
				"base_url": "https://httpbin.org",
				"test_suite": []map[string]interface{}{
					{
						"endpoint": "/post",
						"method":   "POST",
						"body":     map[string]string{"test": "data"},
					},
				},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

// TestSSLValidationHandlerEdgeCases tests additional SSL validation scenarios
func TestSSLValidationHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("SSLValidationInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/network/ssl/validate", bytes.NewBufferString("{invalid"))
		w := httptest.NewRecorder()

		server.handleSSLValidation(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("SSLValidationMissingURL", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body:   map[string]interface{}{},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing URL, got %d", w.Code)
		}
	})

	t.Run("SSLValidationInvalidURL", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body: map[string]interface{}{
				"url": "not-a-valid-url",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 400 or 500 for invalid URL, got %d", w.Code)
		}
	})

	t.Run("SSLValidationNonHTTPS", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body: map[string]interface{}{
				"url": "http://example.com",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for non-HTTPS URL, got %d", w.Code)
		}
	})
}

// TestHealthHandlerAdditional tests additional health check scenarios
func TestHealthHandlerAdditional(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("HealthWithNoDatabase", func(t *testing.T) {
		server.db = nil

		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w, err := makeHTTPRequest(server.handleHealth, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 even without database, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp["status"] != "healthy" {
			t.Error("Expected status=healthy")
		}
	})
}

// TestHTTPRequestHandlerEdgeCases tests additional HTTP request scenarios
func TestHTTPRequestHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("HTTPRequestInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/network/http/request", bytes.NewBufferString("{invalid"))
		w := httptest.NewRecorder()

		server.handleHTTPRequest(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("HTTPRequestWithTimeout", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http/request",
			Body: map[string]interface{}{
				"url":     "https://httpbin.org/delay/1",
				"method":  "GET",
				"timeout": 5,
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("HTTPRequestWithCustomHeaders", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http/request",
			Body: map[string]interface{}{
				"url":    "https://httpbin.org/headers",
				"method": "GET",
				"headers": map[string]string{
					"X-Custom-Header": "test-value",
					"User-Agent":      "NetworkTools/1.0",
				},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

// TestDNSQueryHandlerEdgeCases tests additional DNS query scenarios
func TestDNSQueryHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("DNSQueryInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/network/dns/query", bytes.NewBufferString("{invalid"))
		w := httptest.NewRecorder()

		server.handleDNSQuery(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("DNSQueryAAAARecord", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns/query",
			Body: map[string]interface{}{
				"query":       "google.com",
				"record_type": "AAAA",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// AAAA may or may not be supported
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200 or 400, got %d", w.Code)
		}
	})

	t.Run("DNSQueryNSRecord", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns/query",
			Body: map[string]interface{}{
				"query":       "google.com",
				"record_type": "NS",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// NS may or may not be supported
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200 or 400, got %d", w.Code)
		}
	})
}

// TestConnectivityHandlerEdgeCases tests additional connectivity scenarios
func TestConnectivityHandlerEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("ConnectivityInvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/network/connectivity/test", bytes.NewBufferString("{invalid"))
		w := httptest.NewRecorder()

		server.handleConnectivityTest(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("ConnectivityTraceroute", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/connectivity/test",
			Body: map[string]interface{}{
				"target":     "8.8.8.8",
				"test_type":  "traceroute",
				"max_hops":   15,
				"timeout_ms": 3000,
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Traceroute may succeed or fail depending on network/permissions
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})
}

// TestTestHelpersCoverage ensures test helper functions are covered
func TestTestHelpersCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	t.Run("makeHTTPRequestWithInvalidBody", func(t *testing.T) {
		// Test with a body that can't be marshaled
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/test",
			Body:   make(chan int), // channels can't be marshaled to JSON
		}

		server := env.Server
		_, err := makeHTTPRequest(server.handleHealth, req)
		if err == nil {
			t.Error("Expected error for invalid body type")
		}
	})

	t.Run("assertSuccessResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Response{
			Success: true,
			Data:    map[string]string{"test": "data"},
		})

		result := assertSuccessResponse(t, w, http.StatusOK)

		// Verify the data field exists
		if _, ok := result["data"]; !ok {
			t.Error("Expected data field in response")
		}
	})

	t.Run("assertErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Response{
			Success: false,
			Error:   "test error",
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "test")
	})

	t.Run("containsHelper", func(t *testing.T) {
		if !containsHelper("hello world", "world") {
			t.Error("Expected 'world' to be in 'hello world'")
		}

		if containsHelper("hello", "world") {
			t.Error("Expected 'world' not to be in 'hello'")
		}
	})

	t.Run("assertResponseField", func(t *testing.T) {
		data := map[string]interface{}{
			"status": "healthy",
			"count":  float64(42),
		}

		// Test string field
		assertResponseField(t, data, "status", "healthy")

		// Test numeric field
		assertResponseField(t, data, "count", float64(42))

		// Test with nil expected value (just checks existence)
		assertResponseField(t, data, "status", nil)
	})
}

// TestTestPatternsCoverage ensures test pattern functions are covered
func TestTestPatternsCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("TestScenarioBuilderMethods", func(t *testing.T) {
		builder := NewTestScenarioBuilder()

		// Test AddInvalidPortRange (currently at 0% coverage)
		builder.AddInvalidPortRange("/api/v1/network/scan")

		// Test AddCustom (currently at 0% coverage)
		customPattern := ErrorTestPattern{
			Name:           "CustomTest",
			Description:    "Custom test pattern",
			ExpectedStatus: http.StatusBadRequest,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/custom",
					Body:   map[string]interface{}{"data": "test"},
				}
			},
		}
		builder.AddCustom(customPattern)

		patterns := builder.Build()

		if len(patterns) != 2 {
			t.Errorf("Expected 2 patterns, got %d", len(patterns))
		}
	})

	t.Run("RunConcurrencyTest", func(t *testing.T) {
		pattern := ConcurrencyTestPattern{
			Name:        "ConcurrentHealthChecks",
			Description: "Test concurrent health check requests",
			Concurrency: 5,
			Iterations:  2,
			Setup: func(t *testing.T) interface{} {
				return server
			},
			Execute: func(t *testing.T, setupData interface{}, iteration int) error {
				req := HTTPTestRequest{
					Method: "GET",
					Path:   "/health",
				}
				_, err := makeHTTPRequest(server.handleHealth, req)
				return err
			},
			Validate: func(t *testing.T, setupData interface{}, results []error) {
				for i, err := range results {
					if err != nil {
						t.Errorf("Request %d failed: %v", i, err)
					}
				}
			},
		}

		RunConcurrencyTest(t, pattern)
	})
}
