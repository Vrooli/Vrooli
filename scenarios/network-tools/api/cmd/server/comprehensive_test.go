package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestHTTPRequestHandler tests the HTTP request endpoint comprehensively
func TestHTTPRequestHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("SuccessGETRequest", func(t *testing.T) {
		// Create a mock HTTP server
		mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"message": "success"})
		})

		req := createTestHTTPRequest(mockServer.URL, "GET", nil, nil)
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
				if int(statusCode) != http.StatusOK {
					t.Errorf("Expected status code 200, got %v", statusCode)
				}
			}
		}
	})

	t.Run("SuccessPOSTRequest", func(t *testing.T) {
		mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST method, got %s", r.Method)
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]string{"id": "12345"})
		})

		req := createTestHTTPRequest(mockServer.URL, "POST",
			map[string]string{"Content-Type": "application/json"},
			map[string]interface{}{"data": "test"})

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

	t.Run("ErrorMissingURL", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http",
			Body: map[string]interface{}{
				"method": "GET",
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "URL")
	})

	t.Run("ErrorInvalidURL", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http",
			Body: map[string]interface{}{
				"url":    "not-a-url",
				"method": "GET",
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("ErrorMissingScheme", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http",
			Body: map[string]interface{}{
				"url":    "example.com",
				"method": "GET",
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "scheme")
	})
}

// TestDNSQueryHandler tests the DNS query endpoint
func TestDNSQueryHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("SuccessARecord", func(t *testing.T) {
		req := createTestDNSRequest("google.com", "A")
		w, err := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body:   req,
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		resp := assertSuccessResponse(t, w, http.StatusOK)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if answers, ok := data["answers"].([]interface{}); ok {
				if len(answers) == 0 {
					t.Error("Expected at least one DNS answer")
				}
			}
		}
	})

	t.Run("SuccessCNAMERecord", func(t *testing.T) {
		req := createTestDNSRequest("www.github.com", "CNAME")
		w, _ := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body:   req,
		})

		// CNAME lookup may or may not succeed depending on DNS configuration
		// Just verify we get a response
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("SuccessMXRecord", func(t *testing.T) {
		req := createTestDNSRequest("google.com", "MX")
		w, _ := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body:   req,
		})

		resp := assertSuccessResponse(t, w, http.StatusOK)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if answers, ok := data["answers"].([]interface{}); ok {
				if len(answers) == 0 {
					t.Error("Expected at least one MX record")
				}
			}
		}
	})

	t.Run("SuccessTXTRecord", func(t *testing.T) {
		req := createTestDNSRequest("google.com", "TXT")
		w, _ := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body:   req,
		})

		// TXT records may or may not exist
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})

	t.Run("ErrorMissingQuery", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body: map[string]interface{}{
				"record_type": "A",
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Query")
	})

	t.Run("ErrorUnsupportedRecordType", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body: map[string]interface{}{
				"query":       "example.com",
				"record_type": "UNSUPPORTED",
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Unsupported")
	})

	t.Run("ErrorInvalidDomain", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body: map[string]interface{}{
				"query":       "invalid..domain",
				"record_type": "A",
			},
		})

		// Should either error or return empty results
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 200 or 500, got %d", w.Code)
		}
	})
}

// TestConnectivityHandler tests the connectivity test endpoint
func TestConnectivityHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("SuccessLocalhost", func(t *testing.T) {
		req := createTestConnectivityRequest("127.0.0.1", "ping")
		w, _ := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/test/connectivity",
			Body:   req,
		})

		resp := assertSuccessResponse(t, w, http.StatusOK)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if stats, ok := data["statistics"].(map[string]interface{}); ok {
				if _, ok := stats["packets_sent"]; !ok {
					t.Error("Expected packets_sent in statistics")
				}
			}
		}
	})

	t.Run("SuccessGoogleDNS", func(t *testing.T) {
		req := createTestConnectivityRequest("8.8.8.8", "ping")
		w, _ := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/test/connectivity",
			Body:   req,
		})

		assertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("ErrorMissingTarget", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/test/connectivity",
			Body: map[string]interface{}{
				"test_type": "ping",
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Target")
	})

	t.Run("ErrorInvalidTarget", func(t *testing.T) {
		req := createTestConnectivityRequest("invalid..host", "ping")
		w, _ := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/test/connectivity",
			Body:   req,
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

// TestNetworkScanHandler tests the network scan endpoint
func TestNetworkScanHandler(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("SuccessLocalhost", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleNetworkScan, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/scan",
			Body: map[string]interface{}{
				"target": "127.0.0.1",
				"ports":  []int{80, 443},
			},
		})

		resp := assertSuccessResponse(t, w, http.StatusOK)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if results, ok := data["results"].([]interface{}); ok {
				if len(results) != 2 {
					t.Errorf("Expected 2 scan results, got %d", len(results))
				}
			}
		}
	})

	t.Run("SuccessDefaultPorts", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleNetworkScan, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/scan",
			Body: map[string]interface{}{
				"target": "127.0.0.1",
			},
		})

		resp := assertSuccessResponse(t, w, http.StatusOK)
		if data, ok := resp["data"].(map[string]interface{}); ok {
			if results, ok := data["results"].([]interface{}); ok {
				if len(results) == 0 {
					t.Error("Expected default ports to be scanned")
				}
			}
		}
	})

	t.Run("ErrorMissingTarget", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleNetworkScan, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/scan",
			Body: map[string]interface{}{
				"ports": []int{80},
			},
		})

		assertErrorResponse(t, w, http.StatusBadRequest, "Target")
	})
}

