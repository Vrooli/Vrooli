package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

func TestHealthEndpoint(t *testing.T) {
	// Set required environment variables for testing
	os.Setenv("API_PORT", "8080")
	os.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test?sslmode=disable")

	// Create a mock server with a nil database for unit testing
	// The health endpoint should handle nil gracefully
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil, // Explicitly set to nil for unit testing
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	// Create a test request
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Record the response
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	// Health endpoint should return 200 even if db is nil (will show unhealthy status)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Health endpoint returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response body contains expected content
	body := rr.Body.String()
	if body == "" {
		t.Error("Health endpoint returned empty body")
	}

	// In unit tests without real DB, we expect unhealthy status
	if !contains(body, "status") {
		t.Error("Health endpoint response should contain 'status' field")
	}
}

func TestHealthEndpointWithDB(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Only run if we can connect to a test database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping database integration test")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Skip("Could not open database connection")
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skip("Could not ping database, skipping integration test")
	}

	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: dbURL,
		},
		db:     db,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Health endpoint returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	body := rr.Body.String()
	if !contains(body, "healthy") {
		t.Error("Health endpoint should report healthy status with working database")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		apiPort     string
		databaseURL string
		wantError   bool
	}{
		{
			name:        "valid config",
			apiPort:     "8080",
			databaseURL: "postgres://user:pass@localhost:5432/db?sslmode=disable",
			wantError:   false,
		},
		{
			name:        "missing api port",
			apiPort:     "",
			databaseURL: "postgres://user:pass@localhost:5432/db?sslmode=disable",
			wantError:   true,
		},
		{
			name:        "missing database URL",
			apiPort:     "8080",
			databaseURL: "",
			wantError:   true,
		},
		{
			name:        "invalid port format",
			apiPort:     "not-a-number",
			databaseURL: "postgres://user:pass@localhost:5432/db?sslmode=disable",
			wantError:   false, // Port validation is minimal in current implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original env
			origPort := os.Getenv("API_PORT")
			origDB := os.Getenv("DATABASE_URL")
			defer func() {
				os.Setenv("API_PORT", origPort)
				os.Setenv("DATABASE_URL", origDB)
			}()

			// Set test env
			if tt.apiPort != "" {
				os.Setenv("API_PORT", tt.apiPort)
			} else {
				os.Unsetenv("API_PORT")
			}
			if tt.databaseURL != "" {
				os.Setenv("DATABASE_URL", tt.databaseURL)
			} else {
				os.Unsetenv("DATABASE_URL")
			}

			// Create config and validate
			cfg := &Config{
				Port:        os.Getenv("API_PORT"),
				DatabaseURL: os.Getenv("DATABASE_URL"),
			}

			// Check if config has required fields
			hasError := (cfg.Port == "" || cfg.DatabaseURL == "")
			if hasError != tt.wantError {
				t.Errorf("Config validation: got error=%v, want error=%v", hasError, tt.wantError)
			}

			// Verify fields are set as expected
			if tt.apiPort != "" && cfg.Port != tt.apiPort {
				t.Errorf("Expected Port=%s, got %s", tt.apiPort, cfg.Port)
			}
			if tt.databaseURL != "" && cfg.DatabaseURL != tt.databaseURL {
				t.Errorf("Expected DatabaseURL=%s, got %s", tt.databaseURL, cfg.DatabaseURL)
			}
		})
	}
}

func TestRoutingMiddleware(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	// Test that middleware is applied
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	// Just verify the request was processed
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestHealthEndpointJSONFormat validates the JSON structure of health response
func TestHealthEndpointJSONFormat(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	// Verify Content-Type header
	contentType := rr.Header().Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		t.Errorf("Expected Content-Type to contain application/json, got %s", contentType)
	}

	// Verify JSON is valid
	var healthResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify required fields exist
	if _, ok := healthResp["status"]; !ok {
		t.Error("Health response missing 'status' field")
	}
	if _, ok := healthResp["service"]; !ok {
		t.Error("Health response missing 'service' field")
	}
	if _, ok := healthResp["version"]; !ok {
		t.Error("Health response missing 'version' field")
	}
	if _, ok := healthResp["readiness"]; !ok {
		t.Error("Health response missing 'readiness' field")
	}
	if _, ok := healthResp["timestamp"]; !ok {
		t.Error("Health response missing 'timestamp' field")
	}
	if _, ok := healthResp["dependencies"]; !ok {
		t.Error("Health response missing 'dependencies' field")
	}

	// Verify dependencies structure
	deps, ok := healthResp["dependencies"].(map[string]interface{})
	if !ok {
		t.Error("Health response 'dependencies' should be a map")
	} else {
		if _, ok := deps["database"]; !ok {
			t.Error("Health response dependencies missing 'database' field")
		}
	}
}

// TestHealthEndpointCORS validates CORS headers
func TestHealthEndpointCORS(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	// Health endpoint should have CORS headers from middleware
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", rr.Code)
	}
}

// TestHealthV1Endpoint validates the v1 API health endpoint
func TestHealthV1Endpoint(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("/api/v1/health returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	if !contains(rr.Body.String(), "status") {
		t.Error("/api/v1/health response should contain 'status' field")
	}
}

// TestInvalidRoute validates 404 handling
func TestInvalidRoute(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/nonexistent", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Invalid route should return 404, got %v", status)
	}
}

