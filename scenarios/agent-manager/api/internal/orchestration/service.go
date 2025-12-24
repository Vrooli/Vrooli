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
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/modelregistry"
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
	EnsureProfile(ctx context.Context, req EnsureProfileRequest) (*EnsureProfileResult, error)

	// --- Task Operations ---
	CreateTask(ctx context.Context, task *domain.Task) (*domain.Task, error)
	GetTask(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	ListTasks(ctx context.Context, opts ListOptions) ([]*domain.Task, error)
	UpdateTask(ctx context.Context, task *domain.Task) (*domain.Task, error)
	CancelTask(ctx context.Context, id uuid.UUID) error
	DeleteTask(ctx context.Context, id uuid.UUID) error

	// --- Run Operations ---
	CreateRun(ctx context.Context, req CreateRunRequest) (*domain.Run, error)
	GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	GetRunByTag(ctx context.Context, tag string) (*domain.Run, error)
	ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error)
	DeleteRun(ctx context.Context, id uuid.UUID) error
	StopRun(ctx context.Context, id uuid.UUID) error
	StopRunByTag(ctx context.Context, tag string) error
	StopAllRuns(ctx context.Context, opts StopAllOptions) (*StopAllResult, error)

	// --- Run Resumption Operations (Interruption Resilience) ---
	ResumeRun(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	GetRunProgress(ctx context.Context, id uuid.UUID) (*domain.RunProgress, error)
	ListStaleRuns(ctx context.Context, staleDuration time.Duration) ([]*domain.Run, error)

	// --- Approval Operations ---
	ApproveRun(ctx context.Context, req ApproveRequest) (*ApproveResult, error)
	RejectRun(ctx context.Context, id uuid.UUID, actor, reason string) error
	PartialApprove(ctx context.Context, req PartialApproveRequest) (*ApproveResult, error)

	// --- Event Operations ---
	GetRunEvents(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error)
	StreamRunEvents(ctx context.Context, runID uuid.UUID, opts event.StreamOptions) (<-chan *domain.RunEvent, error)

	// --- Diff Operations ---
	GetRunDiff(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error)

	// --- Model Registry Operations ---
	GetModelRegistry(ctx context.Context) (*modelregistry.Registry, error)
	UpdateModelRegistry(ctx context.Context, registry *modelregistry.Registry) (*modelregistry.Registry, error)

	// --- Status Operations ---
	GetHealth(ctx context.Context) (*HealthStatus, error)
	GetRunnerStatus(ctx context.Context) ([]*RunnerStatus, error)
	ProbeRunner(ctx context.Context, runnerType domain.RunnerType) (*ProbeResult, error)

	// --- Maintenance Operations ---
	PurgeData(ctx context.Context, req PurgeRequest) (*PurgeResult, error)
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
	TagPrefix      string // Filter runs by tag prefix (e.g., "ecosystem-" to get all ecosystem-manager runs)
}

// PurgeTarget identifies entities eligible for purge.
type PurgeTarget int

const (
	PurgeTargetProfiles PurgeTarget = iota + 1
	PurgeTargetTasks
	PurgeTargetRuns
)

// PurgeRequest specifies a purge by regex pattern.
type PurgeRequest struct {
	Pattern string
	Targets []PurgeTarget
	DryRun  bool
}

// PurgeCounts reports matched/deleted counts.
type PurgeCounts struct {
	Profiles int
	Tasks    int
	Runs     int
}

// PurgeResult summarizes a purge operation.
type PurgeResult struct {
	Matched PurgeCounts
	Deleted PurgeCounts
	DryRun  bool
}

// CreateRunRequest contains parameters for creating a new run.
type CreateRunRequest struct {
	TaskID uuid.UUID `json:"taskId"`

	// Profile-based config (optional - can be nil if inline config provided)
	AgentProfileID *uuid.UUID `json:"agentProfileId,omitempty"`

	// ProfileRef resolves a profile by key with fallback defaults.
	ProfileRef *ProfileRef `json:"profileRef,omitempty"`

	// Custom tag for identification (defaults to run ID if not set)
	// Used for agent tracking, log filtering, and external process identification
	// Example: "ecosystem-task-123", "test-genie-abc"
	Tag string `json:"tag,omitempty"`

	// Inline config (optional - used if no profile, or overrides profile)
	RunnerType           *domain.RunnerType           `json:"runnerType,omitempty"`
	Model                *string                      `json:"model,omitempty"`
	ModelPreset          *domain.ModelPreset          `json:"modelPreset,omitempty"`
	MaxTurns             *int                         `json:"maxTurns,omitempty"`
	Timeout              *time.Duration               `json:"timeout,omitempty"`
	FallbackRunnerTypes  []domain.RunnerType          `json:"fallbackRunnerTypes,omitempty"`
	AllowedTools         []string                     `json:"allowedTools,omitempty"`
	DeniedTools          []string                     `json:"deniedTools,omitempty"`
	SkipPermissionPrompt *bool                        `json:"skipPermissionPrompt,omitempty"`
	RequiresSandbox      *bool                        `json:"requiresSandbox,omitempty"`
	RequiresApproval     *bool                        `json:"requiresApproval,omitempty"`
	AllowedPaths         []string                     `json:"allowedPaths,omitempty"`
	DeniedPaths          []string                     `json:"deniedPaths,omitempty"`

	// Sandbox behavior overrides (optional)
	SandboxConfig *domain.SandboxConfig `json:"sandboxConfig,omitempty"`

	// ExistingSandboxID reuses a pre-existing sandbox for the run.
	// Only supported in sandboxed mode.
	ExistingSandboxID *uuid.UUID `json:"existingSandboxId,omitempty"`

	// Execution options
	Prompt       string          `json:"prompt,omitempty"` // Optional override prompt
	RunMode      *domain.RunMode `json:"runMode,omitempty"`
	ForceInPlace bool            `json:"forceInPlace,omitempty"`

	// Force bypasses slot/capacity limits (use for manual user-initiated runs)
	// When true, the run starts even if MaxConcurrentRuns is exceeded.
	Force bool `json:"force,omitempty"`

	// IdempotencyKey enables safe retries of run creation.
	// If provided and a run with this key already exists, the existing run is returned.
	// Format suggestion: "run:{taskID}:{timestamp}" or caller-defined unique string.
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
}

// ProfileRef identifies a profile by key with optional defaults.
type ProfileRef struct {
	ProfileKey string               `json:"profileKey"`
	Defaults   *domain.AgentProfile `json:"defaults,omitempty"`
}

// EnsureProfileRequest resolves a profile by key.
type EnsureProfileRequest struct {
	ProfileKey     string               `json:"profileKey"`
	Defaults       *domain.AgentProfile `json:"defaults,omitempty"`
	UpdateExisting bool                 `json:"updateExisting,omitempty"`
}

