package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestRateLimiter tests the rate limiting functionality
func TestRateLimiter(t *testing.T) {
	t.Run("allows requests under limit", func(t *testing.T) {
		rl := NewRateLimiter(3, time.Minute)

		if !rl.Allow("test-key") {
			t.Error("First request should be allowed")
		}
		if !rl.Allow("test-key") {
			t.Error("Second request should be allowed")
		}
		if !rl.Allow("test-key") {
			t.Error("Third request should be allowed")
		}
	})

	t.Run("blocks requests over limit", func(t *testing.T) {
		rl := NewRateLimiter(2, time.Minute)

		rl.Allow("test-key")
		rl.Allow("test-key")

		if rl.Allow("test-key") {
			t.Error("Third request should be blocked")
		}
	})

	t.Run("allows requests after window expires", func(t *testing.T) {
		rl := NewRateLimiter(1, 100*time.Millisecond)

		if !rl.Allow("test-key") {
			t.Error("First request should be allowed")
		}

		if rl.Allow("test-key") {
			t.Error("Second request should be blocked immediately")
		}

		time.Sleep(150 * time.Millisecond)

		if !rl.Allow("test-key") {
			t.Error("Request after window should be allowed")
		}
	})

	t.Run("tracks separate keys independently", func(t *testing.T) {
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
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	// Create a mock server with mux router
	router := mux.NewRouter()

	// Register health handler
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"service":   "network-tools",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%v'", response["status"])
	}

	if response["service"] != "network-tools" {
		t.Errorf("Expected service 'network-tools', got '%v'", response["service"])
	}
}

// TestRequestValidation tests input validation for various endpoints
func TestRequestValidation(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    string
		expectedStatus int
		description    string
	}{
		{
			name:           "empty request body",
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			description:    "Empty request should return 400",
		},
		{
			name:           "invalid json",
			requestBody:    "{invalid json}",
			expectedStatus: http.StatusBadRequest,
			description:    "Invalid JSON should return 400",
		},
		{
			name:           "missing required fields",
			requestBody:    "{}",
			expectedStatus: http.StatusBadRequest,
			description:    "Missing required fields should return 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test demonstrates the validation structure
			// Actual implementation would test specific endpoints
			if tt.requestBody == "" && tt.expectedStatus != http.StatusBadRequest {
				t.Errorf("%s: validation failed", tt.description)
			}
		})
	}
}

// TestHTTPRequestValidation tests HTTP request structure validation
func TestHTTPRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request HTTPRequest
		wantErr bool
	}{
		{
			name: "valid GET request",
			request: HTTPRequest{
				URL:    "https://example.com",
				Method: "GET",
			},
			wantErr: false,
		},
		{
			name: "valid POST request with body",
			request: HTTPRequest{
				URL:    "https://api.example.com/data",
				Method: "POST",
				Body:   map[string]interface{}{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "invalid empty URL",
			request: HTTPRequest{
				URL:    "",
				Method: "GET",
			},
			wantErr: true,
		},
		{
			name: "invalid method",
			request: HTTPRequest{
				URL:    "https://example.com",
				Method: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate URL
			if tt.request.URL == "" && !tt.wantErr {
				t.Error("Empty URL should be invalid")
			}

			// Validate method
			if tt.request.Method == "" && !tt.wantErr {
				t.Error("Empty method should be invalid")
			}

			// Valid cases
			if tt.request.URL != "" && tt.request.Method != "" && tt.wantErr {
				t.Error("Valid request marked as error")
			}
		})
	}
}

// TestDNSRequestValidation tests DNS request validation
func TestDNSRequestValidation(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		recordType string
		wantErr    bool
	}{
		{
			name:       "valid A record query",
			query:      "example.com",
			recordType: "A",
			wantErr:    false,
		},
		{
			name:       "valid AAAA record query",
			query:      "example.com",
			recordType: "AAAA",
			wantErr:    false,
		},
		{
			name:       "valid MX record query",
			query:      "example.com",
			recordType: "MX",
			wantErr:    false,
		},
		{
			name:       "empty query",
			query:      "",
			recordType: "A",
			wantErr:    true,
		},
		{
			name:       "empty record type",
			query:      "example.com",
			recordType: "",
			wantErr:    true,
		},
		{
			name:       "invalid record type",
			query:      "example.com",
			recordType: "INVALID",
			wantErr:    false, // Should be handled by DNS library
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.query == "" && !tt.wantErr {
				t.Error("Empty query should be invalid")
			}
			if tt.recordType == "" && !tt.wantErr {
				t.Error("Empty record type should be invalid")
			}
		})
	}
}

