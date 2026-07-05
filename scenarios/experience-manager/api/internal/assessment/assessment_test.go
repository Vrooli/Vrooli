package assessment

import (
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
