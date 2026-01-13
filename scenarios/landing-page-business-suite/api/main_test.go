package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/vrooli/api-core/health"
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

		healthHandler := health.New().Version("1.0.0").Check(health.DB(srv.db), health.Critical).Handler()
		healthHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}
		if contentType := w.Header().Get("Content-Type"); contentType != "application/json" {
			t.Fatalf("Expected Content-Type application/json, got %s", contentType)
		}

		var payload health.Response
		if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode health response: %v", err)
		}
		if payload.Status != "healthy" || !payload.Readiness {
			t.Fatalf("expected healthy status and readiness, got status=%s readiness=%v", payload.Status, payload.Readiness)
		}
		dbStatus, ok := payload.Dependencies["database"]
		if !ok || !dbStatus.Connected {
			t.Fatalf("expected database dependency to be connected")
		}
	})

	t.Run("reports unhealthy when database ping fails", func(t *testing.T) {
		srv, cleanup := newHealthTestServer(t)
		defer cleanup()

		// Force a ping failure by closing the connection before the request.
		srv.db.Close()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler := health.New().Version("1.0.0").Check(health.DB(srv.db), health.Critical).Handler()
		healthHandler.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Fatalf("Expected status 503 when unhealthy, got %d", w.Code)
		}

		var payload health.Response
		if err := json.NewDecoder(w.Body).Decode(&payload); err != nil {
			t.Fatalf("failed to decode health response: %v", err)
		}
		if payload.Status != "unhealthy" {
			t.Fatalf("expected unhealthy status, got %s", payload.Status)
		}
		if payload.Readiness {
			t.Fatalf("expected readiness=false when db ping fails")
		}
		dbStatus, ok := payload.Dependencies["database"]
		if !ok || dbStatus.Connected {
			t.Fatalf("expected database dependency marked disconnected")
		}
	})
}

func TestParseEnvBool(t *testing.T) {
	t.Run("truthy values", func(t *testing.T) {
		cases := []string{"1", "true", "TRUE", "yes", "Y", "on", " On "}
		for _, value := range cases {
			if !parseEnvBool(value) {
				t.Fatalf("expected true for %q", value)
			}
		}
	})

	t.Run("falsey values", func(t *testing.T) {
		cases := []string{"", "0", "false", "no", "off", "disabled"}
		for _, value := range cases {
			if parseEnvBool(value) {
				t.Fatalf("expected false for %q", value)
			}
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
