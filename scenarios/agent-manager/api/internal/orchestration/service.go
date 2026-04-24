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
	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/recommendation"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	agentconfig "agent-manager/internal/config"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/modelregistry"
	"agent-manager/internal/policy"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/repository"
	"agent-manager/internal/storage"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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
	CreateInvestigationRun(ctx context.Context, req CreateInvestigationRequest) (*domain.Run, error)
	CreateInvestigationApplyRun(ctx context.Context, req CreateInvestigationApplyRequest) (*domain.Run, error)
	ResumeFromFailedRun(ctx context.Context, req ResumeFromFailedRunRequest) (*domain.Run, error)
	GetRun(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	GetRunByTag(ctx context.Context, tag string) (*domain.Run, error)
	ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error)
	DeleteRun(ctx context.Context, id uuid.UUID) error
	StopRun(ctx context.Context, id uuid.UUID) error
	StopRunByTag(ctx context.Context, tag string) error
	StopAllRuns(ctx context.Context, opts StopAllOptions) (*StopAllResult, error)
	ContinueRun(ctx context.Context, req ContinueRunRequest) (*domain.Run, error)
	DeleteRunMessage(ctx context.Context, runID uuid.UUID, eventID uuid.UUID) (*domain.RunEvent, error)

	// --- Run Resumption Operations (Interruption Resilience) ---
	ResumeRun(ctx context.Context, id uuid.UUID) (*domain.Run, error)
	GetRunProgress(ctx context.Context, id uuid.UUID) (*domain.RunProgress, error)
	ListStaleRuns(ctx context.Context, staleDuration time.Duration) ([]*domain.Run, error)

	// --- Approval Operations ---
	ApproveRun(ctx context.Context, req ApproveRequest) (*ApproveResult, error)
	RejectRun(ctx context.Context, id uuid.UUID, actor, reason string) error
	PartialApprove(ctx context.Context, req PartialApproveRequest) (*ApproveResult, error)
	SyncRunFromSandbox(ctx context.Context, req SandboxSyncRequest) (*domain.Run, error)

	// --- Event Operations ---
	GetRunEvents(ctx context.Context, runID uuid.UUID, opts event.GetOptions) ([]*domain.RunEvent, error)
	StreamRunEvents(ctx context.Context, runID uuid.UUID, opts event.StreamOptions) (<-chan *domain.RunEvent, error)

	// --- Diff Operations ---
	GetRunDiff(ctx context.Context, runID uuid.UUID) (*sandbox.DiffResult, error)

	// --- Model Registry Operations ---
	GetModelRegistry(ctx context.Context) (*modelregistry.Registry, error)
	UpdateModelRegistry(ctx context.Context, registry *modelregistry.Registry) (*modelregistry.Registry, error)
	GetModelRegistryHealth(ctx context.Context) (modelregistry.HealthSnapshot, error)

	// --- Status Operations ---
	GetHealth(ctx context.Context) (*HealthStatus, error)
	GetRunnerStatus(ctx context.Context) ([]*RunnerStatus, error)
	ProbeRunner(ctx context.Context, runnerType domain.RunnerType) (*ProbeResult, error)

	// --- Maintenance Operations ---
	PurgeData(ctx context.Context, req PurgeRequest) (*PurgeResult, error)

	// --- Investigation Settings Operations ---
	GetInvestigationSettings(ctx context.Context) (*domain.InvestigationSettings, error)
	UpdateInvestigationSettings(ctx context.Context, settings *domain.InvestigationSettings) error
	ResetInvestigationSettings(ctx context.Context) error

	// --- Orchestration Settings Operations ---
	GetOrchestrationSettings(ctx context.Context) (*agentconfig.OrchestrationSettings, error)
	UpdateOrchestrationSettings(ctx context.Context, settings *agentconfig.OrchestrationSettings) error
	ResetOrchestrationSettings(ctx context.Context) error

	// --- Recommendation Extraction Operations ---
	ExtractRecommendations(ctx context.Context, runID uuid.UUID) (*domain.ExtractionResult, error)
	RegenerateRecommendations(ctx context.Context, runID uuid.UUID) error

	// --- Path Validation ---
	ValidatePath(ctx context.Context, path string, projectRoot string) (*sandbox.PathValidationResult, error)

	// --- Identity Token Operations ---
	VerifyIdentityToken(ctx context.Context, token string) (*IdentityVerifyResult, error)

	// --- Config Accessors ---
	GetDefaultProjectRoot() string
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
	TaskID                    *uuid.UUID
	AgentProfileID            *uuid.UUID
	Status                    *domain.RunStatus
	TagPrefix                 string // Filter runs by tag prefix (e.g., "ecosystem-" to get all ecosystem-manager runs)
	InvestigatesRunID         *uuid.UUID
	AppliesInvestigationRunID *uuid.UUID
}

// IdentityVerifyResult is the result of verifying an agent identity token.
type IdentityVerifyResult struct {
	Valid     bool             `json:"valid"`
	Claims    *identity.Claims `json:"claims,omitempty"`
	RunStatus domain.RunStatus `json:"run_status,omitempty"`
	Error     string           `json:"error,omitempty"`
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

	// Investigation lineage metadata.
	SourceRunIDs             []uuid.UUID `json:"sourceRunIds,omitempty"`
	SourceInvestigationRunID *uuid.UUID  `json:"sourceInvestigationRunId,omitempty"`

	// Inline config (optional - used if no profile, or overrides profile)
	RunnerType           *domain.RunnerType      `json:"runnerType,omitempty"`
	Model                *string                 `json:"model,omitempty"`
	ModelPreset          *domain.ModelPreset     `json:"modelPreset,omitempty"`
	MaxTurns             *int                    `json:"maxTurns,omitempty"`
	Timeout              *time.Duration          `json:"timeout,omitempty"`
	FallbackRunnerTypes  []domain.RunnerType     `json:"fallbackRunnerTypes,omitempty"`
	AllowedTools         []string                `json:"allowedTools,omitempty"`
	DeniedTools          []string                `json:"deniedTools,omitempty"`
	SkipPermissionPrompt *bool                   `json:"skipPermissionPrompt,omitempty"`
	EnableBrowser        *bool                   `json:"enableBrowser,omitempty"`
	ExtraFlags           domain.RunnerExtraFlags `json:"extraFlags,omitempty"`
	RequiresSandbox      *bool                   `json:"requiresSandbox,omitempty"`
	NetworkAccess        *domain.NetworkAccess   `json:"networkAccess,omitempty"`
	RequiresApproval     *bool                   `json:"requiresApproval,omitempty"`
	AllowedPaths         []string                `json:"allowedPaths,omitempty"`
	DeniedPaths          []string                `json:"deniedPaths,omitempty"`

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

	// Environment passes custom VROOLI_-prefixed environment variables to the
	// agent process. Merged with sandbox env vars; sandbox vars take precedence.
	Environment map[string]string `json:"environment,omitempty"`
}

