package conformance

import "testing"

func TestPassedAssertionsRequiresEveryNamedAssertionToPass(t *testing.T) {
	run := Run{Assertions: []Assertion{
		{Name: "passed", Outcome: OutcomePassed},
		{Name: "failed", Outcome: OutcomeFailed},
		{Name: "unmeasured", Outcome: OutcomeNotMeasured},
	}}

	if !PassedAssertions(run, "passed") {
		t.Fatal("expected observed passed assertion to pass")
	}
	for _, names := range [][]string{{"missing"}, {"failed"}, {"unmeasured"}, {"passed", "missing"}} {
		if PassedAssertions(run, names...) {
			t.Fatalf("PassedAssertions(%v) = true, want false", names)
		}
	}
}

func TestRenamedCaptureBudgetDoesNotAcceptLegacyAssertionName(t *testing.T) {
	run := Run{Assertions: []Assertion{{Name: "per_frame_cost_constant", Outcome: OutcomePassed}}}
	if PassedAssertions(run, "capture_callback_processing_budget") {
		t.Fatal("legacy assertion name must not satisfy the current capture processing budget")
	}
}
