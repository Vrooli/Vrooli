// Package orchestration provides the core orchestration service for agent-manager.
//
// This file contains the RunExecutor which handles the lifecycle of a single
// run execution. It is extracted from the main service to reduce cognitive load
// and make the execution flow explicit and testable.
//
// EXECUTION FLOW:
//   1. UpdateStatusToStarting()
//   2. SetupWorkspace() - creates sandbox if needed
//   3. AcquireRunner() - gets and validates the runner
//   4. Execute() - runs the agent
//   5. HandleResult() - processes the outcome
//   6. Cleanup() - releases resources (on failure)
//
// GRACEFUL DEGRADATION:
// The executor is designed to fail safely and preserve useful state:
// - Sandbox is preserved on failure for inspection
// - Events are flushed before marking failure
// - Errors are classified for actionable recovery hints
// - Partial work is captured in run summary
//
// RESILIENCE PATTERNS (see architectural guides):
// - Idempotency: Operations are idempotent via checkpoint tracking
// - Temporal Flow: Heartbeats, timeouts, and cancellation propagation
// - Progress Continuity: Checkpoints enable safe interruption and resumption

package orchestration

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

// ExecutorConfig holds configuration for run execution.
type ExecutorConfig struct {
	// Timeout is the maximum execution time
	Timeout time.Duration

	// HeartbeatInterval is how often to update heartbeat
	HeartbeatInterval time.Duration

	// CheckpointInterval is how often to save checkpoints
	CheckpointInterval time.Duration

	// MaxRetries is the maximum retries per phase
	MaxRetries int

	// StaleThreshold is how long without heartbeat before considering stale
	StaleThreshold time.Duration
}

// DefaultExecutorConfig returns sensible defaults for execution.
// HeartbeatInterval is set to 15s to ensure multiple heartbeats before stale threshold.
// StaleThreshold is 5 minutes to allow for slow database updates or long tool calls.
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		Timeout:            30 * time.Minute,
		HeartbeatInterval:  15 * time.Second, // More frequent heartbeats for reliability
		CheckpointInterval: 1 * time.Minute,
		MaxRetries:         3,
		StaleThreshold:     5 * time.Minute, // More forgiving threshold
	}
}

// RunExecutor handles the execution lifecycle of a single run.
// It encapsulates all the steps needed to execute an agent run,
// making the flow explicit and each step independently testable.
//
// RESILIENCE FEATURES:
// - Checkpoints: Saves progress at each phase transition
// - Heartbeats: Regular updates to detect stalled runs
// - Timeout handling: Enforces maximum execution time
// - Cancellation: Responds to context cancellation
// - Resumption: Can resume from last checkpoint after interruption
type RunExecutor struct {
	// Dependencies
	runs        repository.RunRepository
	runners     runner.Registry
	sandbox     sandbox.Provider
	events      event.Store
	checkpoints repository.CheckpointRepository // optional: for checkpoint persistence
	broadcaster EventBroadcaster                // optional: for real-time WebSocket updates

	// Configuration
	config ExecutorConfig

	// Execution context
	run     *domain.Run
	task    *domain.Task
	profile *domain.AgentProfile
	prompt  string

	// Workspace state
	sandboxID *uuid.UUID
	workDir   string
	lockID    *uuid.UUID

	// Progress tracking
	checkpoint *domain.RunCheckpoint
	mu         sync.Mutex // protects checkpoint updates

	// Heartbeat management
	heartbeatStop chan struct{}
	heartbeatDone chan struct{}

	// Result state
	outcome domain.RunOutcome
	result  *runner.ExecuteResult
	execErr error

	// Resumption state
	isResuming bool

	// Sandbox finalization state
	sandboxFinalized bool
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
		config:        DefaultExecutorConfig(),
		checkpoint:    domain.NewCheckpoint(run.ID, domain.RunPhaseQueued),
		heartbeatStop: make(chan struct{}),
		heartbeatDone: make(chan struct{}),
	}
}

// WithConfig sets custom executor configuration.
func (e *RunExecutor) WithConfig(config ExecutorConfig) *RunExecutor {
	e.config = config
	return e
}

// WithCheckpointRepository enables checkpoint persistence.
func (e *RunExecutor) WithCheckpointRepository(repo repository.CheckpointRepository) *RunExecutor {
	e.checkpoints = repo
	return e
}

// WithExistingSandbox reuses an existing sandbox for this run.
func (e *RunExecutor) WithExistingSandbox(sandboxID uuid.UUID, workDir string) *RunExecutor {
	e.sandboxID = &sandboxID
	if workDir != "" {
		e.workDir = workDir
	}
	e.checkpoint = e.checkpoint.WithSandbox(sandboxID, workDir)
	return e
}