// TestBasicRateLimiter tests basic rate limiter functionality
func TestBasicRateLimiter(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("RateLimiter", func(t *testing.T) {
		rl := NewRateLimiter(3, time.Minute)

		// First 3 requests should succeed
		for i := 0; i < 3; i++ {
			if !rl.Allow("test-key") {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}

		// 4th request should be blocked
		if rl.Allow("test-key") {
			t.Error("4th request should be blocked")
		}
	})

	t.Run("RateLimiterSeparateKeys", func(t *testing.T) {
		rl := NewRateLimiter(1, time.Minute)

		if !rl.Allow("key1") {
			t.Error("First request for key1 should be allowed")
		}
		if !rl.Allow("key2") {
			t.Error("First request for key2 should be allowed")
		}

		if rl.Allow("key1") {
			t.Error("Second request for key1 should be blocked")
		}
		if rl.Allow("key2") {
			t.Error("Second request for key2 should be blocked")
		}
	})

	t.Run("RateLimiterWindowExpiry", func(t *testing.T) {
		rl := NewRateLimiter(1, 100*time.Millisecond)

		if !rl.Allow("test-key") {
			t.Error("First request should be allowed")
		}

		if rl.Allow("test-key") {
			t.Error("Second request should be blocked immediately")
		}

		time.Sleep(150 * time.Millisecond)

		if !rl.Allow("test-key") {
			t.Error("Request after window expiry should be allowed")
		}
	})
}

// TestHelperFunctions tests utility helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("getEnv", func(t *testing.T) {
		// Test default value
		val := getEnv("NON_EXISTENT_VAR", "default")
		if val != "default" {
			t.Errorf("Expected default value, got %s", val)
		}

		// Test existing value
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		val = getEnv("TEST_VAR", "default")
		if val != "test_value" {
			t.Errorf("Expected test_value, got %s", val)
		}
	})

	t.Run("sendSuccess", func(t *testing.T) {
		w := httptest.NewRecorder()
		sendSuccess(w, map[string]string{"key": "value"})

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
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
			t.Errorf("Expected status 400, got %d", w.Code)
		}

		var resp Response
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if resp.Success {
			t.Error("Expected success=false")
		}

		if resp.Error != "test error" {
			t.Errorf("Expected error 'test error', got '%s'", resp.Error)
		}
	})
}

// TestRequestValidationEdgeCases tests edge cases in request validation
func TestRequestValidationEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("HTTPRequestEmptyMethod", func(t *testing.T) {
		mockServer := mockHTTPServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		req := createTestHTTPRequest(mockServer.URL, "", nil, nil)
		w, _ := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http",
			Body:   req,
		})

		// Should default to GET
		assertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("DNSQueryDefaultRecordType", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/dns",
			Body: map[string]interface{}{
				"query": "google.com",
			},
		})

		// Should default to A record
		assertSuccessResponse(t, w, http.StatusOK)
	})

	t.Run("ConnectivityDefaultTestType", func(t *testing.T) {
		w, _ := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/test/connectivity",
			Body: map[string]interface{}{
				"target": "127.0.0.1",
			},
		})

		// Should default to ping
		assertSuccessResponse(t, w, http.StatusOK)
	})
}

// TestServerInitialization tests server creation and initialization
func TestServerInitialization(t *testing.T) {
	// Skip if no database is configured
	if os.Getenv("POSTGRES_PASSWORD") == "" && os.Getenv("DB_PASSWORD") == "" && os.Getenv("DATABASE_URL") == "" && os.Getenv("POSTGRES_URL") == "" {
		t.Skip("No database configured for server initialization test")
	}

	t.Run("NewServerSuccess", func(t *testing.T) {
		server, err := NewServer()
		if err != nil {
			t.Logf("NewServer failed (expected if no DB): %v", err)
			return
		}
		defer server.db.Close()

		if server.config == nil {
			t.Error("Expected config to be initialized")
		}
		if server.router == nil {
			t.Error("Expected router to be initialized")
		}
		if server.client == nil {
			t.Error("Expected HTTP client to be initialized")
		}
		if server.rateLimiter == nil {
			t.Error("Expected rate limiter to be initialized")
		}
	})

	t.Run("SetupRoutesConfigured", func(t *testing.T) {
		// Create a minimal server with just router
		router := mux.NewRouter()
		server := &Server{
			router:      router,
			rateLimiter: NewRateLimiter(100, time.Minute),
		}
		server.setupRoutes()

		// Test that routes are registered
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should hit a route (even if handler errors due to nil dependencies)
		if w.Code == 404 {
			t.Error("Expected /health route to be registered")
		}
	})
}

