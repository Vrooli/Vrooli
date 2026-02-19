package main

import (
	"testing"
)

// [REQ:ROUTE-001] Route manifest schema - validates the routes table structure
func TestRouteManifestSchema(t *testing.T) {
	db := setupTestDB(t)
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
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)

	_, err := svc.Create(RouteInput{
		Subdomain:    "test-app",
		ScenarioName: "my-scenario",
		LocalPort:    3000,
	})
	if err != nil {
		t.Fatalf("first create: %v", err)
	}

	_, err = svc.Create(RouteInput{
		Subdomain:    "test-app",
		ScenarioName: "other-scenario",
		LocalPort:    3001,
	})
	if err == nil {
		t.Fatal("expected duplicate subdomain error, got nil")
	}
}
