// Responsibility: read runs and apply guarded lifecycle stop, deletion and recovery operations.
package orchestration

import (
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/repository"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"strings"
)

func (o *Orchestrator) GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	run, err := o.runs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.NewNotFoundError("Run", id)
	}
	if err := o.observeInteractiveRun(ctx, run); err != nil {
		return nil, err
	}
	return o.attachRunActions(ctx, run), nil
}

// GetRunByImportProvenance resolves an imported external session without
// exposing its filesystem path. Corpus import uses this before parsing so a
// repeated command is read-only for already adopted evidence.
func (o *Orchestrator) GetRunByImportProvenance(ctx context.Context, sourceHarness, sourceSessionID string) (*domain.Run, error) {
	return o.runs.GetByImportProvenance(ctx, sourceHarness, sourceSessionID)
}

func (o *Orchestrator) ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error) {
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		ListFilter: repository.ListFilter{
			Limit:  opts.Limit,
			Offset: opts.Offset,
		},
		TaskID:                    opts.TaskID,
		AgentProfileID:            opts.AgentProfileID,
		Status:                    opts.Status,
		TagPrefix:                 opts.TagPrefix,
		ScopePrefix:               opts.ScopePrefix,
		InvestigatesRunID:         opts.InvestigatesRunID,
		AppliesInvestigationRunID: opts.AppliesInvestigationRunID,
		ParentRunID:               opts.ParentRunID,
	})
	if err != nil {
		return nil, err
	}
	return o.attachRunActionsList(ctx, runs), nil
}

// ListParkedRuns returns all parked runs with their await-handle populated. The
// pruned list-columns omit the heavy await_handle field, so each parked run is
// reloaded by ID to recover its handle. Used by the await-handle registry's
// restart recovery (re-spawning waiters on boot).
func (o *Orchestrator) ListParkedRuns(ctx context.Context) ([]*domain.Run, error) {
	parked := domain.RunStatusParked
	rows, err := o.runs.List(ctx, repository.RunListFilter{Status: &parked})
	if err != nil {
		return nil, err
	}
	full := make([]*domain.Run, 0, len(rows))
	for _, row := range rows {
		loaded, err := o.runs.Get(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if loaded == nil {
			continue
		}
		full = append(full, loaded)
	}
	return full, nil
}

// ParkedRunsOnHandle returns the parked runs awaiting exactly producer and key,
// so a caller that knows only the await key needs no run ID.
func (o *Orchestrator) ParkedRunsOnHandle(ctx context.Context, producer, key string) ([]*domain.Run, error) {
	parked, err := o.ListParkedRuns(ctx)
	if err != nil {
		return nil, err
	}
	matched := make([]*domain.Run, 0, 1)
	for _, run := range parked {
		if run.AwaitHandle != nil && run.AwaitHandle.Producer == producer && run.AwaitHandle.Key == key {
			matched = append(matched, run)
		}
	}
	return matched, nil
}

func (o *Orchestrator) DeleteRun(ctx context.Context, id uuid.UUID) error {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if allowed, reason := domain.CanDeleteRun(run); !allowed {
		return domain.NewStateError("Run", string(run.Status), "delete", reason)
	}
	if err := o.runs.Delete(ctx, id); err != nil {
		return err
	}
	o.notifyConversationSearch(ctx, "delete_run", id.String(), "")
	return nil
}

// GetRunByTag retrieves a run by its custom tag.
// Returns NotFoundError if no run with that tag exists.
func (o *Orchestrator) GetRunByTag(ctx context.Context, tag string) (*domain.Run, error) {
	// List all runs with matching tag prefix and find exact match
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		TagPrefix: tag,
	})
	if err != nil {
		return nil, err
	}

	// Find exact match
	for _, run := range runs {
		if run.GetTag() == tag {
			return o.attachRunActions(ctx, run), nil
		}
	}

	return nil, domain.NewNotFoundError("Run", uuid.Nil)
}

