package readiness

import (
	"testing"
	"time"
)

func TestAggregateDoesNotTreatMissingSignalsAsPass(t *testing.T) {
	checked := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{{ID: "required-check", Title: "Required", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "must pass"}, {ID: "advisory-check", Title: "Advisory", Category: "mechanical", CleanRequirement: Advisory, GlobalImpact: AdvisoryImpact, AcceptanceCriteria: "should pass"}}}
	verdict, err := Aggregate("demo", "abc", checklist, []Signal{{ItemID: "advisory-check", Status: SignalFailed, Source: "marketing", ObservedAt: checked, Detail: "asset absent"}}, checked)
	if err != nil {
		t.Fatal(err)
	}
	if verdict.Approved || len(verdict.Findings) != 2 {
		t.Fatalf("verdict = %+v", verdict)
	}
	for _, finding := range verdict.Findings {
		if finding.ItemID == "required-check" && finding.Status != SignalUnknown {
			t.Fatalf("missing required signal = %+v", finding)
		}
	}
}

func TestAggregateKeepsAdvisoryFindingVisibleWithoutBlocking(t *testing.T) {
	checked := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{{ID: "advisory-check", Title: "Advisory", Category: "mechanical", CleanRequirement: Advisory, GlobalImpact: AdvisoryImpact, AcceptanceCriteria: "should pass"}}}
	verdict, err := Aggregate("demo", "abc", checklist, []Signal{{ItemID: "advisory-check", Status: SignalFailed, Source: "marketing", RunID: "run-1", ObservedAt: checked, Detail: "asset absent"}}, checked)
	if err != nil || !verdict.Approved || len(verdict.Findings) != 1 || verdict.Findings[0].Signal.RunID != "run-1" {
		t.Fatalf("verdict = %+v, err=%v", verdict, err)
	}
}

func TestAggregateMarksOlderCommitSignalsStale(t *testing.T) {
	checked := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{{ID: "check", Title: "Check", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "must pass"}}}
	verdict, err := Aggregate("demo", "new", checklist, []Signal{{ItemID: "check", Status: SignalPassed, Source: "test-genie", Commit: "old", ObservedAt: checked}}, checked)
	if err != nil || verdict.Approved || len(verdict.Findings) != 1 || verdict.Findings[0].Status != SignalStale {
		t.Fatalf("verdict = %+v, err=%v", verdict, err)
	}
}

func TestAggregateRejectsUnknownAndDuplicateSignals(t *testing.T) {
	checked := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	checklist := Checklist{Version: ChecklistVersion, Items: []Item{{ID: "check", Title: "Check", Category: "mechanical", CleanRequirement: Required, GlobalImpact: CapabilityGap, AcceptanceCriteria: "must pass"}}}
	unknown := []Signal{{ItemID: "other", Status: SignalPassed, Source: "source", ObservedAt: checked}}
	if _, err := Aggregate("demo", "new", checklist, unknown, checked); err == nil {
		t.Fatal("expected unknown signal refusal")
	}
	duplicate := []Signal{{ItemID: "check", Status: SignalPassed, Source: "a", ObservedAt: checked}, {ItemID: "check", Status: SignalFailed, Source: "b", ObservedAt: checked}}
	if _, err := Aggregate("demo", "new", checklist, duplicate, checked); err == nil {
		t.Fatal("expected duplicate signal refusal")
	}
}
