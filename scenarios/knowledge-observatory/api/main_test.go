package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestRequireEnv tests environment variable requirement validation [REQ:KO-API-002]
func TestRequireEnv(t *testing.T) {
	tests := []struct {
		name      string
		key       string
		value     string
		shouldSet bool
		wantPanic bool
	}{
		{
			name:      "returns value when env var is set",
			key:       "TEST_VAR_EXISTS",
			value:     "test-value",
			shouldSet: true,
			wantPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldSet {
				os.Setenv(tt.key, tt.value)
				defer os.Unsetenv(tt.key)
			}

			if tt.wantPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("requireEnv() did not panic")
					}
				}()
			}

			result := requireEnv(tt.key)

			if !tt.wantPanic && result != tt.value {
				t.Errorf("requireEnv() = %v, want %v", result, tt.value)
			}
		})
	}
}

// TestResolveDatabaseURL tests database URL resolution logic [REQ:KO-API-002]
func TestResolveDatabaseURL(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		wantContain string
		wantErr     bool
	}{
		{
			name: "uses DATABASE_URL when set",
			setupEnv: func() {
				os.Setenv("DATABASE_URL", "postgresql://user:pass@localhost/testdb")
			},
			cleanupEnv: func() {
				os.Unsetenv("DATABASE_URL")
			},
			wantContain: "postgresql://user:pass@localhost/testdb",
			wantErr:     false,
		},
		{
			name: "constructs URL from individual env vars",
			setupEnv: func() {
				os.Unsetenv("DATABASE_URL")
				os.Setenv("POSTGRES_USER", "testuser")
				os.Setenv("POSTGRES_PASSWORD", "testpass")
				os.Setenv("POSTGRES_HOST", "testhost")
				os.Setenv("POSTGRES_PORT", "5432")
				os.Setenv("POSTGRES_DB", "testdb")
			},
			cleanupEnv: func() {
				os.Unsetenv("POSTGRES_USER")
				os.Unsetenv("POSTGRES_PASSWORD")
				os.Unsetenv("POSTGRES_HOST")
				os.Unsetenv("POSTGRES_PORT")
				os.Unsetenv("POSTGRES_DB")
			},
			wantContain: "testuser",
			wantErr:     false,
		},
		{
			name: "returns error when no DATABASE_URL and missing POSTGRES_USER",
			setupEnv: func() {
				os.Unsetenv("DATABASE_URL")
				os.Unsetenv("POSTGRES_USER")
			},
			cleanupEnv: func() {},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanupEnv()

			got, err := resolveDatabaseURL()
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveDatabaseURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(got, tt.wantContain) {
				t.Errorf("resolveDatabaseURL() = %v, want to contain %v", got, tt.wantContain)
			}
		})
	}
}

// TestHandleHealth tests the infrastructure health endpoint [REQ:KO-API-002]
func TestHandleHealth(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Create a mock server with minimal setup
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test",
		},
		db: nil, // DB will be nil, which will cause unhealthy status
	}

	srv.handleHealth(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("handleHealth() status = %v, want %v", resp.StatusCode, http.StatusOK)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// When DB is nil, status should be unhealthy but response should still be valid
	if result["status"] == "" {
		t.Error("handleHealth() returned empty status")
	}

	if result["service"] == "" {
		t.Error("handleHealth() returned empty service")
	}
}

// TestLoggingMiddleware tests HTTP request logging middleware [REQ:KO-API-002]
func TestLoggingMiddleware(t *testing.T) {
	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with logging middleware
	wrapped := loggingMiddleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	// Execute request
	wrapped.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("loggingMiddleware() status = %v, want %v", w.Code, http.StatusOK)
	}

	if w.Body.String() != "test response" {
		t.Errorf("loggingMiddleware() body = %v, want 'test response'", w.Body.String())
	}
}

// TestLog tests structured logging function [REQ:KO-API-002]
func TestLog(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port: "8080",
		},
	}

	// Test that log doesn't panic with fields
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("log() panicked: %v", r)
		}
	}()

	srv.log("test message with fields", map[string]interface{}{
		"key":   "value",
		"count": 42,
	})

	// Test that log doesn't panic with nil fields
	srv.log("test message without fields", nil)
}

// TestSetupRoutes tests route registration [REQ:KO-API-002]
func TestSetupRoutes(t *testing.T) {
	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test",
		},
		router: mux.NewRouter(),
	}

	// setupRoutes should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("setupRoutes() panicked: %v", r)
		}
	}()

	srv.setupRoutes()

	// Verify the router was configured
	if srv.router == nil {
		t.Error("setupRoutes() left router nil")
	}
}

// TestHealthResponse tests health endpoint response structure [REQ:KO-QM-004]
func TestHealthResponse(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test",
		},
		db: nil,
	}

	srv.handleHealth(w, req)

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	// Verify required fields exist
	if status, ok := response["status"].(string); !ok || status == "" {
		t.Error("health response missing or empty status field")
	}
	if service, ok := response["service"].(string); !ok || service == "" {
		t.Error("health response missing or empty service field")
	}
	if timestamp, ok := response["timestamp"].(string); !ok || timestamp == "" {
		t.Error("health response missing or empty timestamp field")
	}
	if _, ok := response["dependencies"]; !ok {
		t.Error("health response missing dependencies field")
	}
}

// TestHealthResponseTiming tests that health endpoint responds quickly [REQ:KO-API-002]
func TestHealthResponseTiming(t *testing.T) {
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	srv := &Server{
		config: &Config{
			Port:        "8080",
			DatabaseURL: "postgres://test:test@localhost:5432/test",
		},
		db: nil,
	}

	start := time.Now()
	srv.handleHealth(w, req)
	duration := time.Since(start)

	// Health endpoint should respond in < 100ms for lightweight check
	if duration > 100*time.Millisecond {
		t.Errorf("handleHealth() took %v, want < 100ms", duration)
	}
}

// TestHTTPMethodValidation tests that endpoints validate HTTP methods [REQ:KO-API-003]
func TestHTTPMethodValidation(t *testing.T) {
	tests := []struct {
		name       string
		endpoint   string
		method     string
		handler    http.HandlerFunc
		wantStatus int
	}{
		{
			name:     "GET /health accepts GET",
			endpoint: "/health",
			method:   "GET",
			handler: func(w http.ResponseWriter, r *http.Request) {
				srv := &Server{config: &Config{Port: "8080"}, db: nil}
				srv.handleHealth(w, r)
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint, nil)
			w := httptest.NewRecorder()

			tt.handler(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("HTTP %s %s = %v, want %v", tt.method, tt.endpoint, w.Code, tt.wantStatus)
			}
		})
	}
}

// TestJSONContentType tests that API endpoints return JSON [REQ:KO-API-003]
func TestJSONContentType(t *testing.T) {
	endpoints := []struct {
		name    string
		handler func(w http.ResponseWriter, r *http.Request)
	}{
		{
			name: "/health",
			handler: func(w http.ResponseWriter, r *http.Request) {
				srv := &Server{config: &Config{Port: "8080"}, db: nil}
				srv.handleHealth(w, r)
			},
		},
	}

	for _, ep := range endpoints {
		t.Run(ep.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", ep.name, nil)
			w := httptest.NewRecorder()

			ep.handler(w, req)

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("%s Content-Type = %v, want 'application/json'", ep.name, contentType)
			}
		})
	}
}