// WithResumeFrom configures the executor to resume from a checkpoint.
func (e *RunExecutor) WithResumeFrom(checkpoint *domain.RunCheckpoint) *RunExecutor {
	e.checkpoint = checkpoint
	e.isResuming = true
	// Restore state from checkpoint
	if checkpoint.SandboxID != nil {
		e.sandboxID = checkpoint.SandboxID
		e.workDir = checkpoint.WorkDir
	}
	if checkpoint.LockID != nil {
		e.lockID = checkpoint.LockID
	}
	return e
}

// WithBroadcaster sets the event broadcaster for real-time updates.
func (e *RunExecutor) WithBroadcaster(b EventBroadcaster) *RunExecutor {
	e.broadcaster = b
	return e
}

// Execute runs the full execution lifecycle.
// This is the main entry point that orchestrates all steps.
//
// GRACEFUL DEGRADATION: Each step is wrapped with proper error handling.
// On failure, we capture the error with full context and preserve
// the sandbox for inspection.
//
// RESILIENCE:
// - Context cancellation is propagated to all steps
// - Timeout is enforced via context deadline
// - Heartbeats run in background to detect stalls
// - Checkpoints are saved at each phase transition
// - Resumption skips already-completed phases
func (e *RunExecutor) Execute(ctx context.Context) {
	// Apply timeout to context
	execCtx, cancel := context.WithTimeout(ctx, e.config.Timeout)
	defer cancel()

	// Start heartbeat goroutine
	go e.heartbeatLoop(ctx)
	defer e.stopHeartbeat()

	// Determine starting phase (for resumption)
	startPhase := e.checkpoint.Phase
	if e.isResuming {
		e.emitSystemEvent(ctx, "info", fmt.Sprintf("resuming from phase: %s", startPhase))
	}

	// Step 1: Update to starting (skip if resuming past this phase)
	if !e.shouldSkipPhase(domain.RunPhaseInitializing) {
		if err := e.updateStatusToStarting(execCtx); err != nil {
			e.failWithError(execCtx, &domain.DatabaseError{
				Operation:   "update",
				EntityType:  "Run",
				EntityID:    e.run.ID.String(),
				Cause:       err,
				IsTransient: true, // Status updates are retryable
			})
			return
		}
		e.advancePhase(execCtx, domain.RunPhaseInitializing)
	}

	// Check for cancellation between phases
	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 2: Setup workspace
	if e.run.RunMode == domain.RunModeSandboxed {
		if e.sandboxID == nil {
			e.advancePhase(execCtx, domain.RunPhaseSandboxCreating)
			if err := e.setupWorkspace(execCtx); err != nil {
				// setupWorkspace already returns domain errors
				e.failWithError(execCtx, err)
				e.cleanupOnFailure(execCtx)
				return
			}
		} else {
			if !e.shouldSkipPhase(domain.RunPhaseSandboxCreating) {
				e.advancePhase(execCtx, domain.RunPhaseSandboxCreating)
			}
			if e.workDir == "" && e.sandbox != nil {
				if workDir, err := e.sandbox.GetWorkspacePath(execCtx, *e.sandboxID); err == nil {
					e.workDir = workDir
				}
			}
			if e.workDir == "" {
				e.failWithError(execCtx, domain.NewValidationErrorWithHint("sandboxId", "sandbox workdir not available",
					"ensure the sandbox is active and has a workdir"))
				e.cleanupOnFailure(execCtx)
				return
			}
			e.emitSystemEvent(ctx, "info", "reusing existing sandbox")
		}
	} else {
		if !e.shouldSkipPhase(domain.RunPhaseSandboxCreating) {
			e.advancePhase(execCtx, domain.RunPhaseSandboxCreating)
		}
		if err := e.useInPlaceWorkspace(); err != nil {
			e.failWithError(execCtx, err)
			e.cleanupOnFailure(execCtx)
			return
		}
	}

	// Check for cancellation between phases
	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 3: Acquire runner
	e.advancePhase(execCtx, domain.RunPhaseRunnerAcquiring)
	agentRunner, err := e.acquireRunner(execCtx)
	if err != nil {
		// acquireRunner already returns domain errors
		e.failWithError(execCtx, err)
		e.cleanupOnFailure(execCtx)
		return
	}

	// Check for cancellation between phases
	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 4: Execute agent
	e.advancePhase(execCtx, domain.RunPhaseExecuting)
	e.executeAgent(execCtx, agentRunner)

	// Check for timeout or cancellation
	if err := execCtx.Err(); err != nil {
		e.handleContextError(ctx, err)
		return
	}

	// Step 5: Handle result
	e.advancePhase(execCtx, domain.RunPhaseCollectingResults)
	e.handleResult(execCtx)
}

// =============================================================================
// PHASE MANAGEMENT & CHECKPOINTING
// =============================================================================

// shouldSkipPhase returns true if we're resuming and have already completed this phase.
func (e *RunExecutor) shouldSkipPhase(phase domain.RunPhase) bool {
	if !e.isResuming {
		return false
	}
	// Compare phase ordinals
	return phaseOrdinal(e.checkpoint.Phase) > phaseOrdinal(phase)
}

