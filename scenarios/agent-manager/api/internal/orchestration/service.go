// Package orchestration provides the core orchestration service for agent-manager.
//
// This package is the COORDINATION layer that wires together:
// - Domain entities (Task, Run, AgentProfile)
// - Runner adapters for agent execution
// - Sandbox providers for isolation
// - Policy evaluators for access control
// - Event collectors for activity tracking
//
// The Orchestrator is the primary entry point for all agent management operations.
// It does NOT contain domain logic or infrastructure details - it coordinates.
package orchestration

import (
	"context"
	"fmt"
	"time"

	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/policy"
	"agent-manager/internal/repository"
	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// Service Interface
// -----------------------------------------------------------------------------

// Service defines the orchestration service contract.
// This is the primary API for agent-manager operations.
type Service interface {
	// --- AgentProfile Operations ---
	CreateProfile(ctx context.Context, profile *domain.AgentProfile) (*domain.AgentProfile, error)
	GetProfile(ctx context.Context, id uuid.UUID) (*domain.AgentProfile, error)
	ListProfiles(ctx context.Context, opts ListOptions) ([]*domain.AgentProfile, error)
	UpdateProfile(ctx context.Context, profile *domain.AgentProfile) (*domain.AgentProfile, error)
	DeleteProfile(ctx context.Context, id uuid.UUID) error

	// --- Task Operations ---
	CreateTask(ctx context.Context, task *domain.Task) (*domain.Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	ListTasks(ctx context.Context, opts ListOptions) ([]*domain.Task, error)
	UpdateTask(ctx context.Context, task *domain.Task) (*domain.Task, error)
	CancelTask(ctx context.Context, id uuid.UUID) error

	// --- Run Operations ---
	CreateRun(ctx context.Context, req CreateRunRequest) (*domain.Run, error)
	GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error)
	StopRun(ctx context.Context, id uuid.UUID) error

	// --- Approval Operations ---
	ApproveRun(ctx context.Context, req ApproveRequest) (*ApproveResult, error)
	RejectRun(ctx context.Context, id uuid.UUID, actor, reason string) error
	PartialApprove(ctx context.Context, req PartialApproveRequest) (*ApproveResult, error)

	// --- Event Operations ---
	GetRunEvents(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error)
	StreamRunEvents(ctx context.Context, runID uuid.UUID, opts event.StreamOptions) (<-chan *domain.RunEvent, error)

	// --- Diff Operations ---
	GetRunDiff(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error)

	// --- Status Operations ---
	GetHealth(ctx context.Context) (*HealthStatus, error)
	GetRunnerStatus(ctx context.Context) ([]*RunnerStatus, error)
}

// -----------------------------------------------------------------------------
// Request/Response Types
// -----------------------------------------------------------------------------

// ListOptions specifies pagination and filtering for list operations.
type ListOptions struct {
	Limit  int
	Offset int
}

// RunListOptions extends ListOptions for run-specific filtering.
type RunListOptions struct {
	ListOptions
	TaskID         *uuid.UUID
	AgentProfileID *uuid.UUID
	Status         *domain.RunStatus
}

// CreateRunRequest contains parameters for creating a new run.
type CreateRunRequest struct {
	TaskID         uuid.UUID
	AgentProfileID uuid.UUID
	Prompt         string // Optional override prompt
	RunMode        *domain.RunMode // nil = let policy decide
	ForceInPlace   bool // Request in-place execution
}

// ApproveRequest contains parameters for approving a run.
type ApproveRequest struct {
	RunID     uuid.UUID
	Actor     string
	CommitMsg string
	Force     bool // Force despite conflicts
}

// PartialApproveRequest approves only selected files.
type PartialApproveRequest struct {
	RunID     uuid.UUID
	FileIDs   []uuid.UUID
	Actor     string
	CommitMsg string
}

// ApproveResult contains the approval outcome.
type ApproveResult struct {
	Success    bool
	Applied    int
	Remaining  int
	IsPartial  bool
	CommitHash string
	AppliedAt  time.Time
	ErrorMsg   string
}

// HealthStatus contains system health information.
type HealthStatus struct {
	Status       string                   `json:"status"`
	Database     ComponentStatus          `json:"database"`
	Sandbox      ComponentStatus          `json:"sandbox"`
	Runners      map[string]RunnerStatus  `json:"runners"`
	ActiveRuns   int                      `json:"activeRuns"`
	QueuedTasks  int                      `json:"queuedTasks"`
}

// ComponentStatus describes a component's health.
type ComponentStatus struct {
	Available bool   `json:"available"`
	Message   string `json:"message,omitempty"`
}

// RunnerStatus describes a runner's availability.
type RunnerStatus struct {
	Type         domain.RunnerType    `json:"type"`
	Available    bool                 `json:"available"`
	Message      string               `json:"message,omitempty"`
	Capabilities runner.Capabilities  `json:"capabilities"`
}

// -----------------------------------------------------------------------------
// Orchestrator Implementation
// -----------------------------------------------------------------------------

// Orchestrator coordinates agent execution using injected dependencies.
type Orchestrator struct {
	// Repositories (persistence)
	profiles repository.ProfileRepository
	tasks    repository.TaskRepository
	runs     repository.RunRepository

	// Adapters (external integrations)
	runners   runner.Registry
	sandbox   sandbox.Provider
	events    event.Store
	artifacts artifact.Collector

	// Policy evaluation
	policy policy.Evaluator

	// Lock management
	locks sandbox.LockManager

	// Configuration
	config OrchestratorConfig
}

// OrchestratorConfig holds service configuration.
type OrchestratorConfig struct {
	DefaultTimeout       time.Duration
	MaxConcurrentRuns    int
	DefaultProjectRoot   string
	RequireSandboxByDefault bool
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() OrchestratorConfig {
	return OrchestratorConfig{
		DefaultTimeout:          30 * time.Minute,
		MaxConcurrentRuns:       10,
		RequireSandboxByDefault: true,
	}
}

// Option configures the Orchestrator.
type Option func(*Orchestrator)

// WithConfig sets the configuration.
func WithConfig(cfg OrchestratorConfig) Option {
	return func(o *Orchestrator) {
		o.config = cfg
	}
}

// WithRunners sets the runner registry.
func WithRunners(r runner.Registry) Option {
	return func(o *Orchestrator) {
		o.runners = r
	}
}

// WithSandbox sets the sandbox provider.
func WithSandbox(s sandbox.Provider) Option {
	return func(o *Orchestrator) {
		o.sandbox = s
	}
}

// WithPolicy sets the policy evaluator.
func WithPolicy(p policy.Evaluator) Option {
	return func(o *Orchestrator) {
		o.policy = p
	}
}

// WithEvents sets the event store.
func WithEvents(e event.Store) Option {
	return func(o *Orchestrator) {
		o.events = e
	}
}

// WithArtifacts sets the artifact collector.
func WithArtifacts(a artifact.Collector) Option {
	return func(o *Orchestrator) {
		o.artifacts = a
	}
}

// WithLocks sets the lock manager.
func WithLocks(l sandbox.LockManager) Option {
	return func(o *Orchestrator) {
		o.locks = l
	}
}

// New creates a new Orchestrator with the given dependencies.
func New(
	profiles repository.ProfileRepository,
	tasks repository.TaskRepository,
	runs repository.RunRepository,
	opts ...Option,
) *Orchestrator {
	o := &Orchestrator{
		profiles: profiles,
		tasks:    tasks,
		runs:     runs,
		config:   DefaultConfig(),
	}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

// Verify Orchestrator implements Service interface at compile time.
var _ Service = (*Orchestrator)(nil)

// -----------------------------------------------------------------------------
// AgentProfile Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) CreateProfile(ctx context.Context, profile *domain.AgentProfile) (*domain.AgentProfile, error) {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	profile.CreatedAt = time.Now()
	profile.UpdatedAt = profile.CreatedAt

	if err := o.profiles.Create(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	return profile, nil
}

func (o *Orchestrator) GetProfile(ctx context.Context, id uuid.UUID) (*domain.AgentProfile, error) {
	profile, err := o.profiles.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, domain.NewNotFoundError("AgentProfile", id)
	}
	return profile, nil
}

func (o *Orchestrator) ListProfiles(ctx context.Context, opts ListOptions) ([]*domain.AgentProfile, error) {
	return o.profiles.List(ctx, repository.ListFilter{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

func (o *Orchestrator) UpdateProfile(ctx context.Context, profile *domain.AgentProfile) (*domain.AgentProfile, error) {
	profile.UpdatedAt = time.Now()
	if err := o.profiles.Update(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}
	return profile, nil
}

func (o *Orchestrator) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	return o.profiles.Delete(ctx, id)
}

// -----------------------------------------------------------------------------
// Task Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) CreateTask(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	task.Status = domain.TaskStatusQueued
	task.CreatedAt = time.Now()
	task.UpdatedAt = task.CreatedAt

	if err := o.tasks.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

func (o *Orchestrator) GetTask(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	task, err := o.tasks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, domain.NewNotFoundError("Task", id)
	}
	return task, nil
}

func (o *Orchestrator) ListTasks(ctx context.Context, opts ListOptions) ([]*domain.Task, error) {
	return o.tasks.List(ctx, repository.ListFilter{
		Limit:  opts.Limit,
		Offset: opts.Offset,
	})
}

func (o *Orchestrator) UpdateTask(ctx context.Context, task *domain.Task) (*domain.Task, error) {
	task.UpdatedAt = time.Now()
	if err := o.tasks.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}
	return task, nil
}

func (o *Orchestrator) CancelTask(ctx context.Context, id uuid.UUID) error {
	task, err := o.GetTask(ctx, id)
	if err != nil {
		return err
	}

	if task.Status != domain.TaskStatusQueued && task.Status != domain.TaskStatusRunning {
		return domain.NewStateError("Task", string(task.Status), "cancel", "can only cancel queued or running tasks")
	}

	task.Status = domain.TaskStatusCancelled
	task.UpdatedAt = time.Now()
	return o.tasks.Update(ctx, task)
}

// -----------------------------------------------------------------------------
// Run Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) CreateRun(ctx context.Context, req CreateRunRequest) (*domain.Run, error) {
	// Get task and profile
	task, err := o.GetTask(ctx, req.TaskID)
	if err != nil {
		return nil, err
	}

	profile, err := o.GetProfile(ctx, req.AgentProfileID)
	if err != nil {
		return nil, err
	}

	// Evaluate policies
	var policyDecision *policy.Decision
	if o.policy != nil {
		policyDecision, err = o.policy.EvaluateRunRequest(ctx, policy.EvaluateRequest{
			Task:         task,
			Profile:      profile,
			RequestedMode: valueOrDefault(req.RunMode, domain.RunModeSandboxed),
			ForceInPlace: req.ForceInPlace,
		})
		if err != nil {
			return nil, fmt.Errorf("policy evaluation failed: %w", err)
		}
		if !policyDecision.Allowed {
			return nil, &domain.PolicyViolationError{
				PolicyID:   policyDecision.DenialPolicy.ID,
				PolicyName: policyDecision.DenialPolicy.Name,
				Rule:       "run_request",
				Message:    policyDecision.DenialReason,
			}
		}
	}

	// Determine run mode
	runMode := domain.RunModeSandboxed
	if req.RunMode != nil {
		runMode = *req.RunMode
	} else if policyDecision != nil && !policyDecision.RequiresSandbox {
		runMode = domain.RunModeInPlace
	}

	// Create the run
	run := &domain.Run{
		ID:             uuid.New(),
		TaskID:         task.ID,
		AgentProfileID: profile.ID,
		RunMode:        runMode,
		Status:         domain.RunStatusPending,
		ApprovalState:  domain.ApprovalStateNone,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := o.runs.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}

	// Start execution asynchronously
	go o.executeRun(context.Background(), run, task, profile, req.Prompt)

	return run, nil
}

func (o *Orchestrator) GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	run, err := o.runs.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, domain.NewNotFoundError("Run", id)
	}
	return run, nil
}

