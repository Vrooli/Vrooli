// Package orchestration provides the core orchestration service for agent-manager.
//
// This file contains the RunExecutor, the thin coordinator that owns phase
// ordering and shared per-run state (run record, checkpoint, mutex,
// heartbeat goroutine, identity token). All phase logic lives in
// internal/orchestration/phases — this file is the orchestration entry
// point that wires phases together via the Execute() coordinator.
//
// EXECUTION FLOW (Execute):
//   1. updateStatusToStarting()
//   2. phases.SetupWorkspace()
//   3. phases.AcquireRunner()
//   4. phases.GenerateIdentityToken()
//   5. phases.ExecuteWithModelFallback()
//   6. phases.HandleResult()
//   7. phases.Finalize() — DEFERRED: phase ladder + sandbox teardown
//
// FINALIZATION CONTRACT:
// phases.Finalize is the single terminal seam. It is registered as a
// `defer` at the top of Execute, so it ALWAYS runs — on normal completion,
// panic, or context cancellation. It guarantees:
//   - The phase ladder ends at RunPhaseCompleted.
//   - The sandbox mount is released exactly once via Delete or Stop.
//   - Sandbox teardown is IMMUNE to execCtx cancellation/timeout.
//
// Idempotency for Finalize is enforced by the `finalized` flag on this
// struct so re-entry (panic + recovery, etc.) is a no-op.
//
// 2026-04-28 incident: before Finalize existed, sandbox teardown was
// scattered across the success/failure/cancel handlers, each calling
// teardown with execCtx (the run's 60-min timeout). 11 days of
// accumulation produced 326 orphan mounts holding ~49 GB of RAM.
// Regression gates live in phases/finalize_test.go.

package orchestration

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/emit"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/repository"
	"agent-manager/internal/runstate"

	"github.com/google/uuid"
)

// ModelHealthReporter is an alias for the canonical phase dependency seam.
type ModelHealthReporter = phases.ModelHealthReporter

// RunExecutor handles the execution lifecycle of a single run. It is a
// thin coordinator: it owns shared per-run state and dispatches to phase
// functions. Phase logic does not live here.
type RunExecutor struct {
	// Dependencies
	runs              repository.RunRepository
	runners           runner.Registry
	sandbox           sandbox.Provider
	events            event.Store
	checkpoints       repository.CheckpointRepository
	broadcaster       phases.EventBroadcaster
	workspaceSandbox  phases.WorkspaceSandboxEnsurer
	modelHealth       ModelHealthReporter
	structuredResults phases.StructuredResultResolver
	clock             func() time.Time

	// Configuration
	levers config.Levers

	// gate is the single Emit choke point for events flowing out of this run.
	gate *emit.Gate

	// Per-run inputs
	run          *domain.Run
	task         *domain.Task
	profile      *domain.AgentProfile
	prompt       string
	systemPrompt string
	attachments  []runner.Attachment

	// Workspace state
	sandboxID     *uuid.UUID
	workDir       string
	runState      *runstate.State
	runStateRoot  string
	runStateWrite func()

	// Progress + concurrency
	checkpoint *domain.RunCheckpoint
	mu         sync.Mutex

	// Heartbeat
	heartbeatStop chan struct{}
	heartbeatDone chan struct{}

	// Result state
	outcome domain.RunOutcome
	result  *runner.ExecuteResult
	execErr error

	// Resumption state
	isResuming bool

	// finalized guards the deferred phases.Finalize seam against re-entry.
	finalized bool

	// parked is set when the agent parked mid-turn (the run's persisted status
	// became RunStatusParked while this executor's process was running). It makes
	// the deferred finalize a no-op so the park's preserved sandbox is not torn
	// down — the park owns the run lifecycle from that point.
	parked bool

	// Caller-provided env vars
	customEnv  map[string]string
	sessionEnv map[string]string

	// Identity token state
	identitySecret []byte
	identityToken  string

	// onRunning fires exactly once when the run reaches
	// RunStatusRunning. Wired by the spawn dispatcher to release the
	// startup slot. Nil is safe — the executor still runs.
	onRunning func()

	// onTerminal receives the persisted terminal run after the finalization
	// seam. It is deliberately best-effort so analytics capture cannot alter a
	// completed run's result.
	onTerminal func(*domain.Run)
}

