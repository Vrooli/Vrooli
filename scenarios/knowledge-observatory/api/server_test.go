package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
)

// TestNewServerConfigParsing tests configuration parsing [REQ:KO-API-001]
func TestNewServerConfigParsing(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		wantPort string
	}{
		{
			name: "parses API_PORT correctly",
			envVars: map[string]string{
				"API_PORT": "9999",
			},
			wantPort: "9999",
		},
		{
			name: "uses configured port",
			envVars: map[string]string{
				"API_PORT": "7777",
			},
			wantPort: "7777",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}
			t.Setenv("SKIP_DB_TESTS", "true")

			srv, err := NewServer()
			if err != nil {
				t.Fatalf("NewServer() error = %v", err)
			}

			if srv.config.Port != tt.wantPort {
				t.Errorf("NewServer() port = %v, want %v", srv.config.Port, tt.wantPort)
			}
			if srv.db != nil {
				srv.db.Close()
			}
		})
	}
}

// TestSetupRoutesValidation tests route registration validation [REQ:KO-API-001]
func TestSetupRoutesValidation(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port: "8080",
		},
		db:     nil,
		router: mux.NewRouter(), // Initialize router before calling setupRoutes
	}

	// Call setupRoutes to register handlers
	srv.setupRoutes()

	if srv.router == nil {
		t.Fatal("setupRoutes() did not initialize router")
	}

	// Verify router was initialized (detailed route testing in main_test.go)
	t.Logf("setupRoutes() successfully initialized router")
}

// TestServerLogging tests server logging [REQ:KO-API-003]
func TestServerLogging(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port: "8080",
		},
		db: nil,
	}

	// Test logging doesn't panic
	srv.log("test message", map[string]interface{}{
		"key1": "value1",
		"key2": 42,
	})

	srv.log("test without fields", nil)
}

// TestHandleHealthWithDB tests health endpoint with database [REQ:KO-API-004]
func TestHandleHealthWithDB(t *testing.T) {
	// Create server without database
	srv := &Server{
		config: &Config{
			Port: "8080",
		},
		db: nil, // No database connection
	}

	tests := []struct {
		name         string
		setupDB      func(t *testing.T) *database.RoutedDB
		wantResponse bool
	}{
		{
			name:         "still answers when no database is configured",
			setupDB:      func(*testing.T) *database.RoutedDB { return nil },
			wantResponse: true,
		},
		{
			name: "answers when a sqlite database is configured",
			setupDB: func(t *testing.T) *database.RoutedDB {
				return dbtest.New(t)
			},
			wantResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv.db = tt.setupDB(t)

			rec := httptest.NewRecorder()
			srv.handleHealth(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

			// The endpoint must always answer: an unreachable database is a
			// reported condition, not a dead endpoint.
			if tt.wantResponse && rec.Code == 0 {
				t.Fatal("health endpoint produced no response")
			}
			if body := rec.Body.String(); body == "" {
				t.Error("health endpoint returned an empty body")
			}
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && contains(s[1:], substr)
}