func (o *Orchestrator) ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error) {
	return o.runs.List(ctx, repository.RunListFilter{
		ListFilter: repository.ListFilter{
			Limit:  opts.Limit,
			Offset: opts.Offset,
		},
		TaskID:         opts.TaskID,
		AgentProfileID: opts.AgentProfileID,
		Status:         opts.Status,
	})
}

func (o *Orchestrator) StopRun(ctx context.Context, id uuid.UUID) error {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}

	if run.Status != domain.RunStatusRunning && run.Status != domain.RunStatusStarting {
		return domain.NewStateError("Run", string(run.Status), "stop", "can only stop running or starting runs")
	}

	// Get the runner and stop execution
	if o.runners != nil {
		profile, _ := o.GetProfile(ctx, run.AgentProfileID)
		if profile != nil {
			if r, err := o.runners.Get(profile.RunnerType); err == nil {
				if err := r.Stop(ctx, run.ID); err != nil {
					return fmt.Errorf("failed to stop runner: %w", err)
				}
			}
		}
	}

	// Update status
	now := time.Now()
	run.Status = domain.RunStatusCancelled
	run.EndedAt = &now
	run.UpdatedAt = now
	return o.runs.Update(ctx, run)
}

// executeRun handles the actual agent execution (runs in background).
func (o *Orchestrator) executeRun(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, prompt string) {
	// Update status to starting
	now := time.Now()
	run.Status = domain.RunStatusStarting
	run.StartedAt = &now
	run.UpdatedAt = now
	o.runs.Update(ctx, run)

	var workDir string
	var sandboxID *uuid.UUID

	// Create sandbox if required
	if run.RunMode == domain.RunModeSandboxed && o.sandbox != nil {
		sbx, err := o.sandbox.Create(ctx, sandbox.CreateRequest{
			ScopePath:   task.ScopePath,
			ProjectRoot: task.ProjectRoot,
			Owner:       run.ID.String(),
			OwnerType:   "run",
		})
		if err != nil {
			o.failRun(ctx, run, fmt.Sprintf("sandbox creation failed: %v", err))
			return
		}
		sandboxID = &sbx.ID
		run.SandboxID = sandboxID
		o.runs.Update(ctx, run)

		workDir, err = o.sandbox.GetWorkspacePath(ctx, sbx.ID)
		if err != nil {
			o.failRun(ctx, run, fmt.Sprintf("failed to get workspace path: %v", err))
			return
		}
	} else {
		workDir = task.ProjectRoot
	}

	// Get the runner
	if o.runners == nil {
		o.failRun(ctx, run, "no runner registry configured")
		return
	}

	r, err := o.runners.Get(profile.RunnerType)
	if err != nil {
		o.failRun(ctx, run, fmt.Sprintf("runner not found: %v", err))
		return
	}

	// Update to running
	run.Status = domain.RunStatusRunning
	run.UpdatedAt = time.Now()
	o.runs.Update(ctx, run)

	// Create event sink
	var eventSink runner.EventSink
	if o.events != nil {
		eventSink = &eventStoreAdapter{store: o.events, runID: run.ID}
	} else {
		eventSink = &noOpEventSink{}
	}

	// Execute the agent
	result, err := r.Execute(ctx, runner.ExecuteRequest{
		RunID:      run.ID,
		Profile:    profile,
		Task:       task,
		WorkingDir: workDir,
		Prompt:     prompt,
		EventSink:  eventSink,
	})

	// Handle result
	endTime := time.Now()
	run.EndedAt = &endTime
	run.UpdatedAt = endTime

	if err != nil {
		run.Status = domain.RunStatusFailed
		run.ErrorMsg = err.Error()
	} else if !result.Success {
		run.Status = domain.RunStatusFailed
		run.ErrorMsg = result.ErrorMessage
		run.ExitCode = &result.ExitCode
	} else {
		run.Status = domain.RunStatusNeedsReview
		run.Summary = result.Summary
		run.ExitCode = &result.ExitCode
		run.ApprovalState = domain.ApprovalStatePending
	}

	o.runs.Update(ctx, run)
}

