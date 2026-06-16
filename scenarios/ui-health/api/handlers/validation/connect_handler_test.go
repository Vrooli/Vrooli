package validation

import (
	"os"
	"path/filepath"
	"testing"

	"ui-health/internal/services/manifestvalidation"

	"github.com/vrooli/maturity-go/assessment"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func TestBuildMaturityAssessmentMapsUIFindings(t *testing.T) {
	spec := testMaturitySpec()
	report := manifestvalidation.Report{
		Scenario: "demo",
		Findings: []manifestvalidation.Finding{{
			Severity:   manifestvalidation.SeverityError,
			Code:       "overlay_unknown_slot",
			Location:   "scenarios/demo/.vrooli/ui-manifest.json",
			Message:    "unknown slot",
			Suggestion: "remove slot",
		}},
	}
	got, err := buildMaturityAssessment(report, spec)
	if err != nil {
		t.Fatalf("buildMaturityAssessment returned error: %v", err)
	}
	if got.GetProvider() != "ui-health" || got.GetPhase() != "ui-health" {
		t.Fatalf("assessment identity = %s/%s, want ui-health/ui-health", got.GetProvider(), got.GetPhase())
	}
	if got.GetLocal().GetCurrentLevel() != "L2" || got.GetLocal().GetNextLevel() != "L3" {
		t.Fatalf("local maturity = current %q next %q, want L2/L3", got.GetLocal().GetCurrentLevel(), got.GetLocal().GetNextLevel())
	}
	if got.GetFindings()[0].GetMaturity().GetGlobalImpact() != commonv1.GlobalImpact_GLOBAL_IMPACT_EVOLVABILITY_GAP {
		t.Fatalf("global impact = %v, want EVOLVABILITY_GAP", got.GetFindings()[0].GetMaturity().GetGlobalImpact())
	}
}

func TestMaturitySpecCoversUIHealthFindings(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", ".vrooli", "maturity.json"))
	if err != nil {
		t.Fatal(err)
	}
	spec, err := assessment.ParseSpec(raw)
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		"no_ui_surface",
		"ui_predates_template_layout",
		"service_json_path_invalid",
		"service_json_missing",
		"service_json_invalid",
		"template_id_missing",
		"template_unknown",
		"template_manifest_invalid",
		"contract_kind_mismatch",
		"contract_schema_mismatch",
		"slots_empty",
		"overlay_read_failed",
		"overlay_invalid",
		"overlay_unknown_slot",
		"slot_dir_empty",
		"slot_dir_missing",
		"slot_dir_stat_failed",
		"slot_dir_not_directory",
		"slot_parent_dir_missing",
		"slot_instances_empty",
		"path_pattern_unknown_token",
		"slot_dir_overlap_equal",
	} {
		if _, ok := spec.Findings[code]; !ok {
			t.Fatalf("maturity spec does not map emitted finding code %q", code)
		}
	}
}

func testMaturitySpec() *assessment.Spec {
	spec := &assessment.Spec{
		Provider: "ui-health",
		Phase:    "ui-health",
		Version:  "test",
		Levels: []assessment.Level{
			{ID: "L0", Name: "No UI"},
			{ID: "L1", Name: "Template readable"},
			{ID: "L2", Name: "Contract valid"},
			{ID: "L3", Name: "Overlay compatible"},
		},
		Findings: map[string]assessment.FindingMapping{
			"overlay_unknown_slot": {
				LocalLevelImpact:    "L3",
				GlobalImpact:        assessment.ImpactEvolvabilityGap,
				Dimension:           "ui",
				SeverityDefault:     "SEVERITY_ERROR",
				RecommendedSkillIDs: []string{"ui-health"},
			},
		},
		Fallback: assessment.FallbackPolicy{
			LocalLevelImpact: "L3",
			GlobalImpact:     assessment.ImpactEvolvabilityGap,
			Dimension:        "ui",
			SeverityDefault:  "SEVERITY_ERROR",
		},
	}
	if err := assessment.ValidateSpec(*spec); err != nil {
		panic(err)
	}
	return spec
}
