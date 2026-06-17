package assessment

import (
	"slices"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
)

func validSpec() Spec {
	return Spec{
		Provider: "measures-health",
		Phase:    "measures",
		Version:  "1",
		Levels: []Level{
			{ID: "L0", Name: "No contract"},
			{ID: "L1", Name: "Domains derived"},
			{ID: "L2", Name: "Domains covered"},
			{ID: "L3", Name: "Measures tiered"},
		},
		Findings: map[string]FindingMapping{
			"measures.uncovered-domain": {
				LocalLevelImpact:    "L2",
				GlobalImpact:        ImpactCapabilityGap,
				Dimension:           "measures",
				SeverityDefault:     "ERROR",
				RecommendedSkillIDs: []string{"measures-adoption"},
			},
		},
		Fallback: FallbackPolicy{
			LocalLevelImpact: "L1",
			GlobalImpact:     ImpactUnknown,
			Dimension:        "measures",
			SeverityDefault:  "WARNING",
		},
	}
}

func TestValidateSpecRejectsUnknownDimension(t *testing.T) {
	spec := validSpec()
	spec.Findings["bad.dimension"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     ImpactEvolvabilityGap,
		Dimension:        "not-a-dimension",
	}
	if err := ValidateSpec(spec); err == nil {
		t.Fatal("ValidateSpec succeeded with unknown dimension")
	}
}

func TestValidateSpecRejectsUnknownImpact(t *testing.T) {
	spec := validSpec()
	spec.Findings["bad.impact"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     "not_an_impact",
		Dimension:        "measures",
	}
	if err := ValidateSpec(spec); err == nil {
		t.Fatal("ValidateSpec succeeded with unknown global impact")
	}
}

func TestEveryGlobalImpactHasRungSemanticMapping(t *testing.T) {
	for impact := range validImpacts {
		dims := DimensionsForImpact(impact)
		switch impact {
		case ImpactAdvisory, ImpactUnknown:
			if len(dims) != 0 {
				t.Fatalf("%s dimensions = %#v, want none", impact, dims)
			}
		default:
			if len(dims) == 0 {
				t.Fatalf("%s has no mapped dimensions", impact)
			}
		}
	}
}

func TestLocalMaturityUsesLowestBlockingLevel(t *testing.T) {
	spec := validSpec()
	got := LocalMaturity(spec, []Finding{
		{Code: "measures.uncovered-domain", Severity: "SEVERITY_ERROR"},
	})
	if got.CurrentLevel != "L1" {
		t.Fatalf("CurrentLevel = %q, want L1", got.CurrentLevel)
	}
	if got.NextLevel != "L2" {
		t.Fatalf("NextLevel = %q, want L2", got.NextLevel)
	}
	if len(got.BlockingFindingCodes) != 1 || got.BlockingFindingCodes[0] != "measures.uncovered-domain" {
		t.Fatalf("BlockingFindingCodes = %#v", got.BlockingFindingCodes)
	}
}

func TestNormalizeFindingFallsBackBySourceAndDefaultSeverity(t *testing.T) {
	spec := validSpec()
	spec.Fallback.Dimension = ""
	item := NormalizeFinding(spec, Finding{
		Code:   "unmapped",
		Source: architecturev1.FindingSource_FINDING_SOURCE_SECURITY,
	})
	if item.Mapping.Dimension != "security" {
		t.Fatalf("dimension = %q, want security", item.Mapping.Dimension)
	}
	if item.Severity != architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING {
		t.Fatalf("severity = %v, want WARNING", item.Severity)
	}
}

func TestAdvisoryDoesNotBlockLocalMaturity(t *testing.T) {
	spec := validSpec()
	spec.Findings["measures.context"] = FindingMapping{
		LocalLevelImpact: "L1",
		GlobalImpact:     ImpactAdvisory,
		Dimension:        "measures",
		SeverityDefault:  "ERROR",
	}
	got := LocalMaturity(spec, []Finding{{Code: "measures.context"}})
	if got.CurrentLevel != "L3" {
		t.Fatalf("CurrentLevel = %q, want L3", got.CurrentLevel)
	}
	if got.NextLevel != "" {
		t.Fatalf("NextLevel = %q, want empty", got.NextLevel)
	}
}