func (o *Orchestrator) failRun(ctx context.Context, run *domain.Run, errMsg string) {
	now := time.Now()
	run.Status = domain.RunStatusFailed
	run.ErrorMsg = errMsg
	run.EndedAt = &now
	run.UpdatedAt = now
	o.runs.Update(ctx, run)
}

// -----------------------------------------------------------------------------
// Approval Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) ApproveRun(ctx context.Context, req ApproveRequest) (*ApproveResult, error) {
	run, err := o.GetRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}

	if run.Status != domain.RunStatusNeedsReview {
		return nil, domain.NewStateError("Run", string(run.Status), "approve", "can only approve runs needing review")
	}

	if run.SandboxID == nil {
		return nil, &domain.ValidationError{Field: "sandboxId", Message: "run has no sandbox to approve"}
	}

	if o.sandbox == nil {
		return nil, fmt.Errorf("no sandbox provider configured")
	}

	result, err := o.sandbox.Approve(ctx, sandbox.ApproveRequest{
		SandboxID: *run.SandboxID,
		Actor:     req.Actor,
		CommitMsg: req.CommitMsg,
		Force:     req.Force,
	})
	if err != nil {
		return nil, err
	}

	// Update run status
	now := time.Now()
	run.ApprovalState = domain.ApprovalStateApproved
	run.ApprovedBy = req.Actor
	run.ApprovedAt = &now
	run.UpdatedAt = now
	o.runs.Update(ctx, run)

	return &ApproveResult{
		Success:    result.Success,
		Applied:    result.Applied,
		Remaining:  result.Remaining,
		IsPartial:  result.IsPartial,
		CommitHash: result.CommitHash,
		AppliedAt:  result.AppliedAt,
		ErrorMsg:   result.ErrorMsg,
	}, nil
}

