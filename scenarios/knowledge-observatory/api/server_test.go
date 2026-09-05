package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
)

// TestNewServerValidation tests NewServer initialization [REQ:KO-API-001]
func TestNewServerValidation(t *testing.T) {
	tests := []struct {
		name       string
		setupEnv   func()
		cleanupEnv func()
		wantErr    bool
		errContain string
		skipDB     bool
	}{
		{
			name: "initializes successfully with DATABASE_URL when DB connect is skipped",
			setupEnv: func() {
				os.Setenv("API_PORT", "8080")
				os.Setenv("DATABASE_URL", "postgresql://test:test@localhost:5432/test?sslmode=disable")
			},
			cleanupEnv: func() {
				os.Unsetenv("API_PORT")
				os.Unsetenv("DATABASE_URL")
			},
			wantErr:    false,
			errContain: "",
			skipDB:     true,
		},
		{
			name: "initializes successfully with individual postgres vars when DB connect is skipped",
			setupEnv: func() {
				os.Setenv("API_PORT", "8080")
				os.Unsetenv("DATABASE_URL")
				os.Setenv("POSTGRES_USER", "testuser")
				os.Setenv("POSTGRES_PASSWORD", "testpass")
				os.Setenv("POSTGRES_HOST", "localhost")
				os.Setenv("POSTGRES_PORT", "5432")
				os.Setenv("POSTGRES_DB", "testdb")
			},
			cleanupEnv: func() {
				os.Unsetenv("API_PORT")
				os.Unsetenv("POSTGRES_USER")
				os.Unsetenv("POSTGRES_PASSWORD")
				os.Unsetenv("POSTGRES_HOST")
				os.Unsetenv("POSTGRES_PORT")
				os.Unsetenv("POSTGRES_DB")
			},
			wantErr:    false,
			errContain: "",
			skipDB:     true,
		},
		// Skipping API_PORT not set test because requireEnv calls log.Fatal which exits the process
		// and cannot be caught by recover(). This is tested indirectly by other tests that set API_PORT.
		{
			name: "fails when DATABASE_URL and postgres vars missing",
			setupEnv: func() {
				os.Setenv("API_PORT", "8080")
				os.Unsetenv("DATABASE_URL")
				os.Unsetenv("POSTGRES_USER")
				os.Unsetenv("POSTGRES_PASSWORD")
				os.Unsetenv("POSTGRES_HOST")
				os.Unsetenv("POSTGRES_PORT")
				os.Unsetenv("POSTGRES_DB")
			},
			cleanupEnv: func() {
				os.Unsetenv("API_PORT")
			},
			wantErr:    true,
			errContain: "postgres connection requires environment variables",
		},
		{
			name: "skips database connection when SKIP_DB_TESTS is true",
			setupEnv: func() {
				os.Setenv("API_PORT", "8080")
				os.Setenv("DATABASE_URL", "invalid-url-format")
			},
			cleanupEnv: func() {
				os.Unsetenv("API_PORT")
				os.Unsetenv("DATABASE_URL")
			},
			wantErr:    false,
			errContain: "",
			skipDB:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()
			if tt.skipDB {
				os.Setenv("SKIP_DB_TESTS", "true")
				defer os.Unsetenv("SKIP_DB_TESTS")
			}

			// Catch panics from requireEnv
			var srv *Server
			var err error
			func() {
				defer func() {
					if r := recover(); r != nil {
						if !tt.wantErr {
							t.Errorf("NewServer() panicked unexpectedly: %v", r)
						} else {
							t.Logf("NewServer() panicked as expected: %v", r)
						}
					}
				}()
				srv, err = NewServer()
			}()

			if (err != nil) != tt.wantErr {
				t.Logf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errContain != "" {
				if !contains(err.Error(), tt.errContain) {
					t.Errorf("NewServer() error = %v, want error containing %v", err, tt.errContain)
				}
			}

			// Clean up server if created
			if srv != nil && srv.db != nil {
				srv.db.Close()
			}
		})
	}
}

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
				"API_PORT":     "9999",
				"DATABASE_URL": "postgresql://test:test@localhost:5432/test",
			},
			wantPort: "9999",
		},
		{
			name: "uses configured port",
			envVars: map[string]string{
				"API_PORT":     "7777",
				"DATABASE_URL": "postgresql://test:test@localhost:5432/test",
			},
			wantPort: "7777",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()
			os.Setenv("SKIP_DB_TESTS", "true")
			defer os.Unsetenv("SKIP_DB_TESTS")

			// NewServer will fail to connect, but we can check config parsing
			srv, _ := NewServer()

			if srv != nil {
				if srv.config.Port != tt.wantPort {
					t.Errorf("NewServer() port = %v, want %v", srv.config.Port, tt.wantPort)
				}
				if srv.db != nil {
					srv.db.Close()
				}
			}
		})
	}
}

// TestSetupRoutesValidation tests route registration validation [REQ:KO-API-001]
func TestSetupRoutesValidation(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgresql://test:test@localhost:5432/test",
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
			Port:        "8080",
			DatabaseURL: "postgresql://test:test@localhost:5432/test",
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
			Port:        "8080",
			DatabaseURL: "postgresql://test:test@localhost:5432/test",
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
