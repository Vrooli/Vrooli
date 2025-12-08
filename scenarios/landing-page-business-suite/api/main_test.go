package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func newHealthTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	db := setupTestDB(t)
	return &Server{db: db}, func() { db.Close() }
}

func TestHealthEndpoint(t *testing.T) {
	t.Run("healthy when database reachable", func(t *testing.T) {
		srv, cleanup := newHealthTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		srv.handleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}
		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("Expected Content-Type application/json, got %s", contentType)
		}

		var payload struct {
			Status       string            `json:"status"`
			Readiness    bool              `json:"readiness"`
			Dependencies map[string]string `json:"dependencies"`
		}
		if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode health response: %v", err)
		}
		if payload.Status != "healthy" || !payload.Readiness {
			t.Fatalf("expected healthy status and readiness, got status=%s readiness=%v", payload.Status, payload.Readiness)
		}
		if payload.Dependencies["database"] != "connected" {
			t.Fatalf("expected database dependency to be connected, got %q", payload.Dependencies["database"])
		}
	})

	t.Run("reports unhealthy when database ping fails", func(t *testing.T) {
		srv, cleanup := newHealthTestServer(t)
		defer cleanup()

		// Force a ping failure by closing the connection before the request.
		srv.db.Close()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		srv.handleHealth(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200 even when unhealthy, got %d", w.Code)
		}

		var payload struct {
			Status       string            `json:"status"`
			Readiness    bool              `json:"readiness"`
			Dependencies map[string]string `json:"dependencies"`
		}
		if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode health response: %v", err)
		}
		if payload.Status != "unhealthy" {
			t.Fatalf("expected unhealthy status, got %s", payload.Status)
		}
		if payload.Readiness {
			t.Fatalf("expected readiness=false when db ping fails")
		}
		if payload.Dependencies["database"] != "disconnected" {
			t.Fatalf("expected database dependency marked disconnected, got %q", payload.Dependencies["database"])
		}
	})
}

func TestRequireEnv(t *testing.T) {
	// Test valid environment variable
	t.Run("valid environment variable", func(t *testing.T) {
		os.Setenv("TEST_VAR", "test_value")
		defer os.Unsetenv("TEST_VAR")

		result := requireEnv("TEST_VAR")
		if result != "test_value" {
			t.Errorf("Expected test_value, got %s", result)
		}
	})

	// Test whitespace trimming
	t.Run("whitespace trimming", func(t *testing.T) {
		os.Setenv("TEST_VAR_WS", "  trimmed  ")
		defer os.Unsetenv("TEST_VAR_WS")

		result := requireEnv("TEST_VAR_WS")
		if result != "trimmed" {
			t.Errorf("Expected trimmed, got '%s'", result)
		}
	})
}

func TestResolveDatabaseURL(t *testing.T) {
	t.Run("explicit DATABASE_URL", func(t *testing.T) {
		os.Setenv("DATABASE_URL", "postgres://explicit:5432/db")
		defer os.Unsetenv("DATABASE_URL")

		url, err := resolveDatabaseURL()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if url != "postgres://explicit:5432/db" {
			t.Errorf("Expected explicit URL, got %s", url)
		}
	})

	t.Run("constructed from components", func(t *testing.T) {
		os.Unsetenv("DATABASE_URL")
		os.Setenv("POSTGRES_HOST", "testhost")
		os.Setenv("POSTGRES_PORT", "5432")
		os.Setenv("POSTGRES_USER", "testuser")
		os.Setenv("POSTGRES_PASSWORD", "testpass")
		os.Setenv("POSTGRES_DB", "testdb")
		defer func() {
			os.Unsetenv("POSTGRES_HOST")
			os.Unsetenv("POSTGRES_PORT")
			os.Unsetenv("POSTGRES_USER")
			os.Unsetenv("POSTGRES_PASSWORD")
			os.Unsetenv("POSTGRES_DB")
		}()

		url, err := resolveDatabaseURL()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		expected := "postgres://testuser:testpass@testhost:5432/testdb?sslmode=disable"
		if url != expected {
			t.Errorf("Expected %s, got %s", expected, url)
		}
	})

	t.Run("errors when components missing", func(t *testing.T) {
		keys := []string{"DATABASE_URL", "POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"}
		original := make(map[string]string, len(keys))
		for _, key := range keys {
			original[key] = os.Getenv(key)
			os.Unsetenv(key)
		}
		defer func() {
			for _, key := range keys {
				if original[key] == "" {
					os.Unsetenv(key)
					continue
				}
				os.Setenv(key, original[key])
			}
		}()

		if _, err := resolveDatabaseURL(); err == nil {
			t.Fatalf("expected error when database components are missing")
		}
	})
}

func TestLogStructured(t *testing.T) {
	// Test structured logging (output validation is complex, just ensure no panic)
	logStructured("test_event", map[string]interface{}{
		"key":   "value",
		"count": 42,
	})

	logStructured("test_event_no_fields", nil)
}
