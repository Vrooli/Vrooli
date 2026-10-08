package orchestration

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/testutil/mocks"

	"github.com/google/uuid"
)

func TestContinuationCheckpointStopsManualReviewWithoutDeleting(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   domain.RunStatus
		timedOut bool
		event    domain.SandboxLifecycleEvent
	}{
		{"success", domain.RunStatusComplete, false, domain.SandboxLifecycleTurnCompleted},
		{"failure", domain.RunStatusFailed, false, domain.SandboxLifecycleTurnFailed},
		{"timeout", domain.RunStatusFailed, true, domain.SandboxLifecycleTurnFailed},
		{"cancel", domain.RunStatusCancelled, false, domain.SandboxLifecycleTurnCancelled},
		{"terminal_wildcard", domain.RunStatusComplete, false, domain.SandboxLifecycleTerminal},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stub := mocks.NewFakeSandboxProvider()
			stub.StopFunc = func(ctx context.Context, _ uuid.UUID) error {
				if ctx.Err() != nil {
					t.Error("cleanup inherited cancelled turn context")
				}
				return nil
			}
			id, disabled := uuid.New(), false
			run := &domain.Run{ID: uuid.New(), RunMode: domain.RunModeSandboxed,
				SandboxID: &id, Status: tc.status,
				ResolvedConfig: &domain.RunConfig{SandboxConfig: &domain.SandboxConfig{
					Mode: domain.SandboxModeProtected, ManualReview: true,
					AutoApply: &disabled, ApplyOnFailure: &disabled,
					Lifecycle: domain.SandboxLifecycleConfig{StopOn: []domain.SandboxLifecycleEvent{tc.event}},
				}},
			}
			o := &Orchestrator{sandbox: stub}
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			o.checkpointContinuationTurn(ctx, run, nil, tc.timedOut)
			if stub.StopCallCount() != 1 || stub.DeleteCallCount() != 0 {
				t.Fatalf("stop=%d delete=%d; want one stop with edits retained", stub.StopCallCount(), stub.DeleteCallCount())
			}
			if run.Status != domain.RunStatusNeedsReview || run.Phase != domain.RunPhaseCompleted {
				t.Fatalf("unsettled continuation: status=%s phase=%s", run.Status, run.Phase)
			}
		})
	}
}