// EnsureProfileResult captures profile resolution outcome.
type EnsureProfileResult struct {
	Profile *domain.AgentProfile `json:"profile"`
	Created bool                 `json:"created"`
	Updated bool                 `json:"updated"`
}

// StopAllOptions specifies which runs to stop in a bulk operation.
type StopAllOptions struct {
	TagPrefix string // Only stop runs with this tag prefix (empty = all)
	Force     bool   // Force termination even if graceful stop fails
}

// StopAllResult contains the outcome of a bulk stop operation.
type StopAllResult struct {
	Stopped   int      `json:"stopped"`   // Number of runs successfully stopped
	Failed    int      `json:"failed"`    // Number of runs that failed to stop
	Skipped   int      `json:"skipped"`   // Number of runs that were already stopped
	FailedIDs []string `json:"failedIds"` // IDs of runs that failed to stop
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
// Required fields per health-api.schema.json: status, service, timestamp, readiness
type HealthStatus struct {
	Status       string              `json:"status"`
	Service      string              `json:"service"`
	Timestamp    string              `json:"timestamp"`
	Readiness    bool                `json:"readiness"`
	Dependencies *HealthDependencies `json:"dependencies,omitempty"`
	ActiveRuns   int                 `json:"activeRuns"`
	QueuedTasks  int                 `json:"queuedTasks"`
}

// HealthDependencies contains dependency health status.
type HealthDependencies struct {
	Database *DependencyStatus            `json:"database,omitempty"`
	Sandbox  *DependencyStatus            `json:"sandbox,omitempty"`
	Runners  map[string]*DependencyStatus `json:"runners,omitempty"`
}

// DependencyStatus describes a dependency's health (matches schema).
type DependencyStatus struct {
	Connected bool    `json:"connected"`
	LatencyMs *int64  `json:"latency_ms,omitempty"`
	Error     *string `json:"error,omitempty"`
	Storage   string  `json:"storage,omitempty"`
}

// ComponentStatus describes a component's health.
type ComponentStatus struct {
	Available bool   `json:"available"`
	Message   string `json:"message,omitempty"`
}

// RunnerStatus describes a runner's availability.
type RunnerStatus struct {
	Type         domain.RunnerType   `json:"type"`
	Available    bool                `json:"available"`
	Message      string              `json:"message,omitempty"`
	Capabilities runner.Capabilities `json:"capabilities"`
}

// ProbeResult contains the result of probing a runner.
type ProbeResult struct {
	RunnerType domain.RunnerType `json:"runnerType"`
	Success    bool              `json:"success"`
	Message    string            `json:"message"`
	Response   string            `json:"response,omitempty"`
	DurationMs int64             `json:"durationMs"`
}

// -----------------------------------------------------------------------------
// Orchestrator Implementation
// -----------------------------------------------------------------------------

// Orchestrator coordinates agent execution using injected dependencies.
type Orchestrator struct {
	// Repositories (persistence)
	profiles    repository.ProfileRepository
	tasks       repository.TaskRepository
	runs        repository.RunRepository
	checkpoints repository.CheckpointRepository  // For resumption support
	idempotency repository.IdempotencyRepository // For replay safety

	// Adapters (external integrations)
	runners   runner.Registry
	sandbox   sandbox.Provider
	events    event.Store
	artifacts artifact.Collector

	// Policy evaluation
	policy policy.Evaluator

	// Lock management
	locks sandbox.LockManager

	// Real-time event broadcasting (WebSocket)
	broadcaster EventBroadcaster

	// Robust termination (Phase 2)
	terminator *Terminator

	// Configuration
	config OrchestratorConfig

	// Storage label for health reporting (e.g., postgres, sqlite, memory).
	storageLabel string

	// Model registry for runner model catalogs and presets.
	modelRegistry *modelregistry.Store
}

// OrchestratorConfig holds service configuration.
type OrchestratorConfig struct {
	DefaultTimeout          time.Duration
	MaxConcurrentRuns       int
	DefaultProjectRoot      string
	RequireSandboxByDefault bool
	DefaultSandboxConfig    *domain.SandboxConfig
	RunnerFallbackTypes     []domain.RunnerType
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

// WithCheckpoints sets the checkpoint repository for resumption support.
func WithCheckpoints(c repository.CheckpointRepository) Option {
	return func(o *Orchestrator) {
		o.checkpoints = c
	}
}

// WithIdempotency sets the idempotency repository for replay safety.
func WithIdempotency(i repository.IdempotencyRepository) Option {
	return func(o *Orchestrator) {
		o.idempotency = i
	}
}

// WithBroadcaster sets the event broadcaster for real-time WebSocket updates.
func WithBroadcaster(b EventBroadcaster) Option {
	return func(o *Orchestrator) {
		o.broadcaster = b
	}
}

// WithTerminator sets the terminator for robust process termination.
func WithTerminator(t *Terminator) Option {
	return func(o *Orchestrator) {
		o.terminator = t
	}
}

// WithStorageLabel sets the storage label reported by health checks.
func WithStorageLabel(label string) Option {
	return func(o *Orchestrator) {
		o.storageLabel = strings.TrimSpace(label)
	}
}

// WithModelRegistry sets the model registry store.
func WithModelRegistry(store *modelregistry.Store) Option {
	return func(o *Orchestrator) {
		o.modelRegistry = store
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

	if err := normalizeProfileInput(profile); err != nil {
		return nil, err
	}

	if err := o.profiles.Create(ctx, profile); err != nil {
		return nil, err
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

	if err := normalizeProfileInput(profile); err != nil {
		return nil, err
	}

	if err := o.profiles.Update(ctx, profile); err != nil {
		return nil, err
	}
	return profile, nil
}

func (o *Orchestrator) DeleteProfile(ctx context.Context, id uuid.UUID) error {
	return o.profiles.Delete(ctx, id)
}

// EnsureProfile resolves a profile by key, creating it with defaults if needed.
func (o *Orchestrator) EnsureProfile(ctx context.Context, req EnsureProfileRequest) (*EnsureProfileResult, error) {
	key := strings.TrimSpace(req.ProfileKey)
	if key == "" {
		return nil, domain.NewValidationErrorWithHint("profileKey", "field is required",
			"Provide a stable profile key for lookup or creation")
	}

	existing, err := o.profiles.GetByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	if existing != nil && !req.UpdateExisting {
		return &EnsureProfileResult{Profile: existing}, nil
	}

	if req.Defaults == nil {
		return nil, domain.NewValidationErrorWithHint("defaults", "field is required",
			"Provide default profile settings to create a new profile")
	}

	candidate := *req.Defaults
	candidate.ProfileKey = key
	if strings.TrimSpace(candidate.Name) == "" {
		candidate.Name = key
	}

	now := time.Now()
	if existing == nil {
		if candidate.ID == uuid.Nil {
			candidate.ID = uuid.New()
		}
		candidate.CreatedAt = now
		candidate.UpdatedAt = now

		if err := normalizeProfileInput(&candidate); err != nil {
			return nil, err
		}
		if err := o.profiles.Create(ctx, &candidate); err != nil {
			return nil, err
		}
		return &EnsureProfileResult{Profile: &candidate, Created: true}, nil
	}

	candidate.ID = existing.ID
	candidate.CreatedAt = existing.CreatedAt
	candidate.UpdatedAt = now
	if candidate.CreatedBy == "" {
		candidate.CreatedBy = existing.CreatedBy
	}

	if err := normalizeProfileInput(&candidate); err != nil {
		return nil, err
	}
	if err := o.profiles.Update(ctx, &candidate); err != nil {
		return nil, err
	}

	return &EnsureProfileResult{Profile: &candidate, Updated: true}, nil
}

func normalizeProfileInput(profile *domain.AgentProfile) error {
	if profile == nil {
		return domain.NewValidationError("profile", "cannot be nil")
	}

	name := strings.TrimSpace(profile.Name)
	key := strings.TrimSpace(profile.ProfileKey)
	if key == "" && name != "" {
		profile.ProfileKey = name
		key = name
	}
	if name == "" && key != "" {
		profile.Name = key
	}

	return profile.Validate()
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
		return nil, err
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
	if task == nil {
		return nil, domain.NewValidationError("task", "cannot be nil")
	}

	existing, err := o.tasks.Get(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, domain.NewNotFoundError("Task", task.ID)
	}

	// Preserve immutable/system-managed fields.
	updated := *existing
	updated.Title = task.Title
	updated.Description = task.Description
	updated.ScopePath = task.ScopePath
	updated.ProjectRoot = task.ProjectRoot
	updated.ContextAttachments = task.ContextAttachments
	updated.UpdatedAt = time.Now()

	if err := o.tasks.Update(ctx, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
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

func (o *Orchestrator) DeleteTask(ctx context.Context, id uuid.UUID) error {
	task, err := o.GetTask(ctx, id)
	if err != nil {
		return err
	}

	if task.Status != domain.TaskStatusCancelled {
		return domain.NewStateError("Task", string(task.Status), "delete", "can only delete cancelled tasks")
	}

	return o.tasks.Delete(ctx, id)
}

// -----------------------------------------------------------------------------
// Run Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) CreateRun(ctx context.Context, req CreateRunRequest) (*domain.Run, error) {
	// IDEMPOTENCY: Check if this request has already been processed
	if req.IdempotencyKey != "" && o.idempotency != nil {
		existing, err := o.idempotency.Check(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			// Request already processed - return cached result
			if existing.Status == domain.IdempotencyStatusComplete && existing.EntityID != nil {
				return o.GetRun(ctx, *existing.EntityID)
			}
			if existing.Status == domain.IdempotencyStatusPending {
				// Another request is in progress with this key
				return nil, domain.NewStateError("Run", "creating", "create",
					"a run creation with this idempotency key is already in progress")
			}
			// Failed status - allow retry by falling through
		}

		// Reserve the idempotency key for this operation
		if _, err := o.idempotency.Reserve(ctx, req.IdempotencyKey, 1*time.Hour); err != nil {
			// If reservation fails, another request beat us to it
			return nil, domain.NewStateError("Run", "creating", "create",
				"a run creation with this idempotency key is already in progress")
		}
	}

	// SLOT ENFORCEMENT: Check capacity unless Force is set
	if !req.Force && o.config.MaxConcurrentRuns > 0 && o.runs != nil {
		// Count active runs (both Running and Starting count against the limit)
		runningCount, err := o.runs.CountByStatus(ctx, domain.RunStatusRunning)
		if err != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, err
		}
		startingCount, err := o.runs.CountByStatus(ctx, domain.RunStatusStarting)
		if err != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, err
		}

		activeCount := runningCount + startingCount
		if activeCount >= o.config.MaxConcurrentRuns {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, &domain.CapacityExceededError{
				Resource: "concurrent_runs",
				Current:  activeCount,
				Maximum:  o.config.MaxConcurrentRuns,
			}
		}
	}

	// Get task
	task, err := o.GetTask(ctx, req.TaskID)
	if err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

	if req.AgentProfileID != nil && req.ProfileRef != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, domain.NewValidationErrorWithHint("agentProfileId/profileRef", "only one profile reference is allowed",
			"provide either agentProfileId or profileRef")
	}

	// Resolve configuration: profile (if provided) + inline overrides
	resolvedConfig, profile, err := o.resolveRunConfig(ctx, req)
	if err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

	sandboxConfig, err := o.resolveSandboxConfig(req, profile)
	if err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

	// Evaluate policies
	var policyDecision *policy.Decision
	if o.policy != nil {
		policyDecision, err = o.policy.EvaluateRunRequest(ctx, policy.EvaluateRequest{
			Task:          task,
			Profile:       profile,
			RequestedMode: valueOrDefault(req.RunMode, domain.RunModeSandboxed),
			ForceInPlace:  req.ForceInPlace,
		})
		if err != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewInternalError("policy evaluation failed", err)
		}
		if !policyDecision.Allowed {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
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
	} else if resolvedConfig != nil && !resolvedConfig.RequiresSandbox {
		runMode = domain.RunModeInPlace
	}

	if err := o.preflightScopePath(task, runMode, req.ExistingSandboxID); err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

	existingSandboxWorkDir := ""
	if req.ExistingSandboxID != nil {
		if runMode != domain.RunModeSandboxed {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "existing sandbox requires sandboxed run mode",
				"set runMode to sandboxed or enable requiresSandbox in the profile")
		}
		if o.sandbox == nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
		}

		sbx, err := o.sandbox.Get(ctx, *req.ExistingSandboxID)
		if err != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, err
		}
		switch sbx.Status {
		case sandbox.SandboxStatusDeleted, sandbox.SandboxStatusRejected, sandbox.SandboxStatusApproved, sandbox.SandboxStatusError:
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "sandbox is not reusable",
				fmt.Sprintf("sandbox status is %s", sbx.Status))
		case sandbox.SandboxStatusStopped:
			if err := o.sandbox.Start(ctx, sbx.ID); err != nil {
				o.markIdempotencyFailed(ctx, req.IdempotencyKey)
				return nil, err
			}
		}

		if trimmed := strings.TrimSpace(task.ProjectRoot); trimmed != "" && strings.TrimSpace(sbx.ProjectRoot) != "" && trimmed != sbx.ProjectRoot {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "sandbox project root does not match task",
				fmt.Sprintf("task projectRoot=%q, sandbox projectRoot=%q", trimmed, sbx.ProjectRoot))
		}
		if trimmed := strings.TrimSpace(task.ScopePath); trimmed != "" && strings.TrimSpace(sbx.ScopePath) != "" && trimmed != sbx.ScopePath {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint("existingSandboxId", "sandbox scope path does not match task",
				fmt.Sprintf("task scopePath=%q, sandbox scopePath=%q", trimmed, sbx.ScopePath))
		}

		if sbx.WorkDir != "" {
			existingSandboxWorkDir = sbx.WorkDir
		} else {
			workDir, err := o.sandbox.GetWorkspacePath(ctx, sbx.ID)
			if err != nil {
				o.markIdempotencyFailed(ctx, req.IdempotencyKey)
				return nil, err
			}
			existingSandboxWorkDir = workDir
		}
	}

	// Create the run with progress tracking initialized
	profileID := req.AgentProfileID
	if profile != nil {
		profileID = &profile.ID
	}
	run := &domain.Run{
		ID:              uuid.New(),
		TaskID:          task.ID,
		AgentProfileID:  profileID, // May be nil if inline config used
		Tag:             req.Tag,   // Custom tag for identification
		RunMode:         runMode,
		Status:          domain.RunStatusPending,
		Phase:           domain.RunPhaseQueued,
		ProgressPercent: 0,
		IdempotencyKey:  req.IdempotencyKey,
		ApprovalState:   domain.ApprovalStateNone,
		ResolvedConfig:  resolvedConfig,
		SandboxConfig:   sandboxConfig,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
	if run.ResolvedConfig != nil {
		run.ResolvedConfig.SandboxConfig = sandboxConfig
	}
	if req.ExistingSandboxID != nil {
		run.SandboxID = req.ExistingSandboxID
	}

	if err := o.runs.Create(ctx, run); err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

	// Mark idempotency as complete
	o.markIdempotencyComplete(ctx, req.IdempotencyKey, run.ID, "Run")

	// Determine the prompt: use override if provided, otherwise fall back to task description
	prompt := req.Prompt
	if prompt == "" {
		prompt = task.Description
	}

	// Start execution asynchronously
	go o.executeRun(context.Background(), run, task, profile, prompt, existingSandboxWorkDir)

	return run, nil
}

