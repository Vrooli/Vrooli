package main

import (
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
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
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
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
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
