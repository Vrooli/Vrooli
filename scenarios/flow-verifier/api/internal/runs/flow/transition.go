package flow

import (
	"flow-verifier/internal/runs/flow/generated"
)

// TransitionVerificationRun is the hand-authored wrapper around the
// generated state machine for the flow-verifier.verification-run.api
// flow. The wrapper exists so the host code can drape additional
// behaviour around the transition; here we just delegate.
func TransitionVerificationRun(
	status generated.VerificationRunStatus,
	event generated.VerificationRunEvent,
) (generated.VerificationRunStatus, error) {
	return generated.TransitionVerificationRunStatus(status, event)
}
