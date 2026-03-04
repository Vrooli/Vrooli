package backlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// samplePRDWithTargets returns PRD content with an operational targets section.
func samplePRDWithTargets() string {
	return `# PRD: Test Scenario

## Overview
This is a test PRD.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability
- [x] OT-P0-001 | Core feature | Essential functionality ` + "`[req:REQ-001,REQ-002]`" + `
- [ ] OT-P0-002 | Second feature | Also important

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Nice to have | Improves UX

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Future work | Long-term goal

## Next Steps
Some content after targets.
`
}

func TestSerializeTargetsSection_RoundTrip(t *testing.T) {
	content := samplePRDWithTargets()
	original := parseOperationalTargets(content)
	if len(original) == 0 {
		t.Fatal("expected parsed targets from sample PRD")
	}

	serialized := serializeTargetsSection(original)
	reparsed := parseOperationalTargets(serialized)

	if len(reparsed) != len(original) {
		t.Fatalf("round-trip count mismatch: got %d, want %d", len(reparsed), len(original))
	}

	for i, orig := range original {
		got := reparsed[i]
		if got.ID != orig.ID {
			t.Errorf("target[%d] ID: got %q, want %q", i, got.ID, orig.ID)
		}
		if got.Title != orig.Title {
			t.Errorf("target[%d] Title: got %q, want %q", i, got.Title, orig.Title)
		}
		if got.Criticality != orig.Criticality {
			t.Errorf("target[%d] Criticality: got %q, want %q", i, got.Criticality, orig.Criticality)
		}
		if got.Status != orig.Status {
			t.Errorf("target[%d] Status: got %q, want %q", i, got.Status, orig.Status)
		}
	}
}

func TestReplaceTargetsSection_PreservesOtherContent(t *testing.T) {
	content := samplePRDWithTargets()
	replacement := serializeTargetsSection([]ArchiveTarget{
		{ID: "OT-NEW-001", Criticality: "P0", Title: "New target", Status: "pending"},
	})

	result := replaceTargetsSection(content, replacement)

	// Content before section should be preserved.
	if !strings.Contains(result, "## Overview") {
		t.Error("content before targets section was lost")
	}
	if !strings.Contains(result, "This is a test PRD.") {
		t.Error("overview text was lost")
	}

	// Content after section should be preserved.
	if !strings.Contains(result, "## Next Steps") {
		t.Error("content after targets section was lost")
	}
	if !strings.Contains(result, "Some content after targets.") {
		t.Error("next steps text was lost")
	}

	// New content should be present.
	if !strings.Contains(result, "OT-NEW-001") {
		t.Error("new target not found in result")
	}

	// Old content should be removed.
	if strings.Contains(result, "OT-P0-001") {
		t.Error("old target still present in result")
	}
}

func TestWriteTargets_CreatesSection(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write PRD without targets section.
	prdContent := "# PRD\n\n## Overview\nSome content.\n"
	if err := os.WriteFile(filepath.Join(archiveDir, "PRD.md"), []byte(prdContent), 0o644); err != nil {
		t.Fatal(err)
	}

	targets := []ArchiveTarget{
		{ID: "OT-P0-001", Criticality: "P0", Title: "New target", Status: "pending"},
	}
	if err := WriteTargets(dir, targets); err != nil {
		t.Fatalf("WriteTargets: %v", err)
	}

	// Read back and verify.
	data, err := os.ReadFile(filepath.Join(archiveDir, "PRD.md"))
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)

	if !strings.Contains(result, "## Overview") {
		t.Error("original content lost")
	}
	if !strings.Contains(result, modernOperationalHeader) {
		t.Error("targets section not added")
	}
	if !strings.Contains(result, "OT-P0-001") {
		t.Error("target not found in written PRD")
	}
}

