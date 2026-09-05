package readiness

import (
	"strings"
	"testing"
	"time"
)

func TestBuildGoalSpecSeedsOnlyUnresolvedChecklistItems(t *testing.T) {
	checked := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{
		validTestItem("one", Required, CapabilityGap, "one passes"),
		validTestItem("two", Uncheckable, UnknownImpact, "two observed"),
	}}
	verdict := Verdict{Scenario: "demo", Commit: "abc", CheckedAt: checked, Approved: false, Findings: []Finding{{ItemID: "one", Severity: "error", Status: SignalFailed, Signal: Signal{ItemID: "one", Status: SignalFailed, Source: "test-genie", RunID: "run-1", ObservedAt: checked, Reference: "run:run-1", Detail: "failed"}}}}
	spec, err := BuildGoalSpec("demo", "abc", "demo-offer", "price_changed", checklist, verdict)
	if err != nil || len(spec.Milestones) != 1 || spec.Milestones[0].AcceptanceCriteria[0] != "one passes" || !strings.Contains(spec.Milestones[0].Description, "run=run-1") || spec.ServesDeliverable != "demo-offer" || !strings.Contains(spec.Description, "approved=false") {
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