func (o *Orchestrator) preflightScopePath(task *domain.Task, runMode domain.RunMode, existingSandboxID *uuid.UUID) error {
	if runMode != domain.RunModeSandboxed || existingSandboxID != nil {
		return nil
	}

	scopePath := strings.TrimSpace(task.ScopePath)
	if scopePath == "" {
		return domain.NewValidationError("scopePath", "field is required")
	}

	projectRoot := strings.TrimSpace(task.ProjectRoot)
	if projectRoot == "" {
		projectRoot = strings.TrimSpace(o.config.DefaultProjectRoot)
	}
	if projectRoot == "" && !filepath.IsAbs(scopePath) {
		return domain.NewValidationErrorWithHint("projectRoot", "field is required for sandboxed run",
			"set projectRoot on the task or configure defaultProjectRoot")
	}

	absScopePath := scopePath
	if !filepath.IsAbs(absScopePath) && projectRoot != "" {
		absScopePath = filepath.Join(projectRoot, absScopePath)
	}
	absScopePath = filepath.Clean(absScopePath)

	info, err := os.Stat(absScopePath)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.NewValidationErrorWithHint("scopePath", "scope path does not exist",
				fmt.Sprintf("create the directory: %s", absScopePath))
		}
		return domain.NewValidationErrorWithHint("scopePath", "unable to stat scope path",
			fmt.Sprintf("check permissions for %s", absScopePath))
	}
	if !info.IsDir() {
		return domain.NewValidationErrorWithHint("scopePath", "scope path is not a directory",
			fmt.Sprintf("scope path resolves to %s", absScopePath))
	}

	return nil
}

