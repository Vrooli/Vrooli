package execution

import "testing"

func TestBaselineReceiptIdempotencyKeyChangesWithResolvedPreflight(t *testing.T) {
	base := Execution{
		ID: "execution-1",
		BaselineSet: BaselineSetState{
			Name:            "plan-baseline",
			ScenarioTargets: []string{"portal", "device-control"},
			RepoPaths:       []string{"scenarios/portal/**"},
			SourcePreflight: SourceEvidencePreflight{EligibleFiles: 10, EligibleBytes: 100},
		},
	}
	first := baselineReceiptIdempotencyKey(base)
	base.BaselineSet.SourcePreflight.EligibleBytes++
	second := baselineReceiptIdempotencyKey(base)
	if first == second {
		t.Fatal("content-intent changes must produce a new baseline idempotency key")
	}
	base.BaselineSet.SourcePreflight.EligibleBytes--
	if got := baselineReceiptIdempotencyKey(base); got != first {
		t.Fatalf("identical preflight must replay the same key: got %q want %q", got, first)
	}
}
