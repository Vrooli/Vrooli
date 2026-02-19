package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:CFAPI-006] Mode switching tests

func TestModeSwitcher_SwitchToRemote(t *testing.T) {
	db := setupTestDB(t)
	routeSvc := NewRouteService(db)

	// Seed a route
	seedTestRoute(t, db, "test-app", "test-scenario", 8080)

	// Mock CF API
	var pushedConfig map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			_ = json.NewDecoder(r.Body).Decode(&pushedConfig)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer srv.Close()

	t.Setenv("CF_API_TOKEN", "test-token")
	t.Setenv("CF_ACCOUNT_ID", "acc123")
	t.Setenv("CF_TUNNEL_ID", "tun456")

	cfClient, err := NewCFClient(WithCFBaseURL(srv.URL))
	if err != nil {
		t.Fatalf("NewCFClient: %v", err)
	}

	switcher := NewModeSwitcher(nil, cfClient, routeSvc)
	if err := switcher.SwitchTo(context.Background(), ModeRemote); err != nil {
		t.Fatalf("SwitchTo remote: %v", err)
	}

	if pushedConfig == nil {
		t.Fatal("config was not pushed to CF API")
	}
}

func TestModeSwitcher_SwitchToRemoteRequiresClient(t *testing.T) {
	db := setupTestDB(t)
	routeSvc := NewRouteService(db)

	switcher := NewModeSwitcher(nil, nil, routeSvc)
	err := switcher.SwitchTo(context.Background(), ModeRemote)
	if err == nil {
		t.Fatal("expected error without CF client")
	}
}

func TestModeSwitcher_InvalidMode(t *testing.T) {
	db := setupTestDB(t)
	routeSvc := NewRouteService(db)

	switcher := NewModeSwitcher(nil, nil, routeSvc)
	err := switcher.SwitchTo(context.Background(), "invalid")
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
}