// phaseOrdinal returns the numeric order of a phase for comparison.
func phaseOrdinal(phase domain.RunPhase) int {
	switch phase {
	case domain.RunPhaseQueued:
		return 0
	case domain.RunPhaseInitializing:
		return 1
	case domain.RunPhaseSandboxCreating:
		return 2
	case domain.RunPhaseRunnerAcquiring:
		return 3
	case domain.RunPhaseExecuting:
		return 4
	case domain.RunPhaseCollectingResults:
		return 5
	case domain.RunPhaseAwaitingReview:
		return 6
	case domain.RunPhaseApplying:
		return 7
	case domain.RunPhaseCleaningUp:
		return 8
	case domain.RunPhaseCompleted:
		return 9
	default:
		return 0
	}
}

// advancePhase updates the checkpoint to a new phase and persists it.
func (e *RunExecutor) advancePhase(ctx context.Context, phase domain.RunPhase) {
	e.mu.Lock()
	e.checkpoint = e.checkpoint.Update(phase, 0)
	e.run.UpdateProgress(phase, domain.PhaseToProgress(phase))
	e.mu.Unlock()

	// Persist checkpoint if repository is available
	e.saveCheckpoint(ctx)

	// Update run in database
	if err := e.runs.Update(ctx, e.run); err != nil {
		e.emitSystemEvent(ctx, "warn", "failed to persist phase update: "+err.Error())
	}

	// Emit phase change event
	e.emitSystemEvent(ctx, "info", fmt.Sprintf("phase: %s", phase.Description()))
}

// saveCheckpoint persists the current checkpoint if a repository is configured.
func (e *RunExecutor) saveCheckpoint(ctx context.Context) {
	if e.checkpoints == nil {
		return
	}

	e.mu.Lock()
	cp := *e.checkpoint // copy
	e.mu.Unlock()

	if err := e.checkpoints.Save(ctx, &cp); err != nil {
		// Log but don't fail - checkpoint is best-effort
		e.emitSystemEvent(ctx, "warn", "failed to save checkpoint: "+err.Error())
	}
}

// =============================================================================
// HEARTBEAT MANAGEMENT
// =============================================================================

// heartbeatLoop sends periodic heartbeats to indicate the run is still active.
func (e *RunExecutor) heartbeatLoop(ctx context.Context) {
	defer close(e.heartbeatDone)

	log.Printf("[heartbeat] Starting heartbeat loop for run %s (tag=%s, interval=%v)",
		e.run.ID, e.run.GetTag(), e.config.HeartbeatInterval)

	// Send initial heartbeat immediately
	e.sendHeartbeat(ctx)

	ticker := time.NewTicker(e.config.HeartbeatInterval)
	defer ticker.Stop()

	heartbeatCount := 1
	for {
		select {
		case <-e.heartbeatStop:
			log.Printf("[heartbeat] Stopping heartbeat loop for run %s (sent %d heartbeats)",
				e.run.ID, heartbeatCount)
			return
		case <-ctx.Done():
			log.Printf("[heartbeat] Context cancelled for run %s (sent %d heartbeats)",
				e.run.ID, heartbeatCount)
			return
		case <-ticker.C:
			heartbeatCount++
			e.sendHeartbeat(ctx)
		}
	}
}

// sendHeartbeat updates the run's last heartbeat time.
func (e *RunExecutor) sendHeartbeat(ctx context.Context) {
	e.mu.Lock()
	now := time.Now()
	e.run.LastHeartbeat = &now
	e.checkpoint.LastHeartbeat = now
	runID := e.run.ID
	tag := e.run.GetTag()
	e.mu.Unlock()

	// Update run in database (best-effort)
	if err := e.runs.Update(ctx, e.run); err != nil {
		log.Printf("[heartbeat] ERROR: Failed to update heartbeat for run %s (tag=%s): %v",
			runID, tag, err)
		e.emitSystemEvent(ctx, "warn", "heartbeat update failed: "+err.Error())
	} else {
		log.Printf("[heartbeat] DEBUG: Updated heartbeat for run %s (tag=%s) at %v",
			runID, tag, now.Format(time.RFC3339))
	}

	// Update checkpoint in database (best-effort)
	if e.checkpoints != nil {
		if err := e.checkpoints.Heartbeat(ctx, e.run.ID); err != nil {
			e.emitSystemEvent(ctx, "warn", "heartbeat checkpoint failed: "+err.Error())
		}
	}
}

// stopHeartbeat signals the heartbeat loop to stop.
func (e *RunExecutor) stopHeartbeat() {
	close(e.heartbeatStop)
	<-e.heartbeatDone
}

// =============================================================================
// CONTEXT ERROR HANDLING
// =============================================================================

