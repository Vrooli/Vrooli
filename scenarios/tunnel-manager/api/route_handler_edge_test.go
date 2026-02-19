package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

// [REQ:ROUTE-002] Route handler edge cases

func TestHandlerGetRoute_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleGetRoute(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes/99999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "99999"})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandlerGetRoute_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleGetRoute(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerCreateRoute_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleCreateRoute(svc)
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString("{invalid"))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerCreateRoute_MissingRequiredFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleCreateRoute(svc)
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString(`{"subdomain":"test"}`))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerUpdateRoute_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleUpdateRoute(svc)
	req := httptest.NewRequest("PUT", "/api/v1/routes/abc", bytes.NewBufferString(`{"local_port":3500}`))
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerUpdateRoute_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleUpdateRoute(svc)
	req := httptest.NewRequest("PUT", "/api/v1/routes/99999", bytes.NewBufferString(`{"local_port":3500}`))
	req = mux.SetURLVars(req, map[string]string{"id": "99999"})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandlerUpdateRoute_InvalidJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleUpdateRoute(svc)
	req := httptest.NewRequest("PUT", "/api/v1/routes/1", bytes.NewBufferString("{bad"))
	req = mux.SetURLVars(req, map[string]string{"id": "1"})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerDeleteRoute_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleDeleteRoute(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/routes/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandlerDeleteRoute_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleDeleteRoute(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/routes/99999", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "99999"})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandlerListRoutes_EmptyDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

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
	if len(routes) != 0 {
		t.Errorf("expected empty list, got %d routes", len(routes))
	}
}

func TestHandlerCreateRoute_DuplicateSubdomain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)
	seedTestRoute(t, db, "dup-test", "scenario-a", 3000)

	handler := handleCreateRoute(svc)
	body := `{"subdomain":"dup-test","scenario_name":"scenario-b","local_port":3001,"public_url":"https://dup-test.example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code == http.StatusCreated {
		t.Error("expected error for duplicate subdomain, got 201")
	}
}

func TestHandlerCreateRoute_CustomHealthPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleCreateRoute(svc)
	body := `{"subdomain":"custom-hp","scenario_name":"test","local_port":3000,"health_path":"/api/health","public_url":"https://custom.example.com"}`
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
	if route.HealthPath != "/api/health" {
		t.Errorf("health_path = %q, want %q", route.HealthPath, "/api/health")
	}
}

func TestHandlerUpdateRoute_MultipleFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)
	route := seedTestRoute(t, db, "multi-update", "test-scenario", 3000)

	handler := handleUpdateRoute(svc)
	body := `{"local_port":4000,"health_path":"/healthz","public_url":"https://updated.example.com"}`
	req := httptest.NewRequest("PUT", "/api/v1/routes/1", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": itoa(route.ID)})
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated Route
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if updated.LocalPort != 4000 {
		t.Errorf("local_port = %d, want 4000", updated.LocalPort)
	}
	if updated.HealthPath != "/healthz" {
		t.Errorf("health_path = %q, want /healthz", updated.HealthPath)
	}
	if updated.PublicURL != "https://updated.example.com" {
		t.Errorf("public_url = %q, want https://updated.example.com", updated.PublicURL)
	}
	// Unchanged fields should be preserved
	if updated.Subdomain != "multi-update" {
		t.Errorf("subdomain = %q, want multi-update (should be preserved)", updated.Subdomain)
	}
}

func TestHandlerCreateRoute_DisabledRoute(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	handler := handleCreateRoute(svc)
	body := `{"subdomain":"disabled-app","scenario_name":"test","local_port":3000,"enabled":false,"public_url":"https://disabled.example.com"}`
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
	if route.Enabled {
		t.Error("expected enabled=false")
	}
}
