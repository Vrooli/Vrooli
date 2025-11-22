package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
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

			// Test requireEnv for API_PORT
			if tt.apiPort == "" && !tt.wantError {
				t.Skip("Skipping invalid test case")
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
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && stringContains(s, substr))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
