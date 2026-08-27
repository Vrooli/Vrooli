package lifecycle

import (
	"context"
	"errors"
	"time"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// Registry-backed progress sink (plan Phase 3): translates the typed progress
// stream of ONE top-level start/restart into a durable start-operation record
// other processes can introspect (`scenario status`), attach to (`scenario
// wait`, concurrent `start`), and resume after initiator death.
//
// The record is derived state, never authority: every write error here is
// logged and swallowed so a registry hiccup can never fail a start, and
// readers must treat a dead-initiator record as noise.

// Step names recorded on a start operation, in execution order.
const (
	startStepStop         = "stop"
	startStepDependencies = "dependencies"
	startStepSetup        = "setup"
	startStepDevelop      = "develop"
	startStepHealth       = "health"
)

type startOperationRecorder struct {
	runner   *Runner
	ctx      context.Context
	store    scenarioRuntimeStore // nil once disabled by a write failure
	op       scenarioruntime.StartOperation
	steps    []scenarioruntime.StartOperationStep
	scenario string // top-level scenario; dependency-phase events are filtered by it
	// instanceSlug is InstanceKey{scenario, variant}.Slug(). Stop events carry
	// the instance slug (scenario@variant for non-live), unlike every other
	// event, so they are filtered against this instead of scenario.
	instanceSlug string
	done         chan struct{}
	cancelled    chan struct{}
}

func (rec *startOperationRecorder) context() context.Context {
	if rec == nil || rec.ctx == nil {
		return context.Background()
	}
	return rec.ctx
}

// beginStartOperationRecord opens the registry and creates a running
// operation record for a top-level start. Returns nil (no-op) when the
// registry is unavailable — the start proceeds unrecorded.
func (r *Runner) beginStartOperationRecord(name string, opts StartOptions) *startOperationRecorder {
	deps := r.runtimeDeps()
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	store, err := deps.runtimeRegistry(ctx, r.Home)
	if err != nil {
		r.logWarn("Start-operation record unavailable; starting unrecorded",
			logx.AttrScenario, name, "error", err.Error())
		return nil
	}
	// Capture WHO is starting this, not just the PID. The PID is the liveness
	// signal and dies with the process; argv, the parent, and the OS scope are
	// what let anyone reconstruct the cause of a start later, on a host where
	// many short-lived CLIs start work concurrently.
	initiator := deps.captureInitiator()
	op, err := store.BeginStartOperation(ctx, scenarioruntime.StartOperation{
		Scenario:            name,
		Variant:             opts.Variant,
		Operation:           defaultIfEmpty(opts.Operation, "start"),
		InitiatorPID:        &initiator.PID,
		InitiatorArgv:       initiator.Argv,
		InitiatorParentPID:  &initiator.ParentPID,
		InitiatorParentArgv: initiator.ParentArgv,
		InitiatorScope:      initiator.Scope,
		StartedAt:           deps.now().UTC(),
	})
	if err != nil {
		_ = store.Close()
		r.logWarn("Start-operation record could not be created; starting unrecorded",
			logx.AttrScenario, name, "error", err.Error())
		return nil
	}
	rec := &startOperationRecorder{
		runner: r, ctx: ctx, store: store, op: op, scenario: name,
		instanceSlug: scenarioruntime.InstanceKey{Scenario: name, Variant: opts.Variant}.Slug(),
		done:         make(chan struct{}),
		cancelled:    make(chan struct{}),
	}
	// Cancellation is owned by the operation context. A fresh registry handle
	// avoids racing the recorder's own flush/close path while preserving the
	// abandoned marker when the caller cancels mid-phase.
	go func(operationID, home string, operationCtx context.Context, done <-chan struct{}) {
		select {
		case <-operationCtx.Done():
			if store, err := deps.runtimeRegistry(context.Background(), home); err == nil {
				_, _ = store.MarkStartOperationAbandoned(context.Background(), operationID, "start cancelled")
				_ = store.Close()
			}
			close(rec.cancelled)
		case <-done:
		}
	}(op.OperationID, r.Home, ctx, rec.done)
	return rec
}

// close releases the recorder's registry handle. If the operation never
// reached a terminal state (initiator interrupted between events), the
// running record is marked abandoned so status stays honest.
func (rec *startOperationRecorder) close() {
	if rec == nil {
		return
	}
	if rec.done != nil {
		close(rec.done)
		rec.done = nil
	}
	if rec.store == nil {
		return
	}
	if !rec.op.IsTerminal() {
		if _, err := rec.store.MarkStartOperationAbandoned(context.Background(), rec.op.OperationID, "initiator exited before completion"); err != nil {
			rec.runner.logDebug("Abandon start-operation record failed", "error", err.Error())
		}
	}
	_ = rec.store.Close()
	rec.store = nil
}

//nolint:gocyclo // operation publishing maps event, phase, terminal, and persistence states to one record.
func (rec *startOperationRecorder) Publish(ev ProgressEvent) {
	if rec == nil || rec.store == nil {
		return
	}
	switch ev.Kind {
	case EventStopStarted:
		// Only this instance's stop (restart's leading step, or a mid-plan
		// stop-then-start — the same step semantically). Matching the instance
		// slug also keeps a dependency's stop from opening this record's step.
		if ev.Scenario == rec.instanceSlug {
			rec.beginStep(startStepStop, ev.At)
		}
	case EventOperationStarted:
		if ev.Scenario != rec.scenario {
			return
		}
		rec.completeStep(startStepStop, ev.At, scenarioruntime.StartStepDone)
		rec.op.CurrentStep = startStepDependencies
	case EventDependencyStarting, EventDependencyReused:
		if ev.Scenario != rec.scenario {
			return
		}
		rec.beginStep(startStepDependencies, ev.At)
		rec.op.DependencyCurrent = ev.Dependency
		rec.op.DependencyIndex = ev.Index
		rec.op.DependencyTotal = ev.Total
	case EventPhaseStarted:
		if ev.Scenario != rec.scenario {
			return
		}
		// A mid-plan stop (unfit instance torn down before the fresh start)
		// has no completion event of its own; the next phase beginning is it.
		rec.completeStep(startStepStop, ev.At, scenarioruntime.StartStepDone)
		rec.completeStep(startStepDependencies, ev.At, scenarioruntime.StartStepDone)
		rec.beginStep(ev.Phase, ev.At)
	case EventPhaseCompleted:
		if ev.Scenario != rec.scenario {
			return
		}
		if d, ok := rec.completeStep(ev.Phase, ev.At, scenarioruntime.StartStepDone); ok {
			rec.recordDuration(ev.Phase, d)
		}
	case EventHealthWaiting:
		if ev.Scenario != rec.scenario {
			return
		}
		rec.completeStep(startStepStop, ev.At, scenarioruntime.StartStepDone)
		rec.completeStep(startStepDependencies, ev.At, scenarioruntime.StartStepDone)
		rec.beginStep(startStepHealth, ev.At)
	case EventOperationCompleted:
		if ev.Scenario != rec.scenario {
			return
		}
		if d, ok := rec.completeStep(startStepHealth, ev.At, scenarioruntime.StartStepDone); ok {
			rec.recordDuration(startStepHealth, d)
		}
		rec.op.Status = scenarioruntime.StartOperationStatusSucceeded
		rec.op.Verdict = ev.Verdict
		rec.op.CurrentStep = ""
		rec.op.DependencyCurrent = ""
		at := ev.At.UTC()
		rec.op.FinishedAt = &at
	case EventOperationFailed:
		if ev.Scenario != rec.scenario {
			return
		}
		rec.failRunningSteps(ev.At)
		rec.op.Status = scenarioruntime.StartOperationStatusFailed
		if ev.Err != nil {
			rec.op.Error = ev.Err.Error()
		}
		at := ev.At.UTC()
		rec.op.FinishedAt = &at
	default:
		return
	}
	rec.flush()
}

// beginStep appends a running step (idempotent while it is already running)
// and points CurrentStep at it.
func (rec *startOperationRecorder) beginStep(name string, at time.Time) {
	rec.op.CurrentStep = name
	for i := range rec.steps {
		if rec.steps[i].Name == name && rec.steps[i].Status == scenarioruntime.StartStepRunning {
			return
		}
	}
	rec.steps = append(rec.steps, scenarioruntime.StartOperationStep{
		Name:      name,
		Status:    scenarioruntime.StartStepRunning,
		StartedAt: at.UTC(),
	})
}

// completeStep transitions a running step to the given terminal status and
// returns its duration. No-op (false) when the step is not running.
func (rec *startOperationRecorder) completeStep(name string, at time.Time, status string) (time.Duration, bool) {
	for i := range rec.steps {
		if rec.steps[i].Name == name && rec.steps[i].Status == scenarioruntime.StartStepRunning {
			ended := at.UTC()
			rec.steps[i].Status = status
			rec.steps[i].EndedAt = &ended
			return ended.Sub(rec.steps[i].StartedAt), true
		}
	}
	return 0, false
}

func (rec *startOperationRecorder) failRunningSteps(at time.Time) {
	for i := range rec.steps {
		if rec.steps[i].Status == scenarioruntime.StartStepRunning {
			ended := at.UTC()
			rec.steps[i].Status = scenarioruntime.StartStepFailed
			rec.steps[i].EndedAt = &ended
		}
	}
}

func (rec *startOperationRecorder) recordDuration(phase string, d time.Duration) {
	if d < 0 {
		return
	}
	if err := rec.store.RecordPhaseDuration(rec.context(), rec.op.Scenario, rec.op.Variant, phase, d); err != nil {
		rec.runner.logDebug("Record phase duration failed", logx.AttrScenario, rec.op.Scenario, logx.AttrPhase, phase, "error", err.Error())
	}
}

// flush persists the record. A write failure disables the recorder (the
// start must never fail because progress bookkeeping did). ErrNotFound means
// the record went terminal under us — abandoned by the cancellation watcher or
// superseded by a takeover — and terminal records are immutable, so the
// recorder stops writing without alarming anyone.
func (rec *startOperationRecorder) flush() {
	rec.op.WithSteps(rec.steps)
	updated, err := rec.store.UpdateStartOperation(rec.context(), rec.op)
	if err != nil {
		if errors.Is(err, scenarioruntime.ErrNotFound) {
			rec.runner.logDebug("Start-operation record abandoned or superseded externally; further progress unrecorded",
				logx.AttrScenario, rec.op.Scenario)
		} else {
			rec.runner.logWarn("Start-operation record write failed; further progress unrecorded",
				logx.AttrScenario, rec.op.Scenario, "error", err.Error())
		}
		_ = rec.store.Close()
		rec.store = nil
		return
	}
	rec.op = updated
}

// attachSink registers a sink for the duration of one operation and returns
// the detach closure.
func (r *Runner) attachSink(sink ProgressSink) func() {
	r.sinksMu.Lock()
	if r.sinks == nil {
		r.sinks = []ProgressSink{r}
	}
	r.sinks = append(r.sinks, sink)
	r.sinksMu.Unlock()
	return func() {
		r.sinksMu.Lock()
		defer r.sinksMu.Unlock()
		for i := range r.sinks {
			if r.sinks[i] == sink {
				r.sinks = append(r.sinks[:i], r.sinks[i+1:]...)
				return
			}
		}
	}
}