// NewRunExecutor creates a new executor for the given run.
func NewRunExecutor(
	runs repository.RunRepository,
	runners runner.Registry,
	sandbox sandbox.Provider,
	events event.Store,
	run *domain.Run,
	task *domain.Task,
	profile *domain.AgentProfile,
	prompt string,
	systemPrompt string,
) *RunExecutor {
	return &RunExecutor{
		runs:          runs,
		runners:       runners,
		sandbox:       sandbox,
		events:        events,
		run:           run,
		task:          task,
		profile:       profile,
		prompt:        prompt,
		systemPrompt:  systemPrompt,
		levers:        config.DefaultLevers(),
		checkpoint:    domain.NewCheckpoint(run.ID, domain.RunPhaseQueued),
		heartbeatStop: make(chan struct{}),
		heartbeatDone: make(chan struct{}),
	}
}

// =============================================================================
// BUILDER METHODS
// =============================================================================

func (e *RunExecutor) WithLevers(l config.Levers) *RunExecutor { e.levers = l; return e }

func (e *RunExecutor) WithRunStateRoot(root string) *RunExecutor { e.runStateRoot = root; return e }

func (e *RunExecutor) WithRunStateWriteObserver(observer func()) *RunExecutor {
	e.runStateWrite = observer
	return e
}

// WithClock supplies deterministic timestamps to all execution phases.
func (e *RunExecutor) WithClock(clock func() time.Time) *RunExecutor { e.clock = clock; return e }

// WithTerminalObserver installs a best-effort observer after the final run
// state is persisted. The owner uses this to keep the durable analytics read
// model current for normal executor-driven runs.
func (e *RunExecutor) WithTerminalObserver(observer func(*domain.Run)) *RunExecutor {
	e.onTerminal = observer
	return e
}

func (e *RunExecutor) WithCheckpointRepository(repo repository.CheckpointRepository) *RunExecutor {
	e.checkpoints = repo
	return e
}

func (e *RunExecutor) WithModelHealthReporter(reporter ModelHealthReporter) *RunExecutor {
	e.modelHealth = reporter
	return e
}

func (e *RunExecutor) WithStructuredResultResolver(resolver phases.StructuredResultResolver) *RunExecutor {
	e.structuredResults = resolver
	return e
}

func (e *RunExecutor) WithExistingSandbox(sandboxID uuid.UUID, workDir string) *RunExecutor {
	e.sandboxID = &sandboxID
	if workDir != "" {
		e.workDir = workDir
	}
	e.checkpoint = e.checkpoint.WithSandbox(sandboxID, workDir)
	return e
}

func (e *RunExecutor) WithIdentitySecret(secret []byte) *RunExecutor {
	e.identitySecret = secret
	return e
}

func (e *RunExecutor) WithResumeFrom(checkpoint *domain.RunCheckpoint) *RunExecutor {
	e.checkpoint = checkpoint
	e.isResuming = true
	if checkpoint.SandboxID != nil {
		e.sandboxID = checkpoint.SandboxID
		e.workDir = checkpoint.WorkDir
	}
	return e
}

func (e *RunExecutor) WithBroadcaster(b phases.EventBroadcaster) *RunExecutor {
	e.broadcaster = b
	return e
}

func (e *RunExecutor) WithWorkspaceSandboxEnsurer(ensurer phases.WorkspaceSandboxEnsurer) *RunExecutor {
	e.workspaceSandbox = ensurer
	return e
}

func (e *RunExecutor) WithAttachments(attachments []runner.Attachment) *RunExecutor {
	e.attachments = attachments
	return e
}

