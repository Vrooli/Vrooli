package backlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseArchiveTargets_Modern(t *testing.T) {
	dir := t.TempDir()
	prdContent := `# Test PRD

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Core feature | Essential functionality
- [ ] OT-P0-002 | Auth system | User authentication ` + "`[req:REQ-001]`" + `

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Dashboard | Analytics dashboard

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | API v2 | Next gen API

## Next Section
Some other content.
`
	if err := os.WriteFile(filepath.Join(dir, "PRD.md"), []byte(prdContent), 0o644); err != nil {
		t.Fatalf("failed to write PRD.md: %v", err)
	}

	targets, err := ParseArchiveTargets(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 4 {
		t.Fatalf("expected 4 targets, got %d", len(targets))
	}

	// Check first target (P0, complete).
	if targets[0].ID != "OT-P0-001" {
		t.Errorf("target[0].ID = %q, want %q", targets[0].ID, "OT-P0-001")
	}
	if targets[0].Criticality != "P0" {
		t.Errorf("target[0].Criticality = %q, want %q", targets[0].Criticality, "P0")
	}
	if targets[0].Title != "Core feature" {
		t.Errorf("target[0].Title = %q, want %q", targets[0].Title, "Core feature")
	}
	if targets[0].Notes != "Essential functionality" {
		t.Errorf("target[0].Notes = %q, want %q", targets[0].Notes, "Essential functionality")
	}
	if targets[0].Status != "complete" {
		t.Errorf("target[0].Status = %q, want %q", targets[0].Status, "complete")
	}

	// Check second target (P0, pending, with req link).
	if targets[1].ID != "OT-P0-002" {
		t.Errorf("target[1].ID = %q, want %q", targets[1].ID, "OT-P0-002")
	}
	if targets[1].Status != "pending" {
		t.Errorf("target[1].Status = %q, want %q", targets[1].Status, "pending")
	}
	if len(targets[1].LinkedRequirements) != 1 || targets[1].LinkedRequirements[0] != "REQ-001" {
		t.Errorf("target[1].LinkedRequirements = %v, want [REQ-001]", targets[1].LinkedRequirements)
	}

	// Check P1 target.
	if targets[2].Criticality != "P1" {
		t.Errorf("target[2].Criticality = %q, want %q", targets[2].Criticality, "P1")
	}
	if targets[2].Title != "Dashboard" {
		t.Errorf("target[2].Title = %q, want %q", targets[2].Title, "Dashboard")
	}

	// Check P2 target.
	if targets[3].Criticality != "P2" {
		t.Errorf("target[3].Criticality = %q, want %q", targets[3].Criticality, "P2")
	}
	if targets[3].ID != "OT-P2-001" {
		t.Errorf("target[3].ID = %q, want %q", targets[3].ID, "OT-P2-001")
	}
}

func TestParseArchiveTargets_Legacy(t *testing.T) {
	dir := t.TempDir()
	prdContent := `# Test PRD

### Functional Requirements
- **Authentication (P0)**
- [x] Login system _(basic auth)_
- [ ] OAuth integration

- **Dashboard (P1)**
- [ ] Metrics display
`
	if err := os.WriteFile(filepath.Join(dir, "PRD.md"), []byte(prdContent), 0o644); err != nil {
		t.Fatalf("failed to write PRD.md: %v", err)
	}

	targets, err := ParseArchiveTargets(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(targets))
	}

	// First target: complete, P0 criticality.
	if targets[0].Status != "complete" {
		t.Errorf("target[0].Status = %q, want %q", targets[0].Status, "complete")
	}
	if targets[0].Criticality != "P0" {
		t.Errorf("target[0].Criticality = %q, want %q", targets[0].Criticality, "P0")
	}
	if targets[0].Title != "Login system" {
		t.Errorf("target[0].Title = %q, want %q", targets[0].Title, "Login system")
	}
	if targets[0].Notes != "basic auth" {
		t.Errorf("target[0].Notes = %q, want %q", targets[0].Notes, "basic auth")
	}

	// Second target: pending, P0 criticality.
	if targets[1].Status != "pending" {
		t.Errorf("target[1].Status = %q, want %q", targets[1].Status, "pending")
	}
	if targets[1].Criticality != "P0" {
		t.Errorf("target[1].Criticality = %q, want %q", targets[1].Criticality, "P0")
	}

	// Third target: P1 criticality.
	if targets[2].Criticality != "P1" {
		t.Errorf("target[2].Criticality = %q, want %q", targets[2].Criticality, "P1")
	}
}

func TestParseArchiveTargets_NoPRD(t *testing.T) {
	dir := t.TempDir()

	targets, err := ParseArchiveTargets(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 0 {
		t.Errorf("expected empty slice, got %d targets", len(targets))
	}
}

func TestParseArchiveRequirements(t *testing.T) {
	dir := t.TempDir()
	reqDir := filepath.Join(dir, "requirements")
	coreDir := filepath.Join(reqDir, "01-core")

	if err := os.MkdirAll(coreDir, 0o755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}

	indexJSON := map[string]any{
		"imports": []string{"01-core/module.json"},
		"requirements": []map[string]any{
			{
				"id":          "REQ-ROOT-001",
				"title":       "Root requirement",
				"status":      "draft",
				"description": "A root req",
			},
		},
	}
	writeJSON(t, filepath.Join(reqDir, "index.json"), indexJSON)

	moduleJSON := map[string]any{
		"module_id":   "core",
		"title":       "Core Module",
		"description": "Core requirements",
		"requirements": []map[string]any{
			{
				"id":          "REQ-CORE-001",
				"title":       "Core feature",
				"status":      "complete",
				"description": "A core feature",
				"category":    "backend",
				"prd_ref":     "OT-P0-001",
			},
		},
	}
	writeJSON(t, filepath.Join(coreDir, "module.json"), moduleJSON)

	groups, err := ParseArchiveRequirements(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) != 1 {
		t.Fatalf("expected 1 top-level group, got %d", len(groups))
	}

	root := groups[0]
	if root.Name != "Index" {
		t.Errorf("root group Name = %q, want %q", root.Name, "Index")
	}
	if len(root.Requirements) != 1 {
		t.Fatalf("expected 1 root requirement, got %d", len(root.Requirements))
	}
	if root.Requirements[0].ID != "REQ-ROOT-001" {
		t.Errorf("root req ID = %q, want %q", root.Requirements[0].ID, "REQ-ROOT-001")
	}

	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child group, got %d", len(root.Children))
	}

	child := root.Children[0]
	if child.Name != "Core" {
		t.Errorf("child group Name = %q, want %q", child.Name, "Core")
	}
	if len(child.Requirements) != 1 {
		t.Fatalf("expected 1 child requirement, got %d", len(child.Requirements))
	}
	req := child.Requirements[0]
	if req.ID != "REQ-CORE-001" {
		t.Errorf("child req ID = %q, want %q", req.ID, "REQ-CORE-001")
	}
	if req.Category != "backend" {
		t.Errorf("child req Category = %q, want %q", req.Category, "backend")
	}
	if req.PRDRef != "OT-P0-001" {
		t.Errorf("child req PRDRef = %q, want %q", req.PRDRef, "OT-P0-001")
	}
}

func TestParseArchiveRequirements_NoDir(t *testing.T) {
	dir := t.TempDir()

	groups, err := ParseArchiveRequirements(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(groups) != 0 {
		t.Errorf("expected empty slice, got %d groups", len(groups))
	}
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal JSON: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}