// markIdempotencyFailed marks an idempotency key as failed (allows retry).
func (o *Orchestrator) markIdempotencyFailed(ctx context.Context, key string) {
	if key == "" || o.idempotency == nil {
		return
	}
	_ = o.idempotency.Fail(ctx, key)
}

// markIdempotencyComplete marks an idempotency key as successfully completed.
func (o *Orchestrator) markIdempotencyComplete(ctx context.Context, key string, entityID uuid.UUID, entityType string) {
	if key == "" || o.idempotency == nil {
		return
	}
	_ = o.idempotency.Complete(ctx, key, entityID, entityType, nil)
}

// resolveRunConfig resolves the run configuration from profile and/or inline config.
// Returns the resolved config and the profile (if loaded, may be nil for pure inline config).
func (o *Orchestrator) resolveRunConfig(ctx context.Context, req CreateRunRequest) (*domain.RunConfig, *domain.AgentProfile, error) {
	cfg := domain.DefaultRunConfig()
	var profile *domain.AgentProfile

	// Load profile if provided
	if req.AgentProfileID != nil {
		var err error
		profile, err = o.GetProfile(ctx, *req.AgentProfileID)
		if err != nil {
			return nil, nil, err
		}
		cfg.ApplyProfile(profile)
	}

	// Resolve profile by key if provided
	if req.ProfileRef != nil {
		result, err := o.EnsureProfile(ctx, EnsureProfileRequest{
			ProfileKey: req.ProfileRef.ProfileKey,
			Defaults:   req.ProfileRef.Defaults,
		})
		if err != nil {
			return nil, nil, err
		}
		profile = result.Profile
		if profile != nil {
			cfg.ApplyProfile(profile)
		}
	}

	// Apply inline overrides
	if req.RunnerType != nil {
		cfg.RunnerType = *req.RunnerType
	}
	if req.ModelPreset != nil {
		cfg.ModelPreset = *req.ModelPreset
		if cfg.ModelPreset != domain.ModelPresetUnspecified {
			cfg.Model = ""
		}
	}
	if req.Model != nil {
		cfg.Model = *req.Model
		if strings.TrimSpace(cfg.Model) != "" {
			cfg.ModelPreset = domain.ModelPresetUnspecified
		}
	}
	if req.MaxTurns != nil {
		cfg.MaxTurns = *req.MaxTurns
	}
	if req.Timeout != nil {
		cfg.Timeout = *req.Timeout
	}
	if req.FallbackRunnerTypes != nil {
		cfg.FallbackRunnerTypes = append([]domain.RunnerType(nil), req.FallbackRunnerTypes...)
	}
	if req.AllowedTools != nil {
		cfg.AllowedTools = req.AllowedTools
	}
	if req.DeniedTools != nil {
		cfg.DeniedTools = req.DeniedTools
	}
	if req.SkipPermissionPrompt != nil {
		cfg.SkipPermissionPrompt = *req.SkipPermissionPrompt
	}
	if req.RequiresSandbox != nil {
		cfg.RequiresSandbox = *req.RequiresSandbox
	}
	if req.RequiresApproval != nil {
		cfg.RequiresApproval = *req.RequiresApproval
	}
	if req.AllowedPaths != nil {
		cfg.AllowedPaths = req.AllowedPaths
	}
	if req.DeniedPaths != nil {
		cfg.DeniedPaths = req.DeniedPaths
	}
	if len(cfg.FallbackRunnerTypes) == 0 && o.modelRegistry != nil {
		registry := o.modelRegistry.Get()
		if registry != nil && len(registry.FallbackRunnerTypes) > 0 {
			cfg.FallbackRunnerTypes = make([]domain.RunnerType, 0, len(registry.FallbackRunnerTypes))
			for _, rt := range registry.FallbackRunnerTypes {
				cfg.FallbackRunnerTypes = append(cfg.FallbackRunnerTypes, domain.RunnerType(rt))
			}
		}
	}
	if len(cfg.FallbackRunnerTypes) == 0 && len(o.config.RunnerFallbackTypes) > 0 {
		cfg.FallbackRunnerTypes = append([]domain.RunnerType(nil), o.config.RunnerFallbackTypes...)
	}

	// Validate the resolved config
	if !cfg.RunnerType.IsValid() {
		return nil, nil, domain.NewValidationError("runnerType", "invalid runner type: "+string(cfg.RunnerType))
	}
	for _, rt := range cfg.FallbackRunnerTypes {
		if !rt.IsValid() {
			return nil, nil, domain.NewValidationError("fallbackRunnerTypes", "invalid runner type: "+string(rt))
		}
	}
	if !cfg.ModelPreset.IsValid() {
		return nil, nil, domain.NewValidationError("modelPreset", "invalid model preset")
	}
	if strings.TrimSpace(cfg.Model) != "" && cfg.ModelPreset != domain.ModelPresetUnspecified {
		return nil, nil, domain.NewValidationError("modelPreset", "cannot set model and model preset together")
	}
	if strings.TrimSpace(cfg.Model) == "" && cfg.ModelPreset != domain.ModelPresetUnspecified {
		if o.modelRegistry == nil {
			return nil, nil, domain.NewValidationError("modelPreset", "model registry not configured")
		}
		resolved, ok := o.modelRegistry.ResolvePreset(string(cfg.RunnerType), string(cfg.ModelPreset))
		if !ok {
			return nil, nil, domain.NewValidationError("modelPreset", "preset not mapped for runner")
		}
		cfg.Model = resolved
	}
	return cfg, profile, nil
}

