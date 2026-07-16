package orchestration

import (
	"context"
	"time"

	"agent-manager/internal/domain"
)

type RunStatusTransitionInput struct {
	Run             *domain.Run
	NewStatus       domain.RunStatus
	Phase           domain.RunPhase
	Reason          string
	EndedAt         *time.Time
	ErrorMsg        string
	ExitCode        *int
	LastHeartbeat   *time.Time
	ProgressPercent *int
	Summary         *domain.RunSummary
	Result          *domain.RunResult
}

func (o *Orchestrator) applyRunStatusTransition(ctx context.Context, input RunStatusTransitionInput) (*domain.Run, error) {
	run := input.Run
	if run == nil {
		return nil, domain.NewValidationError("run", "run is required")
	}
	if input.NewStatus == "" {
		return nil, domain.NewValidationError("newStatus", "new status is required")
	}

	previousStatus := run.Status

	// Enforce the run state machine in the single status-mutation helper so no
	// status can be set ad-hoc. A same-status write is a no-op update (heartbeat/
	// progress refresh), not a transition, and is always permitted.
	if previousStatus != input.NewStatus {
		if allowed, reason := previousStatus.CanTransitionTo(input.NewStatus); !allowed {
			return nil, domain.NewStateError(
				"Run", string(previousStatus), "transition to "+string(input.NewStatus), reason)
		}
	}

	now := time.Now()

	run.Status = input.NewStatus
	if input.Phase != "" {
		run.Phase = input.Phase
	}
	if input.ProgressPercent != nil {
		run.ProgressPercent = *input.ProgressPercent
	} else if input.Phase != "" {
		run.ProgressPercent = domain.PhaseToProgress(input.Phase)
	}
	if input.EndedAt != nil {
		run.EndedAt = input.EndedAt
	}
	if input.LastHeartbeat != nil {
		run.LastHeartbeat = input.LastHeartbeat
	}
	if input.ErrorMsg != "" {
		run.ErrorMsg = input.ErrorMsg
	}
	if input.ExitCode != nil {
		run.ExitCode = input.ExitCode
	}
	if input.Summary != nil {
		run.Summary = input.Summary
	}
	if input.Result != nil {
		run.Result = input.Result
	}
	run.UpdatedAt = now

	if err := o.runs.Update(ctx, run); err != nil {
		return nil, err
	}

	if previousStatus != input.NewStatus && o.events != nil {
		statusEvent := domain.NewStatusEvent(
			run.ID,
			string(previousStatus),
			string(input.NewStatus),
			input.Reason,
		)
		if err := o.appendAndBroadcastEvents(ctx, run.ID, statusEvent); err != nil {
			return nil, err
		}
	}

	hydrated := o.attachRunActions(ctx, run)
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(hydrated)
	}

	return hydrated, nil
}
