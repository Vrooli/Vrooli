package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// [REQ:PORT-001] [REQ:PORT-002] Port auditor unit tests

func TestPortAuditor_EmptyRoutes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	auditor := NewPortAuditor(svc, t.TempDir())

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty routes, got %d", len(results))
	}
}

func TestPortAuditor_DisabledRouteSkipped(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	_, err := svc.Create(RouteInput{
		Subdomain:    "disabled-app",
		ScenarioName: "test-scenario",
		LocalPort:    3000,
		Enabled:      boolPtr(false),
	})
	if err != nil {
		t.Fatal(err)
	}

	auditor := NewPortAuditor(svc, t.TempDir())

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("disabled routes should be skipped, got %d results", len(results))
	}
}

func TestPortAuditor_MissingScenario(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	seedTestRoute(t, db, "test-app", "nonexistent", 3000)
	svc := NewRouteService(db)
	auditor := NewPortAuditor(svc, t.TempDir())

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "missing_scenario" {
		t.Errorf("status = %q, want missing_scenario", results[0].Status)
	}
}

func TestPortAuditor_Compliant(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	scenariosDir := t.TempDir()
	scenarioDir := filepath.Join(scenariosDir, "my-app", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeServiceJSON(t, scenarioDir, 3000)

	seedTestRoute(t, db, "my-app", "my-app", 3000)
	svc := NewRouteService(db)
	auditor := NewPortAuditor(svc, scenariosDir)

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "compliant" {
		t.Errorf("status = %q, want compliant", results[0].Status)
	}
	if results[0].ActualPort != 3000 {
		t.Errorf("actual_port = %d, want 3000", results[0].ActualPort)
	}
}

func TestPortAuditor_Mismatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	scenariosDir := t.TempDir()
	scenarioDir := filepath.Join(scenariosDir, "my-app", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeServiceJSON(t, scenarioDir, 4000) // different from route port

	seedTestRoute(t, db, "my-app", "my-app", 3000)
	svc := NewRouteService(db)
	auditor := NewPortAuditor(svc, scenariosDir)

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "mismatch" {
		t.Errorf("status = %q, want mismatch", results[0].Status)
	}
	if results[0].ActualPort != 4000 {
		t.Errorf("actual_port = %d, want 4000", results[0].ActualPort)
	}
}

func TestPortAuditor_InvalidServiceJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	scenariosDir := t.TempDir()
	scenarioDir := filepath.Join(scenariosDir, "bad-app", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "service.json"), []byte("invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	seedTestRoute(t, db, "bad-app", "bad-app", 3000)
	svc := NewRouteService(db)
	auditor := NewPortAuditor(svc, scenariosDir)

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "missing_port" {
		t.Errorf("status = %q, want missing_port", results[0].Status)
	}
}

func TestPortAuditor_NoUIPort(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	scenariosDir := t.TempDir()
	scenarioDir := filepath.Join(scenariosDir, "no-ui", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write service.json without UI port
	if err := os.WriteFile(filepath.Join(scenarioDir, "service.json"), []byte(`{"ports":{"api":{"port":8080}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	seedTestRoute(t, db, "no-ui", "no-ui", 3000)
	svc := NewRouteService(db)
	auditor := NewPortAuditor(svc, scenariosDir)

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "missing_port" {
		t.Errorf("status = %q, want missing_port", results[0].Status)
	}
}

func TestPortAuditor_MultipleRoutes(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	scenariosDir := t.TempDir()
	for _, s := range []struct {
		name string
		port int
	}{
		{"app1", 3000},
		{"app2", 4000},
	} {
		dir := filepath.Join(scenariosDir, s.name, ".vrooli")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		writeServiceJSON(t, dir, s.port)
		seedTestRoute(t, db, s.name, s.name, s.port)
	}

	svc := NewRouteService(db)
	auditor := NewPortAuditor(svc, scenariosDir)

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != "compliant" {
			t.Errorf("route %s: status = %q, want compliant", r.Subdomain, r.Status)
		}
	}
}

func TestPortAuditHandler(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	svc := NewRouteService(db)
	auditor := NewPortAuditor(svc, t.TempDir())

	handler := handlePortAudit(auditor)
	req := httptest.NewRequest("GET", "/api/v1/audit/ports", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Results    []PortAuditResult `json:"results"`
		Total      int               `json:"total"`
		Violations int               `json:"violations"`
		Compliant  int               `json:"compliant"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Total != 0 {
		t.Errorf("total = %d, want 0", resp.Total)
	}
}

func boolPtr(b bool) *bool { return &b }
