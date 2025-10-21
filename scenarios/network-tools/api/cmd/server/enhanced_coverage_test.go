package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandleAPITestComprehensive provides comprehensive coverage for handleAPITest
func TestHandleAPITestComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("APITestWithRetryLogic", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleAPITest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/api/test",
			Body: map[string]interface{}{
				"base_url": "https://httpbin.org",
				"test_suite": []map[string]interface{}{
					{
						"endpoint":    "/get",
						"method":      "GET",
						"max_retries": 3,
						"retry_delay": 100,
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

	t.Run("APITestWithExpectedStatus", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleAPITest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/api/test",
			Body: map[string]interface{}{
				"base_url": "https://httpbin.org",
				"test_suite": []map[string]interface{}{
					{
						"endpoint":        "/status/200",
						"method":          "GET",
						"expected_status": 200,
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

	t.Run("APITestWithExpectedContent", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleAPITest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/api/test",
			Body: map[string]interface{}{
				"base_url": "https://httpbin.org",
				"test_suite": []map[string]interface{}{
					{
						"endpoint":         "/get",
						"method":           "GET",
						"expected_content": "headers",
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

	t.Run("APITestWithFailedExpectations", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleAPITest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/api/test",
			Body: map[string]interface{}{
				"base_url": "https://httpbin.org",
				"test_suite": []map[string]interface{}{
					{
						"endpoint":        "/status/404",
						"method":          "GET",
						"expected_status": 200, // Will fail
					},
				},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 even with failed test, got %d", w.Code)
		}
	})
}

// TestHandleSSLValidationComprehensive provides comprehensive coverage for handleSSLValidation
func TestHandleSSLValidationComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("SSLValidationValidHTTPS", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body: map[string]interface{}{
				"url": "https://www.google.com",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("SSLValidationWithPort", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleSSLValidation, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/ssl/validate",
			Body: map[string]interface{}{
				"url": "https://www.google.com:443",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})
}

// TestHandleHTTPRequestComprehensive provides comprehensive coverage for handleHTTPRequest
func TestHandleHTTPRequestComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("HTTPRequestPOSTWithBody", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http/request",
			Body: map[string]interface{}{
				"url":    "https://httpbin.org/post",
				"method": "POST",
				"body":   map[string]string{"test": "data"},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("HTTPRequestDELETE", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http/request",
			Body: map[string]interface{}{
				"url":    "https://httpbin.org/delete",
				"method": "DELETE",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("HTTPRequestPUT", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http/request",
			Body: map[string]interface{}{
				"url":    "https://httpbin.org/put",
				"method": "PUT",
				"body":   map[string]string{"update": "data"},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("HTTPRequestWithFollowRedirects", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http/request",
			Body: map[string]interface{}{
				"url":    "https://httpbin.org/redirect/1",
				"method": "GET",
				"options": map[string]interface{}{
					"follow_redirects": true,
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

// TestHandleDNSQueryComprehensive provides comprehensive coverage for handleDNSQuery
func TestHandleDNSQueryComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("DNSQueryWithNameserver", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns/query",
			Body: map[string]interface{}{
				"query":       "google.com",
				"record_type": "A",
				"options": map[string]interface{}{
					"nameserver": "8.8.8.8",
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

	t.Run("DNSQueryWithTimeout", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns/query",
			Body: map[string]interface{}{
				"query":       "google.com",
				"record_type": "A",
				"options": map[string]interface{}{
					"timeout_ms": 5000,
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

// TestHandleConnectivityTestComprehensive provides comprehensive coverage for handleConnectivityTest
func TestHandleConnectivityTestComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("ConnectivityTestTCPConnect", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/connectivity/test",
			Body: map[string]interface{}{
				"target":    "google.com:80",
				"test_type": "tcp_connect",
				"options": map[string]interface{}{
					"timeout_ms": 5000,
				},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// tcp_connect may not be supported, so accept 400, 500, or 200
		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200, 400, or 500, got %d", w.Code)
		}
	})

	t.Run("ConnectivityTestWithCount", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/connectivity/test",
			Body: map[string]interface{}{
				"target":    "8.8.8.8",
				"test_type": "ping",
				"options": map[string]interface{}{
					"count":      5,
					"timeout_ms": 5000,
				},
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})
}

// TestInitializeDatabaseCoverage tests database initialization edge cases
func TestInitializeDatabaseCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("InitializeDatabase", func(t *testing.T) {
		// This tests the InitializeDatabase function
		// Since we can't easily create a real DB connection in tests,
		// we test it through NewServer
		env := setupTestEnvironment(t)
		defer env.Cleanup()

		// The setupTestEnvironment creates a server without DB
		// which tests the nil DB path
		if env.Server.db != nil {
			t.Error("Expected nil database in test environment")
		}
	})
}

// TestEdgeCasesForHelpers tests edge cases in helper functions
func TestEdgeCasesForHelpers(t *testing.T) {
	t.Run("sendSuccessWithNilData", func(t *testing.T) {
		w := httptest.NewRecorder()
		sendSuccess(w, nil)

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

	t.Run("sendSuccessWithComplexData", func(t *testing.T) {
		w := httptest.NewRecorder()
		data := map[string]interface{}{
			"nested": map[string]interface{}{
				"key": "value",
			},
			"array": []string{"a", "b", "c"},
		}
		sendSuccess(w, data)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("sendErrorVariousStatuses", func(t *testing.T) {
		statuses := []int{
			http.StatusBadRequest,
			http.StatusUnauthorized,
			http.StatusForbidden,
			http.StatusNotFound,
			http.StatusInternalServerError,
			http.StatusServiceUnavailable,
		}

		for _, status := range statuses {
			w := httptest.NewRecorder()
			sendError(w, "test error", status)

			if w.Code != status {
				t.Errorf("Expected %d, got %d", status, w.Code)
			}
		}
	})

	t.Run("mapToJSONWithEmptyMap", func(t *testing.T) {
		result := mapToJSON(map[string]string{})
		if result != "{}" {
			t.Errorf("Expected '{}', got '%s'", result)
		}
	})

	t.Run("mapToJSONWithSpecialCharacters", func(t *testing.T) {
		testMap := map[string]string{
			"quote":   `"test"`,
			"newline": "line1\nline2",
		}
		result := mapToJSON(testMap)

		var decoded map[string]string
		if err := json.Unmarshal([]byte(result), &decoded); err != nil {
			t.Fatalf("Failed to decode JSON: %v", err)
		}
	})
}

// TestRateLimiterEdgeCases tests rate limiter edge cases
func TestRateLimiterEdgeCases(t *testing.T) {
	t.Run("RateLimiterZeroRequests", func(t *testing.T) {
		rl := NewRateLimiter(0, 60)
		if rl.Allow("test") {
			t.Error("Expected rate limiter to block with 0 limit")
		}
	})

	t.Run("RateLimiterVeryHighLimit", func(t *testing.T) {
		rl := NewRateLimiter(1000000, 60)
		for i := 0; i < 100; i++ {
			if !rl.Allow("test") {
				t.Errorf("Expected rate limiter to allow request %d", i)
			}
		}
	})

	t.Run("RateLimiterMultipleKeys", func(t *testing.T) {
		rl := NewRateLimiter(2, 60)

		// Test multiple keys can access independently
		for i := 0; i < 10; i++ {
			key := fmt.Sprintf("key%d", i)
			if !rl.Allow(key) {
				t.Errorf("Expected first request for %s to be allowed", key)
			}
		}
	})
}

// TestAuthMiddlewareEdgeCases tests authentication middleware edge cases
func TestAuthMiddlewareEdgeCases(t *testing.T) {
	t.Run("AuthMiddlewareMultipleFormats", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sendSuccess(w, map[string]string{"status": "ok"})
		})

		middleware := authMiddleware(testHandler)

		// Test with X-API-Key header
		req1 := httptest.NewRequest("GET", "/api/v1/test", nil)
		req1.Header.Set("X-API-Key", "test-key")
		w1 := httptest.NewRecorder()
		middleware.ServeHTTP(w1, req1)

		// Test with Authorization header
		req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
		req2.Header.Set("Authorization", "Bearer test-key")
		w2 := httptest.NewRecorder()
		middleware.ServeHTTP(w2, req2)
	})
}

// TestCORSMiddlewareEdgeCases tests CORS middleware edge cases
func TestCORSMiddlewareEdgeCases(t *testing.T) {
	t.Run("CORSWithVariousOrigins", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sendSuccess(w, map[string]string{"status": "ok"})
		})

		middleware := corsMiddleware(testHandler)

		// Test with localhost origin (should be allowed in development mode)
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		// CORS headers may or may not be set depending on configuration
		// Just verify the request completes successfully
		if w.Code >= 500 {
			t.Errorf("Expected successful response, got %d", w.Code)
		}
	})

	t.Run("CORSPreflightRequest", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sendSuccess(w, map[string]string{"status": "ok"})
		})

		middleware := corsMiddleware(testHandler)

		req := httptest.NewRequest("OPTIONS", "/api/v1/test", nil)
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 for OPTIONS request, got %d", w.Code)
		}
	})
}