// handleContextError handles context cancellation or timeout.
func (e *RunExecutor) handleContextError(ctx context.Context, err error) {
	if err == context.DeadlineExceeded {
		e.failWithError(ctx, &domain.RunnerError{
			RunnerType:  e.getRunnerType(),
			Operation:   "timeout",
			Cause:       errors.New(fmt.Sprintf("execution exceeded timeout of %v", e.config.Timeout)),
			IsTransient: false,
		})
		e.outcome = domain.RunOutcomeTimeout
	} else if err == context.Canceled {
		// Graceful cancellation - not an error
		e.emitSystemEvent(ctx, "info", "execution cancelled")
		e.outcome = domain.RunOutcomeCancelled
		now := time.Now()
		e.run.Status = domain.RunStatusCancelled
		e.run.EndedAt = &now
		e.run.UpdatedAt = now
		if updateErr := e.runs.Update(ctx, e.run); updateErr != nil {
			e.emitSystemEvent(ctx, "warn", "failed to persist cancellation: "+updateErr.Error())
		}
	}

	e.cleanupOnFailure(ctx)
}

// =============================================================================
// STEP 1: Update Status to Starting
// =============================================================================

func (e *RunExecutor) updateStatusToStarting(ctx context.Context) error {
	now := time.Now()
	e.run.Status = domain.RunStatusStarting
	e.run.StartedAt = &now
	e.run.UpdatedAt = now
	return e.runs.Update(ctx, e.run)
}

// =============================================================================
// STEP 2: Setup Workspace
// =============================================================================

func (e *RunExecutor) setupWorkspace(ctx context.Context) error {
	if e.run.RunMode == domain.RunModeSandboxed {
		return e.createSandboxWorkspace(ctx)
	}
	return e.useInPlaceWorkspace()
}

