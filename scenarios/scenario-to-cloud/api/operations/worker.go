package operations

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/execplan"
	"scenario-to-cloud/faultinject"
	"scenario-to-cloud/persistence"
)

// worker is one ownership attempt over one operation. It holds the fence it
// acquired; every durable write presents that fence and a stale worker
// (superseded by a successor) fails on the first write after losing it.
type worker struct {
	svc     *Service
	op      *domain.CloudOperation
	fence   uint64
	options ExecuteOptions
	resumed bool

	plan      *execplan.Plan
	lost      chan struct{} // closed when a fenced write is refused
	lostOnce  func()
	committed int
}

func (w *worker) run(ctx context.Context) {
	w.lost = make(chan struct{})
	var once bool
	w.lostOnce = func() {
		if !once {
			once = true
			close(w.lost)
		}
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go w.heartbeat(ctx, cancel)

	defer func() {
		if r := recover(); r != nil {
			// A crash mid-step is the classic owner-death case. The record
			// keeps its lease, receipts and active-step marker; nothing
			// terminal is written and a successor reconciles from receipts.
			w.svc.log("operation worker crashed", map[string]interface{}{
				"operation_id": w.op.ID, "fence": w.fence, "panic": fmt.Sprint(r), "stack": string(debug.Stack()),
			})
		}
	}()

	plan, err := DecodePlan(w.op)
	if err != nil {
		w.finish(ctx, Failed, &Result{Outcome: string(Failed), RecoveryOutcome: RecoveryNotAttempted, Message: err.Error()}, apierrors.As(err))
		return
	}
	w.plan = plan
	if receipts, _ := w.op.Receipts(); len(receipts) > 0 {
		w.resumed = true
	}
	// Enter running from reconciling/cancel-requested resumption states.
	switch w.op.State {
	case Reconciling:
		if err := w.setState(ctx, Reconciling, Running, persistence.StatePatch{}); err != nil {
			return
		}
	case Running, CancelRequested:
	default:
		w.svc.log("operation in unexpected state after acquisition", map[string]interface{}{"operation_id": w.op.ID, "state": string(w.op.State)})
		return
	}
	if w.op.CancelRequested {
		// Cancelled before any effect of this attempt: honour it now.
		if w.op.State == CancelRequested {
			w.finish(ctx, Cancelled, &Result{Outcome: string(Cancelled), CompletedSteps: w.completedCount(), Message: "cancelled before resuming"}, nil)
			return
		}
	}
	if w.svc.proj != nil && !w.resumed {
		w.svc.proj.OperationStarted(ctx, w.op, plan)
	}
	w.publish("", "attempt started")

	execErr := w.svc.runner.Execute(ctx, &ExecutionContext{Operation: w.op, Plan: plan, Fence: w.fence, Steps: w, Options: w.options, Resumed: w.resumed})
	select {
	case <-w.lost:
		w.svc.log("operation attempt ended after losing the fence; no terminal write", map[string]interface{}{"operation_id": w.op.ID, "fence": w.fence})
		return
	default:
	}
	w.conclude(ctx, execErr)
}

// conclude maps the runner's outcome onto the state machine.
func (w *worker) conclude(ctx context.Context, execErr error) {
	current, err := w.svc.repo.GetOperation(ctx, w.op.ID)
	if err != nil || current == nil {
		return
	}
	if current.Fence != w.fence {
		w.lostOnce()
		return
	}
	w.op = current
	completed := w.completedCount()
	switch {
	case execErr == nil:
		from := w.op.State
		if from == CancelRequested {
			// Every effect finished before a cancel point was reached; the
			// safe boundary is the end of the plan and verification decides.
			if err := w.setState(ctx, CancelRequested, Verifying, persistence.StatePatch{}); err != nil {
				return
			}
		} else if err := w.setState(ctx, Running, Verifying, persistence.StatePatch{}); err != nil {
			return
		}
		w.finish(ctx, Succeeded, &Result{Outcome: string(Succeeded), CompletedSteps: completed, Message: "all actions verified"}, nil)
	case errors.Is(execErr, ErrCancelled):
		w.finish(ctx, Cancelled, &Result{Outcome: string(Cancelled), CompletedSteps: completed, Message: execErr.Error()}, nil)
	case errors.Is(execErr, ErrWaitingInput):
		typed := apierrors.As(execErr)
		if typed == nil {
			typed = apierrors.New(apierrors.CodeNeedsInput, execErr.Error())
		}
		w.transitionTo(ctx, WaitingInput, persistence.StatePatch{Error: typed})
	case errors.Is(execErr, ErrUnknownEffect):
		typed := apierrors.As(execErr)
		if typed == nil {
			typed = apierrors.New(apierrors.CodeReachUnavailable, execErr.Error())
		}
		w.transitionTo(ctx, Reconciling, persistence.StatePatch{Error: typed, ReleaseLease: true})
	case errors.Is(execErr, ErrRecoveryRequired):
		typed := apierrors.As(execErr)
		if typed == nil {
			typed = apierrors.New(apierrors.CodeInternal, execErr.Error())
		}
		if err := w.transitionTo(ctx, Recovering, persistence.StatePatch{Error: typed}); err != nil {
			return
		}
		// No automatic recovery owner is wired: recovery is unsuccessful by
		// construction and must never read as a deployed service.
		w.finish(ctx, FailedRecovery, &Result{Outcome: string(FailedRecovery), RecoveryOutcome: RecoveryFailed, CompletedSteps: completed, Message: "no recovery owner available: " + execErr.Error()}, typed)
	case errors.Is(execErr, context.DeadlineExceeded), errors.Is(execErr, context.Canceled):
		// Execution bound reached or the pool stopped. Ownership lapses with
		// the lease; the record stays running and is resumed from receipts.
		w.svc.log("operation attempt bound reached; record left for reconciliation", map[string]interface{}{"operation_id": w.op.ID, "fence": w.fence, "error": execErr.Error()})
	default:
		typed := apierrors.As(execErr)
		if typed == nil {
			typed = apierrors.New(apierrors.CodeInternal, execErr.Error())
		}
		w.finish(ctx, Failed, &Result{Outcome: string(Failed), RecoveryOutcome: RecoveryNotAttempted, CompletedSteps: completed, Message: execErr.Error()}, typed)
	}
}

// transitionTo moves from the current durable state to `to` when legal.
func (w *worker) transitionTo(ctx context.Context, to State, patch persistence.StatePatch) error {
	current, err := w.svc.repo.GetOperation(ctx, w.op.ID)
	if err != nil || current == nil {
		return fmt.Errorf("reload operation: %v", err)
	}
	if current.Fence != w.fence {
		w.lostOnce()
		return fenceLost(current, w.fence)
	}
	w.op = current
	return w.setState(ctx, current.State, to, patch)
}

func (w *worker) setState(ctx context.Context, from, to State, patch persistence.StatePatch) error {
	if err := Transition(from, to); err != nil {
		w.svc.log("illegal operation transition refused", map[string]interface{}{"operation_id": w.op.ID, "from": string(from), "to": string(to)})
		return err
	}
	if err := w.svc.repo.SetState(ctx, w.op.ID, w.fence, from, to, patch); err != nil {
		if apierrors.Is(err, apierrors.CodeFenceStale) {
			w.lostOnce()
		}
		w.svc.log("operation state write refused", map[string]interface{}{"operation_id": w.op.ID, "from": string(from), "to": string(to), "error": err.Error()})
		return err
	}
	w.op.State = to
	w.publish("", string(to))
	return nil
}

// finish writes the terminal state, projects it and wakes waiters.
func (w *worker) finish(ctx context.Context, to State, result *Result, failure *apierrors.Error) {
	if err := w.transitionTo(ctx, to, persistence.StatePatch{Result: result, Error: failure}); err != nil {
		return
	}
	if w.svc.proj != nil {
		w.svc.proj.OperationFinished(ctx, w.op, w.plan, to, result, failure)
	}
	w.svc.coord.SignalTerminal(w.op.ID)
}

func (w *worker) completedCount() int {
	receipts, _ := w.op.Receipts()
	n := 0
	for _, r := range receipts {
		if r.Outcome == StepSucceeded || r.Outcome == StepUnchanged {
			n++
		}
	}
	return n
}

func (w *worker) heartbeat(ctx context.Context, cancel context.CancelFunc) {
	ticker := time.NewTicker(w.svc.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if w.svc.heartbeatsPaused.Load() {
				continue
			}
			if err := w.svc.repo.Heartbeat(ctx, w.op.ID, w.svc.cfg.WorkerID, w.fence, w.svc.cfg.LeaseTTL); err != nil {
				if apierrors.Is(err, apierrors.CodeFenceStale) || apierrors.Is(err, apierrors.CodeOperationNotFound) {
					w.svc.log("heartbeat refused; worker lost ownership", map[string]interface{}{"operation_id": w.op.ID, "fence": w.fence, "error": err.Error()})
					w.lostOnce()
					cancel()
					return
				}
				w.svc.log("heartbeat failed", map[string]interface{}{"operation_id": w.op.ID, "error": err.Error()})
			}
		}
	}
}

