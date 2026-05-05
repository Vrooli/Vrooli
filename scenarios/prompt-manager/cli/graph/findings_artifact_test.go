package graph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fixedNow is the canonical pinned time used across every artifact test.
// CI diff scripts compare two artifact files; pinning generated_at in
// tests is the contract that proves the field is wired through and not
// silently re-derived from the system clock.
var fixedNow = time.Date(2026, 5, 4, 12, 34, 56, 0, time.UTC)

func sampleResponse() topicsGraphResponse {
	return topicsGraphResponse{
		Validation: topicValidation{
			Findings: []topicFinding{
				{
					Rule:     "orphan_input",
					Severity: "error",
					Member:   topicMemberRef{Team: "marketing-crew", Member: "researcher"},
					Prefix:   "research-inbox/*",
					Detail:   "no producer",
				},
				{
					Rule:     "unread_required",
					Severity: "warning",
					Member:   topicMemberRef{Team: "marketing-crew", Member: "researcher"},
					Prefix:   "vision-walk-record/*",
					Detail:   "no producer",
				},
			},
			Errors:   1,
			Warnings: 1,
		},
	}
}

func TestBuildFindingsArtifact_PopulatesAllFields(t *testing.T) {
	resp := sampleResponse()
	got := buildFindingsArtifact(resp, "marketing-crew", fixedNow)

	if got.SchemaVersion != findingsArtifactSchemaVersion {
		t.Errorf("SchemaVersion = %d, want %d", got.SchemaVersion, findingsArtifactSchemaVersion)
	}
	if got.GeneratedAt != "2026-05-04T12:34:56Z" {
		t.Errorf("GeneratedAt = %q, want pinned RFC3339 UTC", got.GeneratedAt)
	}
	if got.TeamFilter != "marketing-crew" {
		t.Errorf("TeamFilter = %q, want %q", got.TeamFilter, "marketing-crew")
	}
	if got.Errors != 1 || got.Warnings != 1 {
		t.Errorf("counts = %d errors, %d warnings; want 1, 1", got.Errors, got.Warnings)
	}
	if len(got.Findings) != 2 {
		t.Fatalf("Findings len = %d, want 2", len(got.Findings))
	}
	if got.Findings[0].Rule != "orphan_input" {
		t.Errorf("Findings[0].Rule = %q", got.Findings[0].Rule)
	}
}

