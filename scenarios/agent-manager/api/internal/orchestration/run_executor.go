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

package orchestration

import (
	"context"
	"fmt"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

// RunExecutor handles the execution lifecycle of a single run.
// It encapsulates all the steps needed to execute an agent run,
// making the flow explicit and each step independently testable.
type RunExecutor struct {
	// Dependencies
	runs     repository.RunRepository
	runners  runner.Registry
	sandbox  sandbox.Provider
	events   event.Store

	// Execution context
	run     *domain.Run
	task    *domain.Task
	profile *domain.AgentProfile
	prompt  string

	// Workspace state
	sandboxID *uuid.UUID
	workDir   string

	// Result state
	outcome domain.RunOutcome
	result  *runner.ExecuteResult
	execErr error
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
		runs:    runs,
		runners: runners,
		sandbox: sandbox,
		events:  events,
		run:     run,
		task:    task,
		profile: profile,
		prompt:  prompt,
	}
}

// Execute runs the full execution lifecycle.
// This is the main entry point that orchestrates all steps.
//
// GRACEFUL DEGRADATION: Each step is wrapped with proper error handling.
// On failure, we capture the error with full context and preserve
// the sandbox for inspection.
func (e *RunExecutor) Execute(ctx context.Context) {
	// Step 1: Update to starting
	if err := e.updateStatusToStarting(ctx); err != nil {
		e.failWithError(ctx, &domain.DatabaseError{
			Operation:   "update",
			EntityType:  "Run",
			EntityID:    e.run.ID.String(),
			Cause:       err,
			IsTransient: true, // Status updates are retryable
		})
		return
	}

	// Step 2: Setup workspace
	if err := e.setupWorkspace(ctx); err != nil {
		// setupWorkspace already returns domain errors
		e.failWithError(ctx, err)
		e.cleanupOnFailure(ctx)
		return
	}

	// Step 3: Acquire runner
	agentRunner, err := e.acquireRunner(ctx)
	if err != nil {
		// acquireRunner already returns domain errors
		e.failWithError(ctx, err)
		e.cleanupOnFailure(ctx)
		return
	}

	// Step 4: Execute agent
	e.executeAgent(ctx, agentRunner)

	// Step 5: Handle result
	e.handleResult(ctx)
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

	sbx, err := e.sandbox.Create(ctx, sandbox.CreateRequest{
		ScopePath:   e.task.ScopePath,
		ProjectRoot: e.task.ProjectRoot,
		Owner:       e.run.ID.String(),
		OwnerType:   "run",
	})
	if err != nil {
		return &domain.SandboxError{
			Operation:   "create",
			Cause:       err,
			IsTransient: true, // sandbox service might be temporarily unavailable
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
	if e.runners == nil {
		return nil, &domain.RunnerError{
			RunnerType:  e.profile.RunnerType,
			Operation:   "acquire",
			Cause:       fmt.Errorf("no runner registry configured"),
			IsTransient: false,
		}
	}

	r, err := e.runners.Get(e.profile.RunnerType)
	if err != nil {
		// Try to find an alternative runner to suggest
		alternative := e.findAlternativeRunner()
		return nil, &domain.RunnerError{
			RunnerType:  e.profile.RunnerType,
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
			RunnerType:  e.profile.RunnerType,
			Operation:   "availability_check",
			Cause:       fmt.Errorf(msg),
			IsTransient: true, // runner might become available
			Alternative: alternative,
		}
	}

	return r, nil
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

	for _, rt := range alternatives {
		if rt == e.profile.RunnerType {
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
		RunID:      e.run.ID,
		Profile:    e.profile,
		Task:       e.task,
		WorkingDir: e.workDir,
		Prompt:     e.prompt,
		EventSink:  eventSink,
	}

	// Execute
	e.result, e.execErr = r.Execute(ctx, req)
}

func (e *RunExecutor) createEventSink() runner.EventSink {
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
	e.run.Status = domain.RunStatusNeedsReview
	e.run.ApprovalState = domain.ApprovalStatePending

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
