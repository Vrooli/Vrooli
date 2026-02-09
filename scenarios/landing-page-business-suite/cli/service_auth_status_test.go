package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func withAPIBase(t *testing.T, base string) {
	t.Helper()
	const envKey = "LANDING_PAGE_BUSINESS_SUITE_API_BASE"
	previous, had := os.LookupEnv(envKey)
	if err := os.Setenv(envKey, base); err != nil {
		t.Fatalf("set env %s: %v", envKey, err)
	}
	t.Cleanup(func() {
		if !had {
			_ = os.Unsetenv(envKey)
			return
		}
		_ = os.Setenv(envKey, previous)
	})
}

func TestServiceAuthStatusRequireEnabledPassesWhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/usage/health" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"healthy":true,"database_connected":true,"service_auth_configured":true,"service_auth_mode":"token"}`))
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	if err := app.cmdServiceAuthStatus([]string{"--require-enabled"}); err != nil {
		t.Fatalf("cmdServiceAuthStatus returned error: %v", err)
	}
}

func TestServiceAuthStatusRequireEnabledFailsWhenDisabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/usage/health" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"healthy":true,"database_connected":true,"service_auth_configured":false,"service_auth_mode":"disabled"}`))
	}))
	defer server.Close()

	withAPIBase(t, server.URL)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	if err := app.cmdServiceAuthStatus([]string{"--require-enabled"}); err == nil {
		t.Fatal("expected error when service auth is disabled")
	}
}