func TestWriteTargets_NoPRDCreatesFile(t *testing.T) {
	dir := t.TempDir()

	targets := []ArchiveTarget{
		{ID: "OT-P0-001", Criticality: "P0", Title: "First target", Status: "pending"},
	}
	if err := WriteTargets(dir, targets); err != nil {
		t.Fatalf("WriteTargets: %v", err)
	}

	// Should create PRD.md at item root since no archive/ dir exists.
	data, err := os.ReadFile(filepath.Join(dir, "PRD.md"))
	if err != nil {
		t.Fatalf("read PRD.md: %v", err)
	}
	if !strings.Contains(string(data), "OT-P0-001") {
		t.Error("target not found in created PRD.md")
	}
}

func TestCreateTarget_DuplicateID(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "PRD.md"), []byte(samplePRDWithTargets()), 0o644); err != nil {
		t.Fatal(err)
	}

	err := CreateTarget(dir, ArchiveTarget{
		ID:          "OT-P0-001",
		Criticality: "P0",
		Title:       "Duplicate",
		Status:      "pending",
	})
	if err == nil {
		t.Fatal("expected error for duplicate ID")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestUpdateTarget_NotFound(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "PRD.md"), []byte(samplePRDWithTargets()), 0o644); err != nil {
		t.Fatal(err)
	}

	err := UpdateTarget(dir, "nonexistent-id", ArchiveTarget{
		Criticality: "P0",
		Title:       "Updated",
		Status:      "complete",
	})
	if err == nil {
		t.Fatal("expected error for missing target")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestDeleteTarget_Success(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "PRD.md"), []byte(samplePRDWithTargets()), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := DeleteTarget(dir, "OT-P0-001"); err != nil {
		t.Fatalf("DeleteTarget: %v", err)
	}

	// Read back and verify target is gone.
	targets, _, err := ReadTargetsFromPRD(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, tgt := range targets {
		if tgt.ID == "OT-P0-001" {
			t.Error("deleted target still present")
		}
	}

	// Other targets should remain.
	if len(targets) != 3 {
		t.Errorf("expected 3 remaining targets, got %d", len(targets))
	}

	// PRD.md should still have other content.
	data, err := os.ReadFile(filepath.Join(archiveDir, "PRD.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "## Overview") {
		t.Error("other content lost after delete")
	}
}

func TestWriteTargets_PreservesReqLinks(t *testing.T) {
	dir := t.TempDir()
	archiveDir := filepath.Join(dir, "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		t.Fatal(err)
	}

	targets := []ArchiveTarget{
		{
			ID:                 "OT-P0-001",
			Criticality:        "P0",
			Title:              "Linked target",
			Notes:              "Has links",
			Status:             "pending",
			LinkedRequirements: []string{"REQ-001", "REQ-002"},
		},
	}
	if err := WriteTargets(dir, targets); err != nil {
		t.Fatalf("WriteTargets: %v", err)
	}

	// Read back and verify links survived.
	readBack, _, err := ReadTargetsFromPRD(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(readBack) != 1 {
		t.Fatalf("expected 1 target, got %d", len(readBack))
	}
	if len(readBack[0].LinkedRequirements) != 2 {
		t.Fatalf("expected 2 linked requirements, got %d", len(readBack[0].LinkedRequirements))
	}
	if readBack[0].LinkedRequirements[0] != "REQ-001" || readBack[0].LinkedRequirements[1] != "REQ-002" {
		t.Errorf("linked requirements mismatch: %v", readBack[0].LinkedRequirements)
	}
}

func TestCreateTarget_AutoGeneratesID(t *testing.T) {
	dir := t.TempDir()

	err := CreateTarget(dir, ArchiveTarget{
		Criticality: "P1",
		Title:       "Auto ID Target",
		Status:      "pending",
	})
	if err != nil {
		t.Fatalf("CreateTarget: %v", err)
	}

	targets, _, err := ReadTargetsFromPRD(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].ID != "auto-id-target" {
		t.Errorf("auto-generated ID: got %q, want %q", targets[0].ID, "auto-id-target")
	}
}
