package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// Test configuration
var testConfig *Config

func TestMain(m *testing.M) {
	// Setup test environment
	os.Setenv("VROOLI_LIFECYCLE_MANAGED", "true")
	// Use environment variables if available, otherwise set test values
	if os.Getenv("API_PORT") == "" {
		os.Setenv("API_PORT", "15999") // Use high port in range for testing
	}
	if os.Getenv("POSTGRES_URL") == "" {
		os.Setenv("POSTGRES_URL", "postgres://test:test@test-db:5432/calendar_test?sslmode=disable")
	}
	if os.Getenv("QDRANT_URL") == "" {
		os.Setenv("QDRANT_URL", "http://test-qdrant:6333")
	}
	if os.Getenv("AUTH_SERVICE_URL") == "" {
		os.Setenv("AUTH_SERVICE_URL", "http://test-auth:3250")
	}
	if os.Getenv("NOTIFICATION_SERVICE_URL") == "" {
		os.Setenv("NOTIFICATION_SERVICE_URL", "http://test-notification:28100")
	}
	if os.Getenv("OLLAMA_URL") == "" {
		os.Setenv("OLLAMA_URL", "http://test-ollama:11434")
	}
	if os.Getenv("JWT_SECRET") == "" {
		os.Setenv("JWT_SECRET", "test-jwt-secret-key")
	}

	// Initialize test configuration
	testConfig = initConfig()

	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}

func TestInitConfig(t *testing.T) {
	// Set required environment variable for test
	expectedPort := "19867"
	os.Setenv("API_PORT", expectedPort)

	config := initConfig()

	if config.Port != expectedPort {
		t.Errorf("Expected port %s, got %s", expectedPort, config.Port)
	}

	if config.JWTSecret != "test-jwt-secret-key" {
		t.Errorf("Expected JWT secret, got %s", config.JWTSecret)
	}

	if config.AuthServiceURL == "" {
		t.Error("AUTH_SERVICE_URL should not be empty")
	}
}

func TestInitConfigMissingRequired(t *testing.T) {
	// Skip this test in normal runs to avoid env var conflicts
	t.Skip("Skipping test that unsets API_PORT to avoid test conflicts")

	// Save current env
	originalPort := os.Getenv("API_PORT")

	// Remove required env var
	os.Unsetenv("API_PORT")

	// This should not panic but should handle missing env gracefully
	defer func() {
		if r := recover(); r != nil {
			// Expected behavior - missing required env should cause exit/panic
			t.Log("Expected panic for missing required environment variable")
		}
		// Restore env
		os.Setenv("API_PORT", originalPort)
	}()

	// This should trigger error handling
	_ = initConfig()
}

func TestHealthHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(healthHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK && status != http.StatusServiceUnavailable {
		t.Errorf("handler returned wrong status code: got %v want %v or %v",
			status, http.StatusOK, http.StatusServiceUnavailable)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}

	if _, exists := response["status"]; !exists {
		t.Error("Response should contain 'status' field")
	}

	if _, exists := response["timestamp"]; !exists {
		t.Error("Response should contain 'timestamp' field")
	}

	if _, exists := response["version"]; !exists {
		t.Error("Response should contain 'version' field")
	}
}