func (o *Orchestrator) RejectRun(ctx context.Context, id uuid.UUID, actor, reason string) error {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}

	if run.Status != domain.RunStatusNeedsReview {
		return domain.NewStateError("Run", string(run.Status), "reject", "can only reject runs needing review")
	}

	if run.SandboxID != nil && o.sandbox != nil {
		if err := o.sandbox.Reject(ctx, *run.SandboxID, actor); err != nil {
			return fmt.Errorf("sandbox rejection failed: %w", err)
		}
	}

	now := time.Now()
	run.ApprovalState = domain.ApprovalStateRejected
	run.ErrorMsg = reason
	run.UpdatedAt = now
	return o.runs.Update(ctx, run)
}

func (o *Orchestrator) PartialApprove(ctx context.Context, req PartialApproveRequest) (*ApproveResult, error) {
	run, err := o.GetRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}

	if run.Status != domain.RunStatusNeedsReview {
		return nil, domain.NewStateError("Run", string(run.Status), "partial_approve", "can only approve runs needing review")
	}

	if run.SandboxID == nil {
		return nil, &domain.ValidationError{Field: "sandboxId", Message: "run has no sandbox to approve"}
	}

	if o.sandbox == nil {
		return nil, fmt.Errorf("no sandbox provider configured")
	}

	result, err := o.sandbox.PartialApprove(ctx, sandbox.PartialApproveRequest{
		SandboxID: *run.SandboxID,
		FileIDs:   req.FileIDs,
		Actor:     req.Actor,
		CommitMsg: req.CommitMsg,
	})
	if err != nil {
		return nil, err
	}

	// Update run status
	now := time.Now()
	if result.Remaining == 0 {
		run.ApprovalState = domain.ApprovalStateApproved
	} else {
		run.ApprovalState = domain.ApprovalStatePartiallyApproved
	}
	run.UpdatedAt = now
	o.runs.Update(ctx, run)

	return &ApproveResult{
		Success:    result.Success,
		Applied:    result.Applied,
		Remaining:  result.Remaining,
		IsPartial:  result.IsPartial,
		CommitHash: result.CommitHash,
		AppliedAt:  result.AppliedAt,
		ErrorMsg:   result.ErrorMsg,
	}, nil
}

