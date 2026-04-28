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
	"sync"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/metrics"
	"agent-manager/internal/modelregistry"
	"agent-manager/internal/repository"
	"agent-manager/internal/runstate"

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
		Timeout:            60 * time.Minute,
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
// ModelChainResolver returns the ordered preset chain for a runner+preset pair.
// Implemented by modelregistry.Store. Injected into RunExecutor so model-level fallback
// can walk the chain at execution time without persisting derived state on the run.
type ModelChainResolver interface {
	ResolvePreset(runner string, preset string) (modelregistry.PresetChain, bool)
}

// ModelHealthReporter receives runtime classifications of model availability.
// Implemented in this package by the health-store adapter so the executor does
// not import modelregistry's HealthStore type directly (keeps the executor seam small).
type ModelHealthReporter interface {
	MarkModelHealthy(runnerType, modelID string)
	MarkModelUnavailable(runnerType, modelID, message string)
}

type RunExecutor struct {
	// Dependencies
	runs        repository.RunRepository
	runners     runner.Registry
	sandbox     sandbox.Provider
	events      event.Store
	checkpoints repository.CheckpointRepository // optional: for checkpoint persistence
	broadcaster EventBroadcaster                // optional: for real-time WebSocket updates
	modelChains ModelChainResolver              // optional: for model-level fallback
	modelHealth ModelHealthReporter             // optional: surfaces runtime model verdicts to the health store

	// Configuration
	config ExecutorConfig

	// Execution context
	run          *domain.Run
	task         *domain.Task
	profile      *domain.AgentProfile
	prompt       string
	systemPrompt string
	attachments  []runner.Attachment // image attachments resolved from storage

	// Workspace state
	sandboxID *uuid.UUID
	workDir   string
	lockID    *uuid.UUID
	runState  *runstate.State

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

	// Recommendation extraction gate
	shouldQueueRecommendations func(*domain.Run) bool

	// Resumption state
	isResuming bool

	// Sandbox finalization state
	sandboxFinalized bool

	// Custom environment variables injected by API callers.
	customEnv map[string]string

	// Identity token state
	identitySecret []byte // HMAC secret for token signing (nil = identity disabled)
	identityToken  string // generated token for this run
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
		config:        DefaultExecutorConfig(),
		checkpoint:    domain.NewCheckpoint(run.ID, domain.RunPhaseQueued),
		heartbeatStop: make(chan struct{}),
		heartbeatDone: make(chan struct{}),
		shouldQueueRecommendations: func(run *domain.Run) bool {
			return run.IsInvestigationRun()
		},
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

// WithModelChainResolver wires the preset chain resolver used by model-level fallback.
// When unset, the executor behaves as before — no model-level retry.
func (e *RunExecutor) WithModelChainResolver(resolver ModelChainResolver) *RunExecutor {
	e.modelChains = resolver
	return e
}

// WithModelHealthReporter wires a reporter that receives runtime model-availability
// verdicts. Optional — when unset, classifications are logged but not aggregated.
func (e *RunExecutor) WithModelHealthReporter(reporter ModelHealthReporter) *RunExecutor {
	e.modelHealth = reporter
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

// WithIdentitySecret sets the HMAC secret used to generate identity tokens.
// If not set, no identity token will be injected into runs.
func (e *RunExecutor) WithIdentitySecret(secret []byte) *RunExecutor {
	e.identitySecret = secret
	return e
}

// WithRecommendationQueueFilter sets a custom filter for recommendation extraction.
func (e *RunExecutor) WithRecommendationQueueFilter(filter func(*domain.Run) bool) *RunExecutor {
	if filter != nil {
		e.shouldQueueRecommendations = filter
	}
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

// WithAttachments sets image attachments resolved from storage for the initial execution.
func (e *RunExecutor) WithAttachments(attachments []runner.Attachment) *RunExecutor {
	e.attachments = attachments
	return e
}

// WithCustomEnvironment sets caller-provided environment variables that will
// be merged into the agent process environment. Sandbox variables take
// precedence on key conflicts.
func (e *RunExecutor) WithCustomEnvironment(env map[string]string) *RunExecutor {
	e.customEnv = env
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

	// Step 3.5: Generate identity token (before execution so it's available in env)
	e.generateIdentityToken(execCtx)

	// Step 4: Execute agent, walking the preset chain on model-unavailable errors.
	e.advancePhase(execCtx, domain.RunPhaseExecuting)
	e.executeAgentWithModelFallback(execCtx, agentRunner)

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
		// Preserve session ID for continuation even on timeout.
		// The runner returns a valid result with SessionID populated
		// from stream events received before the process was killed.
		if e.result != nil && e.result.SessionID != "" {
			e.run.SessionID = e.result.SessionID
		}

		e.failWithError(ctx, &domain.RunnerError{
			RunnerType:  e.getRunnerType(),
			Operation:   "timeout",
			Cause:       fmt.Errorf("execution exceeded timeout of %v", e.config.Timeout),
			IsTransient: true, // Timeout is retryable via continuation
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

	// Broadcast terminal status so WebSocket clients see the change
	if e.broadcaster != nil {
		e.broadcaster.BroadcastRunStatus(e.run)
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
	if err := e.runs.Update(ctx, e.run); err != nil {
		return err
	}
	// Broadcast so WebSocket clients see the transition to Starting
	if e.broadcaster != nil {
		e.broadcaster.BroadcastRunStatus(e.run)
	}
	return nil
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

	// Resolve relative project root to absolute (workspace-sandbox requires absolute paths)
	projectRoot := e.task.ProjectRoot
	if projectRoot != "" && !filepath.IsAbs(projectRoot) {
		if absRoot, err := filepath.Abs(projectRoot); err == nil {
			projectRoot = absRoot
		}
	}

	metadata := map[string]string{
		"agent_manager_run_id": e.run.ID.String(),
	}
	sbx, err := e.sandbox.Create(ctx, sandbox.CreateRequest{
		Name:           e.buildSandboxName(),
		ScopePath:      e.task.ScopePath,
		NoLock:         noLockFromSandboxConfig(e.run.SandboxConfig),
		ProjectRoot:    projectRoot,
		Owner:          e.run.ID.String(),
		OwnerType:      "run",
		IdempotencyKey: idempotencyKey,
		Behavior:       e.run.SandboxConfig,
		Metadata:       metadata,
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

// buildSandboxName constructs a descriptive name for the sandbox.
// Priority: run.Tag > task.Title > scope path
// Profile name is appended when available for context.
func (e *RunExecutor) buildSandboxName() string {
	// Use run tag if explicitly set
	if tag := e.run.GetTag(); tag != "" {
		return tag
	}

	// Get profile name for context
	profileName := ""
	if e.profile != nil && e.profile.Name != "" {
		profileName = e.profile.Name
	}

	// Use task title if available
	if e.task.Title != "" {
		if profileName != "" {
			return fmt.Sprintf("%s (%s)", e.task.Title, profileName)
		}
		return e.task.Title
	}

	// Fall back to scope path
	scope := e.task.ScopePath
	if scope == "" {
		scope = "/"
	}
	if profileName != "" {
		return fmt.Sprintf("%s (%s)", scope, profileName)
	}
	return scope
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

// executeAgentWithModelFallback runs the agent, and when the runner rejects the
// current model with a classified "unavailable" error, advances to the next entry in
// the preset chain and retries inside the same Run. The loop is capped at the chain
// length to guarantee termination even if the classifier is overly permissive. On any
// non-model failure (or on success) it returns immediately. The first outcome that is
// not a model-unavailable error determines `Run.ActualModel`.
func (e *RunExecutor) executeAgentWithModelFallback(ctx context.Context, r runner.Runner) {
	chain := e.resolveModelFallbackChain()
	if len(chain) == 0 {
		// No preset chain — keep existing single-shot behavior. Record whatever model
		// the resolved config currently carries as the actual model.
		e.executeAgent(ctx, r)
		e.recordActualModel(e.currentModel())
		return
	}

	for attempt := 0; attempt < len(chain); attempt++ {
		model := chain[attempt]
		e.applyModelForAttempt(ctx, model, attempt, chain)
		e.executeAgent(ctx, r)

		kind := e.classifyExecutionOutcome(r)
		e.reportHealth(model, kind)
		if kind != runner.ModelErrorUnavailable {
			e.recordActualModel(model)
			return
		}

		if attempt == len(chain)-1 {
			// Chain exhausted — leave outcome in place. ActualModel reflects the final
			// attempt so the UI can render "ran on X, degraded through chain".
			e.recordActualModel(model)
			e.emitSystemEvent(ctx, "warn", "model fallback exhausted — all entries in preset chain were rejected")
			return
		}

		next := chain[attempt+1]
		e.emitSystemEvent(ctx, "warn", fmt.Sprintf(
			"model fallback: %s -> %s (runner rejected model)",
			describeModel(model), describeModel(next),
		))
	}
}

// reportHealth forwards the outcome of a single model attempt to the health reporter.
// Only concrete model IDs feed the reporter; the runner-default sentinel (empty string)
// is not a user-addressable entry and is skipped. Non-model failures leave health
// untouched — the failure was not about the model.
func (e *RunExecutor) reportHealth(modelID string, kind runner.ModelErrorKind) {
	if e.modelHealth == nil || modelID == "" {
		return
	}
	runnerType := ""
	if e.run != nil && e.run.ResolvedConfig != nil {
		runnerType = string(e.run.ResolvedConfig.RunnerType)
	}
	if runnerType == "" {
		return
	}
	switch kind {
	case runner.ModelErrorUnavailable:
		message := "runtime classification: model unavailable"
		if e.result != nil && e.result.ErrorMessage != "" {
			message = e.result.ErrorMessage
		}
		e.modelHealth.MarkModelUnavailable(runnerType, modelID, message)
	case runner.ModelErrorNone:
		if e.result != nil && e.result.Success {
			e.modelHealth.MarkModelHealthy(runnerType, modelID)
		}
	}
}

// resolveModelFallbackChain returns the ordered chain the executor will walk.
// Returns an empty slice when the run has no preset (i.e. an explicit model was
// picked) or when no resolver is configured — callers treat that as "single attempt".
func (e *RunExecutor) resolveModelFallbackChain() modelregistry.PresetChain {
	if e.modelChains == nil || e.run == nil || e.run.ResolvedConfig == nil {
		return nil
	}
	cfg := e.run.ResolvedConfig
	if cfg.ModelPreset == domain.ModelPresetUnspecified {
		return nil
	}
	chain, ok := e.modelChains.ResolvePreset(string(cfg.RunnerType), string(cfg.ModelPreset))
	if !ok || len(chain) == 0 {
		return nil
	}
	return chain
}

// applyModelForAttempt mutates the resolved config so the next Execute call uses
// the chosen model. An empty string means "omit the model flag" — runner adapters
// already skip injection when cfg.Model is empty.
func (e *RunExecutor) applyModelForAttempt(ctx context.Context, model string, attempt int, chain modelregistry.PresetChain) {
	if e.run == nil || e.run.ResolvedConfig == nil {
		return
	}
	if e.run.ResolvedConfig.Model == model {
		return
	}
	e.run.ResolvedConfig.Model = model
	if attempt > 0 {
		e.emitSystemEvent(ctx, "info", fmt.Sprintf(
			"model attempt %d/%d: %s",
			attempt+1, len(chain), describeModel(model),
		))
	}
}

// classifyExecutionOutcome inspects the post-Execute state and decides whether
// the failure (if any) was caused by the runner rejecting the model.
func (e *RunExecutor) classifyExecutionOutcome(r runner.Runner) runner.ModelErrorKind {
	if e.result != nil && e.result.Success {
		return runner.ModelErrorNone
	}
	// Prefer the runner's structured error message; fall back to execErr text.
	stderr := ""
	exitCode := 0
	if e.result != nil {
		stderr = e.result.ErrorMessage
		exitCode = e.result.ExitCode
	}
	if stderr == "" && e.execErr != nil {
		stderr = e.execErr.Error()
	}
	runnerType := domain.RunnerTypeClaudeCode
	if e.run != nil && e.run.ResolvedConfig != nil {
		runnerType = e.run.ResolvedConfig.RunnerType
	}
	return runner.ClassifyModelError(runnerType, stderr, exitCode)
}

// recordActualModel persists the model identifier the CLI actually executed with.
// Empty string signals the runner-default sentinel (no --model flag was passed).
func (e *RunExecutor) recordActualModel(model string) {
	if e.run == nil {
		return
	}
	e.run.ActualModel = model
}

// currentModel returns the resolved config's current model, or empty when no run.
func (e *RunExecutor) currentModel() string {
	if e.run == nil || e.run.ResolvedConfig == nil {
		return ""
	}
	return e.run.ResolvedConfig.Model
}

// describeModel returns a human-readable label for a chain entry. The empty string
// (runner-default sentinel) is rendered as "<runner default>" so log lines are clear.
func describeModel(model string) string {
	if model == "" {
		return "<runner default>"
	}
	return model
}

func (e *RunExecutor) executeAgent(ctx context.Context, r runner.Runner) {
	// Update status to running
	e.run.Status = domain.RunStatusRunning
	e.run.UpdatedAt = time.Now()
	if err := e.runs.Update(ctx, e.run); err != nil {
		e.emitSystemEvent(ctx, "warn", "failed to persist run start: "+err.Error())
	}
	// Broadcast so WebSocket clients see the transition to Running
	if e.broadcaster != nil {
		e.broadcaster.BroadcastRunStatus(e.run)
	}

	// Create event sink
	eventSink := e.createEventSink()
	defer eventSink.Close()

	transcriptCfg, err := e.prepareTranscriptConfig(ctx)
	if err != nil {
		e.execErr = err
		return
	}
	defer func() {
		if e.runState != nil {
			_ = e.runState.Close()
		}
	}()

	// Build execution request
	req := runner.ExecuteRequest{
		RunID:          e.run.ID,
		Tag:            e.run.GetTag(), // Custom tag or defaults to ID
		Profile:        e.profile,
		ResolvedConfig: e.run.ResolvedConfig, // Merged config from profile + inline
		Task:           e.task,
		WorkingDir:     e.workDir,
		SandboxID:      e.sandboxID, // populated for sandboxed runs; nil for in-place
		Prompt:         e.prompt,
		SystemPrompt:   e.systemPrompt,
		EventSink:      eventSink,
		Attachments:    e.attachments,
		Environment:    e.MergedEnvVars(),
		Transcript:     transcriptCfg,
	}

	// Execute
	e.result, e.execErr = r.Execute(ctx, req)
}

func (e *RunExecutor) prepareTranscriptConfig(ctx context.Context) (*runner.TranscriptConfig, error) {
	if e.run.ResolvedConfig == nil {
		return nil, nil
	}
	if e.runState == nil {
		startedAt := time.Now().UTC()
		if e.run.StartedAt != nil {
			startedAt = e.run.StartedAt.UTC()
		}
		state, err := runstate.Open(e.run.ID, runstate.OpenOptions{
			RunnerType: e.run.ResolvedConfig.RunnerType,
			WorkingDir: e.workDir,
			StartedAt:  startedAt,
		})
		if err != nil {
			return nil, err
		}
		e.runState = state
		snap := state.Snapshot()
		e.mu.Lock()
		e.run.TranscriptPath = snap.TranscriptPath
		e.run.TranscriptCursor = snap.Cursor.TranscriptCursor
		e.run.TranscriptLastSeq = snap.Cursor.TranscriptLastSeq
		e.mu.Unlock()
		if err := e.runs.Update(ctx, e.run); err != nil {
			return nil, err
		}
	}

	return &runner.TranscriptConfig{
		TranscriptPath: e.run.TranscriptPath,
		StderrPath:     e.runState.Snapshot().StderrPath,
		StdoutFile:     e.runState.TranscriptWriter(),
		StderrFile:     e.runState.StderrWriter(),
		OnProcessStart: func(pid, pgid int) error {
			e.mu.Lock()
			e.run.RunnerPID = pid
			e.run.RunnerPGID = pgid
			e.mu.Unlock()
			if err := e.runState.PersistProcess(pid, pgid); err != nil {
				return err
			}
			return e.runs.Update(context.Background(), e.run)
		},
		OnAdvance: func(cursor, lastSeq int64) error {
			e.mu.Lock()
			if cursor > e.run.TranscriptCursor {
				e.run.TranscriptCursor = cursor
			}
			if lastSeq > e.run.TranscriptLastSeq {
				e.run.TranscriptLastSeq = lastSeq
			}
			e.mu.Unlock()
			if err := e.runState.PersistCursor(e.run.TranscriptCursor, e.run.TranscriptLastSeq); err != nil {
				return err
			}
			return e.runs.Update(context.Background(), e.run)
		},
		OnSessionID: func(sessionID string) error {
			e.mu.Lock()
			if e.run.SessionID == sessionID {
				e.mu.Unlock()
				return nil
			}
			e.run.SessionID = sessionID
			e.mu.Unlock()
			if err := e.runState.PersistSessionID(sessionID); err != nil {
				return err
			}
			return e.runs.Update(context.Background(), e.run)
		},
	}, nil
}

// sandboxEnvVars returns environment variables that enable sandbox-aware scenario
// lifecycle commands (start, stop, restart) inside the agent's process.
//
// # Problem
//
// When an agent runs in an overlayfs sandbox, its file changes are captured in
// the overlay's upper/ layer — the real repo is untouched. But the Vrooli CLI
// lifecycle system (vrooli scenario restart, make start, etc.) reads from the
// real repo by default. Without these environment variables, an agent's code
// changes would be invisible to restarted scenarios, making it impossible to
// test changes while sandboxed.
//
// # Solution
//
// These env vars tell the Vrooli CLI (scripts/lib/scenario/runner.sh) to
// redirect scenario path resolution to the sandbox's merged/ directory. The
// agent doesn't need to pass any flags — it just runs "vrooli scenario restart"
// normally and the CLI handles the redirection transparently.
//
// # Design constraints
//
//   - One instance per slug: restarting a scenario stops any existing instance
//     of that slug, regardless of whether it was started from a sandbox, a
//     different sandbox, or the real repo.
//   - Path-only change: ports, process metadata, health checks, and logs are
//     identical whether started from the sandbox or the real repo.
//   - Scope-narrowed: only scenarios within VROOLI_SANDBOX_SCOPE are redirected.
//     Other scenarios use the real repo, so an agent sandboxing scenario A
//     won't affect scenario B's restarts.
//
// # Variables
//
//   - VROOLI_SANDBOX_ID: sandbox UUID, used for logging and debugging
//   - VROOLI_SANDBOX_MERGED: absolute path to the overlay's merged/ directory
//   - VROOLI_SANDBOX_SCOPE: relative scope path (e.g. "scenarios/my-scenario")
//
// # Example flow
//
//  1. Agent edits main.go inside the sandbox (changes go to overlay upper/)
//  2. Agent runs "vrooli scenario restart my-scenario"
//  3. CLI detects VROOLI_SANDBOX_MERGED and VROOLI_SANDBOX_SCOPE in env
//  4. CLI checks that "my-scenario" falls within the scope
//  5. CLI resolves path to {VROOLI_SANDBOX_MERGED}/scenarios/my-scenario
//  6. Lifecycle rebuilds and starts from the merged directory (agent's changes)
//
// The same mechanism is intentionally allowed for `vrooli scenario restart agent-manager`
// from inside an agent-manager-managed run. Transcript recovery reattaches after restart;
// there is no self-restart guard here.
//
// Returns nil for non-sandboxed runs, so callers can unconditionally assign the
// result to ExecuteRequest.Environment.
func (e *RunExecutor) SandboxEnvVars() map[string]string {
	// Only inject sandbox context for sandboxed runs that have completed
	// sandbox creation (sandboxID and workDir are populated).
	if e.run.RunMode != domain.RunModeSandboxed {
		return nil
	}
	if e.sandboxID == nil || e.workDir == "" {
		return nil
	}

	vars := map[string]string{
		"VROOLI_SANDBOX_ID":     e.sandboxID.String(),
		"VROOLI_SANDBOX_MERGED": e.workDir,
	}
	if e.task != nil && e.task.ScopePath != "" {
		vars["VROOLI_SANDBOX_SCOPE"] = e.task.ScopePath
	}
	return vars
}

// IdentityEnvVars returns identity token env vars for the current run.
// Returns nil if no identity token has been generated.
func (e *RunExecutor) IdentityEnvVars() map[string]string {
	if e.identityToken == "" {
		return nil
	}
	return map[string]string{
		"VROOLI_AGENT_IDENTITY_TOKEN": e.identityToken,
	}
}

// MergedEnvVars returns custom env vars merged with sandbox and identity env vars.
// Sandbox and identity vars take precedence on key conflicts to prevent callers from
// overriding VROOLI_SANDBOX_MERGED or other system-managed variables.
func (e *RunExecutor) MergedEnvVars() map[string]string {
	sandbox := e.SandboxEnvVars()
	identityVars := e.IdentityEnvVars()
	if len(e.customEnv) == 0 && len(sandbox) == 0 && len(identityVars) == 0 {
		return nil
	}
	merged := make(map[string]string, len(e.customEnv)+len(sandbox)+len(identityVars))
	for k, v := range e.customEnv {
		merged[k] = v
	}
	// System vars override custom vars for security.
	for k, v := range sandbox {
		merged[k] = v
	}
	for k, v := range identityVars {
		merged[k] = v
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

// generateIdentityToken creates a signed identity token for this run and
// persists its hash in the database. Non-fatal: if generation fails, the run
// proceeds without an identity token.
func (e *RunExecutor) generateIdentityToken(ctx context.Context) {
	if len(e.identitySecret) == 0 {
		return
	}

	now := time.Now()
	profileKey := ""
	if e.profile != nil {
		profileKey = e.profile.ProfileKey
	}
	scopePath := ""
	if e.task != nil {
		scopePath = e.task.ScopePath
	}

	claims := &identity.Claims{
		RunID:      e.run.ID,
		TaskID:     e.run.TaskID,
		ProfileKey: profileKey,
		ScopePath:  scopePath,
		IssuedAt:   now.Unix(),
		ExpiresAt:  now.Add(identity.DefaultTTL).Unix(),
		Meta:       map[string]string{},
	}

	token, err := identity.GenerateToken(claims, e.identitySecret)
	if err != nil {
		e.emitSystemEvent(ctx, "warn", "failed to generate identity token: "+err.Error())
		return
	}

	e.identityToken = token
	e.run.IdentityTokenHash = identity.HashToken(token)

	if err := e.runs.Update(ctx, e.run); err != nil {
		e.emitSystemEvent(ctx, "warn", "failed to persist identity token hash: "+err.Error())
	}
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

	// Broadcast final status so WebSocket clients see the terminal state
	if e.broadcaster != nil {
		e.broadcaster.BroadcastRunStatus(e.run)
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
	if e.result != nil {
		e.run.Summary = e.result.Summary
		e.run.ExitCode = &e.result.ExitCode
		if e.result.SessionID != "" {
			e.run.SessionID = e.result.SessionID
		}
	}

	// In-place runs (no sandbox) skip the apply workflow entirely. Changes
	// were written directly to the working tree, so there is nothing to
	// apply from a sandbox.
	if e.run.RunMode == domain.RunModeInPlace {
		e.run.Status = domain.RunStatusComplete
		e.run.ApprovalState = domain.ApprovalStateNone
		e.emitSystemEvent(ctx, "info", "in-place run completed — skipping apply (no sandbox to diff)")
	} else {
		// Auditability contract: in-acceptance changes apply at run end.
		// applyAtRunEnd encodes ManualReview / AutoApply / ApplyOnFailure.
		e.applyAtRunEnd(ctx, domain.ContractRunOutcomeSuccess)
	}

	// Queue recommendation extraction for investigation runs
	// The background RecommendationWorker will pick this up and extract recommendations
	if e.shouldQueueRecommendations != nil && e.shouldQueueRecommendations(e.run) {
		now := time.Now()
		e.run.RecommendationStatus = domain.RecommendationStatusPending
		e.run.RecommendationQueuedAt = &now
	}

	e.revokeIdentityToken()
	e.applySandboxLifecycle(ctx, domain.SandboxLifecycleRunCompleted, "run completed")
}

func (e *RunExecutor) handleFailure(ctx context.Context) {
	e.run.Status = domain.RunStatusFailed

	if e.execErr != nil {
		e.run.ErrorMsg = e.execErr.Error()
	} else if e.result != nil && e.result.ErrorMessage != "" {
		e.run.ErrorMsg = e.result.ErrorMessage
		e.run.ExitCode = &e.result.ExitCode
	}

	// Emit error event so clients polling events see the failure reason
	if e.run.ErrorMsg != "" {
		if e.execErr != nil {
			if domainErr, ok := e.execErr.(domain.DomainError); ok {
				e.emitFailureEvent(ctx, domainErr)
			} else {
				e.emitGenericFailureEvent(ctx, e.execErr)
			}
		} else if e.result != nil && e.result.ErrorMessage != "" {
			e.emitGenericFailureEvent(ctx, errors.New(e.result.ErrorMessage))
		}
	}

	// Auditability contract: failed runs that produced useful changes still
	// apply at run end, subject to acceptance, when ApplyOnFailure=true
	// (the default). applyAtRunEnd may flip the run status to Complete on
	// successful apply — that is the contract: the *change* is what matters
	// for auditability, not the run-process exit code.
	if e.run.RunMode == domain.RunModeSandboxed {
		e.applyAtRunEnd(ctx, e.outcome.ToContract())
	}

	e.revokeIdentityToken()
	e.applySandboxLifecycle(ctx, domain.SandboxLifecycleRunFailed, "run failed")
}

func (e *RunExecutor) handleCancellation(ctx context.Context) {
	e.run.Status = domain.RunStatusCancelled
	if e.run.RunMode == domain.RunModeSandboxed {
		e.applyAtRunEnd(ctx, domain.ContractRunOutcomeCancelled)
	}
	e.revokeIdentityToken()
	e.applySandboxLifecycle(ctx, domain.SandboxLifecycleRunCancelled, "run cancelled")
}

// revokeIdentityToken marks the run's identity token as revoked.
func (e *RunExecutor) revokeIdentityToken() {
	if e.run.IdentityTokenHash != "" {
		now := time.Now()
		e.run.IdentityTokenRevokedAt = &now
	}
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

	// Broadcast failure status so WebSocket clients see the change
	if e.broadcaster != nil {
		e.broadcaster.BroadcastRunStatus(e.run)
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
	// Broadcast so WebSocket clients see post-runner events in real-time
	if e.broadcaster != nil {
		e.broadcaster.BroadcastEvent(evt)
	}
}

// emitGenericFailureEvent captures a non-domain error as an event.
// Uses the typed ErrorEventData for type safety.
func (e *RunExecutor) emitGenericFailureEvent(ctx context.Context, err error) {
	if e.events == nil {
		return
	}

	evt := domain.NewErrorEvent(e.run.ID, string(domain.ErrCodeInternal), err.Error(), false)
	_ = e.events.Append(ctx, e.run.ID, evt)
	// Broadcast so WebSocket clients see post-runner events in real-time
	if e.broadcaster != nil {
		e.broadcaster.BroadcastEvent(evt)
	}
}

// emitSystemEvent captures a system-level event (log, status change).
// Uses the typed LogEventData for type safety.
func (e *RunExecutor) emitSystemEvent(ctx context.Context, level, message string) {
	if e.events == nil {
		return
	}

	evt := domain.NewLogEvent(e.run.ID, level, message)
	_ = e.events.Append(ctx, e.run.ID, evt)
	// Broadcast so WebSocket clients see post-runner events in real-time
	if e.broadcaster != nil {
		e.broadcaster.BroadcastEvent(evt)
	}
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

// applyAtRunEnd is the single shared apply-at-run-end seam called from every
// terminal handler (success, failure, cancel, timeout). It encodes the
// auditability contract: in-acceptance changes apply at run end, regardless
// of run outcome, and out-of-acceptance changes are retained as
// state=pending-review on the resulting provenance record.
//
// Returns true iff the run's terminal status should be RunStatusComplete with
// ApprovalState=Approved (i.e., apply succeeded). Returns false in three
// cases: ManualReview=true (sandbox persists for operator approval),
// AutoApply=false (operator opted out of auto-apply), or apply failed
// (warn event emitted; sandbox preserved for inspection).
//
// The contract defaults — locked by AUDITABILITY_CONTRACT.md — are
// AutoApply=true and ApplyOnFailure=true, so the common path is "apply
// regardless of outcome".
func (e *RunExecutor) applyAtRunEnd(ctx context.Context, outcome domain.ContractRunOutcome) bool {
	cfg := e.effectiveSandboxConfig()
	if cfg == nil {
		// Defensive: resolveSandboxConfig guarantees a non-nil config for
		// sandboxed runs since 2026-04-24. If we land here, the orchestrator
		// constructed a run without going through resolveSandboxConfig — a bug.
		e.emitSystemEvent(ctx, "warn", "apply-at-run-end skipped: run has no sandbox config (resolve bug — please report)")
		return false
	}
	if e.sandbox == nil || e.sandboxID == nil {
		e.emitSystemEvent(ctx, "warn", "apply-at-run-end skipped: no sandbox available")
		return false
	}

	// ManualReview=true defers apply until operator approval. The sandbox
	// persists past run end; the run terminates as Complete with
	// ApprovalState=Pending so the AI Changes review queue surfaces it.
	if cfg.ManualReview {
		now := time.Now()
		e.run.ApprovalState = domain.ApprovalStatePending
		e.run.ApprovedAt = &now
		e.run.Status = domain.RunStatusNeedsReview
		e.emitSystemEvent(ctx, "info", "apply deferred: manualReview=true (operator approval required)")
		return false
	}

	if !cfg.GetAutoApply() {
		e.emitSystemEvent(ctx, "info", "apply skipped: autoApply=false")
		return false
	}

	// ApplyOnFailure=false suppresses apply on non-success outcomes. The
	// contract default is true, so this branch is the operator opt-out.
	if outcome != domain.ContractRunOutcomeSuccess && !cfg.GetApplyOnFailure() {
		e.emitSystemEvent(ctx, "info",
			fmt.Sprintf("apply skipped: applyOnFailure=false (outcome=%s)", outcome))
		return false
	}

	conversationID := e.run.ConversationID
	cost := 0.0
	if e.result != nil {
		cost = e.result.Metrics.CostEstimateUSD
	}

	result, err := e.sandbox.ApplyAtRunEnd(ctx, sandbox.ApplyAtRunEndRequest{
		SandboxID:      *e.sandboxID,
		RunID:          e.run.ID.String(),
		ConversationID: conversationID,
		Cost:           cost,
		RunOutcome:     string(outcome),
		Actor:          "applyAtRunEnd",
	})
	if err != nil {
		e.emitSystemEvent(ctx, "warn", "apply-at-run-end failed: "+err.Error())
		metrics.Get().RecordProvenanceSkipped()
		return false
	}

	metrics.Get().RecordProvenanceWrite()

	now := time.Now()
	e.run.ApprovedBy = "applyAtRunEnd"
	e.run.ApprovedAt = &now

	if result != nil && result.IsPartial {
		// Out-of-acceptance files retained as state=pending-review. The
		// in-acceptance subset applied; the sandbox persists for operator
		// review of the remaining files. The run itself is complete — the
		// review queue surfaces the pending entries independently.
		e.run.ApprovalState = domain.ApprovalStateApproved
		e.run.Status = domain.RunStatusComplete
		e.emitSystemEvent(ctx, "info",
			fmt.Sprintf("partial apply: %d applied, %d pending review", result.Applied, result.Remaining))
		return true
	}

	e.run.ApprovalState = domain.ApprovalStateApproved
	e.run.Status = domain.RunStatusComplete
	if result != nil && result.Applied == 0 {
		e.emitSystemEvent(ctx, "info", "apply-at-run-end recorded empty provenance (no changes)")
	}
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

func boolPtr(v bool) *bool { return &v }

// noLockFromSandboxConfig returns the NoLock value from a SandboxConfig,
// or nil if the config doesn't explicitly set it. Returning nil lets the
// workspace-sandbox server apply its own DefaultNoLock setting, rather than
// the agent-manager always overriding with false when NoLock isn't specified.
func noLockFromSandboxConfig(cfg *domain.SandboxConfig) *bool {
	if cfg == nil || !cfg.NoLock {
		return nil // let workspace-sandbox server default apply
	}
	return boolPtr(true)
}
