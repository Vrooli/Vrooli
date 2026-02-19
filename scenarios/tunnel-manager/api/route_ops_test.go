package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

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
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
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