// ProfileRef identifies a profile by key with optional defaults.
//
// When UpdateExisting is true, the supplied Defaults overwrite any existing
// profile row on every CreateRun call — useful for declarative callers
// (e.g. swarm-manager) that treat their code-declared profile as
// authoritative. Otherwise the existing row wins on conflict.
type ProfileRef struct {
	ProfileKey     string               `json:"profileKey"`
	Defaults       *domain.AgentProfile `json:"defaults,omitempty"`
	UpdateExisting bool                 `json:"updateExisting,omitempty"`
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

// ContinueRunRequest contains parameters for continuing an existing run conversation.
type ContinueRunRequest struct {
	RunID         uuid.UUID `json:"runId"`
	Message       string    `json:"message"`
	AttachmentIDs []string  `json:"attachmentIds,omitempty"`
}

// ResumeFromFailedRunRequest contains parameters for creating a new run that
// inherits the original task + profile of a failed/cancelled run and is
// seeded with that run's transcript and diff so the agent can complete the
// remaining work instead of starting over.
type ResumeFromFailedRunRequest struct {
	RunID         uuid.UUID `json:"runId"`
	CustomContext string    `json:"customContext,omitempty"`
	AttachmentIDs []string  `json:"attachmentIds,omitempty"`
}

// CreateInvestigationRequest contains parameters for creating an investigation run.
type CreateInvestigationRequest struct {
	RunIDs        []uuid.UUID               `json:"runIds"`
	CustomContext string                    `json:"customContext,omitempty"`
	Depth         domain.InvestigationDepth `json:"depth,omitempty"`       // Defaults to "standard"
	ProjectRoot   string                    `json:"projectRoot,omitempty"` // Root directory for investigation (explicit, no guessing)
	ScopePaths    []string                  `json:"scopePaths,omitempty"`  // Paths where agent can make changes
	AttachmentIDs []string                  `json:"attachmentIds,omitempty"`
	// Runner + preset overrides for the investigation agent. When nil, the default
	// investigation profile's values apply. Callers use this to pick a runner or
	// preset whose model chain currently works, without editing the registry.
	RunnerType  *domain.RunnerType  `json:"runnerType,omitempty"`
	ModelPreset *domain.ModelPreset `json:"modelPreset,omitempty"`
}

// CreateInvestigationApplyRequest contains parameters for creating an apply run.
type CreateInvestigationApplyRequest struct {
	InvestigationRunID uuid.UUID `json:"investigationRunId"`
	CustomContext      string    `json:"customContext,omitempty"`
	AttachmentIDs      []string  `json:"attachmentIds,omitempty"`
	// Runner + preset overrides for the apply agent; same semantics as investigation.
	RunnerType  *domain.RunnerType  `json:"runnerType,omitempty"`
	ModelPreset *domain.ModelPreset `json:"modelPreset,omitempty"`
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

// SandboxSyncRequest updates run state based on workspace-sandbox approval events.
type SandboxSyncRequest struct {
	RunID      uuid.UUID
	SandboxID  *uuid.UUID
	Status     string
	Actor      string
	Reason     string
	Applied    int
	Remaining  int
	IsPartial  bool
	CommitHash string
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
	profiles              repository.ProfileRepository
	tasks                 repository.TaskRepository
	runs                  repository.RunRepository
	checkpoints           repository.CheckpointRepository            // For resumption support
	idempotency           repository.IdempotencyRepository           // For replay safety
	investigationSettings repository.InvestigationSettingsRepository // For investigation config

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

	// Flag validation
	flagValidator runner.FlagValidator

	// Configuration
	config OrchestratorConfig

	// Storage label for health reporting (e.g., sqlite).
	storageLabel string

	// Model registry for runner model catalogs and presets.
	modelRegistry *modelregistry.Store

	// Model health map (in-memory, populated by runtime classification + startup probe).
	modelHealth *modelregistry.HealthStore

	// Recommendation extractor for investigation outputs.
	recommendationExtractor recommendation.Extractor

	// Prompt-manager client for reading investigation prompts from skills.
	promptClient promptmanager.Client

	// File storage for uploaded attachments.
	storage storage.Service

	// Orchestration settings store (file-backed, hot-reloadable).
	orchestrationSettings *agentconfig.OrchestrationSettingsStore

	// Reconciler reference for hot-reload propagation.
	reconciler *Reconciler

	// Identity signing secret for agent identity tokens.
	identitySecret []byte
}

// OrchestratorConfig holds service configuration.
type OrchestratorConfig struct {
	DefaultTimeout          time.Duration
	MaxConcurrentRuns       int
	DefaultProjectRoot      string
	RequireSandboxByDefault bool
	RunnerFallbackTypes     []domain.RunnerType
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() OrchestratorConfig {
	return OrchestratorConfig{
		DefaultTimeout:          60 * time.Minute,
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

// WithModelHealth sets the model health store used by the executor to flag runtime
// model-unavailable errors. The same store is exposed via GetModelRegistryHealth.
func WithModelHealth(store *modelregistry.HealthStore) Option {
	return func(o *Orchestrator) {
		o.modelHealth = store
	}
}

// WithInvestigationSettings sets the investigation settings repository.
func WithInvestigationSettings(repo repository.InvestigationSettingsRepository) Option {
	return func(o *Orchestrator) {
		o.investigationSettings = repo
	}
}

// WithFlagValidator sets the flag validator for runner-specific flag validation.
func WithFlagValidator(v runner.FlagValidator) Option {
	return func(o *Orchestrator) {
		o.flagValidator = v
	}
}

// WithRecommendationExtractor sets the recommendation extractor for investigation outputs.
func WithRecommendationExtractor(extractor recommendation.Extractor) Option {
	return func(o *Orchestrator) {
		o.recommendationExtractor = extractor
	}
}

// WithPromptClient sets the prompt-manager client for reading investigation prompts from skills.
func WithPromptClient(client promptmanager.Client) Option {
	return func(o *Orchestrator) {
		o.promptClient = client
	}
}

// WithAttachmentStorage sets the file storage service for resolving attachment IDs.
func WithAttachmentStorage(s storage.Service) Option {
	return func(o *Orchestrator) {
		o.storage = s
	}
}

// WithOrchestrationSettings sets the orchestration settings store.
func WithOrchestrationSettings(store *agentconfig.OrchestrationSettingsStore) Option {
	return func(o *Orchestrator) {
		o.orchestrationSettings = store
	}
}

// WithIdentitySecret sets the HMAC secret for signing agent identity tokens.
func WithIdentitySecret(secret []byte) Option {
	return func(o *Orchestrator) {
		o.identitySecret = secret
	}
}

// SetReconciler sets the reconciler reference for hot-reload propagation.
// This is called after construction because the reconciler depends on the orchestrator.
func (o *Orchestrator) SetReconciler(r *Reconciler) {
	o.reconciler = r
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

	// Use built-in defaults for known profile keys if no defaults provided
	defaults := req.Defaults
	if defaults == nil {
		defaults = getBuiltInProfileDefaults(key)
	}
	if defaults == nil {
		return nil, domain.NewValidationErrorWithHint("defaults", "field is required",
			"Provide default profile settings to create a new profile")
	}

	candidate := *defaults
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

	// Resolve relative project root to absolute (workspace-sandbox requires absolute paths).
	// Fall back to DefaultProjectRoot when the task has no project root set.
	if pr := strings.TrimSpace(task.ProjectRoot); pr == "" || !filepath.IsAbs(pr) {
		resolved := pr
		if resolved == "" {
			resolved = strings.TrimSpace(o.config.DefaultProjectRoot)
		}
		if resolved != "" && !filepath.IsAbs(resolved) {
			if abs, err := filepath.Abs(resolved); err == nil {
				resolved = abs
			}
		}
		if resolved != task.ProjectRoot {
			task.ProjectRoot = resolved
			if o.tasks != nil {
				task.UpdatedAt = time.Now()
				_ = o.tasks.Update(ctx, task)
			}
		}
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
		ID:                       uuid.New(),
		TaskID:                   task.ID,
		AgentProfileID:           profileID, // May be nil if inline config used
		Tag:                      req.Tag,   // Custom tag for identification
		SourceRunIDs:             req.SourceRunIDs,
		SourceInvestigationRunID: req.SourceInvestigationRunID,
		RunMode:                  runMode,
		Status:                   domain.RunStatusPending,
		Phase:                    domain.RunPhaseQueued,
		ProgressPercent:          0,
		IdempotencyKey:           req.IdempotencyKey,
		ApprovalState:            domain.ApprovalStateNone,
		ResolvedConfig:           resolvedConfig,
		SandboxConfig:            sandboxConfig,
		// Provenance: requested is the primary model the preset expanded to at creation.
		// Actual is blank until the executor records the model that actually ran.
		RequestedModel: resolvedConfig.Model,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	// Populate PromptPreview so WebSocket broadcasts include display text.
	// This is normally a computed field from the List query JOIN, but we need it
	// for real-time broadcasts during execution.
	if len(task.Description) > 120 {
		run.PromptPreview = task.Description[:120]
	} else {
		run.PromptPreview = task.Description
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

	// Split instructions (system prompt) from context data (user message).
	// Task description contains methodology/instructions → system prompt.
	// Context attachments contain data/evidence → user message.
	// If an override prompt is provided, it replaces the task description as system prompt.
	systemPrompt, userMessage := domain.BuildSplitPrompt(task.Description, task.ContextAttachments, req.Prompt)

	// Resolve image attachments from storage so runners receive file paths
	var imageAttachments []runner.Attachment
	if o.storage != nil {
		for _, att := range task.ContextAttachments {
			if att.Type == "image" && att.AttachmentID != "" {
				meta, err := o.storage.Get(ctx, att.AttachmentID)
				if err != nil {
					continue // skip unresolvable attachments
				}
				imageAttachments = append(imageAttachments, runner.Attachment{
					ID:          meta.ID,
					FileName:    meta.FileName,
					ContentType: meta.ContentType,
					FilePath:    o.storage.GetFilePath(meta.StoragePath),
				})
			}
		}
	}

	// Emit the initial user prompt as the first message event.
	// We emit the user message (context + task), not the system prompt,
	// since the system prompt is runner-internal instructions.
	if o.events != nil && strings.TrimSpace(userMessage) != "" {
		// Build attachment metadata for the event so the UI can render image thumbnails
		var attInfo []domain.MessageAttachmentInfo
		for _, att := range imageAttachments {
			meta, err := o.storage.Get(ctx, att.ID)
			if err == nil {
				attInfo = append(attInfo, domain.MessageAttachmentInfo{
					ID:          meta.ID,
					FileName:    meta.FileName,
					ContentType: meta.ContentType,
					URL:         o.storage.GetServingURL(meta.StoragePath),
				})
			}
		}
		var userEvent *domain.RunEvent
		if len(attInfo) > 0 {
			userEvent = domain.NewMessageEventWithAttachments(run.ID, "user", userMessage, attInfo)
		} else {
			userEvent = domain.NewMessageEvent(run.ID, "user", userMessage)
		}
		if err := o.events.Append(ctx, run.ID, userEvent); err != nil {
			// Log but don't fail
			_ = err
		}
		if o.broadcaster != nil {
			o.broadcaster.BroadcastEvent(userEvent)
		}
	}

	// Start execution asynchronously
	go o.executeRun(context.Background(), run, task, profile, userMessage, systemPrompt, existingSandboxWorkDir, imageAttachments, req.Environment)

	return o.attachRunActions(ctx, run), nil
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
			if mkErr := os.MkdirAll(absScopePath, 0o755); mkErr != nil {
				return domain.NewValidationErrorWithHint("scopePath", "scope path does not exist",
					fmt.Sprintf("create the directory: %s", absScopePath))
			}
			info, err = os.Stat(absScopePath)
			if err != nil {
				return domain.NewValidationErrorWithHint("scopePath", "unable to stat scope path",
					fmt.Sprintf("check permissions for %s", absScopePath))
			}
		}
		if err != nil {
			return domain.NewValidationErrorWithHint("scopePath", "unable to stat scope path",
				fmt.Sprintf("check permissions for %s", absScopePath))
		}
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
			ProfileKey:     req.ProfileRef.ProfileKey,
			Defaults:       req.ProfileRef.Defaults,
			UpdateExisting: req.ProfileRef.UpdateExisting,
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
	// Feature flag overrides
	if req.EnableBrowser != nil {
		cfg.Features.EnableBrowser = *req.EnableBrowser
	}
	// Extra flags overrides (replace per runner type)
	if req.ExtraFlags != nil {
		if cfg.ExtraFlags == nil {
			cfg.ExtraFlags = make(domain.RunnerExtraFlags)
		}
		for rt, flags := range req.ExtraFlags {
			cfg.ExtraFlags[rt] = append([]string(nil), flags...)
		}
	}
	if req.RequiresSandbox != nil {
		cfg.RequiresSandbox = *req.RequiresSandbox
	}
	if req.NetworkAccess != nil {
		cfg.NetworkAccess = *req.NetworkAccess
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
		chain, ok := o.modelRegistry.ResolvePreset(string(cfg.RunnerType), string(cfg.ModelPreset))
		if !ok {
			return nil, nil, domain.NewValidationError("modelPreset", "preset not mapped for runner")
		}
		// cfg.Model carries the primary (first concrete) entry. The full chain is attached
		// to the run by the caller so the executor can walk fallbacks at runtime.
		cfg.Model = chain.Primary()
	}

	// Validate extra flags against runner allowlists (delegate to seam)
	if o.flagValidator != nil {
		for rt, flags := range cfg.ExtraFlags {
			if err := o.flagValidator.ValidateFlags(rt, flags); err != nil {
				return nil, nil, err
			}
		}
	}

	return cfg, profile, nil
}

// resolveSandboxConfig produces the effective SandboxConfig for a run.
//
// Contract: the returned config is always non-nil. Callers (including
// tryAutoApproval) rely on this invariant; a nil return historically caused
// silent fall-through to NEEDS_REVIEW for empty sandboxes because there was
// no acceptance config to consult.
//
// Precedence (later overrides earlier):
//  1. Zero-valued default
//  2. profile.SandboxConfig (if present)
//  3. req.SandboxConfig (inline override, if present)
func (o *Orchestrator) resolveSandboxConfig(req CreateRunRequest, profile *domain.AgentProfile) (*domain.SandboxConfig, error) {
	cfg := &domain.SandboxConfig{}
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

	// Default lifecycle cleanup for auto-approve sandboxes.
	//
	// When autoApprove is enabled, the sandbox changes are applied to the
	// canonical repo and committed automatically — no human reviews the
	// sandbox. Without cleanup, the sandbox stays "active" indefinitely,
	// which:
	//   1. Blocks future runs that target the same scope path (mutual
	//      exclusion via reserved_paths).
	//   2. Leaks overlay mounts and disk space.
	//
	// We default deleteOn to ["terminal"] so the sandbox is cleaned up after
	// any terminal event (run_completed, run_failed, run_cancelled). This
	// matches the intent of auto-approve: the caller trusts the changes and
	// doesn't need the sandbox preserved for inspection.
	//
	// Callers who want different behavior (e.g., keep the sandbox for
	// debugging) can explicitly set lifecycle.deleteOn to override this
	// default.
	if cfg.Acceptance.AutoApprove && len(cfg.Lifecycle.DeleteOn) == 0 && len(cfg.Lifecycle.StopOn) == 0 {
		cfg.Lifecycle.DeleteOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleTerminal}
	}

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
	// Warn when auto_approve is enabled but no allow criteria are configured.
	// This is valid (empty allow = accept all non-denied files), but surprising
	// enough to warrant a log line — especially since an empty deny (from proto
	// serialization) previously caused silent universal denial.
	if cfg.Acceptance.AutoApprove &&
		len(cfg.Acceptance.Allow.PathGlobs) == 0 &&
		len(cfg.Acceptance.Allow.Extensions) == 0 {
		log.Printf("[sandbox-config] auto_approve enabled with no allow criteria — all non-denied files will be approved")
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
	return o.attachRunActions(ctx, run), nil
}

func (o *Orchestrator) ListRuns(ctx context.Context, opts RunListOptions) ([]*domain.Run, error) {
	runs, err := o.runs.List(ctx, repository.RunListFilter{
		ListFilter: repository.ListFilter{
			Limit:  opts.Limit,
			Offset: opts.Offset,
		},
		TaskID:                    opts.TaskID,
		AgentProfileID:            opts.AgentProfileID,
		Status:                    opts.Status,
		TagPrefix:                 opts.TagPrefix,
		InvestigatesRunID:         opts.InvestigatesRunID,
		AppliesInvestigationRunID: opts.AppliesInvestigationRunID,
	})
	if err != nil {
		return nil, err
	}
	return o.attachRunActionsList(ctx, runs), nil
}

func (o *Orchestrator) DeleteRun(ctx context.Context, id uuid.UUID) error {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}
	if allowed, reason := domain.CanDeleteRun(run); !allowed {
		return domain.NewStateError("Run", string(run.Status), "delete", reason)
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
			return o.attachRunActions(ctx, run), nil
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

	if allowed, reason := domain.CanStopRun(run); !allowed {
		return domain.NewStateError("Run", string(run.Status), "stop", reason)
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

// ContinueRun continues an existing run's conversation with a follow-up message.
// The message is appended to the run's event stream and the response is streamed back.
func (o *Orchestrator) ContinueRun(ctx context.Context, req ContinueRunRequest) (*domain.Run, error) {
	// Validate message
	if strings.TrimSpace(req.Message) == "" {
		return nil, domain.NewValidationError("message", "message is required")
	}

	// Get the run
	run, err := o.GetRun(ctx, req.RunID)
	if err != nil {
		return nil, err
	}

	if allowed, reason := domain.CanContinueRun(run); !allowed {
		return nil, domain.NewStateError("Run", string(run.Status), "continue", reason)
	}

	// Get the runner type from resolved config
	var runnerType domain.RunnerType
	if run.ResolvedConfig != nil {
		runnerType = run.ResolvedConfig.RunnerType
	} else if run.AgentProfileID != nil {
		if profile, err := o.GetProfile(ctx, *run.AgentProfileID); err == nil && profile != nil {
			runnerType = profile.RunnerType
		}
	}

	if runnerType == "" {
		return nil, domain.NewStateError("Run", string(run.Status), "continue",
			"cannot determine runner type for this run")
	}

	// Get the runner
	if o.runners == nil {
		return nil, domain.NewConfigMissingError("runners", "runner registry not configured", nil)
	}

	r, err := o.runners.Get(runnerType)
	if err != nil {
		return nil, err
	}

	// Check runner supports continuation
	caps := r.Capabilities()
	if !caps.SupportsContinuation {
		return nil, runner.ErrContinuationNotSupported
	}

	// Update run status to running and reset heartbeat so the reconciler
	// doesn't immediately consider this run stale based on the previous
	// run's last heartbeat (which could be hours old).
	previousStatus := run.Status
	now := time.Now()
	run.Status = domain.RunStatusRunning
	run.Phase = domain.RunPhaseExecuting
	run.ProgressPercent = domain.PhaseToProgress(domain.RunPhaseExecuting)
	run.LastHeartbeat = &now
	run.UpdatedAt = now
	if err := o.runs.Update(ctx, run); err != nil {
		return nil, err
	}

	if o.events != nil {
		statusEvent := domain.NewStatusEvent(
			run.ID,
			string(previousStatus),
			string(domain.RunStatusRunning),
			"Continuation requested",
		)
		if err := o.events.Append(ctx, run.ID, statusEvent); err == nil && o.broadcaster != nil {
			o.broadcaster.BroadcastEvent(statusEvent)
		}
	}
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(run)
	}

	// Emit user message event (with attachment metadata if present, resolved later)
	// Note: We defer emitting until after attachment resolution so we can include URLs.
	emitUserMessage := func(attachments []runner.Attachment) {
		if o.events == nil {
			return
		}
		var attInfo []domain.MessageAttachmentInfo
		if o.storage != nil {
			for _, att := range attachments {
				meta, err := o.storage.Get(ctx, att.ID)
				if err == nil {
					attInfo = append(attInfo, domain.MessageAttachmentInfo{
						ID:          meta.ID,
						FileName:    meta.FileName,
						ContentType: meta.ContentType,
						URL:         o.storage.GetServingURL(meta.StoragePath),
					})
				}
			}
		}
		var userEvent *domain.RunEvent
		if len(attInfo) > 0 {
			userEvent = domain.NewMessageEventWithAttachments(run.ID, "user", req.Message, attInfo)
		} else {
			userEvent = domain.NewMessageEvent(run.ID, "user", req.Message)
		}
		if err := o.events.Append(ctx, run.ID, userEvent); err != nil {
			_ = err
		}
		if o.broadcaster != nil {
			o.broadcaster.BroadcastEvent(userEvent)
		}
	}

	// Get the task for working directory
	task, err := o.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}

	// Determine working directory
	workDir := ""
	if run.SandboxID != nil && o.sandbox != nil {
		workDir, _ = o.sandbox.GetWorkspacePath(ctx, *run.SandboxID)
	}
	if workDir == "" && task != nil {
		if task.ProjectRoot != "" {
			workDir = task.ProjectRoot
		}
	}

	// Create event sink for the continuation
	var eventSink runner.EventSink
	if o.events != nil {
		if o.broadcaster != nil {
			eventSink = &broadcastingEventSink{
				store:       o.events,
				runID:       run.ID,
				broadcaster: o.broadcaster,
			}
		} else {
			eventSink = &eventStoreAdapter{
				store: o.events,
				runID: run.ID,
			}
		}
	} else {
		eventSink = &noOpEventSink{}
	}

	// Resolve attachments
	var attachments []runner.Attachment
	if len(req.AttachmentIDs) > 0 && o.storage != nil {
		metas, err := o.storage.GetMultiple(ctx, req.AttachmentIDs)
		if err != nil {
			// Log but continue without attachments
			_ = err
		}
		for _, meta := range metas {
			attachments = append(attachments, runner.Attachment{
				ID:          meta.ID,
				FileName:    meta.FileName,
				ContentType: meta.ContentType,
				FilePath:    o.storage.GetFilePath(meta.StoragePath),
			})
		}
	}

	// Emit user message event now that attachments are resolved
	emitUserMessage(attachments)

	// Execute continuation asynchronously
	go o.executeContinuation(context.Background(), run, r, eventSink, req.Message, workDir, attachments)

	return o.attachRunActions(ctx, run), nil
}

// DeleteRunMessage appends a message_deleted event for a message event.
// The original message remains in the append-only stream for auditability.
func (o *Orchestrator) DeleteRunMessage(ctx context.Context, runID uuid.UUID, eventID uuid.UUID) (*domain.RunEvent, error) {
	if o.events == nil {
		return nil, domain.NewConfigMissingError("eventStore", "not configured", nil)
	}

	events, err := o.events.Get(ctx, runID, event.GetOptions{
		EventTypes: []domain.RunEventType{domain.EventTypeMessage, domain.EventTypeMessageDeleted},
	})
	if err != nil {
		return nil, err
	}

	var target *domain.RunEvent
	alreadyDeleted := false
	targetID := eventID.String()
	for _, evt := range events {
		if evt == nil {
			continue
		}
		if evt.ID == eventID {
			target = evt
			continue
		}
		if data, ok := evt.Data.(*domain.MessageDeletedEventData); ok && data.TargetEventID == targetID {
			alreadyDeleted = true
		}
	}

	if target == nil {
		return nil, domain.NewNotFoundErrorWithID("RunEvent", targetID)
	}
	if target.EventType != domain.EventTypeMessage {
		return nil, domain.NewValidationError("eventId", "only message events can be deleted")
	}
	if alreadyDeleted {
		return nil, domain.NewStateError("RunEvent", "deleted", "delete", "message already deleted")
	}

	deleteEvent := domain.NewMessageDeletedEvent(runID, targetID)
	if err := o.events.Append(ctx, runID, deleteEvent); err != nil {
		return nil, err
	}
	if o.broadcaster != nil {
		o.broadcaster.BroadcastEvent(deleteEvent)
	}
	return deleteEvent, nil
}

// continuationHeartbeat sends periodic heartbeats for a continuation so the
// reconciler doesn't consider the run stale while it's actively executing.
func (o *Orchestrator) continuationHeartbeat(ctx context.Context, run *domain.Run, stop <-chan struct{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			run.LastHeartbeat = &now
			run.UpdatedAt = now
			if err := o.runs.Update(ctx, run); err != nil {
				log.Printf("[heartbeat] ERROR: continuation heartbeat failed for run %s: %v", run.ID, err)
			}
		}
	}
}

// executeContinuation handles the actual continuation execution (runs in background).
// Each continuation turn gets its own timeout from RunTimeoutMinutes, so a timed-out
// run can be continued indefinitely — each "continue" message resets the clock.
func (o *Orchestrator) executeContinuation(ctx context.Context, run *domain.Run, r runner.Runner, eventSink runner.EventSink, message string, workDir string, attachments []runner.Attachment) {
	// Apply per-turn timeout to continuation
	timeoutMinutes := 30 // default
	if o.orchestrationSettings != nil {
		s := o.orchestrationSettings.Get()
		if s.RunExecution.RunTimeoutMinutes > 0 {
			timeoutMinutes = s.RunExecution.RunTimeoutMinutes
		}
	}
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMinutes)*time.Minute)
	defer cancel()

	// Start heartbeat loop so the reconciler doesn't kill us during execution.
	// Heartbeat uses the parent ctx so it survives after execCtx deadline fires.
	heartbeatStop := make(chan struct{})
	go o.continuationHeartbeat(ctx, run, heartbeatStop)
	defer close(heartbeatStop)

	// Build continue request
	continueReq := runner.ContinueRequest{
		RunID:       run.ID,
		SessionID:   run.SessionID,
		Prompt:      message,
		WorkingDir:  workDir,
		EventSink:   eventSink,
		Attachments: attachments,
	}

	// Execute continuation with per-turn timeout
	result, err := r.Continue(execCtx, continueReq)

	// Update run based on result
	now := time.Now()
	previousStatus := run.Status
	run.UpdatedAt = now

	if execCtx.Err() == context.DeadlineExceeded {
		// Continuation timed out — mark as failed but preserve session ID
		// so the user can continue again with a fresh timeout.
		run.Status = domain.RunStatusFailed
		run.EndedAt = &now
		run.ErrorMsg = fmt.Sprintf("continuation exceeded timeout of %d minutes", timeoutMinutes)
		if result != nil && result.SessionID != "" {
			run.SessionID = result.SessionID
		}
		if o.events != nil {
			errorEvent := domain.NewErrorEvent(run.ID, "continuation_timeout", run.ErrorMsg, true)
			_ = o.events.Append(ctx, run.ID, errorEvent)
			if o.broadcaster != nil {
				o.broadcaster.BroadcastEvent(errorEvent)
			}
		}
	} else if err != nil {
		run.Status = domain.RunStatusFailed
		run.EndedAt = &now
		run.ErrorMsg = err.Error()
		if o.events != nil {
			errorEvent := domain.NewErrorEvent(run.ID, "continuation_error", err.Error(), false)
			_ = o.events.Append(ctx, run.ID, errorEvent)
			if o.broadcaster != nil {
				o.broadcaster.BroadcastEvent(errorEvent)
			}
		}
	} else if result != nil && !result.Success {
		run.Status = domain.RunStatusFailed
		run.EndedAt = &now
		run.ErrorMsg = result.ErrorMessage
		run.ExitCode = &result.ExitCode
		if o.events != nil && result.ErrorMessage != "" {
			errorEvent := domain.NewErrorEvent(run.ID, "continuation_error", result.ErrorMessage, false)
			_ = o.events.Append(ctx, run.ID, errorEvent)
			if o.broadcaster != nil {
				o.broadcaster.BroadcastEvent(errorEvent)
			}
		}
	} else if result != nil {
		run.Status = domain.RunStatusComplete
		run.EndedAt = &now

		// Update summary if available
		if result.Summary != nil {
			run.Summary = result.Summary
		}
	} else {
		run.Status = domain.RunStatusComplete
		run.EndedAt = &now
	}

	// Always preserve session ID from the result for further continuation,
	// regardless of success/failure. The runner populates SessionID from
	// stream events received before process termination.
	if result != nil && result.SessionID != "" {
		run.SessionID = result.SessionID
	}

	// Persist updated run
	if err := o.runs.Update(ctx, run); err != nil {
		// Log but can't do much else
		_ = err
	}

	if o.events != nil && previousStatus != run.Status {
		statusEvent := domain.NewStatusEvent(
			run.ID,
			string(previousStatus),
			string(run.Status),
			"Continuation completed",
		)
		if err := o.events.Append(ctx, run.ID, statusEvent); err == nil && o.broadcaster != nil {
			o.broadcaster.BroadcastEvent(statusEvent)
		}
	}

	// Broadcast status update
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(run)
	}
}

// executeRun handles the actual agent execution (runs in background).
// This delegates to RunExecutor for the actual work.
func (o *Orchestrator) executeRun(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, prompt string, systemPrompt string, existingSandboxWorkDir string, attachments []runner.Attachment, customEnv map[string]string) {
	executor := NewRunExecutor(
		o.runs,
		o.runners,
		o.sandbox,
		o.events,
		run,
		task,
		profile,
		prompt,
		systemPrompt,
	)
	// Apply orchestration settings to executor config if store is available.
	if o.orchestrationSettings != nil {
		s := o.orchestrationSettings.Get()
		executor.WithConfig(ExecutorConfig{
			Timeout:            time.Duration(s.RunExecution.RunTimeoutMinutes) * time.Minute,
			HeartbeatInterval:  time.Duration(s.HealthDetection.HeartbeatIntervalSeconds) * time.Second,
			CheckpointInterval: 1 * time.Minute,
			MaxRetries:         3,
			StaleThreshold:     time.Duration(s.HealthDetection.StaleThresholdSeconds) * time.Second,
		})
	}
	// Configure executor with checkpoint repository if available
	if o.checkpoints != nil {
		executor.WithCheckpointRepository(o.checkpoints)
	}
	// Model-level fallback: hand the executor a resolver so it can walk the preset chain
	// at execution time when the runner rejects a model.
	if o.modelRegistry != nil {
		executor.WithModelChainResolver(o.modelRegistry)
	}
	if o.modelHealth != nil {
		executor.WithModelHealthReporter(newHealthMarkerAdapter(o.modelHealth))
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
	if len(attachments) > 0 {
		executor.WithAttachments(attachments)
	}
	if len(customEnv) > 0 {
		executor.WithCustomEnvironment(customEnv)
	}
	if len(o.identitySecret) > 0 {
		executor.WithIdentitySecret(o.identitySecret)
	}
	executor.WithRecommendationQueueFilter(o.recommendationQueueFilter(ctx))
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

	return o.attachRunActions(ctx, run), nil
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
		"", // No system prompt for resume (session persists instructions)
	)
	// Apply orchestration settings to executor config if store is available.
	if o.orchestrationSettings != nil {
		s := o.orchestrationSettings.Get()
		executor.WithConfig(ExecutorConfig{
			Timeout:            time.Duration(s.RunExecution.RunTimeoutMinutes) * time.Minute,
			HeartbeatInterval:  time.Duration(s.HealthDetection.HeartbeatIntervalSeconds) * time.Second,
			CheckpointInterval: 1 * time.Minute,
			MaxRetries:         3,
			StaleThreshold:     time.Duration(s.HealthDetection.StaleThresholdSeconds) * time.Second,
		})
	}

	// Configure for resumption
	if o.checkpoints != nil {
		executor.WithCheckpointRepository(o.checkpoints)
	}
	// Configure executor with broadcaster for real-time WebSocket updates
	if o.broadcaster != nil {
		executor.WithBroadcaster(o.broadcaster)
	}
	if len(o.identitySecret) > 0 {
		executor.WithIdentitySecret(o.identitySecret)
	}
	executor.WithRecommendationQueueFilter(o.recommendationQueueFilter(ctx))
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

func (o *Orchestrator) GetModelRegistryHealth(ctx context.Context) (modelregistry.HealthSnapshot, error) {
	if o.modelHealth == nil {
		// No health store wired: return an empty snapshot rather than erroring so
		// consumers can render the UI before probes have run.
		return modelregistry.HealthSnapshot{Runners: map[string]map[string]modelregistry.ModelHealth{}}, nil
	}
	return o.modelHealth.Snapshot(), nil
}

// -----------------------------------------------------------------------------
// Config Accessors
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetDefaultProjectRoot() string {
	return o.config.DefaultProjectRoot
}

// ValidatePath delegates path validation to the sandbox provider.
func (o *Orchestrator) ValidatePath(ctx context.Context, path string, projectRoot string) (*sandbox.PathValidationResult, error) {
	if o.sandbox == nil {
		return nil, domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}
	return o.sandbox.ValidatePath(ctx, path, projectRoot)
}

// VerifyIdentityToken validates a signed agent identity token and returns the
// embedded claims along with the current run status.
func (o *Orchestrator) VerifyIdentityToken(ctx context.Context, token string) (*IdentityVerifyResult, error) {
	if len(o.identitySecret) == 0 {
		return &IdentityVerifyResult{Valid: false, Error: "identity system not configured"}, nil
	}

	claims, err := identity.VerifyToken(token, o.identitySecret)
	if err != nil {
		return &IdentityVerifyResult{Valid: false, Error: err.Error()}, nil
	}

	// Look up the run by token hash to get current status.
	tokenHash := identity.HashToken(token)
	run, err := o.runs.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}

	runStatus := domain.RunStatus("unknown")
	if run != nil {
		runStatus = run.Status
	}

	return &IdentityVerifyResult{
		Valid:     true,
		Claims:    claims,
		RunStatus: runStatus,
	}, nil
}

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
			log.Printf("[broadcast-sink] failed to store event for run %s: %v", b.runID, err)
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

// -----------------------------------------------------------------------------
// Investigation Settings Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetInvestigationSettings(ctx context.Context) (*domain.InvestigationSettings, error) {
	var settings *domain.InvestigationSettings
	if o.investigationSettings == nil {
		settings = domain.DefaultInvestigationSettings()
	} else {
		var err error
		settings, err = o.investigationSettings.Get(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Overlay prompt templates from prompt-manager skills (overrides DB values)
	if o.promptClient != nil {
		if prompt, err := o.promptClient.ReadSkill(ctx, "agent-manager-process-investigation", nil, false); err == nil {
			settings.PromptTemplate = prompt
		}
		if applyPrompt, err := o.promptClient.ReadSkill(ctx, "agent-manager-process-investigation-apply", nil, false); err == nil {
			settings.ApplyPromptTemplate = applyPrompt
		}
	}

	return settings, nil
}

func (o *Orchestrator) UpdateInvestigationSettings(ctx context.Context, settings *domain.InvestigationSettings) error {
	if o.investigationSettings == nil {
		return domain.NewConfigMissingError("investigationSettings", "repository not configured", nil)
	}

	// Validate operational settings
	if !settings.DefaultDepth.IsValid() {
		return domain.NewValidationError("defaultDepth", "invalid depth value")
	}

	// Write prompt templates to prompt-manager skills
	if o.promptClient != nil {
		if adminClient, ok := o.promptClient.(promptmanager.AdminClient); ok {
			if settings.PromptTemplate != "" {
				content := settings.PromptTemplate
				if _, err := adminClient.UpdateSkill(ctx, "agent-manager-process-investigation",
					promptmanager.PromptSkillUpdate{Content: &content}); err != nil {
					return fmt.Errorf("update investigation skill: %w", err)
				}
			}
			if settings.ApplyPromptTemplate != "" {
				content := settings.ApplyPromptTemplate
				if _, err := adminClient.UpdateSkill(ctx, "agent-manager-process-investigation-apply",
					promptmanager.PromptSkillUpdate{Content: &content}); err != nil {
					return fmt.Errorf("update apply investigation skill: %w", err)
				}
			}
		}
	}

	// Operational config still saved to local DB
	return o.investigationSettings.Update(ctx, settings)
}

func (o *Orchestrator) ResetInvestigationSettings(ctx context.Context) error {
	if o.investigationSettings == nil {
		return domain.NewConfigMissingError("investigationSettings", "repository not configured", nil)
	}

	// Revert prompt-manager skills to original version
	if o.promptClient != nil {
		if adminClient, ok := o.promptClient.(promptmanager.AdminClient); ok {
			_ = adminClient.RevertSkillVersion(ctx, "agent-manager-process-investigation", 1)
			_ = adminClient.RevertSkillVersion(ctx, "agent-manager-process-investigation-apply", 1)
		}
	}

	// Reset operational config in DB
	return o.investigationSettings.Reset(ctx)
}

// -----------------------------------------------------------------------------
// Orchestration Settings Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetOrchestrationSettings(_ context.Context) (*agentconfig.OrchestrationSettings, error) {
	if o.orchestrationSettings == nil {
		defaults := agentconfig.DefaultOrchestrationSettings()
		return &defaults, nil
	}
	settings := o.orchestrationSettings.Get()
	return &settings, nil
}

func (o *Orchestrator) UpdateOrchestrationSettings(_ context.Context, settings *agentconfig.OrchestrationSettings) error {
	if o.orchestrationSettings == nil {
		return domain.NewConfigMissingError("orchestrationSettings", "store not configured", nil)
	}
	if err := o.orchestrationSettings.Update(*settings); err != nil {
		return err
	}
	o.propagateOrchestrationSettings(settings)
	return nil
}

func (o *Orchestrator) ResetOrchestrationSettings(_ context.Context) error {
	if o.orchestrationSettings == nil {
		return domain.NewConfigMissingError("orchestrationSettings", "store not configured", nil)
	}
	if err := o.orchestrationSettings.Reset(); err != nil {
		return err
	}
	defaults := agentconfig.DefaultOrchestrationSettings()
	o.propagateOrchestrationSettings(&defaults)
	return nil
}

// propagateOrchestrationSettings applies updated settings to running components.
func (o *Orchestrator) propagateOrchestrationSettings(s *agentconfig.OrchestrationSettings) {
	// Update orchestrator config (affects new runs).
	o.config.DefaultTimeout = time.Duration(s.RunExecution.RunTimeoutMinutes) * time.Minute
	o.config.MaxConcurrentRuns = s.RunExecution.MaxConcurrentRuns
	o.config.RequireSandboxByDefault = s.SafetyIsolation.RequireSandbox

	// Propagate to reconciler.
	if o.reconciler != nil {
		o.reconciler.UpdateConfig(ReconcilerConfig{
			Interval:          time.Duration(s.HealthDetection.ReconcilerIntervalSeconds) * time.Second,
			StaleThreshold:    time.Duration(s.HealthDetection.StaleThresholdSeconds) * time.Second,
			MaxRecoveryAge:    time.Duration(s.HealthDetection.MaxRecoveryAgeSeconds) * time.Second,
			OrphanGracePeriod: time.Duration(s.ProcessTermination.OrphanGracePeriodSeconds) * time.Second,
			MaxStaleRuns:      10,
			KillOrphans:       s.ProcessTermination.KillOrphans,
			AutoRecover:       true,
		})
	}

	// Propagate to terminator.
	if o.terminator != nil {
		o.terminator.UpdateConfig(TerminatorConfig{
			GracePeriod:      time.Duration(s.ProcessTermination.GracePeriodSeconds) * time.Second,
			MaxRetries:       s.ProcessTermination.TerminationMaxRetries,
			BaseBackoff:      500 * time.Millisecond,
			MaxBackoff:       5 * time.Second,
			VerifyTimeout:    2 * time.Second,
			KillProcessGroup: s.ProcessTermination.KillProcessGroup,
		})
	}
}
