package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"landing-page-business-suite-api/internal/logx"

	"github.com/vrooli/api-core/health"
)

func newHealthTestServer(t *testing.T) (*Server, func()) {
	t.Helper()
	db := setupTestDB(t)
	return &Server{db: db}, func() { db.Close() }
}

func TestRuntimeSchemaIsDeclarativeAndCoversRuntimeTables(t *testing.T) {
	t.Parallel()

	sql := strings.ToLower(runtimeSchema())
	if strings.Contains(sql, "alter table") || strings.Contains(sql, "drop table") {
		t.Fatal("runtime schema must be declarative; data/schema migrations belong in one-shot operator scripts")
	}
	for _, table := range []string{
		"admin_users", "remote_profiles", "metrics_events", "checkout_sessions",
		"subscriptions", "subscription_schedules", "bundle_products", "bundle_prices",
		"download_apps", "download_assets", "download_artifacts", "credit_wallets",
		"credit_transactions", "payment_settings", "usage_records", "credit_reservations",
		"api_keys", "users", "auth_tokens", "user_sessions", "payment_anomaly_log",
	} {
		if !strings.Contains(sql, "create table if not exists "+table) {
			t.Errorf("missing declarative definition for %s", table)
		}
	}
}

func TestHealthEndpoint(t *testing.T) {
	t.Run("healthy when database reachable", func(t *testing.T) {
		srv, cleanup := newHealthTestServer(t)
		defer cleanup()

		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		healthHandler := health.New().Version("1.0.0").Check(health.DB(srv.primaryDB()), health.Critical).Handler()
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

		healthHandler := health.New().Version("1.0.0").Check(health.DB(srv.primaryDB()), health.Critical).Handler()
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

// NOTE: TestParseEnvBool has been removed - parseEnvBool function was deleted

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
	var output bytes.Buffer
	previousOutput := log.Writer()
	previousFlags := log.Flags()
	log.SetOutput(&output)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
		log.SetFlags(previousFlags)
	})

	logx.Info(`test "event"`, map[string]interface{}{
		"key":   "value",
		"count": 42,
	})

	logx.Info("test_event_no_fields", nil)
	logged := output.String()
	if !strings.Contains(logged, `"msg":"test \"event\""`) || !strings.Contains(logged, `"count":42`) {
		t.Fatalf("structured event log missing expected fields: %s", logged)
	}
	if !strings.Contains(logged, `"msg":"test_event_no_fields"`) {
		t.Fatalf("structured event without fields missing: %s", logged)
	}
	for _, line := range strings.Split(strings.TrimSpace(logged), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("structured log is not valid JSON: %v (%s)", err, line)
		}
	}
}
