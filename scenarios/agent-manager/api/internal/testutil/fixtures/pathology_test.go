package fixtures

import (
	"testing"

	"agent-manager/internal/domain"
)

func TestPathologyReplayFinalizesEachNamedFailureMode(t *testing.T) {
	cases := []struct {
		name       string
		selection  domain.FinalOutputSelectionStatus
		structured domain.StructuredResultStatus
	}{
		{"ambiguous-final-output", domain.FinalOutputSelectionAmbiguous, ""},
		{"unavailable-final-output", domain.FinalOutputSelectionUnavailable, ""},
		{"invalid-structured-result", domain.FinalOutputSelectionSelected, domain.StructuredResultInvalid},
		{"abstained-structured-result", domain.FinalOutputSelectionSelected, domain.StructuredResultAbstained},
		{"tool-failures", domain.FinalOutputSelectionUnavailable, ""},
		{"model-fallback", domain.FinalOutputSelectionSelected, ""},
		{"heartbeat-gap", domain.FinalOutputSelectionUnavailable, ""},
		{"oversized-diff", domain.FinalOutputSelectionUnavailable, ""},
		{"zero-events", domain.FinalOutputSelectionUnavailable, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			replay := ReplayPathology(t, tc.name)
			if replay.Run.Result.Selection.Status != tc.selection {
				t.Fatalf("%s selection=%s want=%s", replay, replay.Run.Result.Selection.Status, tc.selection)
			}
			if tc.structured != "" && (replay.Run.Result.Structured == nil || replay.Run.Result.Structured.Status != tc.structured) {
				t.Fatalf("%s structured=%v want=%s", replay, replay.Run.Result.Structured, tc.structured)
			}
		})
	}
}
