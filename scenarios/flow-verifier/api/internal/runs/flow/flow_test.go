package flow

import (
	"testing"

	"flow-verifier/internal/runs/flow/generated"
)

func TestVerificationRunFormalReplay(t *testing.T) {
	generated.RunReplay(t, func(
		status generated.VerificationRunStatus,
		event generated.VerificationRunEvent,
	) (generated.VerificationRunStatus, error) {
		return TransitionVerificationRun(status, event)
	})
}
