// This file implements interactive controls for active agent runs.
package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

// interactiveDriver is a live interactive coordinator agent-manager owns: either
// the initial Execute turn or a Continue-driven follow-up turn. Its context can
// be cancelled to stop the tail, and done is closed once the coordinator
// goroutine has fully exited (so a waiting Stop knows no further tail/heartbeat
// Update can land).
type interactiveDriver struct {
	cancel context.CancelFunc
	done   chan struct{}
}

// interactiveDriverRegistry tracks live interactive drivers by run id. It is the
// coordination seam that lets StopRun cancel a live coordinator and wait for it
// to exit before finalizing — without it, a late tail cursor Update (which uses
// context.Background) could resurrect a just-Cancelled run.
type interactiveDriverRegistry struct {
	mu sync.Mutex
	m  map[uuid.UUID]*interactiveDriver
}

func newInteractiveDriverRegistry() *interactiveDriverRegistry {
	return &interactiveDriverRegistry{m: map[uuid.UUID]*interactiveDriver{}}
}

// register derives a cancellable context from parent, stores a fresh driver
// under runID, and returns the child context + driver. Any pre-existing driver
// for the same run is cancelled first (defensive; callers already guard against
// concurrent drivers via the terminal-state gate).
func (r *interactiveDriverRegistry) register(parent context.Context, runID uuid.UUID) (context.Context, *interactiveDriver) {
	ctx, cancel := context.WithCancel(parent)
	d := &interactiveDriver{cancel: cancel, done: make(chan struct{})}
	r.mu.Lock()
	if prev, ok := r.m[runID]; ok {
		prev.cancel()
	}
	r.m[runID] = d
	r.mu.Unlock()
	return ctx, d
}

// finish removes d (if it is still the registered driver for runID) and signals
// its done channel so a waiting Stop can proceed. Idempotent; safe to defer.
func (r *interactiveDriverRegistry) finish(runID uuid.UUID, d *interactiveDriver) {
	r.mu.Lock()
	if cur, ok := r.m[runID]; ok && cur == d {
		delete(r.m, runID)
	}
	r.mu.Unlock()
	d.cancel()
	select {
	case <-d.done:
	default:
		close(d.done)
	}
}

// has reports whether a live driver is currently registered for runID.
func (r *interactiveDriverRegistry) has(runID uuid.UUID) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.m[runID]
	return ok
}

// cancelAndWait cancels the live driver for runID (if any) and blocks until its
// coordinator goroutine has fully exited. Returns false when no driver was
// registered. This is the deterministic hand-off that guarantees no further
// coordinator Update lands after Stop finalizes the run.
func (r *interactiveDriverRegistry) cancelAndWait(runID uuid.UUID) bool {
	r.mu.Lock()
	d, ok := r.m[runID]
	r.mu.Unlock()
	if !ok {
		return false
	}
	d.cancel()
	<-d.done
	return true
}

// revokeInteractiveRunIdentity loads and durably revokes a run's currently
// active identity. No-op when no token exists or it was already revoked.
func (o *Orchestrator) revokeInteractiveRunIdentity(ctx context.Context, runID uuid.UUID) error {
	if o.runs == nil {
		return fmt.Errorf("run repository is not configured")
	}
	run, err := o.runs.Get(ctx, runID)
	if err != nil {
		return err
	}
	if run == nil {
		return domain.NewNotFoundError("Run", runID)
	}
	if run.IdentityTokenHash == "" || run.IdentityTokenRevokedAt != nil {
		return nil
	}
	now := o.now()
	run.IdentityTokenRevokedAt = &now
	run.UpdatedAt = now
	return o.runs.Update(ctx, run)
}

