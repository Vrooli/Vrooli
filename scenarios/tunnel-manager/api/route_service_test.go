package main

import (
	"testing"
)

// [REQ:ROUTE-002] RouteService service-level edge case tests

func TestRouteService_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	got, err := svc.GetByID(99999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent route")
	}
}

func TestRouteService_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	err := svc.Delete(99999)
	if err == nil {
		t.Error("expected error deleting non-existent route")
	}
}

func TestRouteService_Update_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	got, err := svc.Update(99999, RouteInput{LocalPort: 3000})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent route update")
	}
}

func TestRouteService_List_EmptyDB(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	routes, err := svc.List()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 0 {
		t.Errorf("expected 0 routes, got %d", len(routes))
	}
}

func TestRouteService_List_OrderedBySubdomain(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	seedTestRoute(t, db, "z-app", "scenario-z", 3002)
	seedTestRoute(t, db, "a-app", "scenario-a", 3000)
	seedTestRoute(t, db, "m-app", "scenario-m", 3001)

	routes, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes))
	}
	if routes[0].Subdomain != "a-app" {
		t.Errorf("first route subdomain = %q, want a-app", routes[0].Subdomain)
	}
	if routes[1].Subdomain != "m-app" {
		t.Errorf("second route subdomain = %q, want m-app", routes[1].Subdomain)
	}
	if routes[2].Subdomain != "z-app" {
		t.Errorf("third route subdomain = %q, want z-app", routes[2].Subdomain)
	}
}

func TestRouteService_Create_TimestampsSet(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	route, err := svc.Create(RouteInput{
		Subdomain:    "ts-test",
		ScenarioName: "test",
		LocalPort:    3000,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if route.CreatedAt.IsZero() {
		t.Error("created_at should be set")
	}
	if route.UpdatedAt.IsZero() {
		t.Error("updated_at should be set")
	}
}

func TestRouteService_Update_PreservesUnchangedFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	original := seedTestRoute(t, db, "preserve-test", "original-scenario", 3000)

	updated, err := svc.Update(original.ID, RouteInput{LocalPort: 4000})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Subdomain != "preserve-test" {
		t.Errorf("subdomain changed: %q", updated.Subdomain)
	}
	if updated.ScenarioName != "original-scenario" {
		t.Errorf("scenario_name changed: %q", updated.ScenarioName)
	}
	if updated.LocalPort != 4000 {
		t.Errorf("local_port = %d, want 4000", updated.LocalPort)
	}
}

// [REQ:ROUTE-004] Validation edge cases

func TestRouteValidation_NegativePort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	_, err := svc.Create(RouteInput{
		Subdomain:    "neg-port",
		ScenarioName: "test",
		LocalPort:    -1,
	})
	if err == nil {
		t.Error("expected error for negative port")
	}
}

func TestRouteValidation_MaxPort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	route, err := svc.Create(RouteInput{
		Subdomain:    "max-port",
		ScenarioName: "test",
		LocalPort:    65535,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if route.LocalPort != 65535 {
		t.Errorf("local_port = %d, want 65535", route.LocalPort)
	}
}

func TestRouteValidation_MinPort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	route, err := svc.Create(RouteInput{
		Subdomain:    "min-port",
		ScenarioName: "test",
		LocalPort:    1,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if route.LocalPort != 1 {
		t.Errorf("local_port = %d, want 1", route.LocalPort)
	}
}

func TestRouteValidation_UpdateInvalidPort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	route := seedTestRoute(t, db, "update-bad-port", "test", 3000)
	_, err := svc.Update(route.ID, RouteInput{LocalPort: 99999})
	if err == nil {
		t.Error("expected error for port > 65535 on update")
	}
}
