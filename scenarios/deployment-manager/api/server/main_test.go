package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"deployment-manager/bundles"
	"deployment-manager/dependencies"
	"deployment-manager/deployments"
	"deployment-manager/fitness"
	"deployment-manager/health"
	"deployment-manager/profiles"
	"deployment-manager/secrets"
	"deployment-manager/swaps"
	"deployment-manager/telemetry"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
)

// TestRequireEnv verifies environment variable validation
func TestRequireEnv(t *testing.T) {
	tests := []struct {
		name      string
		envKey    string
		envValue  string
		shouldErr bool
	}{
		{
			name:      "valid environment variable",
			envKey:    "TEST_VAR",
			envValue:  "valid_value",
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			// RequireEnv calls log.Fatalf on error, so we can't directly test failure cases
			// Instead, we test the success case and verify the function returns correctly
			if !tt.shouldErr {
				result := RequireEnv(tt.envKey)
				if result != tt.envValue {
					t.Errorf("RequireEnv() = %v, want %v", result, tt.envValue)
				}
			}
		})
	}
}

// TestResolveDatabaseURL verifies database URL construction
func TestResolveDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		envVars     map[string]string
		wantURL     string
		wantErr     bool
		wantContain string
	}{
		{
			name: "direct DATABASE_URL",
			envVars: map[string]string{
				"DATABASE_URL": "postgres://user:pass@localhost:5432/dbname?sslmode=disable",
			},
			wantURL: "postgres://user:pass@localhost:5432/dbname?sslmode=disable",
			wantErr: false,
		},
		{
			name: "constructed from components",
			envVars: map[string]string{
				"POSTGRES_HOST":     "testhost",
				"POSTGRES_PORT":     "5433",
				"POSTGRES_USER":     "testuser",
				"POSTGRES_PASSWORD": "testpass",
				"POSTGRES_DB":       "testdb",
			},
			wantContain: "postgres://testuser:testpass@testhost:5433/testdb",
			wantErr:     false,
		},
		{
			name: "missing components",
			envVars: map[string]string{
				"POSTGRES_HOST": "localhost",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all postgres-related env vars
			for _, key := range []string{"DATABASE_URL", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"} {
				os.Unsetenv(key)
			}

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			result, err := ResolveDatabaseURL()

			if tt.wantErr && err == nil {
				t.Errorf("ResolveDatabaseURL() error = nil, wantErr %v", tt.wantErr)
				return
			}

			if !tt.wantErr && err != nil {
				t.Errorf("ResolveDatabaseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.wantURL != "" && result != tt.wantURL {
					t.Errorf("ResolveDatabaseURL() = %v, want %v", result, tt.wantURL)
				}
				if tt.wantContain != "" && !strings.Contains(result, tt.wantContain) {
					t.Errorf("ResolveDatabaseURL() = %v, want to contain %v", result, tt.wantContain)
				}
			}
		})
	}
}

// TestHealthEndpoint tests the health check handler
func TestHealthEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		dbPingErr      error
		wantStatus     string
		wantReadiness  bool
		wantDbStatus   string
		wantHTTPStatus int
	}{
		{
			name:           "healthy database",
			dbPingErr:      nil,
			wantStatus:     "healthy",
			wantReadiness:  true,
			wantDbStatus:   "connected",
			wantHTTPStatus: http.StatusOK,
		},
		{
			name:           "unhealthy database",
			dbPingErr:      sql.ErrConnDone,
			wantStatus:     "unhealthy",
			wantReadiness:  false,
			wantDbStatus:   "disconnected",
			wantHTTPStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock database
			db, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
			if err != nil {
				t.Fatalf("failed to create mock db: %v", err)
			}
			defer db.Close()

			// Configure mock ping expectation
			if tt.dbPingErr != nil {
				mock.ExpectPing().WillReturnError(tt.dbPingErr)
			} else {
				mock.ExpectPing()
			}

			// Create server with the specific mock database (not setupTestServer)
			// so the health handler uses our mock DB with ping expectations
			logFn := func(msg string, fields map[string]interface{}) {}
			profilesRepo := profiles.NewSQLRepository(db)

			srv := &Server{
				Config:              &Config{Port: "8080"},
				DB:                  db,
				Router:              mux.NewRouter(),
				ProfilesRepo:        profilesRepo,
				HealthHandler:       health.NewHandler(db),
				FitnessHandler:      fitness.NewHandler(logFn),
				TelemetryHandler:    telemetry.NewHandler(logFn),
				SecretsHandler:      secrets.NewHandler(profilesRepo, logFn),
				DependenciesHandler: dependencies.NewHandler(logFn),
				SwapsHandler:        swaps.NewHandler(logFn),
				DeploymentsHandler:  deployments.NewHandler(logFn),
				BundlesHandler:      bundles.NewHandler(secrets.NewClient(), logFn),
				ProfilesHandler:     profiles.NewHandler(profilesRepo, logFn),
			}
			srv.setupRoutes()

			// Create test request
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()

			// Execute request
			srv.Router.ServeHTTP(w, req)

			// Verify response
			if w.Code != tt.wantHTTPStatus {
				t.Errorf("HTTP status = %v, want %v", w.Code, tt.wantHTTPStatus)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if response["status"] != tt.wantStatus {
				t.Errorf("status = %v, want %v", response["status"], tt.wantStatus)
			}

			if response["readiness"] != tt.wantReadiness {
				t.Errorf("readiness = %v, want %v", response["readiness"], tt.wantReadiness)
			}

			deps, ok := response["dependencies"].(map[string]interface{})
			if !ok {
				t.Fatal("dependencies field is not a map")
			}

			if deps["database"] != tt.wantDbStatus {
				t.Errorf("database status = %v, want %v", deps["database"], tt.wantDbStatus)
			}

			// Verify all mock expectations were met
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled mock expectations: %v", err)
			}
		})
	}
}

// TestLoggingMiddleware verifies request logging
func TestLoggingMiddleware(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with logging middleware
	wrappedHandler := LoggingMiddleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("HTTP status = %v, want %v", w.Code, http.StatusOK)
	}

	if w.Body.String() != "OK" {
		t.Errorf("body = %v, want %v", w.Body.String(), "OK")
	}
}

// TestServerSetupRoutes verifies route configuration
func TestServerSetupRoutes(t *testing.T) {
	srv, _ := setupTestServer(t)
	defer srv.DB.Close()

	// Test that /health route is registered
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.Router.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Error("/health route not registered")
	}
}

// TestServerLog verifies logging functionality
func TestServerLog(t *testing.T) {
	tests := []struct {
		name   string
		msg    string
		fields map[string]interface{}
	}{
		{
			name:   "log with no fields",
			msg:    "test message",
			fields: nil,
		},
		{
			name: "log with fields",
			msg:  "test message with fields",
			fields: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify it doesn't panic
			LogStructured(tt.msg, tt.fields)
		})
	}
}

// TestServerStartGracefulShutdown verifies graceful shutdown behavior
func TestServerStartGracefulShutdown(t *testing.T) {
	// This test verifies the server can be created but we skip the actual Start()
	// call as it blocks and requires signal handling which is difficult to test
	srv, _ := setupTestServer(t)
	defer srv.DB.Close()

	// Create HTTP server with timeout context
	httpServer := &http.Server{
		Addr:         srv.Config.Port,
		Handler:      srv.Router,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  1 * time.Second,
	}

	// Test that we can create a shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Attempt shutdown (should complete immediately as server is not running)
	err := httpServer.Shutdown(ctx)
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("unexpected shutdown error: %v", err)
	}
}