// stopInteractiveRun stops an ExecutionMode=interactive run. The CLI lives in a
// web-console tmux session, so there is no local process to signal: Stop cancels
// the live coordinator (and waits for it to exit), tears the session down via the
// interrupt-then-archive escalation ladder, then finalizes the run Cancelled —
// mirroring pipe-mode StopRun's terminal status. Finalization happens exactly
// once: the coordinator has been drained before the transition, and a natural
// completion that won the race leaves the run already terminal (Stop is then a
// no-op).
func (o *Orchestrator) stopInteractiveRun(ctx context.Context, run *domain.Run) error {
	// 1. Cancel + drain any live coordinator (Execute or Continue turn) so no
	//    late tail cursor / heartbeat Update can resurrect the run after we
	//    finalize it. Also cancel a recovery-reattached tailer (restart path).
	o.interactiveDrivers.cancelAndWait(run.ID)
	if o.reconciler != nil {
		o.reconciler.CancelInteractiveTail(run.ID)
	}
	// A stopped PTY is a true session retirement even when its run had already
	// reached a resumable terminal turn. Revoke the child bearer durably before
	// teardown; a failed write prevents Stop from claiming retirement.
	if err := o.revokeInteractiveRunIdentity(ctx, run.ID); err != nil {
		return fmt.Errorf("revoke interactive run identity before stop: %w", err)
	}

	// 2. Hard-stop the web-console session: interrupt (soft) then archive (hard).
	//    Only a successful archive (or the adapter's typed already-gone result)
	//    proves retirement. Keep the run retryable and its child identity revoked
	//    when archive fails; do not clean credentials or finalize it as stopped.
	if run.WebConsoleSessionID != "" && o.interactiveSessions != nil {
		sub := interactive.NewSubstrate(o.interactiveSessions, interactive.RegistryLaunchInfo(o.runners))
		if err := sub.Stop(ctx, run.WebConsoleSessionID, interactiveRunSource(run.ID)); err != nil {
			return fmt.Errorf("stop interactive web-console session: %w", err)
		}
		// The session is closing, so any human-session delegation grant minted for
		// it is revoked now — a closed session must confer no further authority
		// (the TTL cap is the backstop; this is the prompt revocation). Non-fatal.
		if owner := humanSpawnOwner(run); owner != "" {
			if err := o.RevokeSessionGrants(ctx, owner, run.WebConsoleSessionID); err != nil {
				obs.Component("interactive").Warn("interactive stop: session grant revoke failed",
					obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
			}
		}
		// The session (and its CLI) is now dead, so the run's private copy of the
		// shared home's credentials can be removed instead of lingering on disk.
		if run.ResolvedConfig != nil {
			root, rootErr := o.resolveRunStateRoot(ctx)
			if rootErr != nil {
				return rootErr
			}
			runDir, rootErr := runstate.RunDir(root, run.ID)
			if rootErr != nil {
				return rootErr
			}
			if err := sub.CleanupCredentials(run.ResolvedConfig.RunnerType, runDir); err != nil {
				obs.Component("interactive").Warn("interactive stop: seeded credential cleanup failed",
					obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
			}
		}
	}

	// 3. Finalize deterministically. Re-read to catch a natural completion that
	//    beat the Stop (coordinator finalized before cancellation) — Stop is then
	//    a no-op on the already-terminal run.
	current, err := o.runs.Get(ctx, run.ID)
	if err != nil {
		return err
	}
	if current == nil {
		return domain.NewNotFoundError("Run", run.ID)
	}
	if current.Status.IsTerminal() {
		return nil
	}
	endedAt := o.now()
	_, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:       current,
		NewStatus: domain.RunStatusCancelled,
		Phase:     domain.RunPhaseCompleted,
		Reason:    "Interactive run stopped by request",
		EndedAt:   &endedAt,
	})
	if err == nil {
		o.finalizeSandboxForTerminalRun(ctx, current)
	}
	return err
}

