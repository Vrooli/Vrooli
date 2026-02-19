package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:PORT-001] Scan scenario service.json for port field
func TestPortAuditorScansServiceJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	tmpDir := t.TempDir()

	// Create a fake scenario with service.json
	scenarioDir := filepath.Join(tmpDir, "test-scenario", ".vrooli")
	os.MkdirAll(scenarioDir, 0o755)
	writeServiceJSON(t, scenarioDir, 35000)

	seedTestRoute(t, db, "test", "test-scenario", 35000)

	auditor := NewPortAuditor(svc, tmpDir)
	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "compliant" {
		t.Errorf("status = %q, want compliant", results[0].Status)
	}
}

// [REQ:PORT-002] Port value match verification
func TestPortAuditorDetectsMismatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	tmpDir := t.TempDir()

	scenarioDir := filepath.Join(tmpDir, "mismatch-scenario", ".vrooli")
	os.MkdirAll(scenarioDir, 0o755)
	writeServiceJSON(t, scenarioDir, 35000) // actual port

	seedTestRoute(t, db, "mismatch", "mismatch-scenario", 36000) // expected port differs

	auditor := NewPortAuditor(svc, tmpDir)
	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "mismatch" {
		t.Errorf("status = %q, want mismatch", results[0].Status)
	}
	if results[0].ActualPort != 35000 {
		t.Errorf("actual_port = %d, want 35000", results[0].ActualPort)
	}
}

// [REQ:PORT-003] Missing scenario detection
func TestPortAuditorDetectsMissingScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	tmpDir := t.TempDir()
	// No scenario directory created

	seedTestRoute(t, db, "ghost", "nonexistent-scenario", 3000)

	auditor := NewPortAuditor(svc, tmpDir)
	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "missing_scenario" {
		t.Errorf("status = %q, want missing_scenario", results[0].Status)
	}
}

// [REQ:PORT-004] Audit result reporting via API
func TestPortAuditAPIReporting(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	tmpDir := t.TempDir()

	// One compliant, one mismatch
	dir1 := filepath.Join(tmpDir, "app-a", ".vrooli")
	os.MkdirAll(dir1, 0o755)
	writeServiceJSON(t, dir1, 35000)

	dir2 := filepath.Join(tmpDir, "app-b", ".vrooli")
	os.MkdirAll(dir2, 0o755)
	writeServiceJSON(t, dir2, 35001)

	seedTestRoute(t, db, "a", "app-a", 35000) // compliant
	seedTestRoute(t, db, "b", "app-b", 36000) // mismatch

	auditor := NewPortAuditor(svc, tmpDir)
	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit: %v", err)
	}

	compliant, violations := 0, 0
	for _, r := range results {
		if r.Status == "compliant" {
			compliant++
		} else {
			violations++
		}
	}
	if compliant != 1 {
		t.Errorf("compliant = %d, want 1", compliant)
	}
	if violations != 1 {
		t.Errorf("violations = %d, want 1", violations)
	}
}

func writeServiceJSON(t *testing.T, dir string, uiPort int) {
	t.Helper()
	svc := map[string]any{
		"ports": map[string]any{
			"ui": map[string]any{
				"port":    uiPort,
				"env_var": "UI_PORT",
			},
		},
	}
	data, _ := json.Marshal(svc)
	if err := os.WriteFile(filepath.Join(dir, "service.json"), data, 0o644); err != nil {
		t.Fatalf("write service.json: %v", err)
	}
}