// StopRunByTag stops a run identified by its custom tag.
func (o *Orchestrator) StopRunByTag(ctx context.Context, tag string) error {
	run, err := o.GetRunByTag(ctx, tag)
	if err != nil {
		return err
	}
	return o.StopRun(ctx, run.ID)
}

// StopAllRuns stops all running runs, optionally filtered by tag prefix.
func (o *Orchestrator) StopAllRuns(ctx context.Context, opts StopAllOptions) (*StopAllResult, error) {
	result := &StopAllResult{
		FailedIDs: []string{},
	}

	// Get all running or starting runs
	runningStatus := domain.RunStatusRunning
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		Status:    &runningStatus,
		TagPrefix: opts.TagPrefix,
	})
	if err != nil {
		return nil, err
	}

	// Also get starting runs
	startingStatus := domain.RunStatusStarting
	startingRuns, err := o.runs.List(ctx, repository.RunListFilter{
		Status:    &startingStatus,
		TagPrefix: opts.TagPrefix,
	})
	if err != nil {
		return nil, err
	}
	runs = append(runs, startingRuns...)

	// Stop each run
	for _, run := range runs {
		// Skip already stopped runs
		if run.Status == domain.RunStatusComplete ||
			run.Status == domain.RunStatusFailed ||
			run.Status == domain.RunStatusCancelled {
			result.Skipped++
			continue
		}

		if err := o.StopRun(ctx, run.ID); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, run.ID.String())
		} else {
			result.Stopped++
		}
	}

	return result, nil
}

func (o *Orchestrator) StopRun(ctx context.Context, id uuid.UUID) error {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return importedRunLifecycleError("stop")
	}

	// An already-cancelled run is an idempotent replay, not a state error. A
	// caller that issued a stop and lost the response may replay it after an
	// owner reopen; the stop already succeeded, so return the same terminal
	// result without a second terminal write. Stop is the only path to
	// cancelled, so observing cancelled means this operation completed.
	if run.Status == domain.RunStatusCancelled {
		return nil
	}

	if allowed, reason := domain.CanStopRun(run); !allowed {
		return domain.NewStateError("Run", string(run.Status), "stop", reason)
	}

	// Interactive runs have no local process to signal — the CLI lives in a
	// web-console tmux session. Stop them via the interrupt-then-delete
	// escalation ladder and finalize deterministically (Cancelled), instead of
	// the pgid/terminator path below.
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		return o.stopInteractiveRun(ctx, run)
	}

	// A parked run has no live process to terminate — stopping it cancels the
	// await (clears the handle) and moves the run to cancelled. The waiter that
	// owns the handle observes the terminal status and deregisters (Phase 3).
	if run.Status == domain.RunStatusParked {
		// Cancel the background watcher first so it observes the cancellation and
		// exits without waking the now-cancelled run.
		if o.awaitRegistry != nil {
			o.awaitRegistry.Cancel(id)
		}
		run.AwaitHandle = nil
		endedAt := o.now()
		_, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
			Run:       run,
			NewStatus: domain.RunStatusCancelled,
			Phase:     domain.RunPhaseCompleted,
			Reason:    "Parked run stopped by request",
			EndedAt:   &endedAt,
		})
		return err
	}

	// Persist the cancellation intent before any process is terminated. A runner
	// process that exits after this stamp (even before the terminator confirms
	// death) reconciles to cancelled instead of resurrecting the run to complete.
	// The stamp is monotonic and status-guarded, so a repeat stop request is a
	// no-op rather than a second terminal write.
	if o.runs != nil {
		if _, err := o.runs.RequestCancellation(ctx, id, o.now()); err != nil {
			return err
		}
	}

	if o.terminator != nil {
		result, err := o.terminator.Terminate(ctx, id)
		if err != nil {
			return err
		}
		if !result.Success {
			return result.Error
		}

		_, err = o.finalizeRunCancellation(ctx, id, "Run stopped by request")
		return err
	}

	// The immutable resolved config is the sole execution authority.
	var runnerType domain.RunnerType
	if run.ResolvedConfig != nil {
		runnerType = run.ResolvedConfig.RunnerType
	}

	// Stop execution if we have a runner type
	if o.runners != nil && runnerType != "" {
		if r, err := o.runners.Get(runnerType); err == nil {
			if err := r.Stop(ctx, run.ID); err != nil {
				return err
			}
		}
	}

	_, err = o.finalizeRunCancellation(ctx, id, "Run stopped by request")
	return err
}

