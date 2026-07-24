package orchestration

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/orchestration/obs"
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

// stopInteractiveRun stops an ExecutionMode=interactive run. The CLI lives in a
// web-console tmux session, so there is no local process to signal: Stop cancels
// the live coordinator (and waits for it to exit), tears the session down via the
// interrupt-then-delete escalation ladder, then finalizes the run Cancelled —
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

	// 2. Hard-stop the web-console session: interrupt (soft) then delete (hard),
	//    idempotent. A teardown error is non-fatal — the session may already be
	//    gone — so we still finalize the run.
	if run.WebConsoleSessionID != "" && o.interactiveSessions != nil {
		sub := interactive.NewSubstrate(o.interactiveSessions, interactive.RegistryLaunchInfo(o.runners))
		if err := sub.Stop(ctx, run.WebConsoleSessionID, interactiveRunSource(run.ID)); err != nil {
			obs.Component("interactive").Warn("interactive stop: session teardown failed",
				obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		}
		// The session (and its CLI) is now dead, so the run's private copy of the
		// shared home's credentials can be removed instead of lingering on disk.
		if run.ResolvedConfig != nil {
			if err := sub.CleanupCredentials(run.ResolvedConfig.RunnerType, runstate.RunDir("", run.ID)); err != nil {
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
	return err
}

// continueInteractiveRun continues an ExecutionMode=interactive run by typing the
// follow-up prompt into the still-live web-console session and reattaching a
// tailer to drive the new turn — never a process respawn (locked decision 6).
// The caller (ContinueRun) has already run the CanContinueRun gate, which admits
// only terminal runs (a mid-turn Continue is rejected there, mirroring pipe-mode).
// The web-console session is kept alive after completion (Phase 4), so the
// follow-up lands in the same CLI; if the session is gone, the run cannot be
// continued in place and an actionable error is returned.
func (o *Orchestrator) continueInteractiveRun(ctx context.Context, run *domain.Run, message string, _ []string) (*domain.Run, error) {
	if o.interactiveSessions == nil {
		return nil, domain.NewConfigMissingError("interactiveSessions", "interactive execution mode is not configured", nil)
	}
	if run.WebConsoleSessionID == "" {
		return nil, domain.NewStateError("Run", string(run.Status), "continue",
			"interactive run has no web-console session to continue")
	}
	// Defensive: a live driver means a turn is still in flight. CanContinueRun
	// already rejects non-terminal runs, so this only fires on a racing
	// double-continue.
	if o.interactiveDrivers.has(run.ID) {
		return nil, domain.NewStateError("Run", string(run.Status), "continue",
			"an interactive turn is already in progress for this run")
	}
	// The live session must still exist to type into.
	if _, err := o.interactiveSessions.GetSession(ctx, run.WebConsoleSessionID); err != nil {
		if errors.Is(err, webconsole.ErrSessionNotFound) {
			return nil, domain.NewStateError("Run", string(run.Status), "continue",
				fmt.Sprintf("the live web-console session %s for this run no longer exists; start a new interactive run to continue",
					run.WebConsoleSessionID))
		}
		return nil, err
	}

	// Reactivate the run (reset the heartbeat in the same transition) and record
	// the follow-up as a user message, mirroring pipe-mode resumeConversation.
	now := o.now()
	run, err := o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:           run,
		NewStatus:     domain.RunStatusRunning,
		Phase:         domain.RunPhaseExecuting,
		Reason:        "Interactive continuation requested",
		LastHeartbeat: &now,
	})
	if err != nil {
		return nil, err
	}
	if o.events != nil {
		if aerr := o.appendAndBroadcastEvents(ctx, run.ID, domain.NewMessageEvent(run.ID, "user", message)); aerr != nil {
			_ = aerr
		}
	}

	// Type the follow-up into the live session (paste + Enter submit). On failure
	// finalize the run Failed rather than leaving it stuck Running.
	if err := o.interactiveSessions.SendPrompt(ctx, run.WebConsoleSessionID, message, interactiveRunSource(run.ID)); err != nil {
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

	// Reattach a live coordinator to drive the new turn to completion. Its
	// turn-boundary debounce closes the follow-up turn exactly like the initial
	// Execute turn.
	o.driveInteractiveContinuation(run)

	return o.attachRunActions(ctx, run), nil
}

// driveInteractiveContinuation reattaches a live interactive coordinator to a run
// whose follow-up prompt has just been typed in, and drives the new turn to
// completion in the background. It borrows the same TailToCompletion + Finalize
// seams as the initial Execute turn and restart recovery, registered in the
// interactive driver registry so StopRun can cancel it deterministically. The
// coordinator mutates its own copy of the run; the persisted DB row is the
// source of truth.
func (o *Orchestrator) driveInteractiveContinuation(run *domain.Run) {
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
		terminal, tailErr := coord.TailToCompletion(ctx, &runCopy)
		if err := coord.Finalize(ctx, &runCopy, terminal, tailErr); err != nil {
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
