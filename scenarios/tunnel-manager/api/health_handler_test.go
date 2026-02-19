package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:OBS-004] Health API endpoint tests

func TestDetailedHealth_EmptyRoutes(t *testing.T) {
	db := setupTestDB(t)
	routeSvc := NewRouteService(db)
	probeSvc := NewProbeService(db, routeSvc)

	mockRunner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("active"), nil
	}
	tunnelHealth := NewTunnelHealthChecker(WithCmdRunner(mockRunner))

	handler := handleDetailedHealth(tunnelHealth, routeSvc, probeSvc)
	req := httptest.NewRequest("GET", "/api/v1/health/detailed", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rr.Code)
	}

	var resp DetailedHealth
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if resp.Timestamp == "" {
		t.Error("timestamp should not be empty")
	}
	if len(resp.Routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(resp.Routes))
	}
}

func TestDetailedHealth_WithRoutes(t *testing.T) {
	db := setupTestDB(t)
	routeSvc := NewRouteService(db)
	probeSvc := NewProbeService(db, routeSvc)

	mockRunner := func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return []byte("active"), nil
	}
	tunnelHealth := NewTunnelHealthChecker(WithCmdRunner(mockRunner))

	// Seed routes
	seedTestRoute(t, db, "app-a", "scenario-a", 8080)
	seedTestRoute(t, db, "app-b", "scenario-b", 9090)

	handler := handleDetailedHealth(tunnelHealth, routeSvc, probeSvc)
	req := httptest.NewRequest("GET", "/api/v1/health/detailed", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: want 200, got %d", rr.Code)
	}

	var resp DetailedHealth
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(resp.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(resp.Routes))
	}
}
