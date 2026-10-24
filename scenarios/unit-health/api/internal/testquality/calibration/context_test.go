package calibration

import "testing"

func TestContextMissingEvidenceAndErrorsCannotPass(t *testing.T) {
	cases := []ContextCase{{ID: "C061", ExpectedOutcome: "unknown", RequiresEvidence: true}}
	for _, observed := range [][]ContextObservation{
		nil,
		{{ID: "C061", Outcome: "unknown", UnknownReason: "owner-unavailable"}},
		{{ID: "C061", Outcome: "unknown", EvidenceRefs: []string{"owner:fixture"}}},
		{{ID: "C061", Outcome: "unknown", UnknownReason: "owner-unavailable", EvidenceRefs: []string{"owner:fixture"}, Error: "parser crashed"}},
	} {
		r := CompareContext(cases, observed)
		if r.MatchedCases != 0 || len(r.Results[0].Differences) == 0 {
			t.Fatalf("invalid observation passed: %+v", r)
		}
	}
	valid := ContextObservation{ID: "C061", Outcome: "unknown", UnknownReason: "owner-unavailable", EvidenceRefs: []string{"owner:fixture"}}
	r := CompareContext(cases, []ContextObservation{valid})
	if r.MatchedCases != 1 || r.TotalCases != 1 || r.UnknownCases != 1 {
		t.Fatalf("explicit owner unknown lost: %+v", r)
	}
	r = CompareContext(cases, []ContextObservation{valid, valid})
	if r.MatchedCases != 0 {
		t.Fatal("duplicate owner observations passed")
	}
}

func TestContextInvertedExpectedOutcomeFails(t *testing.T) {
	cases := []ContextCase{{ID: "C075", ExpectedOutcome: "detected_by_intended_test", RequiresEvidence: true}}
	observed := []ContextObservation{{ID: "C075", Outcome: "invalid_experiment", EvidenceRefs: []string{"run:compiler-error"}}}
	r := CompareContext(cases, observed)
	if r.MatchedCases != 0 {
		t.Fatal("compile failure was accepted as detected behavior")
	}
	observed[0].Outcome = "detected_by_intended_test"
	if r = CompareContext(cases, observed); r.MatchedCases != 1 {
		t.Fatalf("matching control failed: %+v", r)
	}
	cases[0].ExpectedOutcome = "excluded_with_reason"
	if r = CompareContext(cases, observed); r.MatchedCases != 0 {
		t.Fatal("inverted oracle passed")
	}
}

func TestContextInputsRetainBoundaryAndReviewCases(t *testing.T) {
	cases, err := LoadContextCases("../testdata")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 39 {
		t.Fatalf("expected 39 context cases, got %d", len(cases))
	}
	seen := map[string]bool{}
	for _, c := range cases {
		seen[c.ID] = true
	}
	for _, id := range []string{"C043", "C061", "C069", "C072", "C075", "C080"} {
		if !seen[id] {
			t.Fatalf("missing required context boundary %s", id)
		}
	}
}
