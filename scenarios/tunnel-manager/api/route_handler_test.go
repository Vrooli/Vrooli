package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
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

// [REQ:ROUTE-002] Route manifest CRUD via API - get single route
func TestHandlerGetRoute(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	route := seedTestRoute(t, db, "my-app", "my-scenario", 3000)

	handler := handleGetRoute(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": itoa(route.ID)})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// [REQ:ROUTE-002] Route manifest CRUD via API - update route
func TestHandlerUpdateRoute(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	route := seedTestRoute(t, db, "my-app", "my-scenario", 3000)

	handler := handleUpdateRoute(svc)
	body := `{"local_port":3500}`
	req := httptest.NewRequest("PUT", "/api/v1/routes/1", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": itoa(route.ID)})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated Route
	json.Unmarshal(w.Body.Bytes(), &updated)
	if updated.LocalPort != 3500 {
		t.Errorf("local_port = %d, want 3500", updated.LocalPort)
	}
}

// [REQ:ROUTE-002] Route manifest CRUD via API - delete route
func TestHandlerDeleteRoute(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	route := seedTestRoute(t, db, "del-app", "del-scenario", 3000)

	handler := handleDeleteRoute(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/routes/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": itoa(route.ID)})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	got, _ := svc.GetByID(route.ID)
	if got != nil {
		t.Error("route should be deleted")
	}
}

func itoa(i int) string {
	return fmt.Sprintf("%d", i)
}