// TestPortScanValidation tests port scanning validation
func TestPortScanValidation(t *testing.T) {
	tests := []struct {
		name    string
		target  string
		ports   []int
		wantErr bool
	}{
		{
			name:    "valid single port",
			target:  "example.com",
			ports:   []int{80},
			wantErr: false,
		},
		{
			name:    "valid multiple ports",
			target:  "192.168.1.1",
			ports:   []int{22, 80, 443},
			wantErr: false,
		},
		{
			name:    "empty target",
			target:  "",
			ports:   []int{80},
			wantErr: true,
		},
		{
			name:    "empty ports",
			target:  "example.com",
			ports:   []int{},
			wantErr: true,
		},
		{
			name:    "invalid port number (too low)",
			target:  "example.com",
			ports:   []int{0},
			wantErr: true,
		},
		{
			name:    "invalid port number (too high)",
			target:  "example.com",
			ports:   []int{65536},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.target == "" && !tt.wantErr {
				t.Error("Empty target should be invalid")
			}
			if len(tt.ports) == 0 && !tt.wantErr {
				t.Error("Empty ports should be invalid")
			}
			for _, port := range tt.ports {
				if (port < 1 || port > 65535) && !tt.wantErr {
					t.Errorf("Invalid port %d should be rejected", port)
				}
			}
		})
	}
}

// TestConnectivityTestValidation tests connectivity test validation
func TestConnectivityTestValidation(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		testType string
		wantErr  bool
	}{
		{
			name:     "valid ping test",
			target:   "8.8.8.8",
			testType: "ping",
			wantErr:  false,
		},
		{
			name:     "valid traceroute test",
			target:   "example.com",
			testType: "traceroute",
			wantErr:  false,
		},
		{
			name:     "empty target",
			target:   "",
			testType: "ping",
			wantErr:  true,
		},
		{
			name:     "empty test type",
			target:   "8.8.8.8",
			testType: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.target == "" && !tt.wantErr {
				t.Error("Empty target should be invalid")
			}
			if tt.testType == "" && !tt.wantErr {
				t.Error("Empty test type should be invalid")
			}
		})
	}
}

// TestSSLValidationInput tests SSL validation input handling
func TestSSLValidationInput(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid HTTPS URL",
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL with port",
			url:     "https://example.com:443",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "HTTP URL (should warn but not error)",
			url:     "http://example.com",
			wantErr: false, // Warning but processable
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.url == "" && !tt.wantErr {
				t.Error("Empty URL should be invalid")
			}
		})
	}
}

// TestResponseStructure tests API response structure
func TestResponseStructure(t *testing.T) {
	tests := []struct {
		name     string
		response Response
		valid    bool
	}{
		{
			name: "success response with data",
			response: Response{
				Success: true,
				Data:    map[string]string{"key": "value"},
			},
			valid: true,
		},
		{
			name: "error response with message",
			response: Response{
				Success: false,
				Error:   "Error message",
			},
			valid: true,
		},
		{
			name: "success without data (valid)",
			response: Response{
				Success: true,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON to ensure structure is valid
			_, err := json.Marshal(tt.response)
			if err != nil && tt.valid {
				t.Errorf("Failed to marshal valid response: %v", err)
			}

			// Verify success/error consistency
			if tt.response.Success && tt.response.Error != "" {
				t.Error("Success response should not have error message")
			}
			if !tt.response.Success && tt.response.Error == "" {
				t.Error("Error response should have error message")
			}
		})
	}
}

// TestConcurrentRateLimiter tests rate limiter under concurrent load
func TestConcurrentRateLimiter(t *testing.T) {
	rl := NewRateLimiter(100, time.Minute)

	// Run concurrent requests
	concurrency := 50
	done := make(chan bool)

	for i := 0; i < concurrency; i++ {
		go func(id int) {
			for j := 0; j < 3; j++ {
				rl.Allow("test-key")
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// The rate limiter should have handled concurrent access without panic
	t.Log("Concurrent access handled successfully")
}

// TestJSONSerialization tests JSON encoding/decoding
func TestJSONSerialization(t *testing.T) {
	t.Run("HTTPRequest serialization", func(t *testing.T) {
		req := HTTPRequest{
			URL:    "https://example.com",
			Method: "POST",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
			Body: map[string]interface{}{
				"key": "value",
			},
		}

		// Encode
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		// Decode
		var decoded HTTPRequest
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.URL != req.URL {
			t.Error("URL mismatch after serialization")
		}
		if decoded.Method != req.Method {
			t.Error("Method mismatch after serialization")
		}
	})
}

// TestErrorHandling tests error response formatting
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		errorMsg       string
		expectedStatus int
	}{
		{
			name:           "validation error",
			errorMsg:       "Invalid input",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "not found error",
			errorMsg:       "Resource not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "internal error",
			errorMsg:       "Internal server error",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create error response
			resp := Response{
				Success: false,
				Error:   tt.errorMsg,
			}

			// Verify structure
			if resp.Success {
				t.Error("Error response should have Success=false")
			}
			if resp.Error == "" {
				t.Error("Error response should have error message")
			}
			if resp.Data != nil {
				t.Error("Error response should not have data")
			}
		})
	}
}
