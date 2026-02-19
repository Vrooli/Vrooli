package main

import (
	"os"
	"path/filepath"
	"testing"
)

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
	if err := os.MkdirAll(dir1, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeServiceJSON(t, dir1, 35000)

	dir2 := filepath.Join(tmpDir, "app-b", ".vrooli")
	if err := os.MkdirAll(dir2, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
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