// finalizeSandboxForTerminalRun applies terminal provenance and then dispatches
// the sandbox lifecycle policy. Natural completion, operator stop and terminal
// recovery all pass through this seam so delete-on-terminal sandboxes cannot
// remain active after their owner has ended.
func (o *Orchestrator) finalizeSandboxForTerminalRun(ctx context.Context, run *domain.Run) {
	if run == nil || !run.Status.IsTerminal() || run.RunMode != domain.RunModeSandboxed || run.SandboxID == nil || o.sandbox == nil {
		return
	}
	outcome := domain.ContractRunOutcomeSuccess
	switch run.Status {
	case domain.RunStatusFailed:
		outcome = domain.ContractRunOutcomeFailure
	case domain.RunStatusCancelled:
		outcome = domain.ContractRunOutcomeCancelled
	}
	deps := phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster, Levers: o.runLevers(), WorkspaceSandbox: o.workspaceSandbox, ApprovalNotifier: o}
	phases.ApplyAtRunEnd(ctx, phases.ApplyAtRunEndInput{Deps: deps, Run: run, SandboxID: run.SandboxID, Sandbox: o.sandbox, Outcome: outcome})
	if o.runs != nil {
		if err := o.runs.Update(ctx, run); err != nil {
			obs.Component("interactive").Warn("interactive terminal provenance persistence failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		}
	}
	phases.Finalize(phases.FinalizeInput{Deps: deps, Run: run, SandboxID: run.SandboxID, Sandbox: o.sandbox})
}

// continueInteractiveRun continues an ExecutionMode=interactive run by typing the
// follow-up prompt into the still-live web-console session and reattaching a
// tailer to drive the new turn — never a process respawn (locked decision 6).
// The caller (ContinueRun) has already run the CanContinueRun gate, which admits
// only terminal runs (a mid-turn Continue is rejected there, mirroring pipe-mode).
// The web-console session is kept alive after completion (Phase 4), so the
// follow-up lands in the same CLI; if the session is gone, the run cannot be
// continued in place and an actionable error is returned.
func (o *Orchestrator) continueInteractiveRun(ctx context.Context, run *domain.Run, message string, _ []string, reinstallGoal bool) (*domain.Run, error) {
	if err := o.verifyInteractiveSession(ctx, run); err != nil {
		return nil, domain.RefuseBeforeEffects(err)
	}
	tailContext, err := loadInteractiveTailContextForRun(ctx, run, o.resolveRunStateRoot)
	if err != nil {
		obs.Component("interactive").Warn("interactive continuation could not load run-state metadata",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
	if run.ResolvedConfig != nil && run.ResolvedConfig.RunnerType == domain.RunnerTypeCodex && strings.TrimSpace(tailContext.RunDir) == "" {
		return nil, domain.RefuseBeforeEffects(err)
	}
	return o.continueInteractiveRunWithVerifiedSession(ctx, run, message, reinstallGoal, tailContext)
}

// verifyInteractiveSession proves the exact retained session is available
// before a caller cancels a waiter or changes the Run. Wake uses this before
// its durable await bookkeeping; Continue uses it through
// continueInteractiveRun.
func (o *Orchestrator) verifyInteractiveSession(ctx context.Context, run *domain.Run) error {
	if run == nil {
		return domain.NewValidationError("run", "interactive run is required")
	}
	if o.interactiveSessions == nil {
		return domain.NewConfigMissingError("interactiveSessions", "interactive execution mode is not configured", nil)
	}
	if run.WebConsoleSessionID == "" {
		return domain.NewStateError("Run", string(run.Status), "continue",
			"interactive run has no web-console session to continue")
	}
	session, err := o.interactiveSessions.GetSession(ctx, run.WebConsoleSessionID)
	if err != nil {
		if errors.Is(err, webconsole.ErrSessionNotFound) {
			return domain.NewStateError("Run", string(run.Status), "continue",
				fmt.Sprintf("the live web-console session %s for this run no longer exists (ended runs keep their session only for the interactive session retention window); start a new interactive run to continue",
					run.WebConsoleSessionID))
		}
		return err
	}
	if session.ID != run.WebConsoleSessionID {
		return domain.NewStateError("Run", string(run.Status), "continue",
			fmt.Sprintf("web-console lookup returned session %s for retained session %s", session.ID, run.WebConsoleSessionID))
	}
	return nil
}

func (o *Orchestrator) continueInteractiveRunWithVerifiedSession(ctx context.Context, run *domain.Run, message string, reinstallGoal bool, tailContext interactive.TailContext) (*domain.Run, error) {
	completeAfterFirstTurn := shouldCompleteAfterTerminalGoalContinuation(run, reinstallGoal)
	// Defensive: a live driver means a turn is still in flight. ContinueRun's
	// terminal-state gate and WakeRun's cancel-and-drain handoff normally ensure
	// no driver remains here; retain this guard for racing callers.
	if o.interactiveDrivers.has(run.ID) {
		return nil, domain.RefuseBeforeEffects(domain.NewStateError("Run", string(run.Status), "continue",
			"an interactive turn is already in progress for this run"))
	}

	// Reactivate the run (reset the heartbeat in the same transition) and record
	// the follow-up as a user message, mirroring pipe-mode resumeConversation.
	now := o.now()
	transcriptStartCursor := run.TranscriptCursor
	priorInvocationStartedAt := run.InteractiveInvocationStartedAt
	var statusPublicationErr error
	run.InteractiveInvocationStartedAt = &now
	// Persist this exception with the invocation boundary itself so recovery
	// can apply it only to this exact admitted continuation after a restart.
	if run.ResolvedConfig != nil {
		run.ResolvedConfig.InteractiveContinuation = nil
		if completeAfterFirstTurn {
			run.ResolvedConfig.InteractiveContinuation = &domain.InteractiveContinuationPolicy{
				InvocationStartedAt:                               now,
				TranscriptStartCursor:                             &transcriptStartCursor,
				CompleteAfterFirstSuccessfulTurnWithoutGoalMarker: true,
			}
		}
	}
	run, err := o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:                                    run,
		NewStatus:                              domain.RunStatusRunning,
		Phase:                                  domain.RunPhaseExecuting,
		Reason:                                 "Interactive continuation requested",
		LastHeartbeat:                          &now,
		InteractiveContinuation:                true,
		ExpectedInteractiveInvocationStartedAt: priorInvocationStartedAt,
	})
	if err != nil {
		if run == nil {
			return nil, err
		}
		statusPublicationErr = err
		obs.Component("interactive").Error("interactive continuation status committed but event publication failed; proceeding with the retained session",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
	if o.events != nil {
		if aerr := o.appendAndBroadcastEvents(ctx, run.ID, domain.NewMessageEvent(run.ID, "user", message)); aerr != nil {
			_ = aerr
		}
	}

	// Reinstall the harness-native goal in the same session before the follow-up
	// when asked, so a resumed run still has its finish line installed.
	if reinstallGoal {
		if err := o.reinstallNativeGoal(ctx, run); err != nil {
			endedAt := o.now()
			if _, terr := o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
				Run:       run,
				NewStatus: domain.RunStatusFailed,
				Phase:     domain.RunPhaseCompleted,
				Reason:    "Interactive continuation goal reinstall failed",
				EndedAt:   &endedAt,
				ErrorMsg:  err.Error(),
			}); terr != nil {
				obs.Component("interactive").Warn("interactive continue: failed to finalize after goal reinstall error",
					obs.KeyRunID, run.ID.String(), obs.KeyError, terr.Error())
			}
			return nil, domain.NewInternalError("failed to reinstall interactive goal", err)
		}
	}

	// Type the follow-up into the live session (paste + Enter submit). On failure
	// finalize the run Failed rather than leaving it stuck Running.
	submission, err := o.interactiveSessions.SendPrompt(ctx, run.WebConsoleSessionID, message, interactiveRunSource(run.ID))
	if err != nil {
		endedAt := o.now()
		if _, terr := o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
			Run:       run,
			NewStatus: domain.RunStatusFailed,
			Phase:     domain.RunPhaseCompleted,
			Reason:    "Interactive continuation prompt delivery failed",
			EndedAt:   &endedAt,
			ErrorMsg:  err.Error(),
		}); terr != nil {
			obs.Component("interactive").Warn("interactive continue: failed to finalize after send error",
				obs.KeyRunID, run.ID.String(), obs.KeyError, terr.Error())
		}
		return nil, domain.NewInternalError("failed to deliver interactive continuation prompt", err)
	}
	if !submission.Verified {
		o.appendRunLogEvent(ctx, run.ID, "warn", fmt.Sprintf("continuation typed into web-console session %s; submission unverified (the composer could not be observed)", run.WebConsoleSessionID))
	}

	// Reattach a live coordinator to drive the new turn to completion. Its
	// turn-boundary debounce closes the follow-up turn exactly like the initial
	// Execute turn.
	o.driveInteractiveContinuation(run, completeAfterFirstTurn, tailContext)

	return o.attachRunActions(ctx, run), statusPublicationErr
}

