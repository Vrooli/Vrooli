package lifecycle

import (
	"context"
	"errors"
	"strconv"
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
	startStepStop          = "stop"
	startStepDependencies  = "dependencies"
	startStepSetup         = "setup"
	startStepDevelop       = "develop"
	startStepHealth        = "health"
	startStepProviderFence = "provider_fence"
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
	instanceSlug  string
	providerFence *providerFence
	done          chan struct{}
	cancelled     chan struct{}
}

func (rec *startOperationRecorder) operationID() string {
	if rec == nil {
		return ""
	}
	return rec.op.OperationID
}

func (rec *startOperationRecorder) setProviderFence(fence *providerFence) {
	if rec != nil {
		rec.providerFence = fence
	}
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
func (r *Runner) beginStartOperationRecord(name string, opts StartOptions) (*startOperationRecorder, error) {
	deps := r.runtimeDeps()
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	store, err := deps.runtimeRegistry(ctx, r.Home)
	if err != nil {
		if opts.providerFenceOperationID != "" || opts.Operation == lifecycleOperationRestart {
			return nil, errors.Join(errors.New("restart requires a durable start-operation record before preparation or replacement"), err)
		}
		r.logWarn("Start-operation record unavailable; starting unrecorded",
			logx.AttrScenario, name, "error", err.Error())
		return nil, nil
	}
	// Capture WHO is starting this, not just the PID. The PID is the liveness
	// signal and dies with the process; argv, the parent, and the OS scope are
	// what let anyone reconstruct the cause of a start later, on a host where
	// many short-lived CLIs start work concurrently.
	initiator := deps.captureInitiator()
	op, err := store.BeginStartOperation(ctx, scenarioruntime.StartOperation{
		OperationID:         opts.lifecycleOperationID,
		FenceOperationID:    opts.providerFenceOperationID,
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
		if opts.providerFenceOperationID != "" || opts.Operation == lifecycleOperationRestart {
			return nil, errors.Join(errors.New("restart requires a durable start-operation record before preparation or replacement"), err)
		}
		r.logWarn("Start-operation record could not be created; starting unrecorded",
			logx.AttrScenario, name, "error", err.Error())
		return nil, nil
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
	return rec, nil
}

func (rec *startOperationRecorder) linkProviderFence(operationID string) error {
	if rec == nil {
		return errors.New("start operation recorder is unavailable for provider fence recovery link")
	}
	if rec.store == nil {
		return errors.New("start operation registry is unavailable for provider fence recovery link")
	}
	if err := rec.store.SetStartOperationFenceOperationID(rec.context(), rec.op.OperationID, operationID); err != nil {
		return err
	}
	rec.op.FenceOperationID = operationID
	rec.runner.logInfo("Provider lifecycle fence linked to start operation",
		logx.AttrScenario, rec.op.Scenario,
		"start_operation_id", rec.op.OperationID,
		"provider_fence_operation_id", operationID,
	)
	return nil
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
		// A failed progress write disables the recorder and drops its handle
		// (flush) after Publish already moved the in-memory record forward, so
		// the stored record can still say "running" while this operation is
		// over. A retry started right away would refuse to recover a fence whose
		// owner looks live. Settle the stored record through a fresh handle.
		rec.settleWithFreshHandle()
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

// settleWithFreshHandle writes the operation's final state when the recorder's
// own registry handle is already gone: the in-memory terminal record if there is
// one, otherwise an abandoned marker. Both writes only change a record that is
// still running, so ErrNotFound means it already settled (the cancellation
// watcher, a takeover, or an earlier successful flush), which is the outcome
// wanted.
func (rec *startOperationRecorder) settleWithFreshHandle() {
	store, err := rec.runner.runtimeDeps().runtimeRegistry(context.Background(), rec.runner.Home)
	if err != nil {
		rec.runner.logDebug("Settle start-operation record failed", "error", err.Error())
		return
	}
	defer func() { _ = store.Close() }()
	if rec.op.IsTerminal() {
		_, err = store.UpdateStartOperation(context.Background(), rec.op)
	} else {
		_, err = store.MarkStartOperationAbandoned(context.Background(), rec.op.OperationID, "initiator exited before completion")
	}
	if err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
		rec.runner.logDebug("Settle start-operation record failed", "error", err.Error())
	}
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
		identity := map[string]string{
			"runtime_instance_id":                   ev.RuntimeInstanceID,
			"selected_release_id":                   ev.RuntimeReleaseID,
			"release_selection_operation_id":        ev.RuntimeReleaseOperationID,
			"expected_build_identity":               ev.RuntimeBuildIdentity,
			"served_build_identity":                 ev.HealthBuildIdentity,
			"served_release_id":                     ev.HealthRuntimeReleaseID,
			"served_release_selection_operation_id": ev.HealthRuntimeReleaseOperationID,
			"served_start_operation_id":             ev.HealthStartOperationID,
			"health_verdict":                        ev.Verdict,
		}
		if ev.RuntimeInstanceID != "" {
			rec.completeStepWithEvidence(scenarioruntime.StartStepRuntimeIdentity, ev.At, identity)
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
		rec.failRuntimeIdentityStep(ev.At, ev.RollbackSelectionAttempted, ev.RollbackSelectionReleaseID, ev.RollbackSelectionOperationID, ev.RollbackSelectionError)
		rec.op.Status = scenarioruntime.StartOperationStatusFailed
		if ev.Err != nil {
			rec.op.Error = ev.Err.Error()
		}
		at := ev.At.UTC()
		rec.op.FinishedAt = &at
	case EventRuntimeIdentitySelected:
		if ev.Scenario != rec.scenario {
			return
		}
		rec.beginRuntimeIdentityStep(ev)
	default:
		return
	}
	rec.flush()
}

func (rec *startOperationRecorder) beginRuntimeIdentityStep(ev ProgressEvent) {
	if rec == nil {
		return
	}
	evidence := map[string]string{
		"runtime_instance_id":            ev.RuntimeInstanceID,
		"selected_release_id":            ev.RuntimeReleaseID,
		"release_selection_operation_id": ev.RuntimeReleaseOperationID,
		"expected_build_identity":        ev.RuntimeBuildIdentity,
		"start_operation_id":             rec.op.OperationID,
	}
	for i := range rec.steps {
		if rec.steps[i].Name == scenarioruntime.StartStepRuntimeIdentity {
			rec.steps[i].Evidence = evidence
			rec.op.WithSteps(rec.steps)
			rec.flush()
			return
		}
	}
	at := ev.At.UTC()
	rec.steps = append(rec.steps, scenarioruntime.StartOperationStep{
		Name: scenarioruntime.StartStepRuntimeIdentity, Status: scenarioruntime.StartStepRunning, StartedAt: at, Evidence: evidence,
	})
	rec.op.CurrentStep = scenarioruntime.StartStepRuntimeIdentity
	rec.flush()
}

func (rec *startOperationRecorder) failRuntimeIdentityStep(at time.Time, rollbackAttempted bool, rollbackReleaseID, rollbackOperationID, rollbackErr string) {
	if rec == nil {
		return
	}
	for i := range rec.steps {
		if rec.steps[i].Name != scenarioruntime.StartStepRuntimeIdentity {
			continue
		}
		if rec.steps[i].Evidence == nil {
			rec.steps[i].Evidence = map[string]string{}
		}
		rec.steps[i].Evidence["rollback_attempted"] = strconv.FormatBool(rollbackAttempted)
		if rollbackAttempted {
			rec.steps[i].Evidence["rollback_release_id"] = rollbackReleaseID
			rec.steps[i].Evidence["rollback_operation_id"] = rollbackOperationID
			rec.steps[i].Evidence["rollback_error"] = rollbackErr
		}
		ended := at.UTC()
		rec.steps[i].Status = scenarioruntime.StartStepFailed
		rec.steps[i].EndedAt = &ended
		rec.op.WithSteps(rec.steps)
		return
	}
}

func (rec *startOperationRecorder) completeStepWithEvidence(name string, at time.Time, evidence map[string]string) {
	if rec == nil {
		return
	}
	at = at.UTC()
	for i := range rec.steps {
		if rec.steps[i].Name == name {
			for key, value := range evidence {
				if value == "" {
					delete(evidence, key)
				}
			}
			for key, value := range rec.steps[i].Evidence {
				if _, ok := evidence[key]; !ok {
					evidence[key] = value
				}
			}
			rec.steps[i].Status = scenarioruntime.StartStepDone
			rec.steps[i].EndedAt = &at
			rec.steps[i].Evidence = evidence
			rec.op.WithSteps(rec.steps)
			return
		}
	}
	rec.steps = append(rec.steps, scenarioruntime.StartOperationStep{
		Name: name, Status: scenarioruntime.StartStepDone, StartedAt: at, EndedAt: &at, Evidence: evidence,
	})
	rec.op.WithSteps(rec.steps)
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

// markCompletedStep ensures a completed step remains visible even when its
// progress event was not emitted. Replace a previously observed running step
// instead of recording the same stop twice.
func (rec *startOperationRecorder) markCompletedStep(name string, started, ended time.Time) {
	if rec == nil {
		return
	}
	started = started.UTC()
	ended = ended.UTC()
	if _, ok := rec.completeStep(name, ended, scenarioruntime.StartStepDone); ok {
		rec.op.WithSteps(rec.steps)
		return
	}
	rec.steps = append(rec.steps, scenarioruntime.StartOperationStep{
		Name: name, Status: scenarioruntime.StartStepDone,
		StartedAt: started, EndedAt: &ended,
	})
	rec.op.WithSteps(rec.steps)
}

func (rec *startOperationRecorder) beginAndFlushStep(name string, at time.Time) {
	if rec == nil || rec.store == nil {
		return
	}
	rec.beginStep(name, at)
	rec.flush()
}

func (rec *startOperationRecorder) completeAndFlushStep(name string, at time.Time) {
	if rec == nil || rec.store == nil {
		return
	}
	if _, ok := rec.completeStep(name, at, scenarioruntime.StartStepDone); !ok {
		return
	}
	rec.flush()
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
