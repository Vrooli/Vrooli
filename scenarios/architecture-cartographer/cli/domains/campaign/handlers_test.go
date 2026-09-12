package campaign

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// writeTemp writes content to a temp file and returns its path.
func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "findings.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return p
}

// realArtifact mirrors the exact shape test-genie's writeFindingsArtifact
// persists to coverage/runs/<runID>/findings.json. This fixture is the
// producer/consumer contract: if either side drifts, this test fails.
const realArtifact = `{
  "scenario": "web-search",
  "runId": "run-123",
  "verdict": "FAIL",
  "completedAt": "2026-06-13T12:00:00Z",
  "phases": [
    {"name": "structure", "status": "failed", "findingSource": "structure",
     "findings": [{"scenario": "web-search", "source": 1, "code": "missing_field", "severity": 2, "locations": ["./.vrooli/endpoints.json"]}]},
    {"name": "standards", "status": "passed", "findingSource": "standards", "findings": []},
    {"name": "proto", "status": "failed", "findingSource": "proto",
     "findings": [{"scenario": "web-search", "source": 12, "code": "shared_type", "severity": 2, "locations": ["api/proto/x.proto:10"]}]},
    {"name": "tidiness", "status": "skipped", "findingSource": "tidiness", "findings": []},
    {"name": "unit", "status": "passed", "findings": []}
  ]
}`

func TestLoadAuditFindings_RealArtifactContract(t *testing.T) {
	got, err := loadAuditFindings(writeTemp(t, realArtifact))
	if err != nil {
		t.Fatalf("loadAuditFindings: %v", err)
	}
	if got.scenario != "web-search" {
		t.Errorf("scenario = %q, want web-search", got.scenario)
	}
	if len(got.findings) != 2 {
		t.Fatalf("want 2 findings flattened, got %d", len(got.findings))
	}
	if !got.hasSourceTokens {
		t.Error("hasSourceTokens should be true")
	}
	// structure+proto ran (failed); standards passed (covered, zero findings);
	// tidiness skipped (NOT covered); unit has no findingSource (excluded).
	want := []string{"proto", "standards", "structure"}
	if !reflect.DeepEqual(got.coveredSources, want) {
		t.Errorf("coveredSources = %v, want %v", got.coveredSources, want)
	}
}

func TestLoadAuditFindings_MissingPhasesKeyErrors(t *testing.T) {
	// A single per-phase file (no top-level `phases` key) must hard-error.
	_, err := loadAuditFindings(writeTemp(t, `{"phase":"structure","findings":[]}`))
	if err == nil {
		t.Fatal("expected error for document without a phases key")
	}
}

func TestLoadAuditFindings_ZeroFindingsErrors(t *testing.T) {
	doc := `{"scenario":"s","phases":[{"name":"standards","status":"passed","findingSource":"standards","findings":[]}]}`
	_, err := loadAuditFindings(writeTemp(t, doc))
	if err == nil {
		t.Fatal("expected error for a well-formed document with zero findings")
	}
}

func TestLoadAuditFindings_EmptyPathErrors(t *testing.T) {
	if _, err := loadAuditFindings(""); err == nil {
		t.Fatal("expected error for empty --from-audit")
	}
}

func TestLoadAuditFindings_ForeignDocHasNoSourceTokens(t *testing.T) {
	// A hand-built doc with findings but no findingSource tokens: ingestable,
	// but coverage cannot be scoped → reaudit falls back to all-covered.
	doc := `{"phases":[{"name":"x","status":"passed","findings":[{"scenario":"s","source":5,"code":"c","severity":2,"locations":["a.go"]}]}]}`
	got, err := loadAuditFindings(writeTemp(t, doc))
	if err != nil {
		t.Fatalf("loadAuditFindings: %v", err)
	}
	if got.hasSourceTokens {
		t.Error("hasSourceTokens should be false for a doc with no findingSource fields")
	}
	if len(got.coveredSources) != 0 {
		t.Errorf("coveredSources should be empty for a token-less doc, got %v", got.coveredSources)
	}
}