func (w *worker) publish(step, message string) {
	w.svc.publish(w.op, step, message)
}

func fenceLost(current *domain.CloudOperation, presented uint64) error {
	return apierrors.New(apierrors.CodeFenceStale, "worker fence superseded").
		WithDetail("operation_id", current.ID).WithDetail("current_fence", current.Fence).WithDetail("presented_fence", presented)
}

// ---- StepSink -------------------------------------------------------------

// Begin decides whether an action runs. Order matters:
//  1. a committed receipt proves the outcome → skip;
//  2. cancellation at a declared cancel point → stop;
//  3. an in-flight marker or recorded unknown effect from a previous fence
//     means the outcome is unknown → apply the action's retry contract:
//     safe_replay replays; observe_then_replay and recover ask the target for
//     its receipt first and never replay blind;
//  4. otherwise mark the step active (fenced) and run.
func (w *worker) Begin(ctx context.Context, action execplan.Action) (Decision, error) {
	if err := ctx.Err(); err != nil {
		return DecisionSkip, err
	}
	current, err := w.svc.repo.GetOperation(ctx, w.op.ID)
	if err != nil || current == nil {
		return DecisionSkip, fmt.Errorf("reload operation before %s: %v", action.ID, err)
	}
	if current.Fence != w.fence {
		w.lostOnce()
		return DecisionSkip, fenceLost(current, w.fence)
	}
	w.op = current
	if receipt, ok := current.Receipt(action.ID); ok {
		switch receipt.Outcome {
		case StepSucceeded, StepUnchanged, StepSkipped:
			w.publish(action.ID, "already committed; skipping")
			return DecisionSkip, nil
		}
	}
	if current.CancelRequested && action.CancelPoint {
		return DecisionSkip, fmt.Errorf("%w: before %s", ErrCancelled, action.ID)
	}
	if unknown := w.outcomeUnknown(current, action.ID); unknown {
		decision, err := w.resolveUnknown(ctx, action)
		if err != nil || decision == DecisionSkip {
			return decision, err
		}
	}
	marker := &domain.ActiveStep{Step: action.ID, Fence: w.fence, StartedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	if err := w.svc.repo.SetActiveStep(ctx, w.op.ID, w.fence, marker); err != nil {
		if apierrors.Is(err, apierrors.CodeFenceStale) {
			w.lostOnce()
		}
		return DecisionSkip, err
	}
	w.publish(action.ID, "step started")
	return DecisionRun, nil
}

// outcomeUnknown reports whether a previous attempt left the step in flight
// (active marker from an older fence, or a recorded unknown effect with no
// later receipt).
func (w *worker) outcomeUnknown(op *domain.CloudOperation, step string) bool {
	if marker := op.ActiveStepMarker(); marker != nil && marker.Step == step && marker.Fence < w.fence {
		return true
	}
	effects, _ := op.UnknownEffectList()
	for _, e := range effects {
		if e.Step == step {
			return true
		}
	}
	return false
}

// resolveUnknown applies the retry contract of an action whose outcome is
// unknown. It returns DecisionSkip with a committed receipt when the target
// proves success, DecisionRun when replay is allowed, or an ErrUnknownEffect
// / ErrRecoveryRequired stop.
func (w *worker) resolveUnknown(ctx context.Context, action execplan.Action) (Decision, error) {
	switch action.Retry {
	case execplan.RetrySafeReplay:
		w.publish(action.ID, "outcome unknown; safe replay")
		return DecisionRun, nil
	case execplan.RetryObserveThenReplay, execplan.RetryRecover:
	default:
		return DecisionSkip, w.recordUnknown(ctx, action, "unknown retry contract "+action.Retry, "inspect the target and set the retry contract")
	}
	if w.svc.receipts == nil {
		return DecisionSkip, w.recordUnknown(ctx, action, "no target receipt reader configured", "install the native vrooli CLI on the target and reconcile")
	}
	tctx, cancel := context.WithTimeout(ctx, w.svc.cfg.TransportTimeout)
	receipt, err := w.svc.receipts.Read(tctx, w.op, action.ID)
	cancel()
	if err != nil {
		if errors.Is(err, ErrNativeCLIAbsent) {
			return DecisionSkip, w.recordUnknown(ctx, action, err.Error(), "install the native vrooli CLI on the target, then POST /api/v1/operations/reconcile")
		}
		if errors.Is(err, context.DeadlineExceeded) {
			return DecisionSkip, w.recordUnknown(ctx, action, "target receipt read timed out", "restore reach to the target, then POST /api/v1/operations/reconcile")
		}
		return DecisionSkip, w.recordUnknown(ctx, action, "target receipt read failed: "+err.Error(), "restore reach to the target, then POST /api/v1/operations/reconcile")
	}
	if !receipt.Found {
		// No receipt: the target owner never completed the verb. Every
		// target verb is idempotent per (operation, step, fence) and
		// reconciles its own interrupted intent (P08), so re-invoking it is
		// the designed path — the replay is of the invocation, not a blind
		// re-execution of a completed effect.
		w.publish(action.ID, "no target receipt; re-invoking the idempotent target verb")
		return DecisionRun, nil
	}
	switch receipt.Outcome {
	case StepSucceeded, StepUnchanged:
		committed := StepReceipt{Step: action.ID, OwnerOperation: action.OwnerOperation, Outcome: receipt.Outcome, Replayed: true, Source: "target_receipt", Detail: receipt.Detail}
		if err := w.commit(ctx, committed); err != nil {
			return DecisionSkip, err
		}
		w.publish(action.ID, "target receipt proves completion; not replayed")
		return DecisionSkip, nil
	case StepFailed:
		if action.Retry == execplan.RetryRecover {
			return DecisionSkip, fmt.Errorf("%w: target reports %s failed (%s)", ErrRecoveryRequired, action.ID, receipt.Error)
		}
		w.publish(action.ID, "target receipt reports failure; replaying")
		return DecisionRun, nil
	default:
		return DecisionSkip, w.recordUnknown(ctx, action, "target receipt outcome "+string(receipt.Outcome), "inspect the target receipt")
	}
}

func (w *worker) recordUnknown(ctx context.Context, action execplan.Action, reason, next string) error {
	effect := UnknownEffect{Step: action.ID, Reason: reason, Retry: action.Retry, NextAction: next}
	if err := w.svc.repo.RecordUnknownEffect(ctx, w.op.ID, w.fence, effect); err != nil {
		if apierrors.Is(err, apierrors.CodeFenceStale) {
			w.lostOnce()
		}
		return err
	}
	typed := apierrors.New(apierrors.CodeReachUnavailable, fmt.Sprintf("%s: %s", action.ID, reason)).
		WithDetail("step", action.ID).WithDetail("retry", action.Retry).
		WithNextAction(apierrors.NextAction{Owner: "scenario-to-cloud", Kind: "reconcile", Reference: "/api/v1/operations/reconcile", Label: next})
	return &controlError{sentinel: ErrUnknownEffect, typed: typed}
}

// controlError pairs a control sentinel (matched with errors.Is) with the
// typed error stored on the record (extracted with apierrors.As).
type controlError struct {
	sentinel error
	typed    *apierrors.Error
}

func (e *controlError) Error() string   { return e.sentinel.Error() + ": " + e.typed.Error() }
func (e *controlError) Unwrap() []error { return []error{e.sentinel, e.typed} }

// WaitingInputError builds the control error a runner returns when the plan
// needs input that is not durable (for example operator-prompt secrets lost
// with the previous owner). typed should carry the handoff as next_action.
func WaitingInputError(typed *apierrors.Error) error {
	return &controlError{sentinel: ErrWaitingInput, typed: typed}
}

// Commit records the observed outcome of an action under the fence. An
// unknown outcome (the effect may have happened but the reply was lost) is
// never committed as failed: the worker asks the target for its receipt
// first, continues when the receipt proves success, replays only under the
// action's contract, and otherwise records the unknown effect and stops for
// reconciliation.
func (w *worker) Commit(ctx context.Context, action execplan.Action, outcome StepOutcome, detail string, execErr error) error {
	if err := faultinject.Hit(ctx, faultinject.WorkerBeforeCommit); err != nil {
		return err
	}
	if outcome == StepUnknown {
		w.publish(action.ID, "reply lost; reading the target receipt before any replay")
		decision, err := w.resolveUnknown(ctx, action)
		if err != nil {
			return err
		}
		if decision == DecisionRun {
			return ErrReplayStep
		}
		return nil
	}
	receipt := StepReceipt{Step: action.ID, OwnerOperation: action.OwnerOperation, Outcome: outcome, Source: "worker", Detail: detail}
	if execErr != nil {
		receipt.Error = execErr.Error()
	}
	if err := w.commit(ctx, receipt); err != nil {
		return err
	}
	w.committed++
	w.publish(action.ID, "step "+string(outcome))
	if execErr != nil && action.Retry == execplan.RetryRecover {
		return fmt.Errorf("%w: %s failed (%s): %v", ErrRecoveryRequired, action.ID, action.Recovery, execErr)
	}
	return execErr
}

func (w *worker) commit(ctx context.Context, receipt StepReceipt) error {
	err := w.svc.repo.CommitStep(ctx, w.op.ID, w.fence, receipt)
	if err != nil && apierrors.Is(err, apierrors.CodeFenceStale) {
		w.lostOnce()
	}
	return err
}