func (e *RunExecutor) WithCustomEnvironment(env map[string]string) *RunExecutor {
	e.customEnv = env
	return e
}

// workflowIdentityMeta extracts only server-authored workflow lineage injected
// by the workflow launcher. These values become part of the signed identity
// token and are not accepted from an external request body.
func workflowIdentityMeta(env map[string]string) map[string]string {
	meta := make(map[string]string, 7)
	for envKey, claimKey := range map[string]string{
		workflowExecutionEnv:  "workflowExecutionId",
		workflowNodeEnv:       "workflowNodeId",
		workflowAttemptEnv:    "workflowAttemptId",
		workflowExperimentEnv: "workflowExperimentId",
		workflowVariantEnv:    "workflowVariantId",
		workflowPromptHashEnv: "workflowPromptHash",
		personaIDEnv:          "persona_id",
	} {
		if value := strings.TrimSpace(env[envKey]); value != "" {
			meta[claimKey] = value
		}
	}
	return meta
}

// WithOnRunning registers a callback fired exactly once when the run
// transitions to RunStatusRunning. The spawn dispatcher uses this to
// release the startup slot so the next queued run can proceed.
func (e *RunExecutor) WithOnRunning(fn func()) *RunExecutor {
	e.onRunning = fn
	return e
}

// =============================================================================
// MAIN COORDINATOR
// =============================================================================