func (o *Orchestrator) resolveSandboxConfig(req CreateRunRequest, profile *domain.AgentProfile) (*domain.SandboxConfig, error) {
	cfg := cloneSandboxConfig(o.config.DefaultSandboxConfig)
	if profile != nil && profile.SandboxConfig != nil {
		cfg = cloneSandboxConfig(profile.SandboxConfig)
	}
	if req.SandboxConfig != nil {
		cfg = cloneSandboxConfig(req.SandboxConfig)
	}
	cfg = normalizeSandboxConfig(cfg)
	if err := validateSandboxConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func cloneSandboxConfig(cfg *domain.SandboxConfig) *domain.SandboxConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.Lifecycle.StopOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.StopOn...)
	clone.Lifecycle.DeleteOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.DeleteOn...)
	clone.Acceptance.Allow = cloneSandboxCriteria(cfg.Acceptance.Allow)
	clone.Acceptance.Deny = cloneSandboxCriteria(cfg.Acceptance.Deny)
	return &clone
}

func cloneSandboxCriteria(criteria domain.SandboxFileCriteria) domain.SandboxFileCriteria {
	return domain.SandboxFileCriteria{
		PathGlobs:  append([]string(nil), criteria.PathGlobs...),
		Extensions: append([]string(nil), criteria.Extensions...),
	}
}

func normalizeSandboxConfig(cfg *domain.SandboxConfig) *domain.SandboxConfig {
	if cfg == nil {
		return nil
	}
	if cfg.Acceptance.Mode == "" {
		cfg.Acceptance.Mode = "allowlist"
	}
	cfg.Acceptance.Allow = normalizeSandboxCriteria(cfg.Acceptance.Allow)
	cfg.Acceptance.Deny = normalizeSandboxCriteria(cfg.Acceptance.Deny)
	return cfg
}

func normalizeSandboxCriteria(criteria domain.SandboxFileCriteria) domain.SandboxFileCriteria {
	paths := make([]string, 0, len(criteria.PathGlobs))
	seenPaths := make(map[string]bool)
	for _, p := range criteria.PathGlobs {
		p = strings.TrimSpace(p)
		if p == "" || seenPaths[p] {
			continue
		}
		seenPaths[p] = true
		paths = append(paths, p)
	}

	exts := make([]string, 0, len(criteria.Extensions))
	seenExts := make(map[string]bool)
	for _, ext := range criteria.Extensions {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		ext = strings.ToLower(ext)
		if seenExts[ext] {
			continue
		}
		seenExts[ext] = true
		exts = append(exts, ext)
	}

	criteria.PathGlobs = paths
	criteria.Extensions = exts
	return criteria
}

func validateSandboxConfig(cfg *domain.SandboxConfig) error {
	if cfg == nil {
		return nil
	}
	if cfg.Acceptance.Mode != "" && cfg.Acceptance.Mode != "allowlist" {
		return domain.NewValidationError("sandboxConfig.acceptance.mode", "unsupported acceptance mode")
	}
	if cfg.Lifecycle.TTL < 0 {
		return domain.NewValidationError("sandboxConfig.lifecycle.ttl", "ttl cannot be negative")
	}
	if cfg.Lifecycle.IdleTimeout < 0 {
		return domain.NewValidationError("sandboxConfig.lifecycle.idleTimeout", "idleTimeout cannot be negative")
	}
	for _, p := range append(cfg.Acceptance.Allow.PathGlobs, cfg.Acceptance.Deny.PathGlobs...) {
		if filepath.IsAbs(p) || strings.HasPrefix(p, "/") {
			return domain.NewValidationErrorWithHint(
				"sandboxConfig.acceptance.pathGlobs",
				"path globs must be project-root relative",
				"Remove the leading '/' and use project-root relative patterns",
			)
		}
	}
	return nil
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
		TagPrefix:      opts.TagPrefix,
	})
}

func (o *Orchestrator) DeleteRun(ctx context.Context, id uuid.UUID) error {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if run.Status == domain.RunStatusPending ||
		run.Status == domain.RunStatusStarting ||
		run.Status == domain.RunStatusRunning {
		return domain.NewStateError("Run", string(run.Status), "delete", "stop the run before deleting it")
	}
	return o.runs.Delete(ctx, id)
}

// GetRunByTag retrieves a run by its custom tag.
// Returns NotFoundError if no run with that tag exists.
func (o *Orchestrator) GetRunByTag(ctx context.Context, tag string) (*domain.Run, error) {
	// List all runs with matching tag prefix and find exact match
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		TagPrefix: tag,
	})
	if err != nil {
		return nil, err
	}

	// Find exact match
	for _, run := range runs {
		if run.GetTag() == tag {
			return run, nil
		}
	}

	return nil, domain.NewNotFoundError("Run", uuid.Nil)
}

