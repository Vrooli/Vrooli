// This file restores interactive run state during recovery.
package orchestration

import (
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/invocationreadmodel"
	"context"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/orchestration/obs"

	"github.com/google/uuid"
)

// recoverInteractiveRun re-adopts an interactive run (ExecutionMode=interactive)
// after an agent-manager restart or a stale-heartbeat detection. It mirrors the
// codec-pipe recovery trust model (drain the transcript for a terminal marker)
// but resolves liveness from the web-console session rather than a local pgid:
//
//   - a failure terminal already on disk finalizes the run Failed;
//   - otherwise the session decides: gone => (success terminal ? Complete :
//     Failed with an explicit reason so the run is never orphaned); alive =>
//     reattach the interactive tailer from the persisted cursor and let its
//     turn-boundary debounce drive true completion.
//
// When the web-console session controller is not wired, interactive recovery is
// a logged idempotent no-op: it never falsely completes or fails the run.
func (r *Reconciler) recoverInteractiveRun(ctx context.Context, run *domain.Run, allowTail bool) (*RecoverResult, error) {
	recoveryLog := obs.Component("recovery")

	if r.sessions == nil {
		recoveryLog.Warn("interactive recovery not configured; leaving run untouched",
			obs.KeyRunID, run.ID.String())
		return &RecoverResult{Run: run, Idempotent: true, Message: "interactive recovery not configured"}, nil
	}

	// Join an old recovery observer before taking any transcript cut. The
	// continuation owner holds the same mutex while it admits and delivers work.
	if !r.interactiveRecoveryMu.TryLock() {
		return &RecoverResult{Run: run, Idempotent: true, Message: "interactive continuation or recovery owner is in flight"}, nil
	}
	defer r.interactiveRecoveryMu.Unlock()
	if r.interactiveLiveDrivers != nil && r.interactiveLiveDrivers.has(run.ID) {
		return &RecoverResult{Run: run, Idempotent: true, Message: "live interactive driver owns completion"}, nil
	}
	coord := r.interactiveCoordinator()
	if run.ResolvedConfig != nil && strings.TrimSpace(run.ResolvedConfig.Until) != "" {
		joinCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
		hadTailer, joinErr := r.cancelInteractiveTailAndWait(joinCtx, run.ID)
		cancel()
		if joinErr != nil {
			return &RecoverResult{Run: run, Message: "interactive tailer join is pending: " + joinErr.Error()}, nil
		}
		// Tail callbacks may have persisted one final cursor or terminal. Never
		// overwrite that newer state with the pre-cancellation caller snapshot.
		current, err := r.runs.Get(ctx, run.ID)
		if err != nil {
			return nil, err
		}
		run = current
		if run == nil {
			return nil, domain.NewNotFoundErrorWithID("Run", "interactive recovery")
		}
		if !run.Status.LivenessPolicy().ExpectsProcess {
			return &RecoverResult{Run: run, Idempotent: true, Message: "interactive owner already finalized"}, nil
		}
		// Legacy rows predate the atomic invocation boundary. Retained owner start
		// events provide a conservative later cutoff without altering their history.
		inspection := *run
		if inspection.InteractiveInvocationStartedAt == nil && r.events != nil {
			starts, err := r.events.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeStatus}, Limit: 1025})
			if err != nil || len(starts) > 1024 {
				if hadTailer {
					r.startInteractiveTailer(run, coord)
				}
				return &RecoverResult{Run: run, Message: "interactive invocation evidence unavailable or exceeds replay bound"}, nil
			}
			for _, item := range starts {
				if invocationreadmodel.BeginsRunInvocation(item) && !item.Timestamp.IsZero() && (inspection.InteractiveInvocationStartedAt == nil || item.Timestamp.After(*inspection.InteractiveInvocationStartedAt)) {
					stamp := item.Timestamp
					inspection.InteractiveInvocationStartedAt = &stamp
				}
			}
		}
		terminal, inspectErr := coord.InspectRetainedGoal(ctx, &inspection)
		if inspectErr != nil {
			if hadTailer {
				r.startInteractiveTailer(run, coord)
			}
			return &RecoverResult{Run: run, Message: "retained native goal evidence unavailable: " + inspectErr.Error()}, nil
		}
		if terminal != nil {
			if err := coord.Finalize(ctx, run, terminal, nil); err != nil {
				return nil, err
			}
			return &RecoverResult{Run: run, Recovered: true, Message: "reconciled accepted native goal from exact retained session and invocation"}, nil
		}
		// A joined observer owned this run even for a read-triggered recovery call.
		// Preserve that ownership when new work superseded historical completion.
		allowTail = allowTail || hadTailer
	}

	// If a reattached tailer is already driving this run, do not re-drain or
	// restart it every reconcile cycle — it owns completion.
	if r.hasTailer(run.ID) {
		return &RecoverResult{Run: run, Recovered: true, Message: "interactive tail already in flight"}, nil
	}

	// 1. Drain any already-written transcript for a terminal marker, reusing the
	//    codec-pipe drain (same parser seam, same cursor persistence).
	terminal := r.drainInteractiveTranscript(ctx, run)

	// 2. A failure terminal is authoritative regardless of session state.
	if terminal != nil && !terminal.Success {
		return r.finalizeRecoveredRun(ctx, run, terminal)
	}

	// 3. Session liveness decides the rest.
	alive, err := coord.VerifySession(ctx, run)
	if err != nil {
		// Transient RPC failure — leave the run for the next reconcile cycle
		// rather than risk falsely failing a live run.
		recoveryLog.Warn("interactive session check failed",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return &RecoverResult{Run: run, Recovered: false, Message: "interactive session check failed: " + err.Error()}, nil
	}

	if !alive {
		if terminal != nil {
			// Last turn completed, then the session was cleaned up.
			return r.finalizeRecoveredRun(ctx, run, terminal)
		}
		return r.failRecoveredRun(ctx, run, fmt.Sprintf(
			"web-console session %s no longer exists; interactive run cannot be recovered", run.WebConsoleSessionID))
	}

	// 4. Session alive — reattach the tailer to drive true completion.
	if allowTail {
		r.startInteractiveTailer(run, coord)
		return &RecoverResult{Run: run, Recovered: true, Message: fmt.Sprintf(
			"reattached interactive tailer for run %s at offset %d", run.ID, run.TranscriptCursor)}, nil
	}
	return &RecoverResult{Run: run, Recovered: true, Message: "interactive session alive"}, nil
}