func TestBuildProtoAssessmentWithZeroFindings(t *testing.T) {
	got, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     validSpec(),
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	if got.GetScenario() != "demo" || got.GetProvider() != "measures-health" || got.GetPhase() != "measures" {
		t.Fatalf("assessment identity wrong: %+v", got)
	}
	if got.GetLocal().GetCurrentLevel() != "L3" || got.GetLocal().GetNextLevel() != "" {
		t.Fatalf("local maturity = current %q next %q, want L3/empty", got.GetLocal().GetCurrentLevel(), got.GetLocal().GetNextLevel())
	}
	if len(got.GetFindings()) != 0 {
		t.Fatalf("findings = %d, want zero", len(got.GetFindings()))
	}
}

func TestBuildProtoAssessmentIncludesBlockingFindingMetadata(t *testing.T) {
	got, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     validSpec(),
		Findings: []Finding{{
			Code:        "measures.uncovered-domain",
			Severity:    "ERROR",
			Title:       "Domain missing measures",
			Message:     "payments has no measures",
			Location:    "measures.json",
			Remediation: "Add a measure",
			Source:      architecturev1.FindingSource_FINDING_SOURCE_MEASURES,
			Phase:       "measures",
		}},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	if got.GetLocal().GetCurrentLevel() != "L1" || got.GetLocal().GetNextLevel() != "L2" {
		t.Fatalf("local maturity = current %q next %q, want L1/L2", got.GetLocal().GetCurrentLevel(), got.GetLocal().GetNextLevel())
	}
	if got.GetLocal().GetBlockingFindingCodes()[0] != "measures.uncovered-domain" {
		t.Fatalf("blocking codes = %#v", got.GetLocal().GetBlockingFindingCodes())
	}
	finding := got.GetFindings()[0]
	if finding.GetMaturity().GetDimension() != "measures" {
		t.Fatalf("dimension = %q, want measures", finding.GetMaturity().GetDimension())
	}
	if got.GetFindingsByGlobalImpact()["capability_gap"] != 1 {
		t.Fatalf("impact counts = %#v", got.GetFindingsByGlobalImpact())
	}
}

func TestBuildProtoAssessmentRejectsInvalidSpec(t *testing.T) {
	spec := validSpec()
	spec.Provider = ""
	if _, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: spec}); err == nil {
		t.Fatal("BuildProtoAssessment succeeded with invalid spec")
	}
}

func TestValidateAssessmentRejectsMalformedAssessment(t *testing.T) {
	got, err := BuildProtoAssessment(BuildInput{Scenario: "demo", Spec: validSpec()})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	got.Local.CurrentLevel = ""
	if err := ValidateAssessment(got); err == nil {
		t.Fatal("ValidateAssessment succeeded with empty current level")
	}
}

func TestBuildProtoAssessmentDeduplicatesRecommendedSkills(t *testing.T) {
	spec := validSpec()
	spec.Findings["measures.extra"] = FindingMapping{
		LocalLevelImpact:    "L2",
		GlobalImpact:        ImpactCapabilityGap,
		Dimension:           "measures",
		SeverityDefault:     "ERROR",
		RecommendedSkillIDs: []string{"quality-health", "measures-adoption"},
	}
	got, err := BuildProtoAssessment(BuildInput{
		Scenario: "demo",
		Spec:     spec,
		Findings: []Finding{
			{Code: "measures.extra"},
			{Code: "measures.uncovered-domain"},
		},
	})
	if err != nil {
		t.Fatalf("BuildProtoAssessment returned error: %v", err)
	}
	want := []string{"measures-adoption", "quality-health"}
	if !slices.Equal(got.GetRecommendedSkillIds(), want) {
		t.Fatalf("recommended skills = %#v, want %#v", got.GetRecommendedSkillIds(), want)
	}
}