// StopRunByTag stops a run identified by its custom tag.
func (o *Orchestrator) StopRunByTag(ctx context.Context, tag string) error {
	run, err := o.GetRunByTag(ctx, tag)
	if err != nil {
		return err
	}
	return o.StopRun(ctx, run.ID)
}

// StopAllRuns stops all running runs, optionally filtered by tag prefix.
func (o *Orchestrator) StopAllRuns(ctx context.Context, opts StopAllOptions) (*StopAllResult, error) {
	result := &StopAllResult{
		FailedIDs: []string{},
	}

	// Get all running or starting runs
	runningStatus := domain.RunStatusRunning
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		Status:    &runningStatus,
		TagPrefix: opts.TagPrefix,
	})
	if err != nil {
		return nil, err
	}

	// Also get starting runs
	startingStatus := domain.RunStatusStarting
	startingRuns, err := o.runs.List(ctx, repository.RunListFilter{
		Status:    &startingStatus,
		TagPrefix: opts.TagPrefix,
	})
	if err != nil {
		return nil, err
	}
	runs = append(runs, startingRuns...)

	// Stop each run
	for _, run := range runs {
		// Skip already stopped runs
		if run.Status == domain.RunStatusComplete ||
			run.Status == domain.RunStatusFailed ||
			run.Status == domain.RunStatusCancelled {
			result.Skipped++
			continue
		}

		if err := o.StopRun(ctx, run.ID); err != nil {
			result.Failed++
			result.FailedIDs = append(result.FailedIDs, run.ID.String())
		} else {
			result.Stopped++
		}
	}

	return result, nil
}

func (o *Orchestrator) StopRun(ctx context.Context, id uuid.UUID) error {
	// Use the robust terminator if available (Phase 2)
	if o.terminator != nil {
		return o.terminator.StopRunWithRetry(ctx, id)
	}

	// Fallback to simple implementation
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}

	if run.Status != domain.RunStatusRunning && run.Status != domain.RunStatusStarting {
		return domain.NewStateError("Run", string(run.Status), "stop", "can only stop running or starting runs")
	}

	// Get the runner type from resolved config or profile
	var runnerType domain.RunnerType
	if run.ResolvedConfig != nil {
		runnerType = run.ResolvedConfig.RunnerType
	} else if run.AgentProfileID != nil {
		if profile, err := o.GetProfile(ctx, *run.AgentProfileID); err == nil && profile != nil {
			runnerType = profile.RunnerType
		}
	}

	// Stop execution if we have a runner type
	if o.runners != nil && runnerType != "" {
		if r, err := o.runners.Get(runnerType); err == nil {
			if err := r.Stop(ctx, run.ID); err != nil {
				return err
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
// This delegates to RunExecutor for the actual work.
func (o *Orchestrator) executeRun(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, prompt string, existingSandboxWorkDir string) {
	executor := NewRunExecutor(
		o.runs,
		o.runners,
		o.sandbox,
		o.events,
		run,
		task,
		profile,
		prompt,
	)
	// Configure executor with checkpoint repository if available
	if o.checkpoints != nil {
		executor.WithCheckpointRepository(o.checkpoints)
	}
	if run.SandboxID != nil {
		workDir := existingSandboxWorkDir
		if workDir == "" && o.sandbox != nil {
			if resolved, err := o.sandbox.GetWorkspacePath(ctx, *run.SandboxID); err == nil {
				workDir = resolved
			}
		}
		executor.WithExistingSandbox(*run.SandboxID, workDir)
	}
	// Configure executor with broadcaster for real-time WebSocket updates
	if o.broadcaster != nil {
		executor.WithBroadcaster(o.broadcaster)
	}
	executor.Execute(ctx)
}

// -----------------------------------------------------------------------------
// Run Resumption Operations (Interruption Resilience)
// -----------------------------------------------------------------------------

// ResumeRun attempts to resume a stalled or interrupted run from its last checkpoint.
// This enables safe recovery from crashes, network issues, or intentional pauses.
//
// IDEMPOTENCY: Resuming an already-running or completed run is a no-op.
// TEMPORAL FLOW: Validates the run hasn't exceeded its stale threshold.
// PROGRESS CONTINUITY: Uses checkpoints to skip completed phases.
func (o *Orchestrator) ResumeRun(ctx context.Context, id uuid.UUID) (*domain.Run, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}

	// Validate resumability using domain decision helper
	if !run.IsResumable() {
		return nil, domain.NewStateError("Run", string(run.Status), "resume",
			fmt.Sprintf("run in %s state cannot be resumed", run.Status))
	}

	// Get the last checkpoint
	var checkpoint *domain.RunCheckpoint
	if o.checkpoints != nil {
		checkpoint, err = o.checkpoints.Get(ctx, id)
		if err != nil {
		return nil, err
		}
	}

	// If no checkpoint, start from the beginning
	if checkpoint == nil {
		checkpoint = domain.NewCheckpoint(id, domain.RunPhaseQueued)
	}

	// Get associated entities
	task, err := o.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}

	// Get profile if available (may be nil for inline config runs)
	var profile *domain.AgentProfile
	if run.AgentProfileID != nil {
		profile, err = o.GetProfile(ctx, *run.AgentProfileID)
		if err != nil {
			return nil, err
		}
	}

	// Update status to running
	run.Status = domain.RunStatusRunning
	run.UpdatedAt = time.Now()
	if err := o.runs.Update(ctx, run); err != nil {
		return nil, err
	}

	// Start execution asynchronously with resumption
	go o.resumeRun(context.Background(), run, task, profile, checkpoint)

	return run, nil
}

// resumeRun handles the actual agent resumption (runs in background).
func (o *Orchestrator) resumeRun(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, checkpoint *domain.RunCheckpoint) {
	executor := NewRunExecutor(
		o.runs,
		o.runners,
		o.sandbox,
		o.events,
		run,
		task,
		profile,
		"", // No new prompt for resume
	)

	// Configure for resumption
	if o.checkpoints != nil {
		executor.WithCheckpointRepository(o.checkpoints)
	}
	// Configure executor with broadcaster for real-time WebSocket updates
	if o.broadcaster != nil {
		executor.WithBroadcaster(o.broadcaster)
	}
	executor.WithResumeFrom(checkpoint)

	executor.Execute(ctx)
}

// GetRunProgress returns the current progress of a run for display.
// This provides visibility into what phase a run is in and estimated completion.
func (o *Orchestrator) GetRunProgress(ctx context.Context, id uuid.UUID) (*domain.RunProgress, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return nil, err
	}

	progress := &domain.RunProgress{
		Phase:            run.Phase,
		PhaseDescription: run.Phase.Description(),
		PercentComplete:  run.ProgressPercent,
		LastUpdate:       run.UpdatedAt,
	}

	// Calculate elapsed time
	if run.StartedAt != nil {
		progress.ElapsedTime = time.Since(*run.StartedAt)
	}

	// Add current action description based on phase
	switch run.Phase {
	case domain.RunPhaseExecuting:
		progress.CurrentAction = "Agent is working on the task"
	case domain.RunPhaseAwaitingReview:
		progress.CurrentAction = "Changes ready for review"
	case domain.RunPhaseApplying:
		progress.CurrentAction = "Applying approved changes"
	}

	return progress, nil
}

