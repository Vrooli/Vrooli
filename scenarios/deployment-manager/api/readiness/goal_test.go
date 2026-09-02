package readiness

import (
	"strings"
	"testing"
	"time"
)

func TestBuildGoalSpecSeedsEveryChecklistItem(t *testing.T) {
	checked := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{
		{ID: "one", Title: "One", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "one passes"},
		{ID: "two", Title: "Two", Category: "unanchored", CleanRequirement: Uncheckable, GlobalImpact: UnknownImpact, AcceptanceCriteria: "two observed"},
	}}
	verdict := Verdict{Scenario: "demo", Commit: "abc", CheckedAt: checked, Approved: false}
	spec, err := BuildGoalSpec("demo", "abc", "demo-offer", "price_changed", checklist, verdict)
	if err != nil || len(spec.Milestones) != 2 || spec.Milestones[0].AcceptanceCriteria[0] != "one passes" || spec.ServesDeliverable != "demo-offer" || !strings.Contains(spec.Description, "approved=false") {
		t.Fatalf("spec=%+v err=%v", spec, err)
	}
	if spec.Name != "readiness/demo/abc" {
		t.Fatalf("name=%q", spec.Name)
	}
}

func TestBuildGoalSpecRejectsMismatchedVerdict(t *testing.T) {
	checklist := DefaultChecklist()
	_, err := BuildGoalSpec("demo", "abc", "", "", checklist, Verdict{Scenario: "other", Commit: "abc"})
	if err == nil {
		t.Fatal("expected identity mismatch")
	}
}
