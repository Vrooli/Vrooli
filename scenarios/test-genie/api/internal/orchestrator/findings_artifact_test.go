package orchestrator

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// TestWriteFindingsArtifact verifies the combined findings document is written
// to both the per-run path and the latest mirror, that zero-finding phases are
// INCLUDED (with their findingSource so reaudit can derive coverage), and that
// the shape matches the `--from-audit` ingest contract.
func TestWriteFindingsArtifact(t *testing.T) {
	dir := t.TempDir()
	runID := "run-abc"
	completed := time.Date(2026, 6, 13, 12, 0, 0, 0, time.UTC)

	results := []PhaseExecutionResult{
		{
			Name:          "structure",
			Status:        "failed",
			FindingSource: "structure",
			Findings: []*architecturev1.ArchitectureFinding{
				{Scenario: "web-search", Source: architecturev1.FindingSource_FINDING_SOURCE_STRUCTURE, Code: "missing_field", Locations: []string{".vrooli/endpoints.json"}},
			},
		},
		{
			// A zero-finding phase that ran (passed) — must still appear, with
			// its source, so reaudit knows the source was covered.
			Name:          "quality",
			Status:        "passed",
			FindingSource: "standards",
			Findings:      nil,
		},
		{
			// A non-producing phase carries no findingSource.
			Name:   "unit",
			Status: "passed",
		},
	}

	o := &SuiteOrchestrator{}
	if err := o.writeFindingsArtifact(dir, "web-search", runID, SuiteVerdictFail, completed, buildPhaseResultViews("", results)); err != nil {
		t.Fatalf("writeFindingsArtifact: %v", err)
	}

	for _, path := range []string{
		sharedartifacts.RunFindingsArtifactPath(dir, runID),
		sharedartifacts.LatestFindingsArtifactPath(dir),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		var art findingsArtifact
		if err := json.Unmarshal(raw, &art); err != nil {
			t.Fatalf("unmarshal %s: %v", path, err)
		}
		if art.Scenario != "web-search" || art.RunID != runID || art.Verdict != SuiteVerdictFail {
			t.Errorf("%s: header wrong: %+v", path, art)
		}
		if len(art.Phases) != 3 {
			t.Fatalf("%s: want 3 phases (zero-finding included), got %d", path, len(art.Phases))
		}
		if art.Phases[1].Name != "quality" || art.Phases[1].FindingSource != "standards" {
			t.Errorf("%s: zero-finding quality phase missing or unsourced: %+v", path, art.Phases[1])
		}
		if len(art.Phases[1].Findings) != 0 {
			t.Errorf("%s: quality phase should have empty findings, got %d", path, len(art.Phases[1].Findings))
		}
		if art.Phases[2].FindingSource != "" {
			t.Errorf("%s: non-producing unit phase should have no findingSource, got %q", path, art.Phases[2].FindingSource)
		}
		if len(art.Phases[0].Findings) != 1 || art.Phases[0].Findings[0].GetCode() != "missing_field" {
			t.Errorf("%s: structure findings not round-tripped: %+v", path, art.Phases[0].Findings)
		}
	}
}