// ListStaleRuns returns runs that appear to have stalled based on their last heartbeat.
// This enables monitoring and automatic recovery of stuck runs.
func (o *Orchestrator) ListStaleRuns(ctx context.Context, staleDuration time.Duration) ([]*domain.Run, error) {
	// Get all running runs
	runningStatus := domain.RunStatusRunning
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		Status: &runningStatus,
	})
	if err != nil {
		return nil, err
	}

	// Filter to stale runs
	var staleRuns []*domain.Run
	for _, run := range runs {
		if run.IsStale(staleDuration) {
			staleRuns = append(staleRuns, run)
		}
	}

	return staleRuns, nil
}

// NOTE: Approval operations (ApproveRun, RejectRun, PartialApprove)
// are implemented in approval.go for cognitive load reduction.

// -----------------------------------------------------------------------------
// Event Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetRunEvents(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error) {
	if o.events == nil {
		return nil, domain.NewConfigMissingError("eventStore", "not configured", nil)
	}
	return o.events.Get(ctx, runID, opts)
}

func (o *Orchestrator) StreamRunEvents(ctx context.Context, runID uuid.UUID, opts event.StreamOptions) (<-chan *domain.RunEvent, error) {
	if o.events == nil {
		return nil, domain.NewConfigMissingError("eventStore", "not configured", nil)
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
		return nil, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}

	return o.sandbox.GetDiff(ctx, *run.SandboxID)
}

// -----------------------------------------------------------------------------
// Model Registry Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetModelRegistry(ctx context.Context) (*modelregistry.Registry, error) {
	if o.modelRegistry == nil {
		return nil, domain.NewStateError("ModelRegistry", "unconfigured", "get", "model registry not configured")
	}
	return o.modelRegistry.Get(), nil
}

func (o *Orchestrator) UpdateModelRegistry(ctx context.Context, registry *modelregistry.Registry) (*modelregistry.Registry, error) {
	if o.modelRegistry == nil {
		return nil, domain.NewStateError("ModelRegistry", "unconfigured", "update", "model registry not configured")
	}
	return o.modelRegistry.Update(registry)
}

