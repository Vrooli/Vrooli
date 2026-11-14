package candidates

import (
	"testing"

	"brand-manager/internal/candidates"
	"connectrpc.com/connect"
)

func TestTranslateCandidateLifecycleRefusals(t *testing.T) {
	invalid := translate(candidates.ErrInvalidStatusTransition{From: candidates.StatusRejected, To: candidates.StatusPicked})
	if code := connect.CodeOf(invalid); code != connect.CodeFailedPrecondition {
		t.Fatalf("invalid transition code = %s, want failed precondition", code)
	}
	conflict := translate(candidates.ErrStatusConflict{ID: "candidate-1", Expected: candidates.StatusProposed})
	if code := connect.CodeOf(conflict); code != connect.CodeAborted {
		t.Fatalf("status conflict code = %s, want aborted", code)
	}
}