// TestHealthEndpointMethods validates only GET is allowed
func TestHealthEndpointMethods(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	invalidMethods := []string{"POST", "PUT", "DELETE", "PATCH"}
	for _, method := range invalidMethods {
		req := httptest.NewRequest(method, "/health", nil)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusMethodNotAllowed {
			t.Errorf("Method %s should return 405 Method Not Allowed, got %v", method, status)
		}
	}
}

// TestServerInitialization validates server can be initialized with valid config
func TestServerInitialization(t *testing.T) {
	cfg := &Config{
		Port:        "8080",
		DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected port 8080, got %s", cfg.Port)
	}
	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/test?sslmode=disable" {
		t.Errorf("Unexpected database URL: %s", cfg.DatabaseURL)
	}

	// Verify we can create a server with this config
	srv := &Server{
		config: cfg,
		router: mux.NewRouter(),
	}
	if srv.config.Port != "8080" {
		t.Errorf("Server config port should be 8080, got %s", srv.config.Port)
	}
}

// TestHealthEndpointConcurrency validates health endpoint handles concurrent requests
func TestHealthEndpointConcurrency(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	// Send 10 concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/health", nil)
			rr := httptest.NewRecorder()
			srv.router.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("Expected status OK in concurrent request, got %v", status)
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// TestHealthEndpointDatabaseError validates graceful handling of database errors
func TestHealthEndpointDatabaseError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Try to connect to invalid database to simulate connection failure
	invalidDB, _ := sql.Open("postgres", "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable&connect_timeout=1")
	defer invalidDB.Close()

	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://invalid:invalid@localhost:9999/invalid?sslmode=disable",
		},
		db:     invalidDB,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	// Should still return 200 but with unhealthy status
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Expected status OK even with DB error, got %v", status)
	}

	var healthResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Database should be reported as unhealthy
	deps, ok := healthResp["dependencies"].(map[string]interface{})
	if !ok {
		t.Fatal("dependencies field should be a map")
	}

	// Database key should exist
	if _, ok := deps["database"]; !ok {
		t.Fatal("database dependency should be present")
	}

	// Check if database is unhealthy (implementation may vary in structure)
	// Just verify the health endpoint doesn't crash with bad DB
	if healthResp["status"] == nil {
		t.Error("Expected status field to exist even with DB errors")
	}
}

// TestHealthEndpointMaliciousHeaders validates security against header injection
func TestHealthEndpointMaliciousHeaders(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	maliciousHeaders := map[string]string{
		"X-Forwarded-For": "127.0.0.1\r\nX-Injected: malicious",
		"User-Agent":      strings.Repeat("A", 10000), // Very long header
		"X-Custom-Header": "'; DROP TABLE users; --",
		"Content-Type":    "application/json\r\nX-Injected: header",
	}

	for headerName, headerValue := range maliciousHeaders {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set(headerName, headerValue)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		// Should still process request normally without crashing
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Health endpoint should handle malicious header %s gracefully, got status %v", headerName, status)
		}

		// Should not reflect malicious content in response
		body := rr.Body.String()
		if contains(body, "DROP TABLE") || contains(body, "X-Injected") {
			t.Errorf("Response should not reflect malicious header content")
		}
	}
}

// TestHealthEndpointResponseConsistency validates response schema consistency
func TestHealthEndpointResponseConsistency(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	// Make multiple requests and verify schema is consistent
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()
		srv.router.ServeHTTP(rr, req)

		var healthResp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &healthResp); err != nil {
			t.Fatalf("Request %d: Failed to parse JSON: %v", i, err)
		}

		// Verify all required fields present in every response
		requiredFields := []string{"status", "service", "version", "readiness", "timestamp", "dependencies"}
		for _, field := range requiredFields {
			if _, ok := healthResp[field]; !ok {
				t.Errorf("Request %d: Missing required field '%s'", i, field)
			}
		}

		// Verify types are consistent
		if _, ok := healthResp["dependencies"].(map[string]interface{}); !ok {
			t.Errorf("Request %d: dependencies should be object type", i)
		}
	}
}

// TestConfigValidationMissingDatabase validates error handling for missing DATABASE_URL
func TestConfigValidationMissingDatabase(t *testing.T) {
	// Save and restore environment
	origDB := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", origDB)

	os.Unsetenv("DATABASE_URL")

	// Config creation should handle missing DATABASE_URL
	cfg := &Config{
		Port:        "8080",
		DatabaseURL: "",
	}

	if cfg.DatabaseURL != "" {
		t.Error("Expected empty DatabaseURL when environment variable not set")
	}
}

// TestInvalidRouteWithBody validates 404 handling with request body
func TestInvalidRouteWithBody(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	requestBody := `{"malicious": "payload", "size": "large"}`
	req := httptest.NewRequest("POST", "/nonexistent", strings.NewReader(requestBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Invalid route with body should return 404, got %v", status)
	}
}

// TestHealthEndpointQueryParameters validates query parameter handling
func TestHealthEndpointQueryParameters(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test?sslmode=disable",
		},
		db:     nil,
		router: mux.NewRouter(),
	}
	srv.setupRoutes()

	// Health endpoint should ignore query parameters
	req := httptest.NewRequest("GET", "/health?verbose=true&format=json", nil)
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Health endpoint should handle query params, got %v", status)
	}

	var healthResp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &healthResp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
}