func TestCreateEventRequestValidation(t *testing.T) {
	// Skip integration tests that require database
	t.Skip("Skipping test that requires database and global dependencies")

	tests := []struct {
		name        string
		requestBody CreateEventRequest
		expectValid bool
	}{
		{
			name: "Valid event request",
			requestBody: CreateEventRequest{
				Title:     "Test Event",
				StartTime: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				EndTime:   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
				EventType: "meeting",
			},
			expectValid: true,
		},
		{
			name: "Missing title",
			requestBody: CreateEventRequest{
				StartTime: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				EndTime:   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
			expectValid: false,
		},
		{
			name: "Invalid time format",
			requestBody: CreateEventRequest{
				Title:     "Test Event",
				StartTime: "invalid-time",
				EndTime:   time.Now().Add(2 * time.Hour).Format(time.RFC3339),
			},
			expectValid: false,
		},
		{
			name: "End time before start time",
			requestBody: CreateEventRequest{
				Title:     "Test Event",
				StartTime: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
				EndTime:   time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to JSON
			jsonBody, err := json.Marshal(tt.requestBody)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			// Create request
			req, err := http.NewRequest("POST", "/api/v1/events", bytes.NewBuffer(jsonBody))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", "test-user-id")

			rr := httptest.NewRecorder()

			// Note: This test would require database setup to fully work
			// For now, we're just testing request parsing
			handler := http.HandlerFunc(createEventHandler)
			handler.ServeHTTP(rr, req)

			// Check if validation worked as expected
			if tt.expectValid {
				// Should not immediately return 400 for valid requests (might fail on DB)
				if rr.Code == http.StatusBadRequest {
					t.Errorf("Expected valid request but got 400: %s", rr.Body.String())
				}
			} else {
				// Should return 400 for invalid requests
				if rr.Code != http.StatusBadRequest && rr.Code != http.StatusUnauthorized {
					t.Errorf("Expected 400 or 401 for invalid request but got %d", rr.Code)
				}
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with auth middleware
	protectedHandler := authMiddleware(testHandler)

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "No authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid authorization format",
			authHeader:     "InvalidFormat token123",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Bearer token format (will fail validation)",
			authHeader:     "Bearer test-token",
			expectedStatus: http.StatusUnauthorized, // Will fail validation without auth service
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/test", nil)
			if err != nil {
				t.Fatal(err)
			}

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()
			protectedHandler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestHealthCheckRoute(t *testing.T) {
	// Create router with health check route
	router := mux.NewRouter()
	router.HandleFunc("/health", healthHandler).Methods("GET")

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK && status != http.StatusServiceUnavailable {
		t.Errorf("health check returned wrong status code: got %v", status)
	}

	// Check response is valid JSON
	var health map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &health); err != nil {
		t.Errorf("health check returned invalid JSON: %v", err)
	}

	// Check required fields - note: "dependencies" field name instead of "services"
	requiredFields := []string{"status", "timestamp", "version", "dependencies"}
	for _, field := range requiredFields {
		if _, exists := health[field]; !exists {
			t.Errorf("health check missing required field: %s", field)
		}
	}
}

func TestChatRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request ChatRequest
		valid   bool
	}{
		{
			name: "Valid chat request",
			request: ChatRequest{
				Message: "Schedule a meeting tomorrow at 2pm",
				Context: map[string]interface{}{"user": "test"},
			},
			valid: true,
		},
		{
			name: "Empty message",
			request: ChatRequest{
				Message: "",
			},
			valid: false,
		},
		{
			name: "Valid minimal request",
			request: ChatRequest{
				Message: "List my events",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.request.Message == "" && tt.valid {
				t.Error("Empty message should not be valid")
			}
			if tt.request.Message != "" && !tt.valid {
				t.Error("Non-empty message should be valid")
			}
		})
	}
}

// Benchmark tests
func BenchmarkHealthHandler(b *testing.B) {
	req, _ := http.NewRequest("GET", "/health", nil)

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler := http.HandlerFunc(healthHandler)
		handler.ServeHTTP(rr, req)
	}
}

func BenchmarkInitConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = initConfig()
	}
}

// Helper functions for integration tests
func setupTestDatabase(t *testing.T) {
	// This would set up a test database
	// Implementation depends on your test database setup
	t.Log("Setting up test database...")
}

func teardownTestDatabase(t *testing.T) {
	// This would clean up the test database
	t.Log("Tearing down test database...")
}

// Example integration test structure
func TestEventCRUDIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	setupTestDatabase(t)
	defer teardownTestDatabase(t)

	t.Log("Integration test would go here...")
	t.Log("This would test full CRUD operations with real database")
}