func (e *RunExecutor) createSandboxWorkspace(ctx context.Context) error {
	if e.sandbox == nil {
		return domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}

	// Use idempotency key to allow safe retries of sandbox creation
	idempotencyKey := fmt.Sprintf("sandbox:run:%s", e.run.ID.String())

	sbx, err := e.sandbox.Create(ctx, sandbox.CreateRequest{
		ScopePath:      e.task.ScopePath,
		NoLock:         e.run.SandboxConfig != nil && e.run.SandboxConfig.NoLock,
		ProjectRoot:    e.task.ProjectRoot,
		Owner:          e.run.ID.String(),
		OwnerType:      "run",
		IdempotencyKey: idempotencyKey,
		Behavior:       e.run.SandboxConfig,
	})
	if err != nil {
		if _, ok := err.(*domain.SandboxError); ok {
			return err
		}
		return &domain.SandboxError{
			Operation:   "create",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}

	e.sandboxID = &sbx.ID
	e.run.SandboxID = e.sandboxID
	if err := e.runs.Update(ctx, e.run); err != nil {
		return &domain.DatabaseError{
			Operation:   "update",
			EntityType:  "Run",
			EntityID:    e.run.ID.String(),
			Cause:       err,
			IsTransient: true,
		}
	}

	workDir, err := e.sandbox.GetWorkspacePath(ctx, sbx.ID)
	if err != nil {
		return &domain.SandboxError{
			SandboxID:   e.sandboxID,
			Operation:   "get_workspace_path",
			Cause:       err,
			IsTransient: true,
			CanRetry:    true,
		}
	}
	e.workDir = workDir

	// Update checkpoint with sandbox information for resumption
	e.mu.Lock()
	e.checkpoint = e.checkpoint.WithSandbox(sbx.ID, workDir)
	e.mu.Unlock()
	e.saveCheckpoint(ctx)

	return nil
}

func (e *RunExecutor) useInPlaceWorkspace() error {
	if e.task.ProjectRoot == "" {
		return domain.NewValidationErrorWithHint(
			"projectRoot",
			"project root is required for in-place execution",
			"Specify projectRoot in the task or use sandboxed run mode",
		)
	}
	e.workDir = e.task.ProjectRoot
	return nil
}

// =============================================================================
// STEP 3: Acquire Runner
// =============================================================================

func (e *RunExecutor) acquireRunner(ctx context.Context) (runner.Runner, error) {
	// Get runner type from profile or resolved config
	runnerType := e.getRunnerType()

	if e.runners == nil {
		return nil, &domain.RunnerError{
			RunnerType:  runnerType,
			Operation:   "acquire",
			Cause:       domain.NewConfigMissingError("runnerRegistry", "not configured", nil),
			IsTransient: false,
		}
	}

	r, err := e.runners.Get(runnerType)
	if err != nil {
		if fallback := e.tryFallbackRunner(ctx, runnerType); fallback != nil {
			return fallback, nil
		}
		alternative := e.findFallbackAlternative(runnerType)
		return nil, &domain.RunnerError{
			RunnerType:  runnerType,
			Operation:   "acquire",
			Cause:       err,
			IsTransient: false,
			Alternative: alternative,
		}
	}

	// Verify runner is available
	available, msg := r.IsAvailable(ctx)
	if !available {
		if fallback := e.tryFallbackRunner(ctx, runnerType); fallback != nil {
			return fallback, nil
		}
		alternative := e.findFallbackAlternative(runnerType)
		return nil, &domain.RunnerError{
			RunnerType:  runnerType,
			Operation:   "availability_check",
			Cause:       errors.New(msg),
			IsTransient: true, // runner might become available
			Alternative: alternative,
		}
	}

	return r, nil
}

// getRunnerType returns the runner type, preferring profile but falling back to resolved config.
func (e *RunExecutor) getRunnerType() domain.RunnerType {
	if e.profile != nil {
		return e.profile.RunnerType
	}
	if e.run != nil && e.run.ResolvedConfig != nil {
		return e.run.ResolvedConfig.RunnerType
	}
	return domain.RunnerTypeClaudeCode // Default fallback
}

// findAlternativeRunner attempts to find another available runner.
// Returns the runner type as a string, or empty string if none available.
func (e *RunExecutor) findAlternativeRunner() string {
	if e.runners == nil {
		return ""
	}

	// Common runner types to check
	alternatives := []domain.RunnerType{
		domain.RunnerTypeClaudeCode,
		domain.RunnerTypeCodex,
		domain.RunnerTypeOpenCode,
	}

	currentType := e.getRunnerType()
	for _, rt := range alternatives {
		if rt == currentType {
			continue // Skip the one that failed
		}
		if r, err := e.runners.Get(rt); err == nil {
			if available, _ := r.IsAvailable(context.Background()); available {
				return string(rt)
			}
		}
	}

	return ""
}

func (e *RunExecutor) runnerFallbackCandidates(primary domain.RunnerType) []domain.RunnerType {
	if e.run == nil || e.run.ResolvedConfig == nil || len(e.run.ResolvedConfig.FallbackRunnerTypes) == 0 {
		return nil
	}
	seen := make(map[domain.RunnerType]struct{}, len(e.run.ResolvedConfig.FallbackRunnerTypes))
	candidates := make([]domain.RunnerType, 0, len(e.run.ResolvedConfig.FallbackRunnerTypes))
	for _, rt := range e.run.ResolvedConfig.FallbackRunnerTypes {
		if !rt.IsValid() || rt == primary {
			continue
		}
		if _, exists := seen[rt]; exists {
			continue
		}
		seen[rt] = struct{}{}
		candidates = append(candidates, rt)
	}
	return candidates
}

func (e *RunExecutor) findFallbackAlternative(primary domain.RunnerType) string {
	if e.runners == nil {
		return ""
	}
	for _, rt := range e.runnerFallbackCandidates(primary) {
		if r, err := e.runners.Get(rt); err == nil {
			if available, _ := r.IsAvailable(context.Background()); available {
				return string(rt)
			}
		}
	}
	return e.findAlternativeRunner()
}

func (e *RunExecutor) tryFallbackRunner(ctx context.Context, primary domain.RunnerType) runner.Runner {
	if e.runners == nil {
		return nil
	}
	for _, rt := range e.runnerFallbackCandidates(primary) {
		r, err := e.runners.Get(rt)
		if err != nil {
			continue
		}
		available, _ := r.IsAvailable(ctx)
		if !available {
			continue
		}
		e.applyRunnerFallback(ctx, primary, rt)
		return r
	}
	return nil
}

func (e *RunExecutor) applyRunnerFallback(ctx context.Context, from, to domain.RunnerType) {
	if e.run == nil {
		return
	}
	if e.run.ResolvedConfig == nil {
		e.run.ResolvedConfig = domain.DefaultRunConfig()
	}
	e.run.ResolvedConfig.RunnerType = to
	e.run.UpdatedAt = time.Now()
	if err := e.runs.Update(ctx, e.run); err != nil {
		e.emitSystemEvent(ctx, "warn", "failed to persist runner fallback: "+err.Error())
	}
	e.emitSystemEvent(ctx, "warn", fmt.Sprintf("runner fallback: %s -> %s", from, to))
}

// =============================================================================
// STEP 4: Execute Agent
// =============================================================================

func (e *RunExecutor) executeAgent(ctx context.Context, r runner.Runner) {
	// Update status to running
	e.run.Status = domain.RunStatusRunning
	e.run.UpdatedAt = time.Now()
	if err := e.runs.Update(ctx, e.run); err != nil {
		e.emitSystemEvent(ctx, "warn", "failed to persist run start: "+err.Error())
	}

	// Create event sink
	eventSink := e.createEventSink()
	defer eventSink.Close()

	// Build execution request
	req := runner.ExecuteRequest{
		RunID:          e.run.ID,
		Tag:            e.run.GetTag(), // Custom tag or defaults to ID
		Profile:        e.profile,
		ResolvedConfig: e.run.ResolvedConfig, // Merged config from profile + inline
		Task:           e.task,
		WorkingDir:     e.workDir,
		Prompt:         e.prompt,
		EventSink:      eventSink,
	}

	// Execute
	e.result, e.execErr = r.Execute(ctx, req)
}

func (e *RunExecutor) createEventSink() runner.EventSink {
	// If we have a broadcaster, use the broadcasting sink for real-time updates
	if e.broadcaster != nil {
		return &broadcastingEventSink{
			store:       e.events,
			runID:       e.run.ID,
			broadcaster: e.broadcaster,
		}
	}
	// Fallback to just storing events
	if e.events != nil {
		return &eventStoreAdapter{store: e.events, runID: e.run.ID}
	}
	return &noOpEventSink{}
}

// =============================================================================
// STEP 5: Handle Result
// =============================================================================

func (e *RunExecutor) handleResult(ctx context.Context) {
	// Classify the outcome using the domain decision helper
	e.outcome = e.classifyOutcome()

	// Update run based on outcome
	now := time.Now()
	e.run.EndedAt = &now
	e.run.UpdatedAt = now

	switch {
	case e.outcome.RequiresReview():
		e.handleSuccessfulCompletion(ctx)
	case e.outcome.IsTerminalFailure():
		e.handleFailure(ctx)
	case e.outcome == domain.RunOutcomeCancelled:
		e.handleCancellation(ctx)
	default:
		e.handleFailure(ctx) // Fallback
	}

	if err := e.runs.Update(ctx, e.run); err != nil {
		e.emitSystemEvent(ctx, "warn", "failed to persist run result: "+err.Error())
	}
}

func (e *RunExecutor) classifyOutcome() domain.RunOutcome {
	var exitCode *int
	if e.result != nil {
		exitCode = &e.result.ExitCode
	}

	return domain.ClassifyRunOutcome(
		e.execErr,
		exitCode,
		false, // wasCancelled - would be set by StopRun
		false, // timedOut - would be detected during execution
	)
}

func (e *RunExecutor) handleSuccessfulCompletion(ctx context.Context) {
	// Check if approval is required based on resolved config
	requiresApproval := true // default to requiring approval for safety
	if e.run.ResolvedConfig != nil {
		requiresApproval = e.run.ResolvedConfig.RequiresApproval
	}

	if e.result != nil {
		e.run.Summary = e.result.Summary
		e.run.ExitCode = &e.result.ExitCode
		if e.result.SessionID != "" {
			e.run.SessionID = e.result.SessionID
		}
	}

	if err := e.validateExpectedOutputs(ctx); err != nil {
		e.failWithError(ctx, err)
		e.outcome = domain.RunOutcomeException
		e.applySandboxLifecycle(ctx, domain.SandboxLifecycleRunFailed, "expected outputs missing")
		return
	}

	autoApplied := false
	if requiresApproval {
		autoApplied = e.tryAutoApproval(ctx)
		if !autoApplied {
			e.run.Status = domain.RunStatusNeedsReview
			e.run.ApprovalState = domain.ApprovalStatePending
		}
	} else {
		// Skip approval workflow - mark as complete directly
		e.run.Status = domain.RunStatusComplete
		e.run.ApprovalState = domain.ApprovalStateNone
	}

	e.applySandboxLifecycle(ctx, domain.SandboxLifecycleRunCompleted, "run completed")
}

func (e *RunExecutor) validateExpectedOutputs(ctx context.Context) error {
	if e.task == nil {
		return nil
	}
	expectedPaths := extractExpectedPaths(e.task.Title, e.task.Description)
	if len(expectedPaths) == 0 {
		return nil
	}
	if e.run.RunMode != domain.RunModeSandboxed {
		return nil
	}
	if e.sandbox == nil || e.sandboxID == nil {
		return &domain.ValidationError{
			Field:   "expected_files",
			Message: "expected outputs specified but sandbox diff unavailable",
			Hint:    "ensure sandboxed runs have an active sandbox before validation",
		}
	}

	diff, err := e.sandbox.GetDiff(ctx, *e.sandboxID)
	if err != nil {
		return fmt.Errorf("expected output validation failed: %w", err)
	}

	changed := make(map[string]struct{}, len(diff.Files))
	for _, file := range diff.Files {
		path := filepath.ToSlash(filepath.Clean(file.FilePath))
		changed[path] = struct{}{}
	}

	projectRoot := strings.TrimSpace(e.task.ProjectRoot)
	missing := make([]string, 0, len(expectedPaths))
	for _, expected := range expectedPaths {
		normalized := filepath.ToSlash(filepath.Clean(expected))
		if projectRoot != "" && filepath.IsAbs(normalized) {
			if rel, err := filepath.Rel(projectRoot, normalized); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				normalized = filepath.ToSlash(filepath.Clean(rel))
			}
		}
		if _, ok := changed[normalized]; !ok {
			missing = append(missing, normalized)
		}
	}

	if len(missing) > 0 {
		return &domain.ValidationError{
			Field:   "expected_files",
			Message: fmt.Sprintf("expected outputs missing: %s", strings.Join(missing, ", ")),
			Hint:    "verify the agent created the requested files in the sandbox",
		}
	}
	return nil
}

