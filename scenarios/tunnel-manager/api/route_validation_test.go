package main

import (
	"testing"
)

// [REQ:ROUTE-004] Route manifest validation - missing required fields
func TestRouteValidationMissingFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	tests := []struct {
		name  string
		input RouteInput
		errIs string
	}{
		{"missing subdomain", RouteInput{ScenarioName: "x", LocalPort: 3000}, "subdomain"},
		{"missing scenario_name", RouteInput{Subdomain: "x", LocalPort: 3000}, "scenario_name"},
		{"missing local_port", RouteInput{Subdomain: "x", ScenarioName: "x"}, "local_port"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Create(tc.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}

// [REQ:ROUTE-004] Route manifest validation - invalid port range
func TestRouteValidationInvalidPort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	_, err := svc.Create(RouteInput{
		Subdomain:    "test",
		ScenarioName: "test",
		LocalPort:    99999,
	})
	if err == nil {
		t.Fatal("expected error for port > 65535")
	}
}

// [REQ:ROUTE-004] Route manifest validation - default health path
func TestRouteValidationDefaultHealthPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	svc := NewRouteService(db)

	route, err := svc.Create(RouteInput{
		Subdomain:    "test-defaults",
		ScenarioName: "test",
		LocalPort:    3000,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if route.HealthPath != "/health" {
		t.Errorf("health_path = %q, want %q", route.HealthPath, "/health")
	}
	if !route.Enabled {
		t.Error("enabled should default to true")
	}
}