// finalizeRunCancellation moves a run to cancelled after its stop request has
// been reconciled. It re-reads durable state first: if a delayed process exit
// already reconciled the run to a terminal status (for example the executor's
// HandleResult adopted the persisted cancellation intent), this is a no-op
// instead of a second terminal write. That makes stop a replay-safe operation
// whether the terminator or the executor wins the terminal race.
func (o *Orchestrator) finalizeRunCancellation(ctx context.Context, id uuid.UUID, reason string) (*domain.Run, error) {
	current, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.Status.IsTerminal() {
		return current, nil
	}
	endedAt := o.now()
	return o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:       current,
		NewStatus: domain.RunStatusCancelled,
		Phase:     domain.RunPhaseCompleted,
		Reason:    reason,
		EndedAt:   &endedAt,
	})
}

func (o *Orchestrator) RecoverRun(ctx context.Context, id uuid.UUID) (*RecoverResult, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}
	if run.ExecutionMode.Normalized() == domain.ExecutionModeImported {
		return nil, importedRunLifecycleError("recover")
	}
	if run.Status.IsTerminal() && run.FinalizationStatus == domain.RunFinalizationStatusFailed {
		return o.recoverFinalization(ctx, run)
	}
	if recovered, err := o.recoverTerminalSandboxLifecycle(ctx, run); recovered != nil || err != nil {
		return recovered, err
	}
	if run.Status.IsTerminal() && run.FinalizationStatus == domain.RunFinalizationStatusSucceeded {
		return &RecoverResult{Run: run, Idempotent: true, Message: "run execution and sandbox finalization are already complete"}, nil
	}
	if o.reconciler == nil {
		return nil, domain.NewConfigMissingError("reconciler", "reconciler not configured", nil)
	}
	return o.reconciler.RecoverRun(ctx, id)
}