// -----------------------------------------------------------------------------
// Status Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetHealth(ctx context.Context) (*HealthStatus, error) {
	status := &HealthStatus{
		Status:    "healthy",
		Service:   "agent-manager",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Readiness: true,
		Dependencies: &HealthDependencies{
			Runners: make(map[string]*DependencyStatus),
		},
	}

	// Check database (repositories configured)
	if o.profiles != nil && o.tasks != nil && o.runs != nil {
		status.Dependencies.Database = &DependencyStatus{Connected: true, Storage: o.storageLabel}
	} else {
		msg := "not configured"
		status.Dependencies.Database = &DependencyStatus{
			Connected: false,
			Error:     &msg,
			Storage:   o.storageLabel,
		}
	}

	// Check sandbox
	if o.sandbox != nil {
		available, msg := o.sandbox.IsAvailable(ctx)
		status.Dependencies.Sandbox = &DependencyStatus{
			Connected: available,
		}
		if !available && msg != "" {
			status.Dependencies.Sandbox.Error = &msg
		}
	} else {
		msg := "not configured"
		status.Dependencies.Sandbox = &DependencyStatus{
			Connected: false,
			Error:     &msg,
		}
	}

	// Check runners
	if o.runners != nil {
		for _, r := range o.runners.List() {
			available, msg := r.IsAvailable(ctx)
			depStatus := &DependencyStatus{
				Connected: available,
			}
			if !available && msg != "" {
				depStatus.Error = &msg
			}
			status.Dependencies.Runners[string(r.Type())] = depStatus
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

// ProbeRunner sends a real test request to a runner to verify end-to-end functionality.
// This invokes the agent with a minimal prompt to verify CLI + auth + API all work.
func (o *Orchestrator) ProbeRunner(ctx context.Context, runnerType domain.RunnerType) (*ProbeResult, error) {
	if o.runners == nil {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    "no runner registry configured",
		}, nil
	}

	r, err := o.runners.Get(runnerType)
	if err != nil {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("runner not found: %v", err),
		}, nil
	}

	// First check if the runner reports itself as available
	available, msg := r.IsAvailable(ctx)
	if !available {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    msg,
		}, nil
	}

	// Build the probe command - uses a minimal prompt to reduce cost/time
	// The prompt asks for a specific response so we can validate it
	start := time.Now()
	var probeCmd *exec.Cmd
	var cmdName string
	var codexOutputFile string
	probePrompt := "Reply with exactly one word: PROBE_OK"

	// Use a timeout context for the probe (30 seconds should be plenty)
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	switch runnerType {
	case domain.RunnerTypeClaudeCode:
		cmdName = "claude"
		// Use print mode for non-interactive, max tokens to limit response
		probeCmd = exec.CommandContext(probeCtx, cmdName, "-p", "--output-format", "text", probePrompt)
	case domain.RunnerTypeCodex:
		cmdName = "codex"
		// Use exec subcommand for non-interactive execution
		// --skip-git-repo-check allows running from /tmp without a git repo
		// -o writes just the response to a file (avoids session metadata in stdout)
		codexOutputFile = fmt.Sprintf("/tmp/codex-probe-%s.txt", uuid.New().String()[:8])
		probeCmd = exec.CommandContext(probeCtx, cmdName, "exec", "--skip-git-repo-check", "-o", codexOutputFile, probePrompt)
	case domain.RunnerTypeOpenCode:
		cmdName = "opencode"
		// Use run subcommand
		probeCmd = exec.CommandContext(probeCtx, cmdName, "run", probePrompt)
	default:
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("unknown runner type: %s", runnerType),
		}, nil
	}

	// Run from a safe directory (temp) to avoid any project-specific behavior
	probeCmd.Dir = "/tmp"

	output, err := probeCmd.CombinedOutput()
	duration := time.Since(start)

	// For Codex, read the clean output from the file instead of stdout
	var outputStr string
	if codexOutputFile != "" {
		defer os.Remove(codexOutputFile) // Clean up temp file
		if fileContent, readErr := os.ReadFile(codexOutputFile); readErr == nil {
			outputStr = strings.TrimSpace(string(fileContent))
		} else {
			// Fall back to stdout if file read fails
			outputStr = strings.TrimSpace(string(output))
		}
	} else {
		outputStr = strings.TrimSpace(string(output))
	}

	// Strip ANSI escape codes for cleaner output and matching
	outputClean := stripANSI(outputStr)

	// Check for timeout
	if probeCtx.Err() == context.DeadlineExceeded {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("%s probe timed out after 30s", cmdName),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Check for command execution error (non-zero exit code)
	if err != nil {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("%s probe failed: %v", cmdName, err),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Check for error patterns in output (some CLIs return exit 0 on failure)
	outputLower := strings.ToLower(outputClean)
	if strings.Contains(outputLower, "error:") ||
		strings.Contains(outputLower, "unauthorized") ||
		strings.Contains(outputLower, "authentication failed") ||
		strings.Contains(outputLower, "api key") ||
		strings.Contains(outputLower, "rate limit") {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    false,
			Message:    fmt.Sprintf("%s returned error in output", cmdName),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Validate we got a meaningful response
	// The agent should have responded with something containing "PROBE_OK" or similar
	if strings.Contains(strings.ToUpper(outputClean), "PROBE_OK") ||
		strings.Contains(strings.ToUpper(outputClean), "PROBE OK") {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    true,
			Message:    fmt.Sprintf("%s responded correctly", cmdName),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Got a response but not the expected one - still counts as working
	// (the agent might rephrase or add context, which is fine)
	if len(outputClean) > 0 {
		return &ProbeResult{
			RunnerType: runnerType,
			Success:    true,
			Message:    fmt.Sprintf("%s responded (content varies)", cmdName),
			Response:   outputClean,
			DurationMs: duration.Milliseconds(),
		}, nil
	}

	// Empty response is suspicious
	return &ProbeResult{
		RunnerType: runnerType,
		Success:    false,
		Message:    fmt.Sprintf("%s returned empty response", cmdName),
		Response:   "",
		DurationMs: duration.Milliseconds(),
	}, nil
}

// PurgeData deletes profiles, tasks, or runs matching a regex pattern.
func (o *Orchestrator) PurgeData(ctx context.Context, req PurgeRequest) (*PurgeResult, error) {
	pattern := strings.TrimSpace(req.Pattern)
	if pattern == "" {
		return nil, domain.NewValidationError("pattern", "pattern is required")
	}
	if len(req.Targets) == 0 {
		return nil, domain.NewValidationError("targets", "at least one target is required")
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, domain.NewValidationError("pattern", "invalid regex pattern")
	}

	targets := map[PurgeTarget]bool{}
	for _, t := range req.Targets {
		targets[t] = true
	}

	result := &PurgeResult{
		Matched: PurgeCounts{},
		Deleted: PurgeCounts{},
		DryRun:  req.DryRun,
	}

	var profileIDs []uuid.UUID
	if targets[PurgeTargetProfiles] {
		profiles, err := o.profiles.List(ctx, repository.ListFilter{})
		if err != nil {
			return nil, err
		}
		for _, profile := range profiles {
			if re.MatchString(profile.ProfileKey) {
				result.Matched.Profiles++
				profileIDs = append(profileIDs, profile.ID)
			}
		}
	}

	var taskIDs []uuid.UUID
	if targets[PurgeTargetTasks] {
		tasks, err := o.tasks.List(ctx, repository.ListFilter{})
		if err != nil {
			return nil, err
		}
		for _, task := range tasks {
			if re.MatchString(task.Title) {
				result.Matched.Tasks++
				taskIDs = append(taskIDs, task.ID)
			}
		}
	}

	var runIDs []uuid.UUID
	if targets[PurgeTargetRuns] {
		runs, err := o.runs.List(ctx, repository.RunListFilter{})
		if err != nil {
			return nil, err
		}
		for _, run := range runs {
			if re.MatchString(run.GetTag()) {
				result.Matched.Runs++
				runIDs = append(runIDs, run.ID)
			}
		}
	}

	if req.DryRun {
		return result, nil
	}

	for _, id := range runIDs {
		if o.events != nil {
			if err := o.events.Delete(ctx, id); err != nil {
				return nil, err
			}
		}
		if o.checkpoints != nil {
			if err := o.checkpoints.Delete(ctx, id); err != nil {
				return nil, err
			}
		}
		if err := o.runs.Delete(ctx, id); err != nil {
			return nil, err
		}
		result.Deleted.Runs++
	}

	for _, id := range taskIDs {
		if err := o.tasks.Delete(ctx, id); err != nil {
			return nil, err
		}
		result.Deleted.Tasks++
	}

	for _, id := range profileIDs {
		if err := o.profiles.Delete(ctx, id); err != nil {
			return nil, err
		}
		result.Deleted.Profiles++
	}

	return result, nil
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Match ANSI escape sequences: ESC[ followed by params and a letter
	// This handles color codes, cursor movement, etc.
	result := strings.Builder{}
	inEscape := false
	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++ // skip the '['
			continue
		}
		if inEscape {
			// End of escape sequence is a letter (A-Z, a-z)
			if (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= 'a' && s[i] <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteByte(s[i])
	}
	return result.String()
}

// -----------------------------------------------------------------------------
// Helper Types
// -----------------------------------------------------------------------------

// EventBroadcaster is a callback for broadcasting events in real-time.
// This is typically implemented by the WebSocket hub.
type EventBroadcaster interface {
	BroadcastEvent(event *domain.RunEvent)
	BroadcastRunStatus(run *domain.Run)
	BroadcastProgress(runID uuid.UUID, phase domain.RunPhase, percent int, action string)
}

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

// broadcastingEventSink stores events AND broadcasts them via WebSocket.
type broadcastingEventSink struct {
	store       event.Store
	runID       uuid.UUID
	broadcaster EventBroadcaster
}

func (b *broadcastingEventSink) Emit(evt *domain.RunEvent) error {
	// Validate event and log warnings for missing data
	domain.ValidateEvent(evt)

	// Store the event
	if b.store != nil {
		if err := b.store.Append(context.Background(), b.runID, evt); err != nil {
			// Log but don't fail - broadcasting is more important for UX
			_ = err
		}
	}

	// Broadcast the event via WebSocket
	if b.broadcaster != nil {
		b.broadcaster.BroadcastEvent(evt)

		// Also emit progress events for status changes
		if data, ok := evt.Data.(*domain.StatusEventData); ok {
			b.broadcaster.BroadcastProgress(b.runID, domain.RunPhase(data.NewStatus), 0, data.Reason)
		}
		if data, ok := evt.Data.(*domain.ProgressEventData); ok {
			b.broadcaster.BroadcastProgress(b.runID, data.Phase, data.PercentComplete, data.CurrentAction)
		}
	}

	return nil
}

func (b *broadcastingEventSink) Close() error {
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