// shouldCompleteAfterTerminalGoalContinuation scopes the one-turn terminal
// rule to a terminal native goal when the caller did not reinstall it. A
// follow-up can report on a completed or blocked goal without changing that
// historical verdict; a reinstalled goal keeps the normal goal-verdict
// contract.
func shouldCompleteAfterTerminalGoalContinuation(run *domain.Run, reinstallGoal bool) bool {
	return !reinstallGoal &&
		run != nil &&
		run.Status == domain.RunStatusComplete &&
		run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive &&
		run.ResolvedConfig != nil &&
		strings.TrimSpace(run.ResolvedConfig.Until) != "" &&
		(run.ObservedGoalStatus == string(runner.GoalStatusComplete) || run.ObservedGoalStatus == string(runner.GoalStatusBlocked))
}

// reinstallNativeGoal re-sends the harness-native /goal objective into a live
// interactive session. It is a no-op when the run has no completion objective,
// the runner declares no native objective for the run's sandbox, or the session
// controller is not configured. The objective is single-lined because native
// slash commands must be typed as literal keystrokes and explicitly submitted.
func (o *Orchestrator) reinstallNativeGoal(ctx context.Context, run *domain.Run) error {
	if run == nil || run.ResolvedConfig == nil || strings.TrimSpace(run.ResolvedConfig.Until) == "" {
		return nil
	}
	if o.interactiveSessions == nil {
		return domain.NewConfigMissingError("interactiveSessions", "interactive execution mode is not configured", nil)
	}
	selected, _ := o.runners.Get(run.ResolvedConfig.RunnerType)
	if selected == nil {
		return nil
	}
	objective := nativeObjectiveFor(run, selected.Capabilities())
	if strings.TrimSpace(objective) == "" {
		return nil
	}
	command := "/goal " + runner.NativeGoalObjective(objective)
	submission, err := o.interactiveSessions.SendCommand(ctx, run.WebConsoleSessionID, command, interactiveRunSource(run.ID))
	if err != nil {
		// SendCommand may have typed the command or pressed Enter before the
		// transport failed. The caller must withhold the follow-up and leave its
		// idempotency receipt pending for owner reconciliation.
		return fmt.Errorf("native goal command submission outcome is uncertain: %w", err)
	}
	if !submission.Verified {
		return fmt.Errorf("native goal command submission was unverified after %d Enter presses; follow-up withheld", submission.EnterPresses)
	}
	return nil
}

