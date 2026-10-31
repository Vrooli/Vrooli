package dochealth

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewSelection(t *testing.T) {
	if sel, err := newSelection(nil); err != nil || len(sel.enabled) != 0 {
		t.Fatalf("empty selection: sel=%#v err=%v", sel, err)
	}
	sel, err := newSelection([]string{"numbers", "content"})
	if err != nil {
		t.Fatalf("valid selection: %v", err)
	}
	if !sel.enabled[checkNumbers] || !sel.enabled[checkContent] || sel.enabled[checkLinks] {
		t.Fatalf("unexpected enabled set: %#v", sel.enabled)
	}
	if _, err := newSelection([]string{"bogus"}); err == nil {
		t.Fatal("expected error for unknown check")
	}
}

func TestSelectionRuns_ScopeGating(t *testing.T) {
	scenarioTgt := docTarget{isScenario: true}
	genericTgt := docTarget{isScenario: false}

	all := selection{}
	// Default selection: scenario checks gated by target scope.
	if !all.runs(checkStructure, scenarioTgt) {
		t.Fatal("structure should run on scenario target")
	}
	if all.runs(checkStructure, genericTgt) {
		t.Fatal("structure (scenario-scoped) must NOT run on generic target")
	}
	if all.runs(checkManifest, genericTgt) {
		t.Fatal("manifest (scenario-scoped) must NOT run on generic target")
	}
	if !all.runs(checkNumbers, genericTgt) {
		t.Fatal("numbers (generic) should run on generic target")
	}
	if !all.runs(checkNumbers, scenarioTgt) {
		t.Fatal("numbers (generic) should run on scenario target")
	}

	// Explicit filter narrows to named checks (still scope-gated).
	only, _ := newSelection([]string{"numbers"})
	if only.runs(checkContent, scenarioTgt) {
		t.Fatal("content not selected; should not run")
	}
	if !only.runs(checkNumbers, scenarioTgt) {
		t.Fatal("numbers selected; should run")
	}
}

// TestDocHealth_GenericPath_NumbersOnly verifies a project-level path runs the
// generic number lint while skipping scenario-scoped structure/manifest checks.
func TestDocHealth_GenericPath_NumbersOnly(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	if err := os.MkdirAll(scenarios, 0o755); err != nil {
		t.Fatal(err)
	}
	// Project-level docs path, OUTSIDE the scenarios root.
	writeTestFile(t, filepath.Join(root, "docs", "README.md"), "# Docs\n\nWe run four teams here.\n")

	svc, err := NewService(scenarios)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	res, err := svc.DocHealth(context.Background(), "", DocHealthOptions{Scope: "path", Path: filepath.Join(root, "docs")})
	if err != nil {
		t.Fatalf("DocHealth(generic path): %v", err)
	}
	if res.Counts.NumbersFlagged != 1 {
		t.Fatalf("expected 1 number flagged, got %d", res.Counts.NumbersFlagged)
	}
	if !hasFinding(res.ContentFindings, findingUnmarkedNumber) {
		t.Fatalf("expected unmarked_number finding, got %#v", res.ContentFindings)
	}
	// Scenario-scoped structural fields are not evaluated for a generic path.
	if res.ManifestStatus != "not-evaluated" {
		t.Fatalf("expected manifest not-evaluated for generic path, got %q", res.ManifestStatus)
	}
	if len(res.ManifestFindings) != 0 {
		t.Fatalf("manifest check should be skipped for generic path, got %#v", res.ManifestFindings)
	}
}

// TestDocHealth_ChecksFilter confirms --checks narrows the run.
func TestDocHealth_ChecksFilter(t *testing.T) {
	root := t.TempDir()
	scenarios := filepath.Join(root, "scenarios")
	scenarioDir := filepath.Join(scenarios, "demo")
	writeTestFile(t, filepath.Join(scenarioDir, "docs/README.md"), "# Demo\n\nWe run four teams.\n")

	svc, err := NewService(scenarios)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	res, err := svc.DocHealth(context.Background(), "demo", DocHealthOptions{Checks: []string{"numbers"}})
	if err != nil {
		t.Fatalf("DocHealth(checks=numbers): %v", err)
	}
	if res.Counts.NumbersFlagged != 1 {
		t.Fatalf("expected 1 number flagged, got %d", res.Counts.NumbersFlagged)
	}
	// Other checks did not run, so their counters stay zero.
	if res.Counts.FilesChecked != 0 {
		t.Fatalf("content check should be off (FilesChecked=%d)", res.Counts.FilesChecked)
	}
	if res.Counts.MarkedRefsFound != 0 || res.Counts.LocalLinks != 0 {
		t.Fatalf("refs/links checks should be off: %#v", res.Counts)
	}
}
