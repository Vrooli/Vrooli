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
	"fmt"
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
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		Timeout:            30 * time.Minute,
		HeartbeatInterval:  30 * time.Second,
		CheckpointInterval: 1 * time.Minute,
		MaxRetries:         3,
		StaleThreshold:     2 * time.Minute,
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

	// Step 2: Setup workspace (skip if resuming with existing sandbox)
	if !e.shouldSkipPhase(domain.RunPhaseSandboxCreating) || e.sandboxID == nil {
		e.advancePhase(execCtx, domain.RunPhaseSandboxCreating)
		if err := e.setupWorkspace(execCtx); err != nil {
			// setupWorkspace already returns domain errors
			e.failWithError(execCtx, err)
			e.cleanupOnFailure(execCtx)
			return
		}
	} else {
		e.emitSystemEvent(ctx, "info", "reusing existing sandbox from checkpoint")
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
	e.runs.Update(ctx, e.run)

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

	ticker := time.NewTicker(e.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-e.heartbeatStop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
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
	e.mu.Unlock()

	// Update run in database (best-effort)
	if err := e.runs.Update(ctx, e.run); err != nil {
		e.emitSystemEvent(ctx, "warn", "heartbeat update failed: "+err.Error())
	}

	// Update checkpoint in database (best-effort)
	if e.checkpoints != nil {
		e.checkpoints.Heartbeat(ctx, e.run.ID)
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
			Cause:       fmt.Errorf("execution exceeded timeout of %v", e.config.Timeout),
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
		e.runs.Update(ctx, e.run)
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
		return &domain.SandboxError{
			Operation:   "create",
			Cause:       fmt.Errorf("sandbox provider not configured"),
			IsTransient: false,
			CanRetry:    false,
		}
	}

	// Use idempotency key to allow safe retries of sandbox creation
	idempotencyKey := fmt.Sprintf("sandbox:run:%s", e.run.ID.String())

	sbx, err := e.sandbox.Create(ctx, sandbox.CreateRequest{
		ScopePath:      e.task.ScopePath,
		ProjectRoot:    e.task.ProjectRoot,
		Owner:          e.run.ID.String(),
		OwnerType:      "run",
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		sandboxErr := &domain.SandboxError{
			Operation:   "create",
			Cause:       err,
			IsTransient: true, // sandbox service might be temporarily unavailable
			CanRetry:    true,
		}
		// Extract rich error details if available (e.g., conflicting sandboxes)
		if apiErr, ok := err.(*sandbox.SandboxAPIError); ok {
			sandboxErr.CanRetry = apiErr.Retryable
			sandboxErr.IsTransient = apiErr.Retryable
			if apiErr.Details != nil {
				sandboxErr.ExtraDetails = apiErr.Details
			}
		}
		return sandboxErr
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
			Cause:       fmt.Errorf("no runner registry configured"),
			IsTransient: false,
		}
	}

	r, err := e.runners.Get(runnerType)
	if err != nil {
		// Try to find an alternative runner to suggest
		alternative := e.findAlternativeRunner()
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
		// Try to find an alternative runner to suggest
		alternative := e.findAlternativeRunner()
		return nil, &domain.RunnerError{
			RunnerType:  runnerType,
			Operation:   "availability_check",
			Cause:       fmt.Errorf("%s", msg),
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

// =============================================================================
// STEP 4: Execute Agent
// =============================================================================

func (e *RunExecutor) executeAgent(ctx context.Context, r runner.Runner) {
	// Update status to running
	e.run.Status = domain.RunStatusRunning
	e.run.UpdatedAt = time.Now()
	e.runs.Update(ctx, e.run)

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
		e.handleSuccessfulCompletion()
	case e.outcome.IsTerminalFailure():
		e.handleFailure()
	case e.outcome == domain.RunOutcomeCancelled:
		e.handleCancellation()
	default:
		e.handleFailure() // Fallback
	}

	e.runs.Update(ctx, e.run)
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

func (e *RunExecutor) handleSuccessfulCompletion() {
	// Check if approval is required based on resolved config
	requiresApproval := true // default to requiring approval for safety
	if e.run.ResolvedConfig != nil {
		requiresApproval = e.run.ResolvedConfig.RequiresApproval
	}

	if requiresApproval {
		e.run.Status = domain.RunStatusNeedsReview
		e.run.ApprovalState = domain.ApprovalStatePending
	} else {
		// Skip approval workflow - mark as complete directly
		e.run.Status = domain.RunStatusComplete
		e.run.ApprovalState = domain.ApprovalStateNone

		// Cleanup sandbox when approval not required - changes are discarded
		// since the user chose not to review them in sandboxed mode
		if e.run.RunMode == domain.RunModeSandboxed && e.sandboxID != nil && e.sandbox != nil {
			ctx := context.Background()
			if err := e.sandbox.Delete(ctx, *e.sandboxID); err != nil {
				e.emitSystemEvent(ctx, "warn", "failed to cleanup sandbox on completion: "+err.Error())
			} else {
				e.emitSystemEvent(ctx, "info", "sandbox cleaned up (no approval required)")
			}
		}
	}

	if e.result != nil {
		e.run.Summary = e.result.Summary
		e.run.ExitCode = &e.result.ExitCode
	}
}

func (e *RunExecutor) handleFailure() {
	e.run.Status = domain.RunStatusFailed

	if e.execErr != nil {
		e.run.ErrorMsg = e.execErr.Error()
	} else if e.result != nil && e.result.ErrorMessage != "" {
		e.run.ErrorMsg = e.result.ErrorMessage
		e.run.ExitCode = &e.result.ExitCode
	}
}

func (e *RunExecutor) handleCancellation() {
	e.run.Status = domain.RunStatusCancelled
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

// failWithString is a convenience wrapper for simple error messages.
func (e *RunExecutor) failWithString(ctx context.Context, errMsg string) {
	e.failWithError(ctx, fmt.Errorf("%s", errMsg))
}

// classifyErrorOutcome maps errors to RunOutcome for categorization.
func (e *RunExecutor) classifyErrorOutcome(err error) domain.RunOutcome {
	switch err.(type) {
	case *domain.SandboxError:
		return domain.RunOutcomeSandboxFail
	case *domain.RunnerError:
		runnerErr := err.(*domain.RunnerError)
		if runnerErr.Operation == "timeout" {
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
	e.events.Append(ctx, e.run.ID, evt)
}

// emitGenericFailureEvent captures a non-domain error as an event.
// Uses the typed ErrorEventData for type safety.
func (e *RunExecutor) emitGenericFailureEvent(ctx context.Context, err error) {
	if e.events == nil {
		return
	}

	evt := domain.NewErrorEvent(e.run.ID, string(domain.ErrCodeInternal), err.Error(), false)
	e.events.Append(ctx, e.run.ID, evt)
}

// emitSystemEvent captures a system-level event (log, status change).
// Uses the typed LogEventData for type safety.
func (e *RunExecutor) emitSystemEvent(ctx context.Context, level, message string) {
	if e.events == nil {
		return
	}

	evt := domain.NewLogEvent(e.run.ID, level, message)
	e.events.Append(ctx, e.run.ID, evt)
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
	e.emitSystemEvent(ctx, "info", "run failed - sandbox preserved for inspection")
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
