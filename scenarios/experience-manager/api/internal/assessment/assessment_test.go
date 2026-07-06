package assessment

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"experience-manager/internal/spec"

	maturity "github.com/vrooli/maturity-go/assessment"
)

func testSpec() *maturity.Spec {
	return &maturity.Spec{
		Provider: "experience-manager",
		Phase:    "experience",
		Version:  "1",
		Capabilities: []maturity.CapabilitySpec{{
			ID:    "spec_contract",
			Label: "Spec Contract",
			Levels: []maturity.Level{
				{ID: "L0", Name: "Missing"},
				{ID: "L1", Name: "Present"},
			},
		}},
		Findings: map[string]maturity.FindingMapping{
			"experience.schema_invalid": {
				CapabilityID:        "spec_contract",
				LocalLevelImpact:    "L0",
				GlobalImpact:        maturity.ImpactCapabilityGap,
				Dimension:           "ui",
				SeverityDefault:     "SEVERITY_ERROR",
				CleanRequirement:    string(maturity.CleanRequirementRequired),
				RecommendedSkillIDs: []string{"experience-spec-authoring"},
			},
		},
		Fallback: maturity.FallbackPolicy{
			CapabilityID:     "spec_contract",
			LocalLevelImpact: "L0",
			GlobalImpact:     maturity.ImpactUnknown,
			Dimension:        "ui",
			SeverityDefault:  "SEVERITY_WARNING",
			CleanRequirement: string(maturity.CleanRequirementAdvisory),
		},
	}
}

func TestBuildMapsFindingsIntoMaturityAssessment(t *testing.T) {
	builder, err := NewBuilder(testSpec())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	assessment, err := builder.Build("demo", []spec.Finding{{
		Code:       "experience.schema_invalid",
		Severity:   "SEVERITY_ERROR",
		Message:    "schema failed",
		Locations:  []string{"experience/index.json"},
		Suggestion: "fix the JSON",
	}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if assessment.GetScenario() != "demo" {
		t.Fatalf("scenario = %q", assessment.GetScenario())
	}
	if len(assessment.GetFindings()) != 1 {
		t.Fatalf("findings = %d", len(assessment.GetFindings()))
	}
}

func TestDefaultSpecCoversFrozenExperienceFindingVocabulary(t *testing.T) {
	defaultSpec := DefaultSpec()
	if err := maturity.ValidateSpec(*defaultSpec); err != nil {
		t.Fatalf("DefaultSpec invalid: %v", err)
	}
	for _, code := range spec.AllFindingCodes {
		if _, ok := defaultSpec.Findings[code]; !ok {
			t.Fatalf("DefaultSpec missing finding mapping for %s", code)
		}
	}
	if len(defaultSpec.Findings) != len(spec.AllFindingCodes) {
		t.Fatalf("DefaultSpec maps %d findings, want %d", len(defaultSpec.Findings), len(spec.AllFindingCodes))
	}
}

func TestBuildErrorsOnUnmappedExperienceFinding(t *testing.T) {
	builder, err := NewBuilder(testSpec())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	_, err = builder.Build("demo", []spec.Finding{{
		Code:     "experience.unregistered",
		Severity: spec.SeverityWarning,
		Message:  "unknown",
	}})
	if err == nil {
		t.Fatal("Build succeeded with unmapped experience finding")
	}
}

func TestDefaultSpecMatchesTestGenieMaturityLadder(t *testing.T) {
	var descriptor struct {
		Maturity maturity.Spec `json:"maturity"`
	}
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "scenarios", "experience-manager", ".vrooli", "test-genie.json"))
	if err != nil {
		t.Fatalf("read test-genie descriptor: %v", err)
	}
	if err := json.Unmarshal(data, &descriptor); err != nil {
		t.Fatalf("unmarshal test-genie descriptor: %v", err)
	}
	want := DefaultSpec()
	got := descriptor.Maturity
	if got.Version != want.Version {
		t.Fatalf("version = %q, want %q", got.Version, want.Version)
	}
	if !reflect.DeepEqual(got.Capabilities, want.Capabilities) {
		t.Fatalf("capabilities drifted\n got: %#v\nwant: %#v", got.Capabilities, want.Capabilities)
	}
	if !reflect.DeepEqual(got.Findings, want.Findings) {
		t.Fatalf("findings drifted\n got: %#v\nwant: %#v", got.Findings, want.Findings)
	}
	if !reflect.DeepEqual(got.Fallback, want.Fallback) {
		t.Fatalf("fallback drifted\n got: %#v\nwant: %#v", got.Fallback, want.Fallback)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "VISION.md")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