func TestBuildFindingsArtifact_NilFindingsBecomesEmptySlice(t *testing.T) {
	// CONTRACT: clean runs must produce findings: [] not findings: null.
	// CI diff scripts depend on this — null collapses to "missing field"
	// in some JSON consumers and would produce false-positive diffs.
	resp := topicsGraphResponse{
		Validation: topicValidation{
			Findings: nil,
			Errors:   0,
			Warnings: 0,
		},
	}
	got := buildFindingsArtifact(resp, "", fixedNow)
	if got.Findings == nil {
		t.Fatal("Findings is nil; expected empty slice")
	}
	if len(got.Findings) != 0 {
		t.Errorf("Findings len = %d, want 0", len(got.Findings))
	}

	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"findings":[]`) {
		t.Errorf("expected findings: [] in JSON, got: %s", raw)
	}
}

func TestBuildFindingsArtifact_GeneratedAtIsUTC(t *testing.T) {
	// Pinned time in a non-UTC zone must be re-rendered as UTC so two
	// CI runs in different timezones produce diffable artifacts.
	denver, err := time.LoadLocation("America/Denver")
	if err != nil {
		t.Skipf("America/Denver not available: %v", err)
	}
	local := time.Date(2026, 5, 4, 6, 34, 56, 0, denver) // 12:34:56 UTC
	got := buildFindingsArtifact(sampleResponse(), "", local)
	if got.GeneratedAt != "2026-05-04T12:34:56Z" {
		t.Errorf("GeneratedAt = %q, want UTC %q", got.GeneratedAt, "2026-05-04T12:34:56Z")
	}
}

func TestWriteFindingsArtifact_EmptyPathIsNoOp(t *testing.T) {
	dir := t.TempDir()
	if err := writeFindingsArtifact("", sampleResponse(), "", fixedNow); err != nil {
		t.Fatalf("empty path returned error: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read tempdir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("tempdir not empty after empty-path call: %v", entries)
	}
}

func TestWriteFindingsArtifact_WritesStableShape(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "findings.json")

	if err := writeFindingsArtifact(path, sampleResponse(), "marketing-crew", fixedNow); err != nil {
		t.Fatalf("write: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// File ends with newline (matches store convention).
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		t.Errorf("artifact does not end with newline: %q", string(raw))
	}

	// Parses back to the same struct shape.
	var got findingsArtifact
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("re-parse: %v", err)
	}
	if got.SchemaVersion != findingsArtifactSchemaVersion {
		t.Errorf("SchemaVersion mismatch after round-trip: %d", got.SchemaVersion)
	}
	if got.GeneratedAt != "2026-05-04T12:34:56Z" {
		t.Errorf("GeneratedAt mismatch: %q", got.GeneratedAt)
	}
	if got.TeamFilter != "marketing-crew" {
		t.Errorf("TeamFilter mismatch: %q", got.TeamFilter)
	}
	if got.Errors != 1 || got.Warnings != 1 {
		t.Errorf("counts mismatch: %d, %d", got.Errors, got.Warnings)
	}
	if len(got.Findings) != 2 {
		t.Errorf("Findings len mismatch: %d", len(got.Findings))
	}
}

func TestWriteFindingsArtifact_GoldenJSONShape(t *testing.T) {
	// CONTRACT pin: this golden string IS the on-disk shape.
	// Any field reorder, key rename, or indent change is a contract
	// break — see findings_artifact.go CONTRACT comment for the
	// additive-fields rule.
	dir := t.TempDir()
	path := filepath.Join(dir, "findings.json")

	resp := topicsGraphResponse{
		Validation: topicValidation{
			Findings: []topicFinding{
				{
					Rule:     "orphan_input",
					Severity: "error",
					Member:   topicMemberRef{Team: "t", Member: "m"},
					Prefix:   "x/*",
					Detail:   "no producer",
				},
			},
			Errors:   1,
			Warnings: 0,
		},
	}
	if err := writeFindingsArtifact(path, resp, "t", fixedNow); err != nil {
		t.Fatalf("write: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := `{
  "schema_version": 1,
  "generated_at": "2026-05-04T12:34:56Z",
  "team_filter": "t",
  "errors": 1,
  "warnings": 0,
  "findings": [
    {
      "rule": "orphan_input",
      "severity": "error",
      "member": {
        "team": "t",
        "member": "m"
      },
      "prefix": "x/*",
      "detail": "no producer"
    }
  ]
}
`
	if string(raw) != want {
		t.Errorf("artifact shape drift\n--- got ---\n%s--- want ---\n%s", string(raw), want)
	}
}

func TestWriteFindingsArtifact_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "findings.json")

	// First write: clean run.
	if err := writeFindingsArtifact(path, topicsGraphResponse{}, "", fixedNow); err != nil {
		t.Fatalf("first write: %v", err)
	}
	// Second write: with findings, different timestamp.
	later := fixedNow.Add(1 * time.Hour)
	if err := writeFindingsArtifact(path, sampleResponse(), "marketing-crew", later); err != nil {
		t.Fatalf("second write: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got findingsArtifact
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.GeneratedAt != "2026-05-04T13:34:56Z" {
		t.Errorf("expected updated timestamp; got %q", got.GeneratedAt)
	}
	if got.Errors != 1 {
		t.Errorf("expected updated counts; got Errors=%d", got.Errors)
	}
}

func TestWriteFindingsArtifact_LeavesNoTempOnSuccess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "findings.json")
	if err := writeFindingsArtifact(path, sampleResponse(), "", fixedNow); err != nil {
		t.Fatalf("write: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one file (findings.json); got %d entries: %v", len(entries), entries)
	}
	if entries[0].Name() != "findings.json" {
		t.Errorf("unexpected file in dir: %s", entries[0].Name())
	}
}

func TestWriteFindingsArtifact_MissingDirectoryIsHardError(t *testing.T) {
	// Silent mkdir on a missing path would mask a misspelt CI flag value
	// (e.g., --findings-out=/tmp/nonexistnt/findings.json). The contract
	// is to surface this as an error so the misconfiguration is visible.
	missing := filepath.Join(t.TempDir(), "does-not-exist", "findings.json")
	err := writeFindingsArtifact(missing, sampleResponse(), "", fixedNow)
	if err == nil {
		t.Fatal("expected error for missing parent dir; got nil")
	}
}

func TestWriteFindingsArtifact_AbsoluteAndRelativePaths(t *testing.T) {
	dir := t.TempDir()

	// Absolute path.
	abs := filepath.Join(dir, "abs-findings.json")
	if err := writeFindingsArtifact(abs, sampleResponse(), "", fixedNow); err != nil {
		t.Fatalf("absolute path write: %v", err)
	}
	if _, err := os.Stat(abs); err != nil {
		t.Errorf("absolute path file missing: %v", err)
	}

	// Relative path (resolved against CWD via t.Chdir).
	t.Chdir(dir)
	if err := writeFindingsArtifact("rel-findings.json", sampleResponse(), "", fixedNow); err != nil {
		t.Fatalf("relative path write: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "rel-findings.json")); err != nil {
		t.Errorf("relative path file missing: %v", err)
	}
}