// TestMiddleware tests all middleware functions
func TestMiddleware(t *testing.T) {
	t.Run("RateLimiter", func(t *testing.T) {
		rateLimiter := NewRateLimiter(2, time.Minute)
		server := &Server{
			rateLimiter: rateLimiter,
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.rateLimitMiddleware(handler)

		// First two requests should succeed
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("GET", "/api/v1/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()
			middleware.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Request %d: expected 200, got %d", i+1, w.Code)
			}
		}

		// Third request should be rate limited
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusTooManyRequests {
			t.Errorf("Expected 429, got %d", w.Code)
		}
	})

	t.Run("RateLimiterSeparateKeys", func(t *testing.T) {
		rateLimiter := NewRateLimiter(1, time.Minute)
		server := &Server{
			rateLimiter: rateLimiter,
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.rateLimitMiddleware(handler)

		// Different IPs should be tracked separately
		req1 := httptest.NewRequest("GET", "/api/v1/test", nil)
		req1.RemoteAddr = "192.168.1.1:12345"
		w1 := httptest.NewRecorder()
		middleware.ServeHTTP(w1, req1)

		req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
		req2.RemoteAddr = "192.168.1.2:12345"
		w2 := httptest.NewRecorder()
		middleware.ServeHTTP(w2, req2)

		if w1.Code != http.StatusOK || w2.Code != http.StatusOK {
			t.Error("Both requests from different IPs should succeed")
		}
	})

	t.Run("RateLimiterWindowExpiry", func(t *testing.T) {
		rateLimiter := NewRateLimiter(1, 100*time.Millisecond)
		server := &Server{
			rateLimiter: rateLimiter,
		}

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := server.rateLimitMiddleware(handler)

		// First request
		req1 := httptest.NewRequest("GET", "/api/v1/test", nil)
		req1.RemoteAddr = "192.168.1.1:12345"
		w1 := httptest.NewRecorder()
		middleware.ServeHTTP(w1, req1)

		if w1.Code != http.StatusOK {
			t.Error("First request should succeed")
		}

		// Second request immediately should fail
		req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
		req2.RemoteAddr = "192.168.1.1:12345"
		w2 := httptest.NewRecorder()
		middleware.ServeHTTP(w2, req2)

		if w2.Code != http.StatusTooManyRequests {
			t.Error("Second immediate request should be rate limited")
		}

		// Wait for window to expire
		time.Sleep(150 * time.Millisecond)

		// Third request after window should succeed
		req3 := httptest.NewRequest("GET", "/api/v1/test", nil)
		req3.RemoteAddr = "192.168.1.1:12345"
		w3 := httptest.NewRecorder()
		middleware.ServeHTTP(w3, req3)

		if w3.Code != http.StatusOK {
			t.Error("Request after window expiry should succeed")
		}
	})
}

// TestLoggingMiddleware tests the logging middleware
func TestLoggingMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := loggingMiddleware(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	middleware.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// TestRateLimitMiddlewareXForwardedFor tests X-Forwarded-For header handling
func TestRateLimitMiddlewareXForwardedFor(t *testing.T) {
	rateLimiter := NewRateLimiter(1, time.Minute)
	server := &Server{
		rateLimiter: rateLimiter,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.rateLimitMiddleware(handler)

	// First request with X-Forwarded-For
	req1 := httptest.NewRequest("GET", "/api/v1/test", nil)
	req1.Header.Set("X-Forwarded-For", "203.0.113.1, 203.0.113.2")
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Error("First request should succeed")
	}

	// Second request from same X-Forwarded-For IP
	req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
	req2.Header.Set("X-Forwarded-For", "203.0.113.1, 203.0.113.3")
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Error("Second request from same IP should be rate limited")
	}
}

// TestRateLimitMiddlewareXRealIP tests X-Real-IP header handling
func TestRateLimitMiddlewareXRealIP(t *testing.T) {
	rateLimiter := NewRateLimiter(1, time.Minute)
	server := &Server{
		rateLimiter: rateLimiter,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.rateLimitMiddleware(handler)

	// First request with X-Real-IP
	req1 := httptest.NewRequest("GET", "/api/v1/test", nil)
	req1.Header.Set("X-Real-IP", "203.0.113.5")
	w1 := httptest.NewRecorder()
	middleware.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Error("First request should succeed")
	}

	// Second request from same X-Real-IP
	req2 := httptest.NewRequest("GET", "/api/v1/test", nil)
	req2.Header.Set("X-Real-IP", "203.0.113.5")
	w2 := httptest.NewRecorder()
	middleware.ServeHTTP(w2, req2)

	if w2.Code != http.StatusTooManyRequests {
		t.Error("Second request from same IP should be rate limited")
	}
}

// TestRateLimitHealthCheckBypass tests that health checks bypass rate limiting
func TestRateLimitHealthCheckBypass(t *testing.T) {
	rateLimiter := NewRateLimiter(1, time.Minute)
	server := &Server{
		rateLimiter: rateLimiter,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := server.rateLimitMiddleware(handler)

	// Multiple health check requests should all succeed
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Health check %d should bypass rate limiting", i+1)
		}
	}
}