func extractExpectedPaths(parts ...string) []string {
	content := strings.Join(parts, "\n")
	if strings.TrimSpace(content) == "" {
		return nil
	}

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bcreate (?:a |an )?(?:new )?(?:text )?file at ([^\s"'` + "`" + `]+)`),
		regexp.MustCompile("`([^`]+)`"),
	}

	seen := make(map[string]struct{})
	var results []string
	for _, pattern := range patterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			path := strings.TrimSpace(match[1])
			path = strings.TrimRight(path, ".,;:")
			if path == "" {
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			results = append(results, path)
		}
	}
	return results
}

func (e *RunExecutor) handleFailure(ctx context.Context) {
	e.run.Status = domain.RunStatusFailed

	if e.execErr != nil {
		e.run.ErrorMsg = e.execErr.Error()
	} else if e.result != nil && e.result.ErrorMessage != "" {
		e.run.ErrorMsg = e.result.ErrorMessage
		e.run.ExitCode = &e.result.ExitCode
	}

	e.applySandboxLifecycle(ctx, domain.SandboxLifecycleRunFailed, "run failed")
}

func (e *RunExecutor) handleCancellation(ctx context.Context) {
	e.run.Status = domain.RunStatusCancelled
	e.applySandboxLifecycle(ctx, domain.SandboxLifecycleRunCancelled, "run cancelled")
}

