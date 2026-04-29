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

// ModelChainResolver and ModelHealthReporter are aliases for the canonical
// definitions in the phases package. Phases owns these interfaces so per-phase
// functions can reference them without an import cycle; the aliases keep
// orchestration callers compiling without per-site rewrites.
type (
	ModelChainResolver  = phases.ModelChainResolver
	ModelHealthReporter = phases.ModelHealthReporter
)

// RunExecutor handles the execution lifecycle of a single run. It is a
// thin coordinator: it owns shared per-run state and dispatches to phase
// functions. Phase logic does not live here.
type RunExecutor struct {
	// Dependencies
	runs        repository.RunRepository
	runners     runner.Registry
	sandbox     sandbox.Provider
	events      event.Store
	checkpoints repository.CheckpointRepository
	broadcaster phases.EventBroadcaster
	modelChains ModelChainResolver
	modelHealth ModelHealthReporter

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
	sandboxID *uuid.UUID
	workDir   string
	runState  *runstate.State

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

	// Recommendation extraction gate
	shouldQueueRecommendations func(*domain.Run) bool

	// Resumption state
	isResuming bool

	// finalized guards the deferred phases.Finalize seam against re-entry.
	finalized bool

	// Caller-provided env vars
	customEnv map[string]string

	// Identity token state
	identitySecret []byte
	identityToken  string
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
		shouldQueueRecommendations: func(run *domain.Run) bool {
			return run.IsInvestigationRun()
		},
	}
}

// =============================================================================
// BUILDER METHODS
// =============================================================================

func (e *RunExecutor) WithLevers(l config.Levers) *RunExecutor { e.levers = l; return e }

func (e *RunExecutor) WithCheckpointRepository(repo repository.CheckpointRepository) *RunExecutor {
	e.checkpoints = repo
	return e
}

func (e *RunExecutor) WithModelChainResolver(resolver ModelChainResolver) *RunExecutor {
	e.modelChains = resolver
	return e
}

func (e *RunExecutor) WithModelHealthReporter(reporter ModelHealthReporter) *RunExecutor {
	e.modelHealth = reporter
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

func (e *RunExecutor) WithRecommendationQueueFilter(filter func(*domain.Run) bool) *RunExecutor {
	if filter != nil {
		e.shouldQueueRecommendations = filter
	}
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

func (e *RunExecutor) WithAttachments(attachments []runner.Attachment) *RunExecutor {
	e.attachments = attachments
	return e
}

func (e *RunExecutor) WithCustomEnvironment(env map[string]string) *RunExecutor {
	e.customEnv = env
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
	execCtx, cancel := context.WithTimeout(ctx, e.levers.Execution.DefaultTimeout)
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
	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 3: Acquire runner.
	e.advancePhase(execCtx, domain.RunPhaseRunnerAcquiring)
	agentRunner, err := phases.AcquireRunner(execCtx, phases.AcquireRunnerInput{
		Deps: e.deps(), Run: e.run, Profile: e.profile, Runners: e.runners,
	})
	if err != nil {
		e.failWithError(execCtx, err)
		phases.CleanupOnFailure(execCtx, e.deps(), e.run)
		return
	}
	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 3.5: Generate identity token (before execution so it's in env).
	e.identityToken = phases.GenerateIdentityToken(execCtx, phases.GenerateIdentityTokenInput{
		Deps: e.deps(), Run: e.run, Profile: e.profile, Task: e.task, Secret: e.identitySecret,
	})

	// Step 4: Execute, walking the preset chain on model-unavailable errors.
	e.advancePhase(execCtx, domain.RunPhaseExecuting)
	eventSink := e.createEventSink()
	defer eventSink.Close()

	out := phases.ExecuteWithModelFallback(execCtx, phases.ExecuteWithModelFallbackInput{
		ExecuteAgentInput: phases.ExecuteAgentInput{
			Deps:         e.deps(),
			Run:          e.run,
			Task:         e.task,
			Profile:      e.profile,
			Runner:       agentRunner,
			WorkingDir:   e.workDir,
			SandboxID:    e.sandboxID,
			Prompt:       e.prompt,
			SystemPrompt: e.systemPrompt,
			Attachments:  e.attachments,
			EnvVars:      e.MergedEnvVars(),
			EventSink:    eventSink,
			RunState:     e.runState,
			Mu:           &e.mu,
			ModelHealth:  e.modelHealth,
			ModelChains:  e.modelChains,
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

	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 5: Handle result.
	e.advancePhase(execCtx, domain.RunPhaseCollectingResults)
	resultOut := phases.HandleResult(execCtx, phases.HandleResultInput{
		Deps:                       e.deps(),
		Run:                        e.run,
		Result:                     e.result,
		ExecErr:                    e.execErr,
		Sandbox:                    e.sandbox,
		SandboxID:                  e.sandboxID,
		ShouldQueueRecommendations: e.shouldQueueRecommendations,
	})
	e.outcome = resultOut.Outcome
}

// =============================================================================
// COORDINATOR HELPERS
// =============================================================================

// deps returns the bundled dependency struct phase functions consume.
func (e *RunExecutor) deps() phases.Deps {
	return phases.Deps{
		Runs:        e.runs,
		Events:      e.events,
		Broadcaster: e.broadcaster,
		Checkpoints: e.checkpoints,
		Gate:        e.gate,
		Levers:      e.levers,
	}
}

// MergedEnvVars returns custom env vars merged with sandbox + identity.
// Sandbox and identity vars take precedence on key conflicts.
func (e *RunExecutor) MergedEnvVars() map[string]string {
	scope := ""
	if e.task != nil {
		scope = e.task.ScopePath
	}
	return phases.MergeEnvVars(phases.MergeEnvInput{
		Custom: e.customEnv,
		Sandbox: phases.SandboxEnvVars(phases.SandboxEnvInput{
			RunMode:   e.run.RunMode,
			SandboxID: e.sandboxID,
			WorkDir:   e.workDir,
			ScopePath: scope,
		}),
		Identity: phases.IdentityEnvVars(e.identityToken),
	})
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
	now := time.Now()
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
	case e.broadcaster != nil:
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

// finalize delegates to phases.Finalize with idempotency-flag protection.
func (e *RunExecutor) finalize() {
	if e.finalized {
		return
	}
	e.finalized = true
	phases.Finalize(phases.FinalizeInput{
		Deps:      e.deps(),
		Run:       e.run,
		SandboxID: e.sandboxID,
		Sandbox:   e.sandbox,
	})
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