// Execute runs the full execution lifecycle. This is the thin coordinator
// that walks the phases package's seams in order; each phase function
// owns its own logic. The deferred phases.Finalize is the single terminal
// teardown seam — see the file header for the contract it enforces.
func (e *RunExecutor) Execute(ctx context.Context) {
	timeout := e.levers.Execution.DefaultTimeout
	if e.run != nil && e.run.ResolvedConfig != nil && e.run.ResolvedConfig.Timeout > 0 {
		timeout = e.run.ResolvedConfig.Timeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	go phases.RunHeartbeatLoop(ctx, e.heartbeatLoopInput())
	defer e.stopHeartbeat()
	defer e.finalize()

	if e.isResuming {
		e.emitSystem(ctx, "info", fmt.Sprintf("resuming from phase: %s", e.checkpoint.Phase))
	}

	// Step 1: Update to Starting (skip if resuming past).
	if !e.shouldSkipPhase(domain.RunPhaseInitializing) {
		if err := e.updateStatusToStarting(execCtx); err != nil {
			e.failWithError(execCtx, &domain.DatabaseError{
				Operation:   "update",
				EntityType:  "Run",
				EntityID:    e.run.ID.String(),
				Cause:       err,
				IsTransient: true,
			})
			return
		}
		e.advancePhase(execCtx, domain.RunPhaseInitializing)
	}
	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 2: Setup workspace.
	if !e.shouldSkipPhase(domain.RunPhaseSandboxCreating) || e.run.RunMode == domain.RunModeSandboxed {
		e.advancePhase(execCtx, domain.RunPhaseSandboxCreating)
	}
	setupOut, err := phases.SetupWorkspace(execCtx, phases.SetupWorkspaceInput{
		Deps:              e.deps(),
		Run:               e.run,
		Task:              e.task,
		Profile:           e.profile,
		Sandbox:           e.sandbox,
		RunStateRoot:      e.runStateRoot,
		ExistingSandboxID: e.sandboxID,
		ExistingWorkDir:   e.workDir,
	})
	if err != nil {
		e.failWithError(execCtx, err)
		phases.CleanupOnFailure(execCtx, e.deps(), e.run)
		return
	}
	if setupOut.SandboxID != nil {
		e.sandboxID = setupOut.SandboxID
	}
	if setupOut.WorkDir != "" {
		e.workDir = setupOut.WorkDir
	}
	if e.run.RunMode == domain.RunModeSandboxed && e.sandboxID != nil {
		e.mu.Lock()
		e.checkpoint = e.checkpoint.WithSandbox(*e.sandboxID, e.workDir)
		e.mu.Unlock()
		phases.SaveCheckpoint(execCtx, phases.SaveCheckpointInput{
			Deps: e.deps(), Checkpoint: e.checkpoint, Mu: &e.mu, RunID: e.run.ID,
		})
	}
	if e.run.ResolvedConfig != nil {
		sessionEnv, err := PrepareCodecSessionHome(e.runStateRoot, e.run.ID, e.run.ResolvedConfig.RunnerType)
		if err != nil {
			e.failWithError(execCtx, err)
			return
		}
		e.sessionEnv = sessionEnv
	}
	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 3: Acquire runner.
	e.advancePhase(execCtx, domain.RunPhaseRunnerAcquiring)
	var agentRunner runner.Runner
	if e.run.ResolvedConfig != nil && e.run.ResolvedConfig.PolicySnapshot != nil {
		// Snapshot execution owns availability checks and cross-runner
		// transitions so an unavailable initial runner can be recorded as a
		// skipped candidate rather than diverted through legacy fallback state.
		if e.runners != nil {
			agentRunner, _ = e.runners.Get(e.run.ResolvedConfig.RunnerType)
		}
	} else {
		agentRunner, err = phases.AcquireRunner(execCtx, phases.AcquireRunnerInput{
			Deps: e.deps(), Run: e.run, Profile: e.profile, Runners: e.runners,
		})
		if err != nil {
			e.failWithError(execCtx, err)
			phases.CleanupOnFailure(execCtx, e.deps(), e.run)
			return
		}
	}
	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 3.5: Generate identity token (before execution so it's in env).
	e.identityToken = phases.GenerateIdentityToken(execCtx, phases.GenerateIdentityTokenInput{
		Deps: e.deps(), Run: e.run, Profile: e.profile, Task: e.task, Secret: e.identitySecret,
		Meta:            workflowIdentityMeta(e.run.CustomEnv),
		AccountScopes:   e.run.OwnerScopes,
		RequestedScopes: e.run.RequestedScopes,
	})

	// Step 4: Execute, walking the preset chain on model-unavailable errors.
	e.advancePhase(execCtx, domain.RunPhaseExecuting)
	eventSink := e.createEventSink()
	defer eventSink.Close()

	out := phases.ExecuteWithModelFallback(execCtx, phases.ExecuteWithModelFallbackInput{
		ExecuteAgentInput: phases.ExecuteAgentInput{
			Deps:          e.deps(),
			Run:           e.run,
			Task:          e.task,
			Profile:       e.profile,
			Runner:        agentRunner,
			WorkingDir:    e.workDir,
			RunStateRoot:  e.runStateRoot,
			RunStateWrite: e.runStateWrite,
			SandboxID:     e.sandboxID,
			Prompt:        e.prompt,
			SystemPrompt:  e.systemPrompt,
			Attachments:   e.attachments,
			EnvVars:       e.MergedEnvVars(),
			EventSink:     eventSink,
			RunState:      e.runState,
			Mu:            &e.mu,
			ModelHealth:   e.modelHealth,
			Runners:       e.runners,
			OnRunning:     e.onRunning,
		},
	})
	e.result = out.Result
	e.execErr = out.ExecErr
	if out.RunState != nil {
		e.runState = out.RunState
		defer func() {
			if e.runState != nil {
				_ = e.runState.Close()
			}
		}()
	}

	// Park coordination (durable park/resume): if the agent parked mid-turn,
	// ParkRunFromAgent transitioned the run running→parked and terminated this
	// process — which is why execution returned. The park owns the lifecycle from
	// here, so skip result-handling and suppress finalize (do NOT tear down the
	// sandbox the wake will re-acquire, do NOT clobber the parked status).
	if e.detectParked(ctx) {
		return
	}

	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 5: Handle result.
	e.advancePhase(execCtx, domain.RunPhaseCollectingResults)
	resultOut := phases.HandleResult(execCtx, phases.HandleResultInput{
		Deps:      e.deps(),
		Run:       e.run,
		Result:    e.result,
		ExecErr:   e.execErr,
		Sandbox:   e.sandbox,
		SandboxID: e.sandboxID,
	})
	e.outcome = resultOut.Outcome
}

// =============================================================================
// COORDINATOR HELPERS
// =============================================================================

// deps returns the bundled dependency struct phase functions consume.
func (e *RunExecutor) deps() phases.Deps {
	return phases.Deps{
		Runs:              e.runs,
		Events:            e.events,
		Broadcaster:       e.broadcaster,
		Checkpoints:       e.checkpoints,
		Gate:              e.gate,
		Levers:            e.levers,
		WorkspaceSandbox:  e.workspaceSandbox,
		StructuredResults: e.structuredResults,
		Clock:             e.clock,
	}
}

// MergedEnvVars returns custom env vars merged with sandbox + identity.
// Sandbox and identity vars take precedence on key conflicts.
func (e *RunExecutor) MergedEnvVars() map[string]string {
	scope := ""
	if e.task != nil {
		scope = e.task.ScopePath
	}
	env := phases.AssembleRunEnv(phases.AssembleRunEnvInput{
		Custom:        e.customEnv,
		RunMode:       e.run.RunMode,
		SandboxID:     e.sandboxID,
		WorkDir:       e.workDir,
		ScopePath:     scope,
		IdentityToken: e.identityToken,
	})
	for key, value := range e.sessionEnv {
		if env == nil {
			env = make(map[string]string)
		}
		env[key] = value
	}
	return env
}

// shouldSkipPhase returns true if we're resuming and have already completed this phase.
func (e *RunExecutor) shouldSkipPhase(phase domain.RunPhase) bool {
	if !e.isResuming {
		return false
	}
	return phases.PhaseOrdinal(e.checkpoint.Phase) > phases.PhaseOrdinal(phase)
}

// advancePhase delegates to phases.AdvancePhase.
func (e *RunExecutor) advancePhase(ctx context.Context, phase domain.RunPhase) {
	phases.AdvancePhase(ctx, phases.AdvancePhaseInput{
		Deps:       e.deps(),
		Run:        e.run,
		Checkpoint: e.checkpoint,
		Mu:         &e.mu,
		Phase:      phase,
	})
}

// updateStatusToStarting flips the run to RunStatusStarting and broadcasts.
func (e *RunExecutor) updateStatusToStarting(ctx context.Context) error {
	now := e.deps().Now()
	e.run.Status = domain.RunStatusStarting
	e.run.StartedAt = &now
	e.run.UpdatedAt = now
	if err := e.runs.Update(ctx, e.run); err != nil {
		return err
	}
	if e.broadcaster != nil {
		e.broadcaster.BroadcastRunStatus(e.run)
	}
	return nil
}

// failWithError delegates to phases.FailWithError and stores the outcome.
func (e *RunExecutor) failWithError(ctx context.Context, err error) {
	out := phases.FailWithError(ctx, phases.FailWithErrorInput{
		Deps: e.deps(),
		Run:  e.run,
		Err:  err,
	})
	e.outcome = out.Outcome
}

// handleContextError delegates to phases.HandleContextError and stores
// the resulting outcome.
func (e *RunExecutor) handleContextError(ctx context.Context, err error) {
	sessionID := ""
	if e.result != nil {
		sessionID = e.result.SessionID
	}
	out := phases.HandleContextError(ctx, phases.HandleContextErrorInput{
		Deps:      e.deps(),
		Run:       e.run,
		Profile:   e.profile,
		SandboxID: e.sandboxID,
		Sandbox:   e.sandbox,
		SessionID: sessionID,
		Err:       err,
		Levers:    e.levers,
	})
	e.outcome = out.Outcome
}

// emitSystem is a thin wrapper around phases.EmitSystemEvent.
func (e *RunExecutor) emitSystem(ctx context.Context, level, message string) {
	phases.EmitSystemEvent(ctx, e.deps(), e.run.ID, level, message)
}

// =============================================================================
// HEARTBEAT
// =============================================================================

// heartbeatLoopInput builds the input struct phases.RunHeartbeatLoop consumes.
func (e *RunExecutor) heartbeatLoopInput() phases.HeartbeatLoopInput {
	return phases.HeartbeatLoopInput{
		Deps:        e.deps(),
		Run:         e.run,
		Checkpoint:  e.checkpoint,
		Mu:          &e.mu,
		Levers:      e.levers,
		Stop:        e.heartbeatStop,
		Done:        e.heartbeatDone,
		Checkpoints: e.checkpoints,
	}
}

// stopHeartbeat signals the heartbeat loop to stop and waits for it.
func (e *RunExecutor) stopHeartbeat() {
	close(e.heartbeatStop)
	<-e.heartbeatDone
}

// =============================================================================
// EVENT SINK + FINALIZE
// =============================================================================

// createEventSink picks the underlying sink (broadcaster, store, or no-op)
// and wraps it in the per-run emit.Gate.
func (e *RunExecutor) createEventSink() runner.EventSink {
	if e.gate != nil {
		return e.gate
	}
	var underlying runner.EventSink
	switch {
	case e.events != nil && e.broadcaster != nil:
		underlying = &broadcastingEventSink{
			store:       e.events,
			runID:       e.run.ID,
			broadcaster: e.broadcaster,
		}
	case e.events != nil:
		underlying = &eventStoreAdapter{store: e.events, runID: e.run.ID}
	default:
		underlying = &noOpEventSink{}
	}
	e.gate = emit.NewGate(underlying)
	return e.gate
}

// detectParked reports whether the run was parked mid-turn by re-reading its
// persisted status. On a positive detection it sets e.parked so the deferred
// finalize becomes a no-op (preserving the sandbox the wake re-acquires).
// Best-effort: a read error returns false so normal terminal handling proceeds.
func (e *RunExecutor) detectParked(ctx context.Context) bool {
	if e.runs == nil {
		return false
	}
	cur, err := e.runs.Get(ctx, e.run.ID)
	if err != nil || cur == nil {
		return false
	}
	if cur.Status == domain.RunStatusParked {
		e.parked = true
		return true
	}
	return false
}

// finalize delegates to phases.Finalize with idempotency-flag protection. It is
// a no-op for a parked run (the park preserves the sandbox and owns lifecycle).
func (e *RunExecutor) finalize() {
	if e.finalized || e.parked {
		return
	}
	e.finalized = true
	if e.run != nil && e.run.ResolvedConfig != nil {
		if err := CleanupCodecSessionHomeCredentials(e.runStateRoot, e.run.ID, e.run.ResolvedConfig.RunnerType); err != nil {
			e.emitSystem(context.Background(), "warn", "failed to clean run-scoped session credentials: "+err.Error())
		}
	}
	phases.Finalize(phases.FinalizeInput{
		Deps:      e.deps(),
		Run:       e.run,
		SandboxID: e.sandboxID,
		Sandbox:   e.sandbox,
	})
	if e.onTerminal != nil && e.run != nil && e.run.Status.IsTerminal() {
		e.onTerminal(e.run)
	}
}

// =============================================================================
// QUERY METHODS
// =============================================================================

// Outcome returns the execution outcome after Execute() completes.
func (e *RunExecutor) Outcome() domain.RunOutcome { return e.outcome }

// SandboxID returns the sandbox ID if one was created.
func (e *RunExecutor) SandboxID() *uuid.UUID { return e.sandboxID }

// WorkDir returns the working directory used for execution.
func (e *RunExecutor) WorkDir() string { return e.workDir }