// =============================================================================
// ERROR HANDLING & GRACEFUL DEGRADATION
// =============================================================================

// failWithError marks the run as failed with proper error classification.
// This is the central failure handler that ensures:
// - Errors are captured with full context
// - Events are preserved (sandbox not deleted)
// - Failure reason is machine-readable
func (e *RunExecutor) failWithError(ctx context.Context, err error) {
	now := time.Now()
	e.run.Status = domain.RunStatusFailed
	e.run.EndedAt = &now
	e.run.UpdatedAt = now

	// Classify the error for structured storage
	if domainErr, ok := err.(domain.DomainError); ok {
		e.run.ErrorMsg = domainErr.UserMessage()
		// Store the error code in a structured way for filtering/alerting
		e.emitFailureEvent(ctx, domainErr)
	} else {
		e.run.ErrorMsg = err.Error()
		e.emitGenericFailureEvent(ctx, err)
	}

	// Classify the outcome based on error type
	e.outcome = e.classifyErrorOutcome(err)

	// Persist the failure state
	if updateErr := e.runs.Update(ctx, e.run); updateErr != nil {
		// Log but don't override - the original error is more important
		e.emitSystemEvent(ctx, "error", "failed to persist failure state: "+updateErr.Error())
	}
}

// classifyErrorOutcome maps errors to RunOutcome for categorization.
func (e *RunExecutor) classifyErrorOutcome(err error) domain.RunOutcome {
	switch err := err.(type) {
	case *domain.SandboxError:
		return domain.RunOutcomeSandboxFail
	case *domain.ConfigError:
		if err.Missing && err.Setting == "sandbox" {
			return domain.RunOutcomeSandboxFail
		}
		return domain.RunOutcomeException
	case *domain.RunnerError:
		if err.Operation == "timeout" {
			return domain.RunOutcomeTimeout
		}
		return domain.RunOutcomeRunnerFail
	default:
		return domain.RunOutcomeException
	}
}

// emitFailureEvent captures a domain error as a structured event.
// Uses the typed ErrorEventData for type safety.
func (e *RunExecutor) emitFailureEvent(ctx context.Context, err domain.DomainError) {
	if e.events == nil {
		return
	}

	evt := domain.NewErrorEventFromDomainError(e.run.ID, err)
	_ = e.events.Append(ctx, e.run.ID, evt)
}

// emitGenericFailureEvent captures a non-domain error as an event.
// Uses the typed ErrorEventData for type safety.
func (e *RunExecutor) emitGenericFailureEvent(ctx context.Context, err error) {
	if e.events == nil {
		return
	}

	evt := domain.NewErrorEvent(e.run.ID, string(domain.ErrCodeInternal), err.Error(), false)
	_ = e.events.Append(ctx, e.run.ID, evt)
}

// emitSystemEvent captures a system-level event (log, status change).
// Uses the typed LogEventData for type safety.
func (e *RunExecutor) emitSystemEvent(ctx context.Context, level, message string) {
	if e.events == nil {
		return
	}

	evt := domain.NewLogEvent(e.run.ID, level, message)
	_ = e.events.Append(ctx, e.run.ID, evt)
}

// =============================================================================
// CLEANUP OPERATIONS
// =============================================================================

// cleanupOnFailure performs cleanup when a run fails.
// NOTE: We intentionally do NOT delete the sandbox on failure.
// This allows inspection of partial work and debugging.
func (e *RunExecutor) cleanupOnFailure(ctx context.Context) {
	// Release any acquired locks
	// (Future: implement lock cleanup when lock manager is wired up)

	// Emit final status event
	if e.shouldPreserveSandbox(domain.SandboxLifecycleRunFailed) {
		e.emitSystemEvent(ctx, "info", "run failed - sandbox preserved for inspection")
	}

	e.applySandboxLifecycle(ctx, domain.SandboxLifecycleRunFailed, "failure cleanup")
}