// -----------------------------------------------------------------------------
// Event Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetRunEvents(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error) {
	if o.events == nil {
		return nil, fmt.Errorf("no event store configured")
	}
	return o.events.Get(ctx, runID, opts)
}

func (o *Orchestrator) StreamRunEvents(ctx context.Context, runID uuid.UUID, opts event.StreamOptions) (<-chan *domain.RunEvent, error) {
	if o.events == nil {
		return nil, fmt.Errorf("no event store configured")
	}
	return o.events.Stream(ctx, runID, opts)
}

// -----------------------------------------------------------------------------
// Diff Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetRunDiff(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}

	if run.SandboxID == nil {
		return nil, &domain.ValidationError{Field: "sandboxId", Message: "run has no sandbox"}
	}

	if o.sandbox == nil {
		return nil, fmt.Errorf("no sandbox provider configured")
	}

	return o.sandbox.GetDiff(ctx, *run.SandboxID)
}

// -----------------------------------------------------------------------------
// Status Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetHealth(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Status:  "healthy",
		Runners: make(map[string]RunnerStatus),
	}

	// Check sandbox
	if o.sandbox != nil {
		available, msg := o.sandbox.IsAvailable(ctx)
		status.Sandbox = ComponentStatus{Available: available, Message: msg}
	} else {
		status.Sandbox = ComponentStatus{Available: false, Message: "not configured"}
	}

	// Check runners
	if o.runners != nil {
		for _, r := range o.runners.List() {
			available, msg := r.IsAvailable(ctx)
			status.Runners[string(r.Type())] = RunnerStatus{
				Type:         r.Type(),
				Available:    available,
				Message:      msg,
				Capabilities: r.Capabilities(),
			}
		}
	}

	// Count active runs
	if o.runs != nil {
		runningStatus := domain.RunStatusRunning
		runs, _ := o.runs.List(ctx, repository.RunListFilter{Status: &runningStatus})
		status.ActiveRuns = len(runs)
	}

	// Count queued tasks
	if o.tasks != nil {
		tasks, _ := o.tasks.List(ctx, repository.ListFilter{})
		var queued int
		for _, t := range tasks {
			if t.Status == domain.TaskStatusQueued {
				queued++
			}
		}
		status.QueuedTasks = queued
	}

	return status, nil
}

