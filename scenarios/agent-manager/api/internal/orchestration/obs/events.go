// Lifecycle event taxonomy + emission helpers.
//
// Lifecycle events are the stable spine of the run timeline: the same
// six transitions appear on every run, in the same order, with the
// same payload shape. This package owns construction so a renamed
// phase or a missing transition is a single-file change rather than a
// search-and-replace.
//
// Contract (see SEAMS.md, decision "Lifecycle events are emitted
// through obs/events.go only"):
//   - Lifecycle events MUST be constructed via the helpers below.
//   - Helpers route through a [runner.EventSink]-shaped sink so the
//     same Gate / event-store / WebSocket pipeline carries them; no
//     side channels.
//   - When emission fails (sink == nil, downstream error) the helper
//     also writes a structured log line via [Logger] so the transition
//     is at least visible in stderr/journal.

package obs

import (
	"log/slog"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// Sink is the minimal interface obs needs to publish a lifecycle event.
// It matches the [runner.EventSink] shape (Emit + Close) but is declared
// here to avoid an import cycle (runner imports nothing from
// orchestration today, and we want to keep it that way).
//
// Callers pass any concrete value satisfying this interface — typically
// the per-run [emit.Gate] for in-band runs, or the recovery event
// sink for the reconciler tail path.
type Sink interface {
	Emit(event *domain.RunEvent) error
}

// SpawnEnqueuedFields are the timing/dispatcher-state fields available
// at the moment a run is enqueued for spawn.
type SpawnEnqueuedFields struct {
	RunMode      domain.RunMode
	RunnerType   domain.RunnerType
	QueueDepth   int
	ActiveCount  int
	StartingSlot int
}

// EmitSpawnEnqueued records that a run has entered the spawn dispatcher
// queue. Phase 3 (spawn dispatcher) wires the producer; for Phase 2
// the helper exists so service.go can call it from the bare-goroutine
// site too.
func EmitSpawnEnqueued(sink Sink, runID uuid.UUID, fields SpawnEnqueuedFields) {
	emitLifecycle(sink, runID, domain.LifecycleEventData{
		Phase:       domain.LifecyclePhaseSpawnEnqueued,
		Message:     "spawn enqueued",
		RunnerType:  string(fields.RunnerType),
		QueueDepth:  fields.QueueDepth,
		ActiveCount: fields.ActiveCount,
	}, slog.Group("spawn",
		KeyRunMode, string(fields.RunMode),
		KeyRunnerType, string(fields.RunnerType),
		KeyQueueDepth, fields.QueueDepth,
		KeyActiveCount, fields.ActiveCount,
	))
}

// SpawnStartedFields capture the queue-residency time and the worker
// state at the moment a queued job actually starts executing.
type SpawnStartedFields struct {
	RunMode     domain.RunMode
	RunnerType  domain.RunnerType
	QueuedFor   time.Duration
	ActiveCount int
}

// EmitSpawnStarted records that the dispatcher has picked up a queued
// job and is about to invoke the executor.
func EmitSpawnStarted(sink Sink, runID uuid.UUID, fields SpawnStartedFields) {
	emitLifecycle(sink, runID, domain.LifecycleEventData{
		Phase:       domain.LifecyclePhaseSpawnStarted,
		Message:     "spawn started",
		DurationMS:  fields.QueuedFor.Milliseconds(),
		RunnerType:  string(fields.RunnerType),
		ActiveCount: fields.ActiveCount,
	}, slog.Group("spawn",
		KeyRunMode, string(fields.RunMode),
		KeyRunnerType, string(fields.RunnerType),
		KeyDuration, fields.QueuedFor.Milliseconds(),
		KeyActiveCount, fields.ActiveCount,
	))
}

// RunnerAcquiredFields describe the runner that has been bound to the
// run after AcquireRunner / launcher selection.
type RunnerAcquiredFields struct {
	RunnerType   domain.RunnerType
	LauncherType string
	SandboxID    *uuid.UUID
}

// EmitRunnerAcquired records that AcquireRunner picked a runner +
// launcher pair. Always emit before the runner subprocess is started so
// the timeline shows "we got here, then this happened" rather than the
// ambiguous bare timeout.
func EmitRunnerAcquired(sink Sink, runID uuid.UUID, fields RunnerAcquiredFields) {
	sandboxID := ""
	if fields.SandboxID != nil && *fields.SandboxID != uuid.Nil {
		sandboxID = fields.SandboxID.String()
	}
	emitLifecycle(sink, runID, domain.LifecycleEventData{
		Phase:        domain.LifecyclePhaseRunnerAcquired,
		Message:      "runner acquired",
		RunnerType:   string(fields.RunnerType),
		LauncherType: fields.LauncherType,
		SandboxID:    sandboxID,
	}, slog.Group("runner",
		KeyRunnerType, string(fields.RunnerType),
		KeyLauncherType, fields.LauncherType,
		KeySandboxID, sandboxID,
	))
}

// RunnerExitedFields capture the terminal state observed when the
// runner subprocess returns.
type RunnerExitedFields struct {
	RunnerType   domain.RunnerType
	ExitCode     *int
	Duration     time.Duration
	TerminalCode string
	Success      bool
}

// EmitRunnerExited records that core.Runner.Execute has returned a
// classified result. terminalCode is the structured error code (when
// any) so an operator can grep "RUNNER_SESSION_STATE_LOST" without
// joining against the error event.
func EmitRunnerExited(sink Sink, runID uuid.UUID, fields RunnerExitedFields) {
	exitAttr := slog.Attr{}
	if fields.ExitCode != nil {
		exitAttr = slog.Int(KeyExitCode, *fields.ExitCode)
	}
	emitLifecycle(sink, runID, domain.LifecycleEventData{
		Phase:        domain.LifecyclePhaseRunnerExited,
		Message:      runnerExitedMessage(fields),
		DurationMS:   fields.Duration.Milliseconds(),
		RunnerType:   string(fields.RunnerType),
		ExitCode:     fields.ExitCode,
		TerminalCode: fields.TerminalCode,
	}, slog.Group("runner",
		KeyRunnerType, string(fields.RunnerType),
		KeyDuration, fields.Duration.Milliseconds(),
		KeyTerminalCode, fields.TerminalCode,
		exitAttr,
	))
}

func runnerExitedMessage(fields RunnerExitedFields) string {
	if fields.Success {
		return "runner exited cleanly"
	}
	if fields.TerminalCode != "" {
		return "runner exited: " + fields.TerminalCode
	}
	return "runner exited with failure"
}

// FinalizeFields capture which sandbox-lifecycle action finalize took
// (delete / stop / preserve). Phase 5 documents the values; for Phase 2
// we just propagate the raw string from finalize so the consumer side
// is informational.
type FinalizeFields struct {
	SandboxID *uuid.UUID
	Action    string // "delete" | "stop" | "preserve" | ""
}

// EmitFinalizeStarted records the entry into the deferred terminal
// seam. Pair with [EmitFinalizeCompleted] so duration is observable.
func EmitFinalizeStarted(sink Sink, runID uuid.UUID, fields FinalizeFields) {
	sandboxID := ""
	if fields.SandboxID != nil && *fields.SandboxID != uuid.Nil {
		sandboxID = fields.SandboxID.String()
	}
	emitLifecycle(sink, runID, domain.LifecycleEventData{
		Phase:     domain.LifecyclePhaseFinalizeStarted,
		Message:   "finalize started",
		SandboxID: sandboxID,
	}, slog.Group("finalize",
		KeySandboxID, sandboxID,
	))
}

// EmitFinalizeCompleted records the exit from the deferred terminal
// seam, including the action finalize took on the sandbox.
func EmitFinalizeCompleted(sink Sink, runID uuid.UUID, fields FinalizeFields, took time.Duration) {
	sandboxID := ""
	if fields.SandboxID != nil && *fields.SandboxID != uuid.Nil {
		sandboxID = fields.SandboxID.String()
	}
	msg := "finalize completed"
	if fields.Action != "" {
		msg = "finalize completed: sandbox " + fields.Action
	}
	emitLifecycle(sink, runID, domain.LifecycleEventData{
		Phase:      domain.LifecyclePhaseFinalizeCompleted,
		Message:    msg,
		DurationMS: took.Milliseconds(),
		SandboxID:  sandboxID,
	}, slog.Group("finalize",
		KeySandboxID, sandboxID,
		KeyDuration, took.Milliseconds(),
		"action", fields.Action,
	))
}

// emitLifecycle is the shared body for every helper above: build the
// typed event, push it into the sink, and (always) log the transition
// so it is visible regardless of sink wiring.
func emitLifecycle(sink Sink, runID uuid.UUID, data domain.LifecycleEventData, logAttrs ...any) {
	evt := domain.NewLifecycleEvent(runID, data)
	if sink != nil {
		_ = sink.Emit(evt)
	}
	args := append([]any{KeyRunID, runID.String(), KeyPhase, string(data.Phase)}, logAttrs...)
	Logger().Info(data.Message, args...)
}
