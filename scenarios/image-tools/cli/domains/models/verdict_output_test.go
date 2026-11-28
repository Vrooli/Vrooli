package models

import (
	"strings"
	"testing"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

func verdictModel() *modelsv1.Model {
	return &modelsv1.Model{
		Id: "flux-2-klein-4b", Name: "FLUX.2 [klein] 4B", Tier: "quality", Backend: "diffusers",
		Operations: []string{"edit_instruct", "upscale"}, Enabled: true,
		Verdict: &modelsv1.Verdict{Grade: modelsv1.VerdictGrade_VERDICT_GRADE_RECOMMENDED, Reason: "edit_instruct: kept identity"},
		Verdicts: []*modelsv1.Verdict{
			{Operation: "edit_instruct", Grade: modelsv1.VerdictGrade_VERDICT_GRADE_RECOMMENDED, Reason: "kept identity",
				Evidence: []*modelsv1.VerdictEvidence{{Kind: "judged_comparison", Ref: "research:RESULTS.md", Date: "2026-11-21"}}},
			{Operation: "upscale", Grade: modelsv1.VerdictGrade_VERDICT_GRADE_UNTESTED, Reason: "no qualification, job history or rating yet"},
		},
		Override: &modelsv1.StateOverride{Enabled: true, SetBy: "operator", Relation: "agrees", SeedEnabled: true},
	}
}

// TestModelLinesLeadWithTheVerdict is the at-a-glance contract: every model line
// starts with its verdict, and the detail view shows each operation's verdict,
// its evidence pointers and the override provenance.
func TestModelLinesLeadWithTheVerdict(t *testing.T) {
	line := formatModel(verdictModel())
	if !strings.HasPrefix(line, "recommended     flux-2-klein-4b") {
		t.Fatalf("line = %q", line)
	}
	if !strings.HasPrefix(formatModel(&modelsv1.Model{Id: "x"}), "untested        x") {
		t.Fatal("a model with no verdict reads as untested, never blank")
	}
	detail := formatModelDetail(verdictModel())
	for _, want := range []string{
		"verdict: recommended: edit_instruct: kept identity",
		"edit_instruct: recommended: kept identity",
		"evidence: judged_comparison: research:RESULTS.md (2026-11-21)",
		"upscale: untested",
		"override: enabled by operator",
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail lacks %q:\n%s", want, detail)
		}
	}
}