// drainInteractiveTranscript drains the run's on-disk transcript from the
// persisted cursor and returns a terminal marker if one is already present.
// Reuses the codec-pipe recovery drain (recoveryParser + drainTranscript) so
// interactive runs share the same parser + cursor-persistence machinery.
func (r *Reconciler) drainInteractiveTranscript(ctx context.Context, run *domain.Run) *runner.TranscriptTerminal {
	if run.TranscriptPath == "" {
		return nil
	}
	parser, transcriptPath, state, err := r.recoveryParser(ctx, run)
	if err != nil || parser == nil {
		return nil
	}
	terminal, err := r.drainTranscript(ctx, run, transcriptPath, state, parser)
	if err != nil {
		return nil
	}
	return terminal
}

// interactiveCoordinator builds a recovery-flavoured interactive coordinator: no
// substrate (reattach never launches), heartbeat disabled (the run is not owned
// live), and the recovery event sink + summary builder wired in.
func (r *Reconciler) interactiveCoordinator() *interactive.Coordinator {
	return interactive.NewCoordinator(interactive.CoordinatorDeps{
		Tailer:      interactive.NewTailer(interactive.RegistryParser(r.runners)),
		Sessions:    r.sessions,
		Runs:        r.runs,
		Broadcaster: r.broadcaster,
		NewSink: func(runID uuid.UUID) runner.EventSink {
			return r.recoveryEventSink(runID)
		},
		Result:                r.recoveredResult,
		Heartbeat:             -1,
		Debounce:              r.interactiveDebounce,
		SessionPoll:           r.interactiveSessionPoll,
		SessionReattachWindow: r.interactiveSessionReattachWindow,
	})
}