func (e *RunExecutor) applySandboxLifecycle(ctx context.Context, event domain.SandboxLifecycleEvent, reason string) {
	if e.sandboxFinalized {
		return
	}
	if e.run.RunMode != domain.RunModeSandboxed || e.sandboxID == nil || e.sandbox == nil {
		return
	}

	cfg := e.effectiveSandboxConfig()
	if cfg == nil {
		return
	}

	events := []domain.SandboxLifecycleEvent{event}
	if event == domain.SandboxLifecycleRunCompleted || event == domain.SandboxLifecycleRunFailed || event == domain.SandboxLifecycleRunCancelled {
		events = append(events, domain.SandboxLifecycleTerminal)
	}

	if hasLifecycleEvent(cfg.Lifecycle.DeleteOn, events) {
		if err := e.sandbox.Delete(ctx, *e.sandboxID); err != nil {
			e.emitSystemEvent(ctx, "warn", "failed to delete sandbox: "+err.Error())
		} else {
			e.emitSystemEvent(ctx, "info", "sandbox deleted ("+reason+")")
			e.sandboxFinalized = true
		}
		return
	}

	if hasLifecycleEvent(cfg.Lifecycle.StopOn, events) {
		if err := e.sandbox.Stop(ctx, *e.sandboxID); err != nil {
			e.emitSystemEvent(ctx, "warn", "failed to stop sandbox: "+err.Error())
		} else {
			e.emitSystemEvent(ctx, "info", "sandbox stopped ("+reason+")")
		}
	}
}

func (e *RunExecutor) effectiveSandboxConfig() *domain.SandboxConfig {
	if e.run != nil && e.run.SandboxConfig != nil {
		return e.run.SandboxConfig
	}
	return nil
}

func hasLifecycleEvent(events []domain.SandboxLifecycleEvent, candidates []domain.SandboxLifecycleEvent) bool {
	for _, candidate := range candidates {
		for _, event := range events {
			if event == candidate {
				return true
			}
		}
	}
	return false
}

func (e *RunExecutor) shouldPreserveSandbox(event domain.SandboxLifecycleEvent) bool {
	cfg := e.effectiveSandboxConfig()
	if cfg == nil {
		return true
	}
	events := []domain.SandboxLifecycleEvent{event, domain.SandboxLifecycleTerminal}
	if hasLifecycleEvent(cfg.Lifecycle.StopOn, events) || hasLifecycleEvent(cfg.Lifecycle.DeleteOn, events) {
		return false
	}
	return true
}

func (e *RunExecutor) tryAutoApproval(ctx context.Context) bool {
	cfg := e.effectiveSandboxConfig()
	if cfg == nil {
		return false
	}
	if cfg.Acceptance.AutoReject {
		return e.autoReject(ctx)
	}
	if cfg.Acceptance.AutoApprove {
		return e.autoApprove(ctx)
	}
	return false
}

func (e *RunExecutor) autoApprove(ctx context.Context) bool {
	if e.sandbox == nil || e.sandboxID == nil {
		e.emitSystemEvent(ctx, "warn", "auto-approve skipped: no sandbox available")
		return false
	}
	actor := "auto-approve"
	_, err := e.sandbox.Approve(ctx, sandbox.ApproveRequest{
		SandboxID: *e.sandboxID,
		Actor:     actor,
	})
	if err != nil {
		e.emitSystemEvent(ctx, "warn", "auto-approve failed: "+err.Error())
		return false
	}
	now := time.Now()
	e.run.ApprovalState = domain.ApprovalStateApproved
	e.run.ApprovedBy = actor
	e.run.ApprovedAt = &now
	e.run.Status = domain.RunStatusComplete
	return true
}

func (e *RunExecutor) autoReject(ctx context.Context) bool {
	if e.sandbox == nil || e.sandboxID == nil {
		e.emitSystemEvent(ctx, "warn", "auto-reject skipped: no sandbox available")
		return false
	}
	actor := "auto-reject"
	if err := e.sandbox.Reject(ctx, *e.sandboxID, actor); err != nil {
		e.emitSystemEvent(ctx, "warn", "auto-reject failed: "+err.Error())
		return false
	}
	now := time.Now()
	e.run.ApprovalState = domain.ApprovalStateRejected
	e.run.ApprovedBy = actor
	e.run.ApprovedAt = &now
	e.run.Status = domain.RunStatusComplete
	return true
}

// =============================================================================
// QUERY METHODS
// =============================================================================

// Outcome returns the execution outcome after Execute() completes.
func (e *RunExecutor) Outcome() domain.RunOutcome {
	return e.outcome
}

// SandboxID returns the sandbox ID if one was created.
func (e *RunExecutor) SandboxID() *uuid.UUID {
	return e.sandboxID
}

// WorkDir returns the working directory used for execution.
func (e *RunExecutor) WorkDir() string {
	return e.workDir
}