// recoverTerminalSandboxLifecycle repairs the historical gap where a terminal
// sandboxed run with autoApply=false never reached the shared finalization seam.
// It only acts when the persisted policy explicitly requests terminal stop or
// delete and cannot apply changes, reopen a manual-review workspace, or resume
// execution. A nil result means the run needs the ordinary recovery path.
func (o *Orchestrator) recoverTerminalSandboxLifecycle(ctx context.Context, run *domain.Run) (*RecoverResult, error) {
	if run == nil || !run.Status.IsTerminal() || run.RunMode != domain.RunModeSandboxed || run.SandboxID == nil || o.sandbox == nil {
		return nil, nil
	}
	if run.Status != domain.RunStatusComplete && run.Status != domain.RunStatusFailed && run.Status != domain.RunStatusCancelled {
		return nil, nil
	}
	if run.FinalizationStatus != "" && run.FinalizationStatus != domain.RunFinalizationStatusNone && run.FinalizationStatus != domain.RunFinalizationStatusSucceeded {
		return nil, nil
	}
	cfg := phases.EffectiveSandboxConfig(run)
	if cfg == nil || cfg.ManualReview {
		return nil, nil
	}
	terminalEvents := []domain.SandboxLifecycleEvent{
		domain.SandboxLifecycleRunCompleted,
		domain.SandboxLifecycleRunFailed,
		domain.SandboxLifecycleRunCancelled,
		domain.SandboxLifecycleTerminal,
	}
	if !phases.HasLifecycleEvent(cfg.Lifecycle.DeleteOn, terminalEvents) && !phases.HasLifecycleEvent(cfg.Lifecycle.StopOn, terminalEvents) {
		return nil, nil
	}
	if run.FinalizationStatus == domain.RunFinalizationStatusSucceeded {
		sandboxState, err := o.sandbox.Get(ctx, *run.SandboxID)
		if err != nil {
			var notFound *domain.NotFoundError
			if errors.As(err, &notFound) {
				return &RecoverResult{Run: run, Idempotent: true, Message: "run execution and sandbox finalization are already complete"}, nil
			}
			return nil, fmt.Errorf("inspect finalized sandbox lifecycle: %w", err)
		}
		if sandboxState == nil {
			return nil, fmt.Errorf("inspect finalized sandbox lifecycle: provider returned no sandbox")
		}
		// Finalization was already successful, but an older process may have
		// crashed between provenance persistence and lifecycle teardown. Re-run
		// only the idempotent lifecycle seam; never re-apply or execute.
		phases.Finalize(phases.FinalizeInput{
			Deps:      phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster, Levers: o.runLevers(), WorkspaceSandbox: o.workspaceSandbox},
			Run:       run,
			SandboxID: run.SandboxID,
			Sandbox:   o.sandbox,
		})
		return &RecoverResult{Run: run, Recovered: true, Message: "reconciled terminal sandbox lifecycle after successful finalization"}, nil
	}
	// Older runs may omit AutoApply from their persisted config even though the
	// contract default is true. Never delete such a sandbox merely because its
	// executor is gone: first prove that the live overlay contains no changes.
	// A non-empty or unavailable diff remains an owner-review case.
	if cfg.GetAutoApply() {
		diff, err := o.sandbox.GetDiff(ctx, *run.SandboxID)
		if err != nil {
			return nil, fmt.Errorf("inspect terminal sandbox diff before recovery: %w", err)
		}
		if !sandboxDiffEmpty(diff) {
			return &RecoverResult{
				Run:        run,
				Idempotent: true,
				Message:    "terminal sandbox retains changes; owner review is required before lifecycle cleanup",
			}, nil
		}
		// An empty diff proves the required apply effect is a successful no-op.
		// Mark it succeeded so the lifecycle dispatcher is allowed to delete;
		// `Skipped` intentionally preserves auto-apply runs for owner recovery.
		domain.MarkFinalizationSucceeded(run, o.now())
		if o.runs != nil {
			if err := o.runs.Update(ctx, run); err != nil {
				return nil, fmt.Errorf("persist empty-sandbox recovery: %w", err)
			}
		}
		phases.Finalize(phases.FinalizeInput{
			Deps:      phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster, Levers: o.runLevers(), WorkspaceSandbox: o.workspaceSandbox},
			Run:       run,
			SandboxID: run.SandboxID,
			Sandbox:   o.sandbox,
		})
		return &RecoverResult{
			Run:       run,
			Recovered: true,
			Message:   "reconciled terminal sandbox with an empty diff without repeating execution",
		}, nil
	}
	o.finalizeSandboxForTerminalRun(ctx, run)
	return &RecoverResult{
		Run:       run,
		Recovered: true,
		Message:   "reconciled terminal sandbox lifecycle without repeating execution",
	}, nil
}

// sandboxDiffEmpty is deliberately conservative. A nil diff is unknown and
// therefore retained; every populated field must agree that no change exists.
func sandboxDiffEmpty(diff *sandbox.DiffResult) bool {
	if diff == nil {
		return false
	}
	return len(diff.Files) == 0 &&
		diff.Stats.FilesChanged == 0 &&
		diff.Stats.FilesAdded == 0 &&
		diff.Stats.FilesModified == 0 &&
		diff.Stats.FilesDeleted == 0 &&
		diff.Stats.TotalLines == 0 &&
		diff.Stats.LinesAdded == 0 &&
		diff.Stats.LinesRemoved == 0 &&
		diff.Stats.TotalBytes == 0 &&
		strings.TrimSpace(diff.UnifiedDiff) == ""
}

// ContinueRun continues an existing run's conversation with a follow-up message.
// The message is appended to the run's event stream and the response is streamed back.