func (r *Reconciler) recoveredResult(ctx context.Context, runID uuid.UUID, success bool, exitCode int, terminalReason string) (*domain.RunResult, *domain.RunSummary) {
	result, summary, err := r.buildRecoveredResult(ctx, runID, success, exitCode, terminalReason)
	if err != nil {
		return nil, nil
	}
	return result, summary
}

// startInteractiveTailer reattaches an interactive tailer to a live run in the
// background, mirroring startTailer's dedupe-by-cancel bookkeeping. The tailer
// drives the run to completion (terminal marker + turn-boundary debounce) and
// finalizes it.
func (r *Reconciler) startInteractiveTailer(run *domain.Run, coord *interactive.Coordinator) {
	r.recoveryMu.Lock()
	if cancel, ok := r.tailers[run.ID]; ok {
		cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	if r.interactiveTailDone == nil {
		r.interactiveTailDone = make(map[uuid.UUID]chan struct{})
	}
	r.interactiveTailDone[run.ID] = done
	r.tailers[run.ID] = cancel
	if r.tailerOwners == nil {
		r.tailerOwners = make(map[uuid.UUID]context.Context)
	}
	r.tailerOwners[run.ID] = ctx
	r.recoveryMu.Unlock()

	runCopy := *run
	run = &runCopy
	go func() {
		defer close(done)
		defer func() {
			r.recoveryMu.Lock()
			if r.tailerOwners[run.ID] == ctx {
				delete(r.tailers, run.ID)
				delete(r.tailerOwners, run.ID)
				delete(r.interactiveTailDone, run.ID)
			}
			r.recoveryMu.Unlock()
		}()
		// Log-only containment: the session may still be healthy, so the run
		// must not be failed here; the stale sweep is the backstop.
		defer obs.RecoverToFailure("interactive reattach tailer", nil)
		terminal, tailErr := coord.TailToCompletion(ctx, run)
		if err := coord.Finalize(ctx, run, terminal, tailErr); err != nil {
			obs.Component("recovery").Warn("interactive reattach finalize failed",
				obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		}
	}()
}

// hasTailer reports whether a transcript tailer (codec-pipe or interactive) is
// currently attached to the run.
func (r *Reconciler) hasTailer(runID uuid.UUID) bool {
	r.recoveryMu.Lock()
	defer r.recoveryMu.Unlock()
	_, ok := r.tailers[runID]
	return ok
}

// CancelInteractiveTail cancels and deregisters a recovery-reattached tailer for
// a run, joining its final callbacks before returning. It lets StopRun tear down a run whose interactive
// tailer was reattached by restart recovery (rather than the live Execute path).
// Returns whether a tailer was present. The cancelled tailer's coordinator
// observes context cancellation and finalizes as a no-op, leaving StopRun to
// write the terminal status.
func (r *Reconciler) CancelInteractiveTail(runID uuid.UUID) bool {
	present, _ := r.cancelInteractiveTailAndWait(context.Background(), runID)
	return present
}

// Cancellation alone is not an ownership transfer: onAdvance can still be
// writing under context.Background. Retain registration until the owner joins.
func (r *Reconciler) cancelInteractiveTailAndWait(ctx context.Context, runID uuid.UUID) (bool, error) {
	r.recoveryMu.Lock()
	cancel, ok := r.tailers[runID]
	done, owner := r.interactiveTailDone[runID], r.tailerOwners[runID]
	r.recoveryMu.Unlock()
	if !ok {
		return false, nil
	}
	cancel()
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return true, ctx.Err()
		}
	}
	r.recoveryMu.Lock()
	if r.tailerOwners[runID] == owner {
		delete(r.tailers, runID)
		delete(r.tailerOwners, runID)
		delete(r.interactiveTailDone, runID)
	}
	r.recoveryMu.Unlock()
	return true, nil
}
