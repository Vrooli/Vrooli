package orchestrator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"test-genie/internal/executionevidence"

	sharedartifacts "test-genie/internal/shared/artifacts"
)

// TestWriteFindingsArtifact verifies the combined findings document is written
// once to the immutable per-run path, that zero-finding phases are INCLUDED
// (with their findingSource so reaudit can derive coverage), and that the shape
// matches the `--from-audit` ingest contract.
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

	for _, path := range []string{sharedartifacts.RunFindingsArtifactPath(dir, runID)} {
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
	if _, err := os.Stat(filepath.Join(sharedartifacts.LatestDirPath(dir), sharedartifacts.FindingsArtifactFile)); !os.IsNotExist(err) {
		t.Fatalf("latest findings duplicate exists: %v", err)
	}
	if err := writeEvidenceManifest(dir, runID, "web-search", SuiteVerdictFail, completed, buildPhaseResultViews("", results)); err != nil {
		t.Fatalf("writeEvidenceManifest: %v", err)
	}
	manifestData, err := os.ReadFile(filepath.Join(sharedartifacts.RunDir(dir, runID), executionevidence.ManifestFile))
	if err != nil {
		t.Fatalf("read evidence manifest: %v", err)
	}
	var manifest executionevidence.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("decode evidence manifest: %v", err)
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("validate evidence manifest: %v", err)
	}
	if manifest.Phases[0].Findings == nil || manifest.Phases[1].Findings != nil {
		t.Fatalf("manifest findings references = %+v", manifest.Phases)
	}
}

// TestFindingsArtifactCarriesStandingAndStaysIngestible verifies (a) the per-phase
// maturity standing round-trips through findings.json, and (b) the additive
// standing does not break architecture-cartographer's --from-audit ingest, which
// reads only phases[].findings and ignores unknown fields.
func TestFindingsArtifactCarriesStandingAndStaysIngestible(t *testing.T) {
	dir := t.TempDir()
	runID := "run-standing"
	completed := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)

	results := []PhaseExecutionResult{
		{
			Name:          "contracts",
			Status:        "passed",
			FindingSource: "cli",
			Findings: []*architecturev1.ArchitectureFinding{
				{Scenario: "cli-health", Source: architecturev1.FindingSource_FINDING_SOURCE_CLI, Code: "arch.primitive_unverified", Locations: []string{"cli/manifest.json"}},
			},
			Assessment: &commonv1.MaturityAssessment{RecommendedSkillIds: []string{"scientific-debugging", "unit-testing-architecture-steer"}},
			PhasePresentation: &commonv1.PhasePresentation{
				Provider:             "cli-health",
				Phase:                "contracts",
				CurrentLevel:         "L3",
				NextLevel:            "L4",
				CeilingLevel:         "L4",
				NorthStar:            "Verified renderer-separated primitives.",
				NextAction:           "Prove each declared primitive with cli-core evidence.",
				BlockingFindingCodes: []string{"arch.primitive_unverified"},
			},
			FindingsSummary: &runspb.PhaseFindingsSummary{Warnings: 1, Total: 1},
		},
	}

	o := &SuiteOrchestrator{}
	if err := o.writeFindingsArtifact(dir, "cli-health", runID, SuiteVerdictPass, completed, buildPhaseResultViews("", results)); err != nil {
		t.Fatalf("writeFindingsArtifact: %v", err)
	}
	raw, err := os.ReadFile(sharedartifacts.RunFindingsArtifactPath(dir, runID))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// (a) Standing round-trips.
	var art findingsArtifact
	if err := json.Unmarshal(raw, &art); err != nil {
		t.Fatalf("unmarshal full: %v", err)
	}
	st := art.Phases[0].PhasePresentation
	if st == nil {
		t.Fatal("maturity standing dropped from findings.json")
	}
	if st.GetCurrentLevel() != "L3" || st.GetNextLevel() != "L4" || st.GetNorthStar() == "" {
		t.Fatalf("standing fields not round-tripped: %+v", st)
	}
	if art.Phases[0].FindingsSummary.GetTotal() != 1 {
		t.Fatalf("findings summary not round-tripped: %+v", art.Phases[0].FindingsSummary)
	}
	if got, want := art.Phases[0].Assessment.GetRecommendedSkillIds(), []string{"scientific-debugging", "unit-testing-architecture-steer"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("recommended skills not round-tripped: got %#v want %#v", got, want)
	}

	// (b) Backward-compat: the cartographer ingest contract (phases[].findings)
	// still parses when the record only knows the pre-standing fields.
	var legacy struct {
		Phases []struct {
			Name     string `json:"name"`
			Status   string `json:"status"`
			Findings []struct {
				Code string `json:"code"`
			} `json:"findings"`
		} `json:"phases"`
	}
	if err := json.Unmarshal(raw, &legacy); err != nil {
		t.Fatalf("legacy ingest unmarshal broke: %v", err)
	}
	if len(legacy.Phases) != 1 || len(legacy.Phases[0].Findings) != 1 || legacy.Phases[0].Findings[0].Code != "arch.primitive_unverified" {
		t.Fatalf("cartographer ingest shape regressed: %+v", legacy)
	}
}