func (o *Orchestrator) GetRunnerStatus(ctx context.Context) ([]*RunnerStatus, error) {
	if o.runners == nil {
		return nil, nil
	}

	var statuses []*RunnerStatus
	for _, r := range o.runners.List() {
		available, msg := r.IsAvailable(ctx)
		statuses = append(statuses, &RunnerStatus{
			Type:         r.Type(),
			Available:    available,
			Message:      msg,
			Capabilities: r.Capabilities(),
		})
	}
	return statuses, nil
}

// -----------------------------------------------------------------------------
// Helper Types
// -----------------------------------------------------------------------------

// eventStoreAdapter adapts event.Store to runner.EventSink
type eventStoreAdapter struct {
	store event.Store
	runID uuid.UUID
}

func (e *eventStoreAdapter) Emit(evt *domain.RunEvent) error {
	return e.store.Append(context.Background(), e.runID, evt)
}

func (e *eventStoreAdapter) Close() error {
	return nil
}

// noOpEventSink discards events
type noOpEventSink struct{}

func (n *noOpEventSink) Emit(_ *domain.RunEvent) error { return nil }
func (n *noOpEventSink) Close() error                  { return nil }

// valueOrDefault returns the pointer value or default
func valueOrDefault(ptr *domain.RunMode, def domain.RunMode) domain.RunMode {
	if ptr != nil {
		return *ptr
	}
	return def
}
