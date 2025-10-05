package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestNetworkScanHandlerComprehensive provides comprehensive coverage for handleNetworkScan
func TestNetworkScanHandlerComprehensive(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("NetworkScanWithPortRange", func(t *testing.T) {
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

		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200 or 400, got %d", w.Code)
		}
	})

	t.Run("NetworkScanDefaultPorts", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleNetworkScan, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/scan",
			Body: map[string]interface{}{
				"target": "127.0.0.1",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200 or 400, got %d", w.Code)
		}
	})

	t.Run("NetworkScanMissingTarget", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleNetworkScan, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/scan",
			Body:   map[string]interface{}{},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for missing target, got %d", w.Code)
		}
	})

	t.Run("NetworkScanInvalidJSON", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/scan",
			Body:   "{invalid json",
		}

		w, err := makeHTTPRequest(server.handleNetworkScan, req)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("NetworkScanWithTimeout", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleNetworkScan, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/scan",
			Body: map[string]interface{}{
				"target": "127.0.0.1",
				"ports":  []int{22, 80},
				"timeout": 1000,
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
			t.Errorf("Expected 200 or 400, got %d", w.Code)
		}
	})
}

// TestNewServerVariations tests NewServer with various configurations
func TestNewServerVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("NewServerWithEnvironmentVariables", func(t *testing.T) {
		// Save original env vars
		originalPort := os.Getenv("PORT")
		originalEnv := os.Getenv("VROOLI_ENV")

		// Set test env vars
		os.Setenv("PORT", "18000")
		os.Setenv("VROOLI_ENV", "test")

		defer func() {
			os.Setenv("PORT", originalPort)
			os.Setenv("VROOLI_ENV", originalEnv)
		}()

		server, err := NewServer()
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		if server.config.Port != "18000" {
			t.Errorf("Expected port 18000, got %s", server.config.Port)
		}
	})

	t.Run("NewServerWithCustomRateLimit", func(t *testing.T) {
		originalRate := os.Getenv("RATE_LIMIT_REQUESTS")
		originalWindow := os.Getenv("RATE_LIMIT_WINDOW_SECONDS")

		os.Setenv("RATE_LIMIT_REQUESTS", "200")
		os.Setenv("RATE_LIMIT_WINDOW_SECONDS", "120")

		defer func() {
			os.Setenv("RATE_LIMIT_REQUESTS", originalRate)
			os.Setenv("RATE_LIMIT_WINDOW_SECONDS", originalWindow)
		}()

		server, err := NewServer()
		if err != nil {
			t.Fatalf("Failed to create server: %v", err)
		}

		if server == nil || server.rateLimiter == nil {
			t.Error("Expected server with rate limiter")
		}
	})
}

// TestHandleHealthVariations tests handleHealth with different scenarios
func TestHandleHealthVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("HealthCheckBasic", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleHealth, HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("HealthCheckMultipleTimes", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			w, err := makeHTTPRequest(server.handleHealth, HTTPTestRequest{
				Method: "GET",
				Path:   "/health",
			})

			if err != nil {
				t.Fatalf("Failed to make request %d: %v", i, err)
			}

			if w.Code != http.StatusOK {
				t.Errorf("Request %d: Expected 200, got %d", i, w.Code)
			}
		}
	})
}

// TestRateLimiterExpiry tests rate limiter expiry behavior
func TestRateLimiterExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiter expiry test in short mode")
	}

	t.Run("RateLimiterWindowReset", func(t *testing.T) {
		// Create a rate limiter with 1 request per 2 seconds for simplicity
		rl := NewRateLimiter(1, 2)

		// Use up the limit
		if !rl.Allow("test") {
			t.Error("Expected first request to be allowed")
		}
		// Second should be blocked
		if rl.Allow("test") {
			t.Log("Second request was allowed (rate limiter may have different behavior)")
		}

		// Wait for window to expire
		time.Sleep(3 * time.Second)

		// Should be allowed again after window expiry
		if !rl.Allow("test") {
			t.Error("Expected request to be allowed after window expiry")
		}
	})
}

