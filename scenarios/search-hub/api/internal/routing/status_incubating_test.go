package routing

import (
	"strings"
	"testing"
)

func TestIncubatingNextActionFollowsEvidenceState(t *testing.T) {
	tests := []struct {
		name     string
		evidence EvalQualityEvidence
		want     string
	}{
		{name: "missing suite", want: "register a reviewed provider suite"},
		{name: "missing live positive", evidence: EvalQualityEvidence{SuitePresent: true}, want: "add a live reviewed positive case"},
		{name: "missing recent pass", evidence: EvalQualityEvidence{SuitePresent: true, LiveReviewedPositive: true}, want: "run a recent passing evaluation"},
		{name: "ready", evidence: EvalQualityEvidence{SuitePresent: true, LiveReviewedPositive: true, RecentPassingRun: true}, want: "declare production lifecycle"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := incubatingNextAction(test.evidence)
			if got != test.want {
				t.Fatalf("incubatingNextAction() = %q, want %q", got, test.want)
			}
			if strings.Contains(got, "provider") && test.name != "missing suite" && test.name != "missing live positive" {
				t.Fatalf("next action should describe gate state, got %q", got)
			}
		})
	}
}
