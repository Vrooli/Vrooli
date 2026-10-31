// This file owns run lifecycle transitions shared by orchestration operations.
package orchestration

import (
	"context"
	"time"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/runsignal"
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
	// StopReason, when set on a terminal transition, is written with the
	// status in the same update (with its terminal class).
	StopReason domain.RunStopReason
	// InteractiveContinuation is set only by the retained-session continuation
	// path, which must atomically replace the exact prior invocation fence while
	// moving a parked or terminal run back to Running.
	InteractiveContinuation                bool
	ExpectedInteractiveInvocationStartedAt *time.Time
}

// applyRunStatusTransition returns a nonnil run with an error only when the
// run-row update committed but its subsequent status-event append failed. A
// nil run with an error means the row update did not commit. Callers that own
// post-commit obligations can therefore finish them without hiding the
// publication error or probing durable state a second time.
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
	if input.InteractiveContinuation && (previousStatus != input.NewStatus && ((!previousStatus.IsTerminal() && previousStatus != domain.RunStatusParked) || input.NewStatus != domain.RunStatusRunning || run.ExecutionMode.Normalized() != domain.ExecutionModeInteractive)) {
		return nil, domain.NewValidationError("interactiveContinuation", "requires a parked or terminal interactive run transitioning to Running")
	}

	now := o.now()

	run.Status = input.NewStatus
	if input.NewStatus == domain.RunStatusRunning && previousStatus != domain.RunStatusRunning {
		// These fields describe the current turn. Prior output, accounting,
		// handoff, and the original start time remain available across turns.
		run.EndedAt = nil
		run.ErrorMsg = ""
		run.ExitCode = nil
		run.TerminalClass = ""
		run.StopReason = ""
	}
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
	if input.NewStatus.IsTerminal() {
		o.accountSummaryTokens(ctx, run)
	}
	if input.Result != nil {
		run.Result = input.Result
	}
	if input.StopReason != "" && input.NewStatus.IsTerminal() {
		run.StopReason = input.StopReason
		run.TerminalClass = domain.TerminalClassForStopReason(input.StopReason)
	}
	run.UpdatedAt = now

	var updateErr error
	if input.InteractiveContinuation {
		updateErr = o.runs.UpdateInteractiveContinuation(ctx, run, previousStatus, input.ExpectedInteractiveInvocationStartedAt)
	} else if previousStatus.IsTerminal() && input.NewStatus == domain.RunStatusRunning && run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		return nil, domain.NewValidationError("interactiveContinuation", "terminal interactive runs require the explicit continuation path")
	} else {
		updateErr = o.runs.Update(ctx, run)
	}
	if updateErr != nil {
		return nil, updateErr
	}

	if previousStatus != input.NewStatus && o.events != nil {
		statusEvent := domain.NewStatusEvent(
			run.ID,
			string(previousStatus),
			string(input.NewStatus),
			input.Reason,
		)
		if err := o.appendAndBroadcastEvents(ctx, run.ID, statusEvent); err != nil {
			// The Run update above has committed. Return that committed projection
			// with the publication error so lifecycle owners can finish any
			// post-commit obligations without treating the transition as rolled
			// back. Repository update failures still return nil before this point.
			return o.attachRunActions(ctx, run), err
		}
	}

	hydrated := o.attachRunActions(ctx, run)
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(hydrated)
	}
	if previousStatus != input.NewStatus && hydrated.Status.IsTerminal() {
		o.projectTerminalInvocationReadModel(hydrated)
		if o.credentialUseReleaser != nil {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := o.credentialUseReleaser.RevokeRunCredentialUse(cleanupCtx, hydrated.ID, string(hydrated.Status))
			cancel()
			if err != nil {
				obs.Component("credential-use").Warn("terminal credential cleanup failed", obs.KeyRunID, hydrated.ID.String(), obs.KeyError, err.Error())
			}
		}
	}

	return hydrated, nil
}

// accountSummaryTokens writes the run's usage-account total into its summary,
// so a terminal summary agrees with the read model whichever path produced it
// (runner stream metrics, transcript recovery, interactive tail).
func (o *Orchestrator) accountSummaryTokens(ctx context.Context, run *domain.Run) {
	if o.events == nil || run.Summary == nil {
		return
	}
	events, err := o.meteredRunEvents(ctx, run.ID)
	if err != nil {
		obs.Component("run-lifecycle").Warn("terminal usage account unavailable; summary keeps stream metrics", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return
	}
	for _, event := range events {
		if _, ok := event.Data.(*domain.UsageEventData); ok {
			run.Summary.TokensUsed = runsignal.AccountUsage(events).TotalTokens()
			return
		}
	}
}
