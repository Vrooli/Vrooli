//go:build integration

package store_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"tunnel-manager/domain"
	"tunnel-manager/handler"
	"tunnel-manager/service"
	"tunnel-manager/store"
	"tunnel-manager/testutil"

	"github.com/gorilla/mux"
)

// [REQ:ROUTE-001] Route manifest schema - validates the routes table structure
func TestRouteManifestSchema(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	// Verify the routes table has the expected columns
	rows, err := db.Query(`SELECT column_name, data_type FROM information_schema.columns WHERE table_name = 'routes' ORDER BY ordinal_position`)
	if err != nil {
		t.Fatalf("query columns: %v", err)
	}
	defer rows.Close()

	expected := map[string]bool{
		"id":            false,
		"subdomain":     false,
		"scenario_name": false,
		"local_port":    false,
		"health_path":   false,
		"public_url":    false,
		"enabled":       false,
		"created_at":    false,
		"updated_at":    false,
	}

	for rows.Next() {
		var col, dtype string
		if err := rows.Scan(&col, &dtype); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		if _, ok := expected[col]; ok {
			expected[col] = true
		}
	}

	for col, found := range expected {
		if !found {
			t.Errorf("missing column: %s", col)
		}
	}
}

// [REQ:ROUTE-001] Route manifest schema - validates subdomain uniqueness
func TestRouteManifestSubdomainUniqueness(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	routeStore := store.NewRouteStore(db)
	svc := service.NewRouteService(routeStore)

	_, err := svc.Create(domain.RouteInput{
		Subdomain:    "test-app",
		ScenarioName: "my-scenario",
		LocalPort:    3000,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = svc.Create(domain.RouteInput{
		Subdomain:    "test-app",
		ScenarioName: "other-scenario",
		LocalPort:    3001,
	})
	if err == nil {
		t.Fatal("expected duplicate subdomain error, got nil")
	}
}

// [REQ:ROUTE-002] Route manifest CRUD via API - create route
func TestHandlerCreateRoute(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	routeStore := store.NewRouteStore(db)
	svc := service.NewRouteService(routeStore)
	h := handler.HandleCreateRoute(svc)

	body := `{"subdomain":"agent-manager","scenario_name":"agent-manager","local_port":35001,"public_url":"https://agent-manager.example.com"}`
	req := httptest.NewRequest("POST", "/api/v1/routes", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var route domain.Route
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
	db := testutil.SetupTestDB(t)
	defer db.Close()

	routeStore := store.NewRouteStore(db)
	svc := service.NewRouteService(routeStore)
	testutil.SeedTestRoute(t, db, "app-a", "scenario-a", 3000)
	testutil.SeedTestRoute(t, db, "app-b", "scenario-b", 3001)

	h := handler.HandleListRoutes(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes", nil)
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var routes []domain.Route
	if err := json.Unmarshal(w.Body.Bytes(), &routes); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
}

// [REQ:ROUTE-002] Route manifest CRUD via API - get single route
func TestHandlerGetRoute(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	routeStore := store.NewRouteStore(db)
	svc := service.NewRouteService(routeStore)
	route := testutil.SeedTestRoute(t, db, "my-app", "my-scenario", 3000)

	h := handler.HandleGetRoute(svc)
	req := httptest.NewRequest("GET", "/api/v1/routes/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": testutil.Itoa(route.ID)})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// [REQ:ROUTE-002] Route manifest CRUD via API - update route
func TestHandlerUpdateRoute(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	routeStore := store.NewRouteStore(db)
	svc := service.NewRouteService(routeStore)
	route := testutil.SeedTestRoute(t, db, "my-app", "my-scenario", 3000)

	h := handler.HandleUpdateRoute(svc)
	body := `{"local_port":3500}`
	req := httptest.NewRequest("PUT", "/api/v1/routes/1", bytes.NewBufferString(body))
	req = mux.SetURLVars(req, map[string]string{"id": testutil.Itoa(route.ID)})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated domain.Route
	if err := json.Unmarshal(w.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if updated.LocalPort != 3500 {
		t.Errorf("local_port = %d, want 3500", updated.LocalPort)
	}
}

// [REQ:ROUTE-002] Route manifest CRUD via API - delete route
func TestHandlerDeleteRoute(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	routeStore := store.NewRouteStore(db)
	svc := service.NewRouteService(routeStore)
	route := testutil.SeedTestRoute(t, db, "del-app", "del-scenario", 3000)

	h := handler.HandleDeleteRoute(svc)
	req := httptest.NewRequest("DELETE", "/api/v1/routes/1", nil)
	req = mux.SetURLVars(req, map[string]string{"id": testutil.Itoa(route.ID)})
	w := httptest.NewRecorder()
	h(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify deleted
	got, _ := svc.GetByID(route.ID)
	if got != nil {
		t.Error("route should be deleted")
	}
}

// [REQ:ROUTE-002] Store-level CRUD tests

func TestRouteStore_CreateAndGet(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	rs := store.NewRouteStore(db)

	created, err := rs.Create("store-test", "test-scenario", 4000, "/health", "https://store-test.example.com", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected non-zero ID")
	}

	got, err := rs.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Subdomain != "store-test" {
		t.Errorf("subdomain = %q, want store-test", got.Subdomain)
	}
	if got.LocalPort != 4000 {
		t.Errorf("local_port = %d, want 4000", got.LocalPort)
	}
}

func TestRouteStore_List(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	rs := store.NewRouteStore(db)

	_, _ = rs.Create("list-a", "scenario-a", 3000, "/health", "", true)
	_, _ = rs.Create("list-b", "scenario-b", 3001, "/health", "", true)

	routes, err := rs.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(routes))
	}
}

func TestRouteStore_Update(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	rs := store.NewRouteStore(db)

	created, err := rs.Create("update-test", "test-scenario", 3000, "/health", "", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updated, err := rs.Update(created.ID, "update-test", "test-scenario", 4000, "/healthz", "", true)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.LocalPort != 4000 {
		t.Errorf("local_port = %d, want 4000", updated.LocalPort)
	}
	if updated.HealthPath != "/healthz" {
		t.Errorf("health_path = %q, want /healthz", updated.HealthPath)
	}
}

func TestRouteStore_Delete(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	rs := store.NewRouteStore(db)

	created, err := rs.Create("delete-test", "test-scenario", 3000, "/health", "", true)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	err = rs.Delete(created.ID)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	got, err := rs.GetByID(created.ID)
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after delete")
	}
}

func TestRouteStore_DeleteNotFound(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	rs := store.NewRouteStore(db)

	err := rs.Delete(99999)
	if err == nil {
		t.Error("expected error deleting non-existent route")
	}
}
