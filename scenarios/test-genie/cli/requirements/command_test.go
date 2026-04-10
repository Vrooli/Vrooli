package requirements

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPlaceholdersReplacesTemplateTokensAndAppendsNewline(t *testing.T) {
	got := string(applyPlaceholders(
		[]byte("scenario=__SCENARIO_NAME__ date=__GENERATED_DATE__ owner=__CONTACT__"),
		"demo",
		"2026-03-28",
		"engineering@demo.local",
	))

	if !strings.Contains(got, "scenario=demo") || !strings.Contains(got, "date=2026-03-28") || !strings.Contains(got, "owner=engineering@demo.local") {
		t.Fatalf("expected placeholders to be replaced, got %q", got)
	}
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("expected generated content to end with newline, got %q", got)
	}
}

func TestScenarioNameFromDirPrefersExplicitName(t *testing.T) {
	if got := scenarioNameFromDir("/tmp/example", "named"); got != "named" {
		t.Fatalf("expected explicit name override, got %q", got)
	}
	if got := scenarioNameFromDir("/tmp/example", ""); got != "example" {
		t.Fatalf("expected directory basename fallback, got %q", got)
	}
}

func TestEnsureDirValidatesScenarioDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := ensureDir(dir); err != nil {
		t.Fatalf("expected temp dir to be accepted, got %v", err)
	}

	filePath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	err := ensureDir(filePath)
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected non-directory validation error, got %v", err)
	}
}

func TestDetectDriftReportsMissingSnapshot(t *testing.T) {
	dir := t.TempDir()
	result, err := detectDrift(dir)
	if err != nil {
		t.Fatalf("detectDrift returned error: %v", err)
	}
	if result.Status != "missing_snapshot" || !result.MissingSnapshot {
		t.Fatalf("expected missing snapshot result, got %+v", result)
	}
	wantPath := filepath.Join(dir, "coverage", "requirements-sync", "latest.json")
	if result.SnapshotPath != wantPath {
		t.Fatalf("expected snapshot path %q, got %q", wantPath, result.SnapshotPath)
	}
}