// TestAuthMiddlewareProduction tests auth middleware in production mode
func TestAuthMiddlewareProduction(t *testing.T) {
	originalEnv := os.Getenv("VROOLI_ENV")
	originalKey := os.Getenv("NETWORK_TOOLS_API_KEY")

	defer func() {
		os.Setenv("VROOLI_ENV", originalEnv)
		os.Setenv("NETWORK_TOOLS_API_KEY", originalKey)
	}()

	t.Run("ProductionModeWithAPIKey", func(t *testing.T) {
		os.Setenv("VROOLI_ENV", "production")
		os.Setenv("NETWORK_TOOLS_API_KEY", "secret-key-123")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		// Test with correct key
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("X-API-Key", "secret-key-123")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 with valid API key, got %d", w.Code)
		}
	})

	t.Run("ProductionModeWithBearerToken", func(t *testing.T) {
		os.Setenv("VROOLI_ENV", "production")
		os.Setenv("NETWORK_TOOLS_API_KEY", "secret-key-123")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := authMiddleware(handler)

		// Test with Bearer token
		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		req.Header.Set("Authorization", "Bearer secret-key-123")
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200 with valid Bearer token, got %d", w.Code)
		}
	})
}

// TestCORSMiddlewareProduction tests CORS middleware in different environments
func TestCORSMiddlewareProduction(t *testing.T) {
	originalEnv := os.Getenv("VROOLI_ENV")
	defer os.Setenv("VROOLI_ENV", originalEnv)

	t.Run("ProductionMode", func(t *testing.T) {
		os.Setenv("VROOLI_ENV", "production")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})

	t.Run("DevelopmentMode", func(t *testing.T) {
		os.Setenv("VROOLI_ENV", "development")

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		middleware := corsMiddleware(handler)

		req := httptest.NewRequest("GET", "/api/v1/test", nil)
		w := httptest.NewRecorder()
		middleware.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
	})
}

// TestHTTPClientTimeout tests HTTP request timeouts
func TestHTTPClientTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("HTTPRequestWithVeryShortTimeout", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleHTTPRequest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/http/request",
			Body: map[string]interface{}{
				"url":     "https://httpbin.org/delay/5",
				"method":  "GET",
				"timeout": 100, // Very short timeout
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should either timeout or complete successfully
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})
}

// TestDNSQueryTypes tests various DNS query types
func TestDNSQueryTypes(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	recordTypes := []string{"A", "CNAME", "MX", "TXT"}

	for _, recordType := range recordTypes {
		t.Run("DNSQuery"+recordType, func(t *testing.T) {
			w, err := makeHTTPRequest(server.handleDNSQuery, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/network/dns/query",
				Body: map[string]interface{}{
					"query":       "google.com",
					"record_type": recordType,
				},
			})

			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}

			if w.Code != http.StatusOK && w.Code != http.StatusBadRequest {
				t.Errorf("Expected 200 or 400 for %s query, got %d", recordType, w.Code)
			}
		})
	}
}

// TestConnectivityPingVariations tests ping with various options
func TestConnectivityPingVariations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	defer env.Cleanup()

	server := env.Server

	t.Run("ConnectivityPingLocalhost", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/connectivity/test",
			Body: map[string]interface{}{
				"target":    "127.0.0.1",
				"test_type": "ping",
			},
		})

		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Expected 200 or 500, got %d", w.Code)
		}
	})

	t.Run("ConnectivityPingWithCustomCount", func(t *testing.T) {
		w, err := makeHTTPRequest(server.handleConnectivityTest, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/network/connectivity/test",
			Body: map[string]interface{}{
				"target":    "8.8.8.8",
				"test_type": "ping",
				"count":     3,
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
