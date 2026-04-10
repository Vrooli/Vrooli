package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tunnel-manager/domain"
)

// writeServiceJSON creates a minimal service.json with a UI port for testing.
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

// [REQ:PORT-001] [REQ:PORT-002] Port auditor unit tests

func TestPortAuditor_EmptyRoutes(t *testing.T) {
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{}, nil
		},
	}
	auditor := NewPortAuditor(lister, t.TempDir())

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty routes, got %d", len(results))
	}
}

func TestPortAuditor_DisabledRouteSkipped(t *testing.T) {
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "disabled-app", ScenarioName: "test-scenario", LocalPort: 3000, Enabled: false},
			}, nil
		},
	}
	auditor := NewPortAuditor(lister, t.TempDir())

	results, err := auditor.Audit()
	if err != nil {
		t.Fatalf("audit error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("disabled routes should be skipped, got %d results", len(results))
	}
}

func TestPortAuditor_MissingScenario(t *testing.T) {
	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "test-app", ScenarioName: "nonexistent", LocalPort: 3000, Enabled: true},
			}, nil
		},
	}
	auditor := NewPortAuditor(lister, t.TempDir())

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
	scenariosDir := t.TempDir()
	scenarioDir := filepath.Join(scenariosDir, "my-app", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeServiceJSON(t, scenarioDir, 3000)

	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "my-app", ScenarioName: "my-app", LocalPort: 3000, Enabled: true},
			}, nil
		},
	}
	auditor := NewPortAuditor(lister, scenariosDir)

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
	scenariosDir := t.TempDir()
	scenarioDir := filepath.Join(scenariosDir, "my-app", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeServiceJSON(t, scenarioDir, 4000) // different from route port

	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "my-app", ScenarioName: "my-app", LocalPort: 3000, Enabled: true},
			}, nil
		},
	}
	auditor := NewPortAuditor(lister, scenariosDir)

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
	scenariosDir := t.TempDir()
	scenarioDir := filepath.Join(scenariosDir, "bad-app", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scenarioDir, "service.json"), []byte("invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "bad-app", ScenarioName: "bad-app", LocalPort: 3000, Enabled: true},
			}, nil
		},
	}
	auditor := NewPortAuditor(lister, scenariosDir)

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
	scenariosDir := t.TempDir()
	scenarioDir := filepath.Join(scenariosDir, "no-ui", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write service.json without UI port
	if err := os.WriteFile(filepath.Join(scenarioDir, "service.json"), []byte(`{"ports":{"api":{"port":8080}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "no-ui", ScenarioName: "no-ui", LocalPort: 3000, Enabled: true},
			}, nil
		},
	}
	auditor := NewPortAuditor(lister, scenariosDir)

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
	}

	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "app1", ScenarioName: "app1", LocalPort: 3000, Enabled: true},
				{Subdomain: "app2", ScenarioName: "app2", LocalPort: 4000, Enabled: true},
			}, nil
		},
	}
	auditor := NewPortAuditor(lister, scenariosDir)

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

// [REQ:PORT-003] Missing scenario detection
func TestPortAuditorDetectsMissingScenario(t *testing.T) {
	tmpDir := t.TempDir()
	// No scenario directory created

	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "ghost", ScenarioName: "nonexistent-scenario", LocalPort: 3000, Enabled: true},
			}, nil
		},
	}

	auditor := NewPortAuditor(lister, tmpDir)
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

	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "a", ScenarioName: "app-a", LocalPort: 35000, Enabled: true},
				{Subdomain: "b", ScenarioName: "app-b", LocalPort: 36000, Enabled: true},
			}, nil
		},
	}

	auditor := NewPortAuditor(lister, tmpDir)
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

// [REQ:PORT-001] Scan scenario service.json for port field
func TestPortAuditorScansServiceJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a fake scenario with service.json
	scenarioDir := filepath.Join(tmpDir, "test-scenario", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeServiceJSON(t, scenarioDir, 35000)

	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "test", ScenarioName: "test-scenario", LocalPort: 35000, Enabled: true},
			}, nil
		},
	}

	auditor := NewPortAuditor(lister, tmpDir)
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
	tmpDir := t.TempDir()

	scenarioDir := filepath.Join(tmpDir, "mismatch-scenario", ".vrooli")
	if err := os.MkdirAll(scenarioDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeServiceJSON(t, scenarioDir, 35000) // actual port

	lister := &mockRouteLister{
		listFn: func() ([]domain.Route, error) {
			return []domain.Route{
				{Subdomain: "mismatch", ScenarioName: "mismatch-scenario", LocalPort: 36000, Enabled: true},
			}, nil
		},
	}

	auditor := NewPortAuditor(lister, tmpDir)
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
