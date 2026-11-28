package apply

import (
	"testing"

	applyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply"
	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
)

func TestInterruptedRunUsesExistingIndeterminateWireState(t *testing.T) {
	if got := runState("interrupted"); got != applyv1.ApplyRunState_APPLY_RUN_STATE_INDETERMINATE {
		t.Fatalf("wire status = %v, want indeterminate", got)
	}
	if got := runProto(applydomain.Run{ID: "apply-interrupted", Status: "interrupted"}).GetLegacyStatus(); got != "interrupted" {
		t.Fatalf("legacy status = %q, want persisted interrupted label", got)
	}
}

func TestIndeterminateStepPreservesUnknownEffectDetails(t *testing.T) {
	got := stepProto(applydomain.Step{
		Item:          applydomain.Item{ID: "resource:postgres", Kind: "resource", Name: "postgres"},
		State:         "indeterminate",
		LegacyOutcome: "indeterminate",
		Disposition:   "blocked",
		Error:         "the control plane cannot confirm whether the effect completed",
		ErrorCode:     "effect_outcome_unknown",
		Remediation:   "Review target state before retrying",
	})
	if got.GetLegacyOutcome() != "indeterminate" || got.GetErrorCode() != "effect_outcome_unknown" || got.GetError() == "" || got.GetRemediation() == "" {
		t.Fatalf("step projection lost unknown-effect details: %+v", got)
	}
}