// driveInteractiveContinuation reattaches a live interactive coordinator to a run
// whose follow-up prompt has just been typed in, and drives the new turn to
// completion in the background. It borrows the same TailToCompletion + Finalize
// seams as the initial Execute turn and restart recovery, registered in the
// interactive driver registry so StopRun can cancel it deterministically. The
// coordinator mutates its own copy of the run; the persisted DB row is the
// source of truth.
func (o *Orchestrator) driveInteractiveContinuation(run *domain.Run, completeAfterFirstTurn bool, tailContext interactive.TailContext) {
	coord := interactive.NewCoordinator(interactive.CoordinatorDeps{
		Tailer:      interactive.NewTailer(interactive.RegistryParser(o.runners)),
		Sessions:    o.interactiveSessions,
		Runs:        o.runs,
		Broadcaster: o.broadcaster,
		NewSink:     o.interactiveEventSink,
		Result:      o.persistedResultBuilder,
	})

	runCopy := *run
	ctx, driver := o.interactiveDrivers.register(context.Background(), run.ID)
	go func() {
		defer o.interactiveDrivers.finish(run.ID, driver)
		defer obs.RecoverToFailure("interactive continuation driver", func(failure obs.PanicFailure) {
			o.recoverPanickedRun(&runCopy, failure)
		})
		if err := coord.TailAndFinalizeLive(ctx, &runCopy, tailContext, interactive.TailCompletionOptions{
			CompleteAfterFirstSuccessfulTurnWithoutGoalMarker: completeAfterFirstTurn,
		}); err != nil {
			obs.Component("interactive").Warn("interactive continuation finalize failed",
				obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		}
	}()
}

// interactiveRunSource is the diagnostic SendInput source attribution for a run's
// interactive session traffic (locked decision 4 — attribution only, no lease).
func interactiveRunSource(runID uuid.UUID) string {
	return "agent-manager:run-" + runID.String()
}
