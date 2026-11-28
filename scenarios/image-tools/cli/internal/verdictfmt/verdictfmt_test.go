package verdictfmt

import (
	"strings"
	"testing"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

func TestLabelsAreThePlainWordsForEveryGrade(t *testing.T) {
	want := map[modelsv1.VerdictGrade]string{
		modelsv1.VerdictGrade_VERDICT_GRADE_RECOMMENDED:     "recommended",
		modelsv1.VerdictGrade_VERDICT_GRADE_WORKS:           "works",
		modelsv1.VerdictGrade_VERDICT_GRADE_EXPERIMENTAL:    "experimental",
		modelsv1.VerdictGrade_VERDICT_GRADE_UNTESTED:        "untested",
		modelsv1.VerdictGrade_VERDICT_GRADE_NOT_RECOMMENDED: "not recommended",
		modelsv1.VerdictGrade_VERDICT_GRADE_BROKEN:          "broken",
		modelsv1.VerdictGrade_VERDICT_GRADE_UNSPECIFIED:     "untested",
	}
	for grade, label := range want {
		if got := Label(grade); got != label {
			t.Errorf("Label(%s) = %q, want %q", grade, got, label)
		}
		if got := Badge(&modelsv1.Verdict{Grade: grade}); len(got) != len("not recommended") || strings.TrimSpace(got) != label {
			t.Errorf("Badge(%s) = %q, want %q padded to the longest label", grade, got, label)
		}
	}
}

func TestOverrideShowsWhoWhenWhyAndTheContradiction(t *testing.T) {
	got := Override(&modelsv1.StateOverride{
		Enabled: false, SetBy: "agent-7", SetAt: "2026-11-10T04:00:00Z", Reason: "judge flagged it",
		Relation: "contradicts_seed", SeedEnabled: true, Note: "the seed says enabled=true; this override says enabled=false and wins until the seed's flag changes",
	})
	for _, want := range []string{"disabled by agent-7", "2026-11-10", "judge flagged it", "contradicts the seed, which says enabled=true", "until the seed's flag changes"} {
		if !strings.Contains(got, want) {
			t.Errorf("override line %q lacks %q", got, want)
		}
	}
	if !strings.Contains(Override(&modelsv1.StateOverride{Enabled: true, SetBy: "operator", Pinned: true, Reason: "licence", Relation: "agrees"}), "[pinned]") {
		t.Error("a pinned override must say so")
	}
	if !strings.Contains(Override(&modelsv1.StateOverride{SetBy: "agent-9", Relation: "superseded"}), "superseded, not applied") {
		t.Error("a superseded override must say it is not applied")
	}
	if Override(nil) != "" {
		t.Error("no override renders as nothing")
	}
}

func TestEvidenceListsEveryPointerWithItsDate(t *testing.T) {
	lines := Evidence(&modelsv1.Verdict{Evidence: []*modelsv1.VerdictEvidence{
		{Kind: "judged_comparison", Ref: "research:RESULTS.md", Date: "2026-11-21"},
		{Kind: "operator_rating", Ref: "image-tools jobs ratings"},
	}}, "  ")
	if len(lines) != 2 || lines[0] != "  judged_comparison: research:RESULTS.md (2026-11-21)" || lines[1] != "  operator_rating: image-tools jobs ratings" {
		t.Fatalf("lines = %q", lines)
	}
}
