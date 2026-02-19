package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// [REQ:ROUTE-002] Route manifest CRUD via API - create route
func TestHandlerCreateRoute(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	handler := handleCreateRoute(svc)

	body := `{"subdomain":"agent-manager","scenario_name":"agent-manager","local_port":35001,"public_url":"https://agent-manager.example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var route Route
	if err := json.Unmarshal(w.Body.Bytes(), &route); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if route.Subdomain != "agent-manager" {
		t.Errorf("subdomain = %q, want %q", route.Subdomain, "agent-manager")
	}
	if route.LocalPort != 35001 {
		t.Errorf("local_port = %d, want %d", route.LocalPort, 35001)
	}
}

// [REQ:ROUTE-002] Route manifest CRUD via API - list routes
func TestHandlerListRoutes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	seedTestRoute(t, db, "app-a", "scenario-a", 3000)
	seedTestRoute(t, db, "app-b", "scenario-b", 3001)

	handler := handleListRoutes(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var routes []Route
	if err := json.Unmarshal(w.Body.Bytes(), &routes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
}
