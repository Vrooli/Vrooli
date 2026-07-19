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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/health"
	"agent-manager/internal/identity"
	"agent-manager/internal/metrics"
	"agent-manager/internal/orchestration/interactive"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/policy"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/runstate"
	"agent-manager/internal/storage"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/workflowruntime"

	agentconfig "agent-manager/internal/config"

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
	ReconcileScenarioProfiles(ctx context.Context, req ReconcileScenarioProfilesRequest) (*ReconcileScenarioProfilesResult, error)
	ReconcileScenarioDeclarations(ctx context.Context, req ReconcileScenarioDeclarationsRequest) (*ReconcileScenarioDeclarationsResult, error)
	ReconcileDeclaringScenarios(ctx context.Context, repoRoot string) SweepSummary
	ReconcileSelfDeclarations(ctx context.Context, repoRoot string) (*ReconcileScenarioDeclarationsResult, error)
	ValidateWorkflow(ctx context.Context, data []byte) (*WorkflowValidationResult, error)
	ReconcileScenarioWorkflows(ctx context.Context, req ReconcileScenarioWorkflowsRequest) (*ReconcileScenarioWorkflowsResult, error)
	ListWorkflowRevisions(ctx context.Context, owner, key string, opts ListOptions) ([]*domain.WorkflowRevision, error)
	GetWorkflowRevision(ctx context.Context, owner, key, digest string) (*domain.WorkflowRevision, error)
	StartWorkflowExecution(ctx context.Context, req StartWorkflowExecutionRequest) (*domain.WorkflowExecution, error)
	ListWorkflowExecutions(ctx context.Context, req ListWorkflowExecutionsRequest) ([]*domain.WorkflowExecution, error)
	GetWorkflowExecution(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error)
	AdvanceWorkflowExecution(ctx context.Context, id uuid.UUID) (*domain.WorkflowExecution, error)
	WaitWorkflowExecution(ctx context.Context, id uuid.UUID, timeout time.Duration) (*WaitWorkflowExecutionResult, error)
	GetWorkflowExecutionTrace(ctx context.Context, id uuid.UUID, afterSequence int64, limit int) (*WorkflowExecutionTrace, error)
	ListWorkflowExecutionRuns(ctx context.Context, id uuid.UUID) ([]*domain.WorkflowNodeAttempt, error)
	SignalWorkflowExecution(ctx context.Context, req WorkflowExecutionSignalRequest) (*WorkflowExecutionOperationResult, error)
	CancelWorkflowExecution(ctx context.Context, req WorkflowExecutionOperationRequest) (*WorkflowExecutionOperationResult, error)
	RetryWorkflowExecution(ctx context.Context, req WorkflowExecutionOperationRequest) (*WorkflowExecutionOperationResult, error)
	ResumeWorkflowExecution(ctx context.Context, req WorkflowExecutionOperationRequest) (*WorkflowExecutionOperationResult, error)
	RecoverWorkflowExecutions(ctx context.Context) error
	SimulateWorkflow(ctx context.Context, req SimulateWorkflowRequest) (*WorkflowSimulation, error)

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
	QuiesceScenario(ctx context.Context, opts QuiesceOptions) (*QuiesceResult, error)
	ContinueRun(ctx context.Context, req ContinueRunRequest) (*domain.Run, error)
	ParkRunFromAgent(ctx context.Context, req ParkRunFromAgentRequest) (*ParkRunResult, error)
	GetAwaitResult(ctx context.Context, runID uuid.UUID) (*AwaitResult, error)
	WakeRun(ctx context.Context, in WakeRunInput) (*domain.Run, error)
	RecoverRun(ctx context.Context, id uuid.UUID) (*RecoverResult, error)
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

	// --- Model Policy Operations ---
	GetModelHealthSnapshot(ctx context.Context) (health.Snapshot, error)
	ExplainProfilePolicy(ctx context.Context, profileID uuid.UUID) (*domain.ExecutionPolicySnapshot, error)
	ExplainRunPolicy(ctx context.Context, runID uuid.UUID) (*domain.ExecutionPolicySnapshot, error)

	// --- Status Operations ---
	GetHealth(ctx context.Context) (*HealthStatus, error)
	GetRunnerStatus(ctx context.Context) ([]*RunnerStatus, error)
	ProbeRunner(ctx context.Context, runnerType domain.RunnerType) (*ProbeResult, error)
	SpawnStats() spawn.Stats

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
	TagPrefix                 string // Filter runs by tag prefix (e.g., "ecosystem-" to get all swarm-manager runs)
	ScopePrefix               string // Filter runs by the joined task's scope_path prefix (e.g., "scenarios/agent-manager")
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
	RoleRef              *string                 `json:"roleRef,omitempty"`
	MaxTurns             *int                    `json:"maxTurns,omitempty"`
	Timeout              *time.Duration          `json:"timeout,omitempty"`
	AllowedTools         []string                `json:"allowedTools,omitempty"`
	DeniedTools          []string                `json:"deniedTools,omitempty"`
	SkipPermissionPrompt *bool                   `json:"skipPermissionPrompt,omitempty"`
	EnableBrowser        *bool                   `json:"enableBrowser,omitempty"`
	ExtraFlags           domain.RunnerExtraFlags `json:"extraFlags,omitempty"`
	NetworkAccess        *domain.NetworkAccess   `json:"networkAccess,omitempty"`
	AllowedPaths         []string                `json:"allowedPaths,omitempty"`
	DeniedPaths          []string                `json:"deniedPaths,omitempty"`
	ResultSpec           *domain.ResultSpec      `json:"resultSpec,omitempty"`

	// Sandbox behavior overrides (optional)
	SandboxConfig *domain.SandboxConfig `json:"sandboxConfig,omitempty"`

	// ExistingSandboxID reuses a pre-existing sandbox for the run.
	// Only supported in sandboxed mode.
	ExistingSandboxID *uuid.UUID `json:"existingSandboxId,omitempty"`

	// Execution options
	Prompt       string          `json:"prompt,omitempty"` // Optional override prompt
	RunMode      *domain.RunMode `json:"runMode,omitempty"`
	ForceInPlace bool            `json:"forceInPlace,omitempty"`

	// ExecutionMode selects the CLI-driving substrate (codec-pipe vs
	// interactive web-console session). Empty defaults to codec-pipe. This is
	// the internal domain-level seam; the public proto/API surface that lets a
	// caller request interactive mode is added in Phase 6, which sets this field.
	// Interactive mode is gated to non-protected (in-place) runs at creation —
	// see domain.ValidateInteractiveRunMode.
	ExecutionMode domain.ExecutionMode `json:"executionMode,omitempty"`

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

	// ConversationID and ParentRunID encode the agent-thread linkage per
	// Decision D7 of the auditability contract. Spawn surfaces SHOULD
	// populate at least one explicitly so provenance readers can group by
	// conversation. agent-manager applies the locked precedence at
	// run-creation time (spawner > parent inheritance > fresh UUID) — see
	// domain.ResolveConversationID.
	ConversationID string     `json:"conversationId,omitempty"`
	ParentRunID    *uuid.UUID `json:"parentRunId,omitempty"`
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

// ReconcileScenarioProfilesRequest resolves all profile sources declared by a scenario.
type ReconcileScenarioProfilesRequest struct {
	Scenario string `json:"scenario"`
	DryRun   bool   `json:"dryRun,omitempty"`
}

// ProfileReconcileStatus classifies one source profile reconciliation result.
type ProfileReconcileStatus string

const (
	ProfileReconcileStatusCreated                 ProfileReconcileStatus = "created"
	ProfileReconcileStatusUpdated                 ProfileReconcileStatus = "updated"
	ProfileReconcileStatusUnchanged               ProfileReconcileStatus = "unchanged"
	ProfileReconcileStatusSkipped                 ProfileReconcileStatus = "skipped"
	ProfileReconcileStatusConflictedLocalOverride ProfileReconcileStatus = "conflicted_local_override"
	ProfileReconcileStatusFailedValidation        ProfileReconcileStatus = "failed_validation"
)

// ProfileReconcileResult reports one profile source outcome.
type ProfileReconcileResult struct {
	ProfileKey string                 `json:"profileKey,omitempty"`
	SourcePath string                 `json:"sourcePath,omitempty"`
	SourceHash string                 `json:"sourceHash,omitempty"`
	ProfileID  string                 `json:"profileId,omitempty"`
	Status     ProfileReconcileStatus `json:"status"`
	Message    string                 `json:"message,omitempty"`
}

// ReconcileScenarioProfilesResult captures a scenario-level reconciliation report.
type ReconcileScenarioProfilesResult struct {
	Scenario   string                   `json:"scenario"`
	Results    []ProfileReconcileResult `json:"results"`
	Created    int                      `json:"created"`
	Updated    int                      `json:"updated"`
	Unchanged  int                      `json:"unchanged"`
	Skipped    int                      `json:"skipped"`
	Conflicted int                      `json:"conflicted"`
	Failed     int                      `json:"failed"`
	DryRun     bool                     `json:"dryRun"`
}

type WorkflowValidationResult struct {
	Valid       bool                        `json:"valid"`
	Digest      string                      `json:"digest,omitempty"`
	Definition  *domain.WorkflowDefinition  `json:"definition,omitempty"`
	Diagnostics []domain.WorkflowDiagnostic `json:"diagnostics,omitempty"`
}

type ReconcileScenarioWorkflowsRequest struct {
	Scenario     string `json:"scenario"`
	DryRun       bool   `json:"dryRun,omitempty"`
	ValidateOnly bool   `json:"validateOnly,omitempty"`
}

type WorkflowReconcileStatus string

const (
	WorkflowReconcileCreated          WorkflowReconcileStatus = "created"
	WorkflowReconcileActivated        WorkflowReconcileStatus = "activated"
	WorkflowReconcileUnchanged        WorkflowReconcileStatus = "unchanged"
	WorkflowReconcileSkipped          WorkflowReconcileStatus = "skipped"
	WorkflowReconcileFailedValidation WorkflowReconcileStatus = "failed_validation"
)

type WorkflowReconcileResult struct {
	WorkflowKey string                      `json:"workflowKey,omitempty"`
	Version     string                      `json:"version,omitempty"`
	Digest      string                      `json:"digest,omitempty"`
	SourcePath  string                      `json:"sourcePath,omitempty"`
	Status      WorkflowReconcileStatus     `json:"status"`
	Message     string                      `json:"message,omitempty"`
	Diagnostics []domain.WorkflowDiagnostic `json:"diagnostics,omitempty"`
}

type ReconcileScenarioWorkflowsResult struct {
	Scenario     string                    `json:"scenario"`
	Results      []WorkflowReconcileResult `json:"results"`
	Created      int                       `json:"created"`
	Activated    int                       `json:"activated"`
	Unchanged    int                       `json:"unchanged"`
	Skipped      int                       `json:"skipped"`
	Failed       int                       `json:"failed"`
	DryRun       bool                      `json:"dryRun"`
	ValidateOnly bool                      `json:"validateOnly"`
}

// ReconcileScenarioDeclarationsRequest reconciles a scenario's unified
// declaration block (profiles and workflows) in one call.
type ReconcileScenarioDeclarationsRequest struct {
	Scenario     string `json:"scenario"`
	DryRun       bool   `json:"dryRun,omitempty"`
	ValidateOnly bool   `json:"validateOnly,omitempty"`
}

// ReconcileScenarioDeclarationsResult aggregates the per-kind reconciliation
// outcomes; the legacy profile/workflow RPCs project their halves from it.
type ReconcileScenarioDeclarationsResult struct {
	Scenario        string                    `json:"scenario"`
	ProfileResults  []ProfileReconcileResult  `json:"profileResults"`
	WorkflowResults []WorkflowReconcileResult `json:"workflowResults"`

	ProfilesCreated    int `json:"profilesCreated"`
	ProfilesUpdated    int `json:"profilesUpdated"`
	ProfilesUnchanged  int `json:"profilesUnchanged"`
	ProfilesSkipped    int `json:"profilesSkipped"`
	ProfilesConflicted int `json:"profilesConflicted"`
	ProfilesFailed     int `json:"profilesFailed"`

	WorkflowsCreated   int `json:"workflowsCreated"`
	WorkflowsActivated int `json:"workflowsActivated"`
	WorkflowsUnchanged int `json:"workflowsUnchanged"`
	WorkflowsSkipped   int `json:"workflowsSkipped"`
	WorkflowsFailed    int `json:"workflowsFailed"`

	DryRun       bool `json:"dryRun"`
	ValidateOnly bool `json:"validateOnly"`
}

// SweepSummary is the aggregate of a startup declaration sweep across every
// scenario that declares the unified block.
type SweepSummary struct {
	Scanned    int                   `json:"scanned"`
	Declaring  int                   `json:"declaring"`
	Reconciled int                   `json:"reconciled"`
	Failed     int                   `json:"failed"`
	Scenarios  []ScenarioSweepResult `json:"scenarios,omitempty"`
}

// ScenarioSweepResult is one scenario's outcome within a startup sweep.
type ScenarioSweepResult struct {
	Scenario           string `json:"scenario"`
	ProfilesCreated    int    `json:"profilesCreated"`
	ProfilesUpdated    int    `json:"profilesUpdated"`
	WorkflowsCreated   int    `json:"workflowsCreated"`
	WorkflowsActivated int    `json:"workflowsActivated"`
	Failed             int    `json:"failed"`
	Err                string `json:"err,omitempty"`
}

type StartWorkflowExecutionRequest struct {
	Owner            string          `json:"owner"`
	WorkflowKey      string          `json:"workflowKey"`
	DefinitionDigest string          `json:"definitionDigest,omitempty"`
	Input            json.RawMessage `json:"input"`
	IdempotencyKey   string          `json:"idempotencyKey"`
}

type ListWorkflowExecutionsRequest struct {
	Owner       string
	WorkflowKey string
	Status      domain.WorkflowExecutionStatus
	Limit       int
	Offset      int
}

type WorkflowExecutionTrace struct {
	Execution *domain.WorkflowExecution
	Attempts  []*domain.WorkflowNodeAttempt
	Journal   []*domain.WorkflowJournalEntry
}

type WorkflowExecutionSignalRequest struct {
	ExecutionID     uuid.UUID
	Signal          string
	Payload         json.RawMessage
	IdempotencyKey  string
	ExpectedVersion int64
}

type WorkflowExecutionOperationRequest struct {
	ExecutionID     uuid.UUID
	IdempotencyKey  string
	ExpectedVersion int64
	Reason          string
}

type WorkflowExecutionOperationResult struct {
	Execution  *domain.WorkflowExecution
	Idempotent bool
}

type (
	SimulateWorkflowRequest struct {
		Owner, WorkflowKey, DefinitionDigest string
		Input                                json.RawMessage
	}
	WorkflowNodePlan struct {
		NodeID               string                  `json:"nodeId"`
		Kind                 domain.WorkflowNodeKind `json:"kind"`
		ExecutionStrategy    string                  `json:"executionStrategy,omitempty"`
		ProfileKey           string                  `json:"profileKey,omitempty"`
		RoleRef              string                  `json:"roleRef,omitempty"`
		ContinuationSource   string                  `json:"continuationSource,omitempty"`
		ChildWorkflowKey     string                  `json:"childWorkflowKey,omitempty"`
		ChildWorkflowVersion string                  `json:"childWorkflowVersion,omitempty"`
		WaitSignal           string                  `json:"waitSignal,omitempty"`
		WaitTimeoutSeconds   int                     `json:"waitTimeoutSeconds,omitempty"`
		JoinStrategy         string                  `json:"joinStrategy,omitempty"`
		JoinQuorum           int                     `json:"joinQuorum,omitempty"`
		Parallel             bool                    `json:"parallel,omitempty"`
	}
	WorkflowSimulation struct {
		Valid                 bool                        `json:"valid"`
		DefinitionDigest      string                      `json:"definitionDigest"`
		Nodes                 []WorkflowNodePlan          `json:"nodes"`
		PossibleTerminalNodes []string                    `json:"possibleTerminalNodes"`
		Diagnostics           []domain.WorkflowDiagnostic `json:"diagnostics,omitempty"`
	}
)

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
	RunID          uuid.UUID          `json:"runId"`
	Message        string             `json:"message"`
	AttachmentIDs  []string           `json:"attachmentIds,omitempty"`
	IdempotencyKey string             `json:"idempotencyKey,omitempty"`
	MaxTurns       *int               `json:"maxTurns,omitempty"`
	Timeout        *time.Duration     `json:"timeout,omitempty"`
	ResultSpec     *domain.ResultSpec `json:"resultSpec,omitempty"`
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
	// RoleRef overrides the portable role on the default investigation profile.
	// Resource-owned role resolution selects the concrete runner/model snapshot.
	RoleRef *string `json:"roleRef,omitempty"`
	// Environment carries custom VROOLI_-prefixed variables (e.g.
	// VROOLI_SHADOW_SCENARIOS) into the investigation runner process — same
	// contract as CreateRunRequest.Environment. Without this, an investigation
	// triggered for a run under a shadow engagement would route its lifecycle
	// ops to the live variant.
	Environment map[string]string `json:"environment,omitempty"`
}

// CreateInvestigationApplyRequest contains parameters for creating an apply run.
type CreateInvestigationApplyRequest struct {
	InvestigationRunID uuid.UUID `json:"investigationRunId"`
	// Decision is the operator's approval decision on the investigation:
	// "completed" (approve and apply), "rejected", or "abstained". Empty
	// defaults to "completed" for backward compatibility with the apply action.
	Decision string `json:"decision,omitempty"`
	// Selected is the list of approved recommendation texts the operator chose to
	// apply. Empty means apply every recommendation in the approved findings.
	Selected      []string `json:"selected,omitempty"`
	CustomContext string   `json:"customContext,omitempty"`
	AttachmentIDs []string `json:"attachmentIds,omitempty"`
	// RoleRef overrides the portable role on the default apply profile.
	RoleRef *string `json:"roleRef,omitempty"`
	// Environment carries custom VROOLI_-prefixed variables into the apply runner
	// process; same contract as CreateInvestigationRequest.Environment.
	Environment map[string]string `json:"environment,omitempty"`
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
	Database        *DependencyStatus            `json:"database,omitempty"`
	WorkflowRuntime *DependencyStatus            `json:"workflow_runtime,omitempty"`
	Sandbox         *DependencyStatus            `json:"sandbox,omitempty"`
	Runners         map[string]*DependencyStatus `json:"runners,omitempty"`
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
	workflows             repository.WorkflowRepository
	workflowExecutions    repository.WorkflowExecutionRepository
	tasks                 repository.TaskRepository
	runs                  repository.RunRepository
	checkpoints           repository.CheckpointRepository            // For resumption support
	idempotency           repository.IdempotencyRepository           // For replay safety
	investigationSettings repository.InvestigationSettingsRepository // For investigation config

	// Adapters (external integrations)
	runners          runner.Registry
	sandbox          sandbox.Provider
	workspaceSandbox phases.WorkspaceSandboxEnsurer
	events           event.Store
	artifacts        artifact.Collector

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

	rolePolicy   *rolepolicy.State
	roleResolver rolepolicy.Resolver

	// Model + runner health audit (SQLite-persisted, populated by runtime
	// classification + the periodic probe). Snapshots derive from
	// MAX(timestamp) over the audit tables; nil store means health
	// observability is disabled (writes are silent no-ops, snapshots
	// return empty).
	healthStore *health.Store

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

	// dispatcher serializes runner startups and exposes queue depth.
	// All run-spawn paths (CreateRun, ResumeRun) MUST go through it —
	// see contract decision 2 in scenarios/agent-manager/docs/internal/SEAMS.md.
	dispatcher *spawn.Dispatcher

	// awaitRegistry drives durable park/wait (Phase 3): it owns a background
	// waiter per parked run's await-handle, wakes the run on resolve/deadline,
	// and re-spawns waiters for persisted parked runs after a restart. Nil
	// disables the auto-waiter (park/wake transitions still work; nothing drives
	// them) — wired post-construction via SetAwaitRegistry to break the
	// registry↔orchestrator construction cycle (mirrors SetReconciler).
	awaitRegistry *AwaitRegistry

	// interactiveSessions is the web-console session controller that drives
	// interactive runs (ExecutionMode=interactive): agent-manager launches the
	// real interactive CLI in a web-console tmux session and tails its
	// agent-owned transcript. Nil when interactive mode is not wired — an
	// interactive run then fails cleanly at execute time rather than hanging.
	interactiveSessions webconsole.SessionController

	// webConsoleUIBase is the resolved web-console UI base URL (browser-facing
	// origin), captured once at wiring time. Used to build the run-detail deep
	// link (Run.WebConsoleSessionURL) for interactive runs. Empty when the UI
	// base could not be resolved — the deep link is then omitted and clients
	// fall back to the session id.
	webConsoleUIBase string

	// interactiveDrivers tracks the live interactive coordinators agent-manager
	// owns (the initial Execute turn and any Continue-driven follow-up turn) so
	// StopRun can cancel the coordinator deterministically and wait for it to
	// exit before finalizing — see stopInteractiveRun. Always non-nil (set in
	// New).
	interactiveDrivers *interactiveDriverRegistry

	// structuredResults owns the deterministic-first typed-output projection.
	// It is always present; its optional extractor is selected by portable role.
	structuredResults phases.StructuredResultResolver
	workflowEngine    *workflowruntime.Engine

	// workflowNudger drives a parent execution forward when one of its runs (or
	// child workflows) reaches terminal, so no consumer polls Advance. Nil
	// disables completion-driven advance — the reconciler recovery sweep still
	// re-drives non-terminal executions. Wired post-construction via
	// SetWorkflowNudger to break the nudger↔orchestrator construction cycle
	// (mirrors SetAwaitRegistry).
	workflowNudger *WorkflowNudger

	// workflowWaiters backs the blocking WaitWorkflowExecution RPC: an
	// event-driven notifier the drive path fires when an execution settles
	// terminal. Always non-nil (set in New) so waits work even before the
	// nudger is wired.
	workflowWaiters *workflowWaitRegistry
}

// OrchestratorConfig holds service configuration.
type OrchestratorConfig struct {
	DefaultTimeout          time.Duration
	MaxConcurrentRuns       int
	DefaultProjectRoot      string
	RequireSandboxByDefault bool
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

// WithWorkspaceSandboxEnsurer wires the run-time availability seam used by
// sandboxed run setup/finalization when workspace-sandbox is unavailable.
func WithWorkspaceSandboxEnsurer(e phases.WorkspaceSandboxEnsurer) Option {
	return func(o *Orchestrator) {
		o.workspaceSandbox = e
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

func WithWorkflowRepository(repo repository.WorkflowRepository) Option {
	return func(o *Orchestrator) { o.workflows = repo }
}

func WithWorkflowExecutionRepository(repo repository.WorkflowExecutionRepository) Option {
	return func(o *Orchestrator) { o.workflowExecutions = repo }
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

// WithRolePolicyState wires portable role resolution. The resolver is an
// explicit seam so tests never need installed resource CLIs.
func WithRolePolicyState(state *rolepolicy.State, resolver rolepolicy.Resolver) Option {
	return func(o *Orchestrator) {
		o.rolePolicy = state
		o.roleResolver = resolver
	}
}

// WithStructuredExtractor wires the optional constrained extraction backend.
// The backend receives the portable role from ResultSpec and cannot bypass
// local schema validation.
func WithStructuredExtractor(extractor structuredresult.Extractor) Option {
	return func(o *Orchestrator) {
		o.structuredResults = structuredresult.Resolver{Extractor: extractor}
	}
}

// WithHealthStore wires the persisted health audit store. The executor
// records every model-availability classification (ok or failed) here
// via the ModelHealthReporter seam, and GetModelHealthSnapshot reads
// the current snapshot from the same store. Pass nil to disable health
// observability (writes become no-ops).
func WithHealthStore(store *health.Store) Option {
	return func(o *Orchestrator) {
		o.healthStore = store
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

// WithSpawnDispatcher installs the runner-startup dispatcher. The
// orchestrator routes every spawn (CreateRun, ResumeRun) through it
// to enforce startup serialization and surface queue depth in
// CreateRunResponse. Supplying nil leaves the dispatcher unset, which
// is only valid in tests that mock CreateRun's spawn path.
func WithSpawnDispatcher(d *spawn.Dispatcher) Option {
	return func(o *Orchestrator) {
		o.dispatcher = d
	}
}

// WithInteractiveSessions wires the web-console session controller that drives
// interactive-mode runs. Without it, a run created with
// ExecutionMode=interactive fails cleanly at execute time.
func WithInteractiveSessions(sessions webconsole.SessionController) Option {
	return func(o *Orchestrator) {
		o.interactiveSessions = sessions
	}
}

// WithWebConsoleUIBase sets the resolved web-console UI base URL used to build
// the run-detail deep link for interactive runs. Empty disables the deep link
// (clients fall back to the session id).
func WithWebConsoleUIBase(base string) Option {
	return func(o *Orchestrator) {
		o.webConsoleUIBase = base
	}
}

// SetReconciler sets the reconciler reference for hot-reload propagation.
// This is called after construction because the reconciler depends on the orchestrator.
func (o *Orchestrator) SetReconciler(r *Reconciler) {
	o.reconciler = r
	if r != nil {
		r.structuredResults = o.structuredResults
	}
}

// SetAwaitRegistry wires the durable park/wait registry after construction. The
// registry takes the orchestrator as its waker (WakeRun/ListParkedRuns), so it
// must be built once the orchestrator exists — mirroring SetReconciler.
func (o *Orchestrator) SetAwaitRegistry(r *AwaitRegistry) {
	o.awaitRegistry = r
}

// SetWorkflowNudger wires the completion-nudge queue. The nudger's drive
// function is o.driveWorkflowExecution, so it must be built once the
// orchestrator exists — mirroring SetAwaitRegistry. Nil leaves completion-
// driven advance disabled (the reconciler recovery sweep still re-drives).
func (o *Orchestrator) SetWorkflowNudger(n *WorkflowNudger) {
	o.workflowNudger = n
}

// SpawnStats returns the current spawn-dispatcher state. Safe to call
// from the HTTP response path. Returns the zero Stats when the
// dispatcher is unset (test-only).
func (o *Orchestrator) SpawnStats() spawn.Stats {
	if o.dispatcher == nil {
		return spawn.Stats{}
	}
	return o.dispatcher.Stats()
}

// New creates a new Orchestrator with the given dependencies.
//
// If no spawn dispatcher is supplied via [WithSpawnDispatcher], a
// default one is constructed with single-slot serialization and queue
// capacity = MaxConcurrentRuns * 2. This keeps tests that don't care
// about the dispatcher working without per-test boilerplate, while
// production wiring still passes an explicitly-tuned dispatcher.
func New(
	profiles repository.ProfileRepository,
	tasks repository.TaskRepository,
	runs repository.RunRepository,
	opts ...Option,
) *Orchestrator {
	o := &Orchestrator{
		profiles:           profiles,
		tasks:              tasks,
		runs:               runs,
		config:             DefaultConfig(),
		interactiveDrivers: newInteractiveDriverRegistry(),
		structuredResults:  structuredresult.Resolver{},
		workflowWaiters:    newWorkflowWaitRegistry(),
	}

	for _, opt := range opts {
		opt(o)
	}
	if o.workflowExecutions != nil && o.workflows != nil {
		expressions, _ := workflowruntime.NewExpressionEvaluator()
		o.workflowEngine = &workflowruntime.Engine{Store: o.workflowExecutions, Catalog: o.workflows, Children: workflowChildLauncher{o: o}, Subworkflows: workflowSubworkflowLauncher{o: o}, Expressions: expressions}
	}

	if o.dispatcher == nil {
		queueCap := o.config.MaxConcurrentRuns * 2
		if queueCap < 1 {
			queueCap = 8
		}
		o.dispatcher = spawn.New(spawn.Config{
			MaxStartingConcurrency: 1,
			QueueCapacity:          queueCap,
		})
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
	existing, err := o.profiles.Get(ctx, profile.ID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.SourcePath != "" {
		profile.OwnerScenario = existing.OwnerScenario
		profile.SourcePath = existing.SourcePath
		profile.SourceHash = existing.SourceHash
		profile.LastAppliedHash = existing.LastAppliedHash
		profile.SourceUpdatedAt = existing.SourceUpdatedAt
		profile.LocalOverride = true
	}
	profile.UpdatedAt = time.Now()
	if existing != nil {
		profile.CreatedAt = existing.CreatedAt
	}

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

	defaults := req.Defaults
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

	// Determine run mode.
	//
	// SandboxConfig.Mode is the single source of truth; DeriveRunMode
	// translates the resolved Mode to a RunMode without consulting any
	// other input. See docs/internal/SEAMS.md (RunMode decision boundary)
	// and docs/internal/INVARIANTS.md.
	//
	// Decision priority (highest first):
	//   1. Explicit caller override via req.RunMode
	//   2. ForceInPlace (policy must permit; orchestrator validates that
	//      the resolved sandbox mode is at or above policy's required
	//      minimum below)
	//   3. Derived from sandboxConfig.Mode
	runMode := domain.DeriveRunMode(sandboxConfig)
	if req.RunMode != nil {
		runMode = *req.RunMode
	} else if req.ForceInPlace {
		runMode = domain.RunModeInPlace
	}

	// Policy gate (locked decision 5): interactive execution mode is only
	// available for non-protected (in-place) runs. Reject at creation time with
	// an actionable error before the run is persisted or dispatched. The
	// executeInteractiveRun path re-checks this as a backstop.
	if err := domain.ValidateInteractiveRunMode(req.ExecutionMode, runMode); err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

	// Enforce the policy-declared minimum sandbox mode. The policy layer
	// expresses sandbox requirements as a minimum SandboxMode rather
	// than a bool so a higher-strictness policy can require Protected
	// while still allowing Tracking-mode runs through other paths.
	if policyDecision != nil && policyDecision.RequiredSandboxMode != domain.SandboxModeUnspecified {
		resolvedMode := domain.SandboxModeOff
		if sandboxConfig != nil {
			resolvedMode = sandboxConfig.Mode.Effective()
		}
		if !resolvedMode.AtLeast(policyDecision.RequiredSandboxMode) {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationErrorWithHint(
				"sandboxConfig.mode",
				"resolved sandbox mode is below the policy-required minimum",
				fmt.Sprintf("policy requires Mode >= %q; resolved Mode is %q",
					policyDecision.RequiredSandboxMode, resolvedMode),
			)
		}
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
				"set runMode to sandboxed or set sandboxConfig.mode to a sandbox-enabled value (tracking/protected)")
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
		ExecutionMode:            req.ExecutionMode,
		Status:                   domain.RunStatusPending,
		Phase:                    domain.RunPhaseQueued,
		ProgressPercent:          0,
		IdempotencyKey:           req.IdempotencyKey,
		ApprovalState:            domain.ApprovalStateNone,
		ResolvedConfig:           resolvedConfig,
		SandboxConfig:            sandboxConfig,
		ConversationID:           req.ConversationID,
		ParentRunID:              req.ParentRunID,
		// Persist caller-supplied custom env so the continue/wake path can
		// re-inject it. Already VROOLI_*-validated at the API boundary.
		CustomEnv: req.Environment,
		// Provenance: requested is the primary model the preset expanded to at creation.
		// Actual is blank until the executor records the model that actually ran.
		RequestedModel: resolvedConfig.Model,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	// Apply Decision D7 precedence (spawner > parent inheritance > fresh
	// UUID). When the spawn surface populates ConversationID directly,
	// step (1) wins; otherwise we inherit from ParentRunID's run when set,
	// or mint a fresh UUID.
	run.ConversationID = domain.ResolveConversationID(run, func(parentID uuid.UUID) (string, bool) {
		parent, perr := o.runs.Get(ctx, parentID)
		if perr != nil || parent == nil {
			return "", false
		}
		return parent.ConversationID, true
	})
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

	// Sandbox-default rollout adoption metrics (Phase D of
	// agent-sandbox-audit-foundation). Three labels capture the rollout
	// state per run: run_mode, sandbox_mode, manual_review.
	sandboxModeLabel := "n/a"
	manualReviewLabel := "false"
	if run.SandboxConfig != nil {
		sandboxModeLabel = string(run.SandboxConfig.Mode.Effective())
		if run.SandboxConfig.ManualReview {
			manualReviewLabel = "true"
		}
	}
	metrics.Get().RecordRunCreated(string(resolvedConfig.RunnerType), string(run.RunMode))
	metrics.Get().RecordSandboxAdoption(string(run.RunMode), sandboxModeLabel, manualReviewLabel)

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
		if err := o.appendAndBroadcastEvents(ctx, run.ID, userEvent); err != nil {
			// Log but don't fail
			_ = err
		}
	}

	// Hand the executor body to the spawn dispatcher. Enqueue is the
	// only path through which a run begins — direct goroutine spawning
	// would skip startup serialization (codex SQLite WAL contention)
	// and queue-depth surfacing.
	if err := o.dispatcher.Enqueue(&spawn.Job{
		RunID:      run.ID,
		RunMode:    run.RunMode,
		RunnerType: runnerTypeOrEmpty(run),
		Sink:       o.dispatcherSink(run.ID),
		Fn: func(started spawn.StartedFn) {
			o.executeRun(context.Background(), run, task, profile, userMessage, systemPrompt, existingSandboxWorkDir, imageAttachments, req.Environment, started)
		},
	}); err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}

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
		if req.ProfileRef.Defaults == nil && !req.ProfileRef.UpdateExisting {
			key := strings.TrimSpace(req.ProfileRef.ProfileKey)
			if key == "" {
				return nil, nil, domain.NewValidationErrorWithHint("profileRef.profileKey", "field is required",
					"Provide a stable profile key or inline profile defaults")
			}
			var err error
			profile, err = o.profiles.GetByKey(ctx, key)
			if err != nil {
				return nil, nil, err
			}
			if profile == nil {
				return nil, nil, domain.NewValidationErrorWithHint("profileRef.profileKey", "profile not found",
					"Start the owning scenario so agent-manager can reconcile its manifest-declared profiles")
			}
		} else {
			result, err := o.EnsureProfile(ctx, EnsureProfileRequest{
				ProfileKey:     req.ProfileRef.ProfileKey,
				Defaults:       req.ProfileRef.Defaults,
				UpdateExisting: req.ProfileRef.UpdateExisting,
			})
			if err != nil {
				return nil, nil, err
			}
			profile = result.Profile
		}
		if profile != nil {
			cfg.ApplyProfile(profile)
		}
	}

	// Apply inline overrides
	if req.RoleRef != nil {
		cfg.RoleRef = strings.TrimSpace(*req.RoleRef)
	}
	if req.MaxTurns != nil {
		cfg.MaxTurns = *req.MaxTurns
	}
	if req.Timeout != nil {
		cfg.Timeout = *req.Timeout
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
	if req.NetworkAccess != nil {
		cfg.NetworkAccess = *req.NetworkAccess
	}
	if req.AllowedPaths != nil {
		cfg.AllowedPaths = req.AllowedPaths
	}
	if req.DeniedPaths != nil {
		cfg.DeniedPaths = req.DeniedPaths
	}
	if req.ResultSpec != nil {
		normalized, err := structuredresult.NormalizeSpec(req.ResultSpec)
		if err != nil {
			return nil, nil, domain.NewValidationErrorWithHint("resultSpec", err.Error(),
				"Use result-spec/v1 with the documented bounded JSON Schema subset")
		}
		cfg.ResultSpec = normalized
	}
	// Validate the resolved config
	if strings.TrimSpace(cfg.RoleRef) == "" {
		return nil, nil, domain.NewValidationErrorWithHint("roleRef", "field is required",
			"Select a portable role from the active role-policy catalog")
	}
	if err := o.resolveExecutionPolicy(ctx, cfg); err != nil {
		return nil, nil, err
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

// resolveExecutionPolicy converts the final profile-plus-override selection
// into a run-owned immutable snapshot. A named policy is resolved once; no
// runtime decision reads mutable catalog state after this function returns.
func (o *Orchestrator) resolveExecutionPolicy(ctx context.Context, cfg *domain.RunConfig) error {
	if cfg == nil {
		return domain.NewValidationError("runConfig", "field is required")
	}
	if strings.TrimSpace(cfg.RoleRef) != "" {
		if o.rolePolicy == nil || o.roleResolver == nil {
			return domain.NewValidationError("rolePolicyCatalog", "role policy state or resource resolver is not configured")
		}
		resolution, err := o.rolePolicy.Resolve(ctx, o.roleResolver, cfg.RoleRef)
		if err != nil {
			return err
		}
		snapshot := resolution.Snapshot()
		if snapshot == nil || len(snapshot.Candidates) == 0 {
			return domain.NewValidationError("rolePolicyCatalog", "role resolution produced no candidates")
		}
		selectedIndex, preflight, err := o.selectInitialCandidate(ctx, snapshot.Candidates)
		if err != nil {
			return err
		}
		snapshot.SelectedIndex = selectedIndex
		snapshot.SelectedCandidate = snapshot.Candidates[selectedIndex]
		snapshot.Explanation.Preflight = preflight
		snapshot.Explanation.Summary = fmt.Sprintf(
			"%s; selected candidate %d (%s/%s)",
			snapshot.Explanation.Summary,
			selectedIndex,
			snapshot.SelectedCandidate.RunnerType,
			snapshot.SelectedCandidate.Model,
		)
		cfg.PolicySnapshot = snapshot
		cfg.RunnerType = snapshot.SelectedCandidate.RunnerType
		cfg.Model = snapshot.SelectedCandidate.Model
		return nil
	}
	return domain.NewValidationError("roleRef", "field is required")
}

func (o *Orchestrator) selectInitialCandidate(ctx context.Context, candidates []domain.ExecutionCandidate) (int, []domain.CandidatePreflight, error) {
	if len(candidates) == 0 {
		return -1, nil, domain.NewValidationError("rolePolicyCatalog", "resolution produced no candidates")
	}
	if o.runners == nil {
		// Minimal unit orchestrators omit adapters. Production always injects
		// the registry before accepting traffic.
		return 0, nil, nil
	}

	checks := make([]domain.CandidatePreflight, 0, len(candidates))
	for index, candidate := range candidates {
		check := domain.CandidatePreflight{Index: index, Candidate: candidate}
		// Availability is resource-resolution evidence for portable roles.
		// Legacy snapshots predate that field, so their zero value must not
		// make every historical/direct candidate unavailable.
		if candidate.ResourceRole != "" && !candidate.Available {
			check.Reason = candidate.Failure
			if check.Reason == "" {
				check.Reason = candidate.FailureCode
			}
			if check.Reason == "" {
				check.Reason = "resource role is unavailable"
			}
			checks = append(checks, check)
			continue
		}
		resolvedRunner, err := o.runners.Get(candidate.RunnerType)
		if err != nil || resolvedRunner == nil {
			check.Reason = "runner is not registered"
			checks = append(checks, check)
			continue
		}
		available, message := resolvedRunner.IsAvailable(ctx)
		if !available {
			check.Reason = strings.TrimSpace(message)
			if check.Reason == "" {
				check.Reason = "runner is unavailable"
			}
			checks = append(checks, check)
			continue
		}
		switch candidate.SelectionType {
		case domain.ModelSelectionTypeModel:
			if err := resolvedRunner.ProbeModel(ctx, candidate.Model); err != nil {
				check.Reason = err.Error()
				checks = append(checks, check)
				continue
			}
		case domain.ModelSelectionTypeRunnerDefault:
			// Catalog/codec conformance already proves runner-default support.
		default:
			check.Reason = "candidate selection type is invalid"
			checks = append(checks, check)
			continue
		}
		check.Available = true
		checks = append(checks, check)
		return index, checks, nil
	}

	reasons := make([]string, 0, len(checks))
	for _, check := range checks {
		reasons = append(reasons, fmt.Sprintf("candidate %d %s/%s: %s", check.Index, check.Candidate.RunnerType, check.Candidate.SelectionType, check.Reason))
	}
	return -1, checks, domain.NewValidationErrorWithHint(
		"rolePolicyCatalog",
		"no policy candidate passed runner/model preflight",
		strings.Join(reasons, "; "),
	)
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
//
// Phase G: Profile/req AllowedPaths/DeniedPaths are merged into the resolved
// SandboxConfig.Acceptance so they become *enforced* at apply-at-run-end
// rather than passed as advisory env vars to runners. This is the
// agent-sandbox-audit-foundation policy-to-sandbox handoff.
func (o *Orchestrator) resolveSandboxConfig(req CreateRunRequest, profile *domain.AgentProfile) (*domain.SandboxConfig, error) {
	// Start from the auditability-contract defaults (Mode=Protected,
	// AutoApply=true, ApplyOnFailure=true, NetworkMode=localhost,
	// NoLock=true). Profile and request overrides clone over the top.
	// Without this baseline, a request with no profile and no inline
	// config would zero-value the struct, dropping Mode to unspecified
	// and silently downgrading to host-tracked execution.
	defaults := domain.DefaultSandboxConfig()
	cfg := defaults
	if profile != nil && profile.SandboxConfig != nil {
		cfg = cloneSandboxConfig(profile.SandboxConfig)
	}
	if req.SandboxConfig != nil {
		cfg = cloneSandboxConfig(req.SandboxConfig)
	}

	// Backfill enum/string fields that the override left at the proto
	// zero-value. Callers (notably swarm-manager) often send a partial
	// SandboxConfig containing only Acceptance overrides; without this
	// backfill the wholesale-replace clone above would silently strip
	// Mode and NetworkMode to "unspecified", silently downgrading
	// protected runs to tracking. Pointer-typed fields (AutoApply,
	// ApplyOnFailure) and structural fields (Lifecycle, Acceptance) are
	// left intentional-explicit; bool fields (ManualReview, NoLock) are
	// left at the override's value because zero is operator-visible
	// "off" rather than "not provided".
	if cfg.Mode == domain.SandboxModeUnspecified {
		cfg.Mode = defaults.Mode
	}
	if cfg.NetworkMode == "" {
		cfg.NetworkMode = defaults.NetworkMode
	}

	// Push path policy from profile/request into the acceptance layer so
	// workspace-sandbox actually enforces it at apply time. The runner-side
	// advisory env vars are kept for the tracking-mode capability matrix
	// but the load-bearing enforcement now lives at the sandbox boundary.
	allowedPaths := profilePaths(profile, func(p *domain.AgentProfile) []string { return p.AllowedPaths })
	if req.AllowedPaths != nil {
		allowedPaths = req.AllowedPaths
	}
	deniedPaths := profilePaths(profile, func(p *domain.AgentProfile) []string { return p.DeniedPaths })
	if req.DeniedPaths != nil {
		deniedPaths = req.DeniedPaths
	}
	cfg.Acceptance.Allow.PathGlobs = mergeUnique(cfg.Acceptance.Allow.PathGlobs, allowedPaths)
	cfg.Acceptance.Deny.PathGlobs = mergeUnique(cfg.Acceptance.Deny.PathGlobs, deniedPaths)

	cfg = normalizeSandboxConfig(cfg)
	if err := validateSandboxConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func profilePaths(p *domain.AgentProfile, get func(*domain.AgentProfile) []string) []string {
	if p == nil {
		return nil
	}
	return get(p)
}

func mergeUnique(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(a)+len(b))
	out := make([]string, 0, len(a)+len(b))
	for _, v := range a {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	for _, v := range b {
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	return out
}

func cloneSandboxConfig(cfg *domain.SandboxConfig) *domain.SandboxConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	clone.Lifecycle.CheckpointOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.CheckpointOn...)
	clone.Lifecycle.StopOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.StopOn...)
	clone.Lifecycle.DeleteOn = append([]domain.SandboxLifecycleEvent(nil), cfg.Lifecycle.DeleteOn...)
	clone.Acceptance.Allow = cloneSandboxCriteria(cfg.Acceptance.Allow)
	clone.Acceptance.Deny = cloneSandboxCriteria(cfg.Acceptance.Deny)
	if cfg.AutoApply != nil {
		v := *cfg.AutoApply
		clone.AutoApply = &v
	}
	if cfg.ApplyOnFailure != nil {
		v := *cfg.ApplyOnFailure
		clone.ApplyOnFailure = &v
	}
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

	// Default lifecycle cleanup for auto-apply sandboxes.
	//
	// Under the auditability contract (Phase 3b), AutoApply=true (the
	// contract default unless ManualReview=true) means the sandbox is
	// applied at run end. Once applied, leaving the sandbox active
	// indefinitely blocks future runs on the same scope path and leaks
	// overlay mounts. We default deleteOn to ["terminal"] so the sandbox
	// is cleaned up after any terminal event when ManualReview is off.
	//
	// ManualReview=true sandboxes intentionally persist past run end so
	// operators can review; their TTL GC is owned by workspace-sandbox
	// LifecycleReconciler (Phase 4).
	if cfg.GetAutoApply() && !cfg.ManualReview &&
		len(cfg.Lifecycle.CheckpointOn) == 0 && len(cfg.Lifecycle.DeleteOn) == 0 && len(cfg.Lifecycle.StopOn) == 0 {
		cfg.Lifecycle.CheckpointOn = []domain.SandboxLifecycleEvent{
			domain.SandboxLifecycleTurnCompleted,
			domain.SandboxLifecycleTurnFailed,
			domain.SandboxLifecycleTurnCancelled,
		}
		cfg.Lifecycle.DeleteOn = []domain.SandboxLifecycleEvent{domain.SandboxLifecycleRunFinalized}
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
	// Warn when AutoApply is on (the contract default) but no allow
	// criteria are configured. This is valid (empty allow = accept all
	// non-denied files), but surprising enough to warrant a log line —
	// especially since an empty deny (from proto serialization)
	// previously caused silent universal denial.
	if cfg.GetAutoApply() && !cfg.ManualReview &&
		len(cfg.Acceptance.Allow.PathGlobs) == 0 &&
		len(cfg.Acceptance.Allow.Extensions) == 0 {
		obs.Component("sandbox-config").Info("autoApply enabled with no allow criteria; all non-denied files will be applied at run end")
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
		ScopePrefix:               opts.ScopePrefix,
		InvestigatesRunID:         opts.InvestigatesRunID,
		AppliesInvestigationRunID: opts.AppliesInvestigationRunID,
	})
	if err != nil {
		return nil, err
	}
	return o.attachRunActionsList(ctx, runs), nil
}

// ListParkedRuns returns all parked runs with their await-handle populated. The
// pruned list-columns omit the heavy await_handle field, so each parked run is
// reloaded by ID to recover its handle. Used by the await-handle registry's
// restart recovery (re-spawning waiters on boot).
func (o *Orchestrator) ListParkedRuns(ctx context.Context) ([]*domain.Run, error) {
	parked := domain.RunStatusParked
	rows, err := o.runs.List(ctx, repository.RunListFilter{Status: &parked})
	if err != nil {
		return nil, err
	}
	full := make([]*domain.Run, 0, len(rows))
	for _, row := range rows {
		loaded, err := o.runs.Get(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		if loaded == nil {
			continue
		}
		full = append(full, loaded)
	}
	return full, nil
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
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return err
	}

	if allowed, reason := domain.CanStopRun(run); !allowed {
		return domain.NewStateError("Run", string(run.Status), "stop", reason)
	}

	// Interactive runs have no local process to signal — the CLI lives in a
	// web-console tmux session. Stop them via the interrupt-then-delete
	// escalation ladder and finalize deterministically (Cancelled), instead of
	// the pgid/terminator path below.
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		return o.stopInteractiveRun(ctx, run)
	}

	// A parked run has no live process to terminate — stopping it cancels the
	// await (clears the handle) and moves the run to cancelled. The waiter that
	// owns the handle observes the terminal status and deregisters (Phase 3).
	if run.Status == domain.RunStatusParked {
		// Cancel the background watcher first so it observes the cancellation and
		// exits without waking the now-cancelled run.
		if o.awaitRegistry != nil {
			o.awaitRegistry.Cancel(id)
		}
		run.AwaitHandle = nil
		endedAt := time.Now()
		_, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
			Run:       run,
			NewStatus: domain.RunStatusCancelled,
			Phase:     domain.RunPhaseCompleted,
			Reason:    "Parked run stopped by request",
			EndedAt:   &endedAt,
		})
		return err
	}

	if o.terminator != nil {
		result, err := o.terminator.Terminate(ctx, id)
		if err != nil {
			return err
		}
		if !result.Success {
			return result.Error
		}

		endedAt := time.Now()
		_, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
			Run:       run,
			NewStatus: domain.RunStatusCancelled,
			Phase:     domain.RunPhaseCompleted,
			Reason:    "Run stopped by request",
			EndedAt:   &endedAt,
		})
		return err
	}

	// The immutable resolved config is the sole execution authority.
	var runnerType domain.RunnerType
	if run.ResolvedConfig != nil {
		runnerType = run.ResolvedConfig.RunnerType
	}

	// Stop execution if we have a runner type
	if o.runners != nil && runnerType != "" {
		if r, err := o.runners.Get(runnerType); err == nil {
			if err := r.Stop(ctx, run.ID); err != nil {
				return err
			}
		}
	}

	endedAt := time.Now()
	_, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:       run,
		NewStatus: domain.RunStatusCancelled,
		Phase:     domain.RunPhaseCompleted,
		Reason:    "Run stopped by request",
		EndedAt:   &endedAt,
	})
	return err
}

func (o *Orchestrator) RecoverRun(ctx context.Context, id uuid.UUID) (*RecoverResult, error) {
	if o.reconciler == nil {
		return nil, domain.NewConfigMissingError("reconciler", "reconciler not configured", nil)
	}
	return o.reconciler.RecoverRun(ctx, id)
}

// ContinueRun continues an existing run's conversation with a follow-up message.
// The message is appended to the run's event stream and the response is streamed back.
func (o *Orchestrator) ContinueRun(ctx context.Context, req ContinueRunRequest) (*domain.Run, error) {
	if req.IdempotencyKey != "" && o.idempotency != nil {
		existing, err := o.idempotency.Check(ctx, req.IdempotencyKey)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.Status == domain.IdempotencyStatusComplete && existing.EntityID != nil {
			return o.GetRun(ctx, *existing.EntityID)
		}
		if existing != nil && existing.Status == domain.IdempotencyStatusPending {
			return nil, domain.NewStateError("Run", "continuing", "continue", "a continuation with this idempotency key is already in progress")
		}
	}
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
	if req.IdempotencyKey != "" && o.idempotency != nil {
		if _, err := o.idempotency.Reserve(ctx, req.IdempotencyKey, time.Hour); err != nil {
			return nil, domain.NewStateError("Run", "continuing", "continue", "a continuation with this idempotency key is already in progress")
		}
	}

	// Interactive runs continue by typing the follow-up into the still-live
	// web-console session (never a process respawn) and reattaching a tailer to
	// drive the new turn to completion — see continueInteractiveRun.
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		if req.MaxTurns != nil || req.Timeout != nil || req.ResultSpec != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, domain.NewValidationError("continuationOverrides", "interactive continuation does not support per-turn workflow overrides")
		}
		continued, err := o.continueInteractiveRun(ctx, run, req.Message, req.AttachmentIDs)
		if err != nil {
			o.markIdempotencyFailed(ctx, req.IdempotencyKey)
			return nil, err
		}
		o.markIdempotencyComplete(ctx, req.IdempotencyKey, continued.ID, "Run")
		return continued, nil
	}
	continued, err := o.resumeConversation(ctx, run, req.Message, req.AttachmentIDs, "Continuation requested", continuationOverrides{MaxTurns: req.MaxTurns, Timeout: req.Timeout, ResultSpec: req.ResultSpec})
	if err != nil {
		o.markIdempotencyFailed(ctx, req.IdempotencyKey)
		return nil, err
	}
	o.markIdempotencyComplete(ctx, req.IdempotencyKey, continued.ID, "Run")
	return continued, nil
}

// resumeConversation drives a single session-resume turn shared by both
// operator-driven continuation (ContinueRun) and waiter-driven wake (WakeRun):
// it resolves the runner, re-acquires the checkpointed sandbox, transitions the
// run back to running (resetting the heartbeat in the same transition — the
// faefb9cb54 invariant), re-assembles the full process env + a freshly
// regenerated identity token (Phase 0 assembler), and spawns executeContinuation
// with `message` injected as the next user turn. Centralising it keeps continue
// and wake from ever diverging on env/identity/heartbeat handling. Callers are
// responsible for the precondition gate (CanContinueRun for continue, the parked
// guard for wake) before calling this.
type continuationOverrides struct {
	MaxTurns   *int
	Timeout    *time.Duration
	ResultSpec *domain.ResultSpec
}

func (o *Orchestrator) resumeConversation(ctx context.Context, run *domain.Run, message string, attachmentIDs []string, reason string, overrides continuationOverrides) (*domain.Run, error) {
	// The immutable resolved config is the sole execution authority.
	var runnerType domain.RunnerType
	if run.ResolvedConfig != nil {
		runnerType = run.ResolvedConfig.RunnerType
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

	task, err := o.GetTask(ctx, run.TaskID)
	if err != nil {
		return nil, err
	}

	// Profile is best-effort: it only supplies ProfileKey to the regenerated
	// identity token (a continuation can still run without it).
	var profile *domain.AgentProfile
	if run.AgentProfileID != nil {
		if p, perr := o.GetProfile(ctx, *run.AgentProfileID); perr == nil {
			profile = p
		}
	}

	workDir, err := o.prepareContinuationSandbox(ctx, run, task)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	run, err = o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
		Run:           run,
		NewStatus:     domain.RunStatusRunning,
		Phase:         domain.RunPhaseExecuting,
		Reason:        reason,
		LastHeartbeat: &now,
	})
	if err != nil {
		return nil, err
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
			userEvent = domain.NewMessageEventWithAttachments(run.ID, "user", message, attInfo)
		} else {
			userEvent = domain.NewMessageEvent(run.ID, "user", message)
		}
		if err := o.appendAndBroadcastEvents(ctx, run.ID, userEvent); err != nil {
			_ = err
		}
	}

	eventSink := o.runEventSink(run.ID)

	// Resolve attachments
	var attachments []runner.Attachment
	if len(attachmentIDs) > 0 && o.storage != nil {
		metas, err := o.storage.GetMultiple(ctx, attachmentIDs)
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

	transcriptCfg, cleanupTranscript, err := o.prepareRunTranscript(ctx, run, workDir)
	if err != nil {
		return nil, err
	}

	// Re-assemble the full process env for this turn synchronously (before the
	// background goroutine spawns) so the run mutation from identity
	// regeneration happens on this goroutine — the same one that returns
	// attachRunActions(run) below — and never races the executeContinuation
	// goroutine. The original Execute call injected custom env + sandbox routing
	// + an identity token; a continuation that left ContinueRequest.Environment
	// nil silently dropped all three (the latent bug this fixes). We:
	//   1. regenerate the identity token (the plaintext is never stored, only
	//      the hash, so the original is unrecoverable — and a long-parked run
	//      could otherwise outlive its 24h TTL). GenerateIdentityToken
	//      re-persists the hash so /identity/verify keeps working.
	//   2. re-derive VROOLI_SANDBOX_* from the live (resumed) sandbox.
	//   3. re-inject the persisted custom env.
	// Both Execute and this path build env through phases.AssembleRunEnv so they
	// can never diverge again.
	continueEnv := o.assembleContinuationEnv(ctx, run, task, profile, workDir)

	// Hand the background turn its OWN *Run value. executeContinuation mutates
	// the run in place (status via applyRunStatusTransition, SessionID,
	// heartbeat) on a separate goroutine, while this goroutine returns
	// attachRunActions(run) below — sharing the same pointer is a data race
	// (the deferred -race failure in
	// TestContinueRun_ProtectedSandboxCarriesLauncherInputsAndLifecycleEvents).
	// A shallow copy is sufficient: the racing fields are scalars / freshly
	// reassigned pointers, and the persisted DB row remains the single source of
	// truth, so the background copy and the returned snapshot converge on reload.
	runForExec := *run
	if run.ResolvedConfig != nil {
		resolved := *run.ResolvedConfig
		if overrides.MaxTurns != nil {
			resolved.MaxTurns = *overrides.MaxTurns
		}
		if overrides.Timeout != nil {
			resolved.Timeout = *overrides.Timeout
		}
		if overrides.ResultSpec != nil {
			resolved.ResultSpec = overrides.ResultSpec
		}
		runForExec.ResolvedConfig = &resolved
	}

	// Execute continuation asynchronously
	go o.executeContinuation(context.Background(), &runForExec, r, eventSink, message, workDir, attachments, continueEnv, transcriptCfg, cleanupTranscript)

	return o.attachRunActions(ctx, run), nil
}

// assembleContinuationEnv regenerates the run's identity token and builds the
// full process env (custom + sandbox + identity) for a continuation/wake turn.
// Called synchronously on the request goroutine so the identity-hash persist
// does not race the background executeContinuation goroutine.
func (o *Orchestrator) assembleContinuationEnv(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, workDir string) map[string]string {
	scopePath := ""
	if task != nil {
		scopePath = task.ScopePath
	}
	identityToken := ""
	if len(o.identitySecret) > 0 {
		identityToken = phases.GenerateIdentityToken(ctx, phases.GenerateIdentityTokenInput{
			Deps:    phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster},
			Run:     run,
			Profile: profile,
			Task:    task,
			Secret:  o.identitySecret,
		})
	}
	return phases.AssembleRunEnv(phases.AssembleRunEnvInput{
		Custom:        run.CustomEnv,
		RunMode:       run.RunMode,
		SandboxID:     run.SandboxID,
		WorkDir:       workDir,
		ScopePath:     scopePath,
		IdentityToken: identityToken,
	})
}

func (o *Orchestrator) prepareContinuationSandbox(ctx context.Context, run *domain.Run, task *domain.Task) (string, error) {
	if run == nil {
		return "", domain.NewValidationError("run", "run is required")
	}
	if run.SandboxID == nil {
		if task != nil && task.ProjectRoot != "" {
			return task.ProjectRoot, nil
		}
		return "", nil
	}
	if o.sandbox == nil {
		return "", domain.NewConfigMissingError("sandbox", "provider not configured", nil)
	}

	sb, err := o.sandbox.Get(ctx, *run.SandboxID)
	if err != nil {
		return "", err
	}
	switch sb.Status {
	case sandbox.SandboxStatusCheckpointed:
		sb, err = o.sandbox.Resume(ctx, sb.ID)
		if err != nil {
			return "", err
		}
	case sandbox.SandboxStatusStopped:
		if err := o.sandbox.Start(ctx, sb.ID); err != nil {
			return "", err
		}
		sb, err = o.sandbox.Get(ctx, sb.ID)
		if err != nil {
			return "", err
		}
	case sandbox.SandboxStatusActive:
	case sandbox.SandboxStatusDeleted, sandbox.SandboxStatusRejected, sandbox.SandboxStatusApproved, sandbox.SandboxStatusError:
		return "", domain.NewStateError("Sandbox", string(sb.Status), "continue", "sandbox is not resumable")
	default:
		return "", domain.NewStateError("Sandbox", string(sb.Status), "continue", "sandbox is not active or checkpointed")
	}

	workDir := sb.WorkDir
	if workDir == "" {
		workDir, err = o.sandbox.GetWorkspacePath(ctx, sb.ID)
		if err != nil {
			return "", err
		}
	}
	if workDir == "" {
		return "", domain.NewStateError("Sandbox", string(sb.Status), "continue", "resumed sandbox did not provide a workspace path")
	}
	return workDir, nil
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
	if err := o.appendAndBroadcastEvents(ctx, runID, deleteEvent); err != nil {
		return nil, err
	}
	return deleteEvent, nil
}

func (o *Orchestrator) prepareRunTranscript(ctx context.Context, run *domain.Run, workDir string) (*runner.TranscriptConfig, func(), error) {
	if run == nil || run.ResolvedConfig == nil {
		return nil, nil, nil
	}

	startedAt := time.Now().UTC()
	if run.StartedAt != nil {
		startedAt = run.StartedAt.UTC()
	}
	state, err := runstate.Open(run.ID, runstate.OpenOptions{
		RunnerType: run.ResolvedConfig.RunnerType,
		WorkingDir: workDir,
		StartedAt:  startedAt,
	})
	if err != nil {
		return nil, nil, err
	}

	snap := state.Snapshot()
	run.TranscriptPath = snap.TranscriptPath
	run.TranscriptCursor = snap.Cursor.TranscriptCursor
	run.TranscriptLastSeq = snap.Cursor.TranscriptLastSeq
	if err := o.runs.Update(ctx, run); err != nil {
		_ = state.Close()
		return nil, nil, err
	}

	cfg := &runner.TranscriptConfig{
		TranscriptPath: snap.TranscriptPath,
		StderrPath:     snap.StderrPath,
		StdoutFile:     state.TranscriptWriter(),
		StderrFile:     state.StderrWriter(),
		OnProcessStart: func(pid, pgid int) error {
			run.RunnerPID = pid
			run.RunnerPGID = pgid
			if err := state.PersistProcess(pid, pgid); err != nil {
				return err
			}
			return o.runs.Update(context.Background(), run)
		},
		OnAdvance: func(cursor, lastSeq int64) error {
			if cursor > run.TranscriptCursor {
				run.TranscriptCursor = cursor
			}
			if lastSeq > run.TranscriptLastSeq {
				run.TranscriptLastSeq = lastSeq
			}
			if err := state.PersistCursor(run.TranscriptCursor, run.TranscriptLastSeq); err != nil {
				return err
			}
			return o.runs.Update(context.Background(), run)
		},
		OnSessionID: func(sessionID string) error {
			if sessionID == "" || run.SessionID == sessionID {
				return nil
			}
			run.SessionID = sessionID
			if err := state.PersistSessionID(sessionID); err != nil {
				return err
			}
			return o.runs.Update(context.Background(), run)
		},
	}

	return cfg, func() { _ = state.Close() }, nil
}

// executeContinuation handles the actual continuation execution (runs in background).
// Each continuation turn gets its own timeout from RunTimeoutMinutes, so a timed-out
// run can be continued indefinitely — each "continue" message resets the clock.
func (o *Orchestrator) executeContinuation(ctx context.Context, run *domain.Run, r runner.Runner, eventSink runner.EventSink, message string, workDir string, attachments []runner.Attachment, continueEnv map[string]string, transcript *runner.TranscriptConfig, cleanupTranscript func()) {
	if cleanupTranscript != nil {
		defer cleanupTranscript()
	}

	levers := o.runLevers()
	timeout := levers.Execution.DefaultTimeout
	if run.ResolvedConfig != nil && run.ResolvedConfig.Timeout > 0 {
		timeout = run.ResolvedConfig.Timeout
	}
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Start heartbeat loop so the reconciler doesn't kill us during execution.
	// Heartbeat uses the parent ctx so it survives after execCtx deadline fires.
	heartbeatStop := make(chan struct{})
	heartbeatDone := make(chan struct{})
	go phases.RunHeartbeatLoop(ctx, phases.HeartbeatLoopInput{
		Deps: phases.Deps{
			Runs:        o.runs,
			Events:      o.events,
			Broadcaster: o.broadcaster,
			Levers:      levers,
		},
		Run:    run,
		Levers: levers,
		Stop:   heartbeatStop,
		Done:   heartbeatDone,
	})
	// stopHeartbeat is idempotent (sync.Once): it must run BEFORE the terminal
	// status transition below, because the heartbeat loop mutates the same *run
	// (LastHeartbeat) that applyRunStatusTransition writes — concurrent writes to
	// the shared run are a data race (the deferred -race failure). The defer is a
	// safety net for the early-return paths.
	var stopOnce sync.Once
	stopHeartbeat := func() {
		stopOnce.Do(func() {
			close(heartbeatStop)
			<-heartbeatDone
		})
	}
	defer stopHeartbeat()

	// Build continue request — pass the run's ResolvedConfig and SandboxID
	// so launcherSelector.PickFor routes the continuation through the same
	// host-or-sandbox path as the original Execute call. Without these,
	// protected runs would silently downgrade to host on continuation.
	continueReq := runner.ContinueRequest{
		RunID:          run.ID,
		SessionID:      run.SessionID,
		Prompt:         message,
		WorkingDir:     workDir,
		EventSink:      eventSink,
		Environment:    continueEnv,
		Attachments:    attachments,
		Transcript:     transcript,
		ResolvedConfig: run.ResolvedConfig,
		SandboxID:      run.SandboxID,
	}

	// Execute continuation with per-turn timeout
	result, err := r.Continue(execCtx, continueReq)
	if result != nil && result.Result != nil && run.ResolvedConfig != nil && o.structuredResults != nil {
		result.Result.Structured = o.structuredResults.Resolve(execCtx, run.ResolvedConfig.ResultSpec, result.Result)
	}

	// Stop the heartbeat loop before the terminal transition so its run writes
	// cannot race the transition's run writes (see stopHeartbeat above).
	stopHeartbeat()

	// Park coordination (durable park/resume): if the agent parked mid-turn,
	// ParkRunFromAgent transitioned the run running→parked and terminated this
	// process — which is why r.Continue returned. The park owns the lifecycle
	// from here; applying a terminal transition would clobber the park
	// (parked→failed is an allowed edge for waiter errors) and the continuation
	// checkpoint would tear down the sandbox the wake needs. Skip both.
	if o.isRunParked(ctx, run.ID) {
		obs.Component("continuation").Info("continuation ended on a parked run; leaving park intact",
			obs.KeyRunID, run.ID.String())
		return
	}

	now := time.Now()
	transition := RunStatusTransitionInput{
		Run:     run,
		EndedAt: &now,
		Reason:  "Continuation completed",
	}
	if result != nil {
		transition.Result = result.Result
		transition.Summary = result.Summary
	}

	if execCtx.Err() == context.DeadlineExceeded {
		// Continuation timed out — mark as failed but preserve session ID
		// so the user can continue again with a fresh timeout.
		transition.NewStatus = domain.RunStatusFailed
		transition.Phase = domain.RunPhaseCompleted
		transition.ErrorMsg = fmt.Sprintf("continuation exceeded timeout of %s", timeout)
		run.ErrorMsg = transition.ErrorMsg
		if result != nil && result.SessionID != "" {
			run.SessionID = result.SessionID
		}
		if o.events != nil {
			errorEvent := domain.NewErrorEvent(run.ID, "continuation_timeout", run.ErrorMsg, true)
			_ = o.appendAndBroadcastEvents(ctx, run.ID, errorEvent)
		}
	} else if err != nil {
		transition.NewStatus = domain.RunStatusFailed
		transition.Phase = domain.RunPhaseCompleted
		transition.ErrorMsg = err.Error()
		run.ErrorMsg = transition.ErrorMsg
		if o.events != nil {
			errorEvent := domain.NewErrorEvent(run.ID, "continuation_error", err.Error(), false)
			_ = o.appendAndBroadcastEvents(ctx, run.ID, errorEvent)
		}
	} else if result != nil && !result.Success {
		transition.NewStatus = domain.RunStatusFailed
		transition.Phase = domain.RunPhaseCompleted
		transition.ErrorMsg = result.ErrorMessage
		transition.ExitCode = &result.ExitCode
		transition.Result = result.Result
		transition.Summary = result.Summary
		run.ErrorMsg = transition.ErrorMsg
		if o.events != nil && result.ErrorMessage != "" {
			errorEvent := domain.NewErrorEvent(run.ID, "continuation_error", result.ErrorMessage, false)
			_ = o.appendAndBroadcastEvents(ctx, run.ID, errorEvent)
		}
	} else if result != nil {
		transition.NewStatus = domain.RunStatusComplete
		transition.Phase = domain.RunPhaseCompleted
		transition.Summary = result.Summary
		transition.Result = result.Result
	} else {
		transition.NewStatus = domain.RunStatusComplete
		transition.Phase = domain.RunPhaseCompleted
	}

	// Always preserve session ID from the result for further continuation,
	// regardless of success/failure. The runner populates SessionID from
	// stream events received before process termination.
	if result != nil && result.SessionID != "" {
		run.SessionID = result.SessionID
	}

	updatedRun, err := o.applyRunStatusTransition(ctx, transition)
	if err != nil {
		obs.Component("continuation").Error("continuation status transition failed",
			obs.KeyRunID, run.ID.String(),
			obs.KeyError, err.Error(),
		)
		return
	}
	run = updatedRun
	o.checkpointContinuationTurn(ctx, run, result, execCtx.Err() == context.DeadlineExceeded)
	// Completion-driven advance: a workflow continue-node run just reached
	// terminal (the parked path returned above, keeping its attempt open).
	o.nudgeWorkflowForRun(run.ID)
}

func (o *Orchestrator) checkpointContinuationTurn(ctx context.Context, run *domain.Run, result *runner.ExecuteResult, timedOut bool) {
	if run == nil || run.RunMode != domain.RunModeSandboxed || run.SandboxID == nil || o.sandbox == nil {
		return
	}
	outcome := domain.ContractRunOutcomeSuccess
	switch {
	case timedOut:
		outcome = domain.ContractRunOutcomeTimeout
	case run.Status == domain.RunStatusCancelled:
		outcome = domain.ContractRunOutcomeCancelled
	case run.Status == domain.RunStatusFailed:
		outcome = domain.ContractRunOutcomeFailure
	}
	cost := 0.0
	if result != nil {
		cost = result.Metrics.CostEstimateUSD
	}
	phases.ApplyAtRunEnd(ctx, phases.ApplyAtRunEndInput{
		Deps:      phases.Deps{Runs: o.runs, Events: o.events, Broadcaster: o.broadcaster, Levers: o.runLevers(), WorkspaceSandbox: o.workspaceSandbox},
		Run:       run,
		SandboxID: run.SandboxID,
		Sandbox:   o.sandbox,
		Outcome:   outcome,
		Cost:      cost,
	})
	if o.runs != nil {
		if err := o.runs.Update(ctx, run); err != nil {
			obs.Component("continuation").Error("continuation checkpoint status update failed",
				obs.KeyRunID, run.ID.String(),
				obs.KeyError, err.Error(),
			)
		}
	}
}

// executeRun handles the actual agent execution (runs in background).
// This delegates to RunExecutor for the actual work. `started` is the
// spawn dispatcher's slot-release callback, fired the moment the run
// reaches RunStatusRunning so the next queued run can begin its
// codex-bootstrap window.
func (o *Orchestrator) executeRun(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, prompt string, systemPrompt string, existingSandboxWorkDir string, attachments []runner.Attachment, customEnv map[string]string, started spawn.StartedFn) {
	// Interactive runs take the parallel execution path: agent-manager launches
	// the real interactive CLI in a web-console session and tails its transcript
	// to completion, instead of owning a codec stdout pipe. Selected by the run's
	// ExecutionMode (design §1).
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		o.executeInteractiveRun(ctx, run, task, interactiveInitialPrompt(systemPrompt, prompt), started)
		return
	}

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
	executor.WithStructuredResultResolver(o.structuredResults)
	// Apply orchestration-settings overrides to executor levers when a store
	// is wired. Defaults come from config.DefaultLevers().
	if levers, ok := o.executorLevers(); ok {
		executor.WithLevers(levers)
	}
	// Configure executor with checkpoint repository if available
	if o.checkpoints != nil {
		executor.WithCheckpointRepository(o.checkpoints)
	}
	if o.healthStore != nil {
		executor.WithModelHealthReporter(newHealthMarkerAdapter(o.healthStore, run.ID.String()))
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
	if o.workspaceSandbox != nil {
		executor.WithWorkspaceSandboxEnsurer(o.workspaceSandbox)
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
	executor.WithOnRunning(started)
	executor.Execute(ctx)
	// Completion-driven advance: a workflow run just reached terminal (a parked
	// run keeps its attempt open — the wake leg nudges when it later completes).
	if !executor.parked {
		o.nudgeWorkflowForRun(run.ID)
	}
}

// interactiveInitialPrompt reconstructs the single-channel prompt to type into
// an interactive CLI. The codec-pipe path splits a task into a system prompt
// (instructions) and a user message (context + question); an interactive CLI
// launched raw has no separate system channel, so both halves are recombined
// into one prompt. When there is no system prompt (the common no-attachment
// case) the user message already carries the full task.
func interactiveInitialPrompt(systemPrompt, userMessage string) string {
	systemPrompt = strings.TrimSpace(systemPrompt)
	userMessage = strings.TrimSpace(userMessage)
	switch {
	case systemPrompt == "":
		return userMessage
	case userMessage == "":
		return systemPrompt
	default:
		return systemPrompt + "\n\n" + userMessage
	}
}

// executeInteractiveRun drives an interactive run to completion via the
// interactive.Coordinator: it launches the real interactive CLI in a web-console
// session, tails the agent-owned transcript, and finalizes the run on the
// terminal marker (design §1). It is the parallel path to the codec-pipe
// RunExecutor and leaves the Continue/Stop seam (Substrate.Stop, SendText) for
// Phase 5.
func (o *Orchestrator) executeInteractiveRun(ctx context.Context, run *domain.Run, task *domain.Task, initialPrompt string, started spawn.StartedFn) {
	// The spawn slot must be released even if we fail before reaching Running.
	releaseOnce := sync.Once{}
	release := func() {
		if started != nil {
			releaseOnce.Do(started)
		}
	}
	defer release()

	if o.interactiveSessions == nil {
		o.failInteractiveRun(ctx, run, "interactive execution mode is not configured (no web-console session controller wired)")
		return
	}
	if run.ResolvedConfig == nil {
		o.failInteractiveRun(ctx, run, "interactive run has no resolved config")
		return
	}
	if !interactive.SupportsInteractive(run.ResolvedConfig.RunnerType) {
		o.failInteractiveRun(ctx, run, fmt.Sprintf("runner %q is not supported in interactive mode", run.ResolvedConfig.RunnerType))
		return
	}
	// Policy-gate backstop (locked decision 5): interactive mode is never allowed
	// for protected (sandboxed) runs. CreateRun rejects this at validation time;
	// this defends the execution path against any run that reached here mislabeled.
	if err := domain.ValidateInteractiveRunMode(run.ExecutionMode, run.RunMode); err != nil {
		o.failInteractiveRun(ctx, run, err.Error())
		return
	}

	workDir, err := phases.UseInPlaceWorkspace(task)
	if err != nil {
		o.failInteractiveRun(ctx, run, fmt.Sprintf("resolve interactive working directory: %v", err))
		return
	}

	coord := interactive.NewCoordinator(interactive.CoordinatorDeps{
		Substrate:   interactive.NewSubstrate(o.interactiveSessions, interactive.RegistryLaunchInfo(o.runners)),
		Tailer:      interactive.NewTailer(interactive.RegistryParser(o.runners)),
		Sessions:    o.interactiveSessions,
		Runs:        o.runs,
		Broadcaster: o.broadcaster,
		NewSink:     o.interactiveEventSink,
		Result:      o.persistedResultBuilder,
	})

	// Register the live coordinator so StopRun can cancel it deterministically and
	// wait for it to exit before finalizing (no late tail Update can resurrect a
	// stopped run). The context is cancellable via the registry; unregister +
	// signal done when Execute returns.
	runCtx, driver := o.interactiveDrivers.register(ctx, run.ID)
	defer o.interactiveDrivers.finish(run.ID, driver)

	if err := coord.Execute(runCtx, run, interactive.LaunchParams{
		RunID:        run.ID,
		RunnerType:   run.ResolvedConfig.RunnerType,
		Tag:          run.GetTag(),
		WorkingDir:   workDir,
		RunDir:       runstate.RunDir("", run.ID),
		DisplayLabel: run.GetTag(),
		Prompt:       initialPrompt,
	}, release); err != nil {
		obs.Component("interactive").Warn("interactive run finalize failed",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

// interactiveEventSink builds the per-run event sink interactive tail events are
// emitted into, mirroring the codec-pipe executor's sink selection.
func (o *Orchestrator) interactiveEventSink(runID uuid.UUID) runner.EventSink {
	switch {
	case o.events != nil && o.broadcaster != nil:
		return &broadcastingEventSink{store: o.events, runID: runID, broadcaster: o.broadcaster}
	case o.events != nil:
		return &eventStoreAdapter{store: o.events, runID: runID}
	default:
		return &noOpEventSink{}
	}
}

// failInteractiveRun marks an interactive run failed with an explicit reason
// (used for pre-launch misconfiguration/validation failures).
func (o *Orchestrator) failInteractiveRun(ctx context.Context, run *domain.Run, reason string) {
	now := time.Now()
	run.Status = domain.RunStatusFailed
	run.Phase = domain.RunPhaseCompleted
	run.ErrorMsg = reason
	run.EndedAt = &now
	run.UpdatedAt = now
	if err := o.runs.Update(ctx, run); err != nil {
		obs.Component("interactive").Warn("failed to persist interactive run failure",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(run)
	}
}

// executorLevers folds the runtime OrchestrationSettings store (when wired)
// onto the static config.DefaultLevers() to produce the per-run lever set
// the executor reads. Returns ok=false when no override store is configured —
// callers fall through to the executor's built-in defaults.
//
// Only fields that the OrchestrationSettings model actually exposes are
// overridden. Other levers (recovery polls, scanner buffers, diagnostics)
// stay at compile-time defaults until they are surfaced as runtime knobs.
func (o *Orchestrator) executorLevers() (agentconfig.Levers, bool) {
	if o.orchestrationSettings == nil {
		return agentconfig.Levers{}, false
	}
	return o.runLevers(), true
}

func (o *Orchestrator) runLevers() agentconfig.Levers {
	levers := agentconfig.DefaultLevers()
	if o.orchestrationSettings == nil {
		return levers
	}
	s := o.orchestrationSettings.Get()
	if s.RunExecution.RunTimeoutMinutes > 0 {
		levers.Execution.DefaultTimeout = time.Duration(s.RunExecution.RunTimeoutMinutes) * time.Minute
	}
	if s.HealthDetection.HeartbeatIntervalSeconds > 0 {
		levers.Heartbeat.RunHeartbeatInterval = time.Duration(s.HealthDetection.HeartbeatIntervalSeconds) * time.Second
	}
	if s.HealthDetection.StaleThresholdSeconds > 0 {
		levers.Heartbeat.StaleThreshold = time.Duration(s.HealthDetection.StaleThresholdSeconds) * time.Second
	}
	return levers
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

	// Resume goes through the same dispatcher as initial spawn —
	// per contract decision 2 in SEAMS.md, no goroutine spawn outside
	// spawn.Dispatcher.Enqueue.
	if err := o.dispatcher.Enqueue(&spawn.Job{
		RunID:      run.ID,
		RunMode:    run.RunMode,
		RunnerType: runnerTypeOrEmpty(run),
		Sink:       o.dispatcherSink(run.ID),
		Fn: func(started spawn.StartedFn) {
			o.resumeRun(context.Background(), run, task, profile, checkpoint, started)
		},
	}); err != nil {
		return nil, err
	}

	return o.attachRunActions(ctx, run), nil
}

// resumeRun handles the actual agent resumption (runs in background).
// `started` is the spawn dispatcher's slot-release callback.
func (o *Orchestrator) resumeRun(ctx context.Context, run *domain.Run, task *domain.Task, profile *domain.AgentProfile, checkpoint *domain.RunCheckpoint, started spawn.StartedFn) {
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
	executor.WithStructuredResultResolver(o.structuredResults)
	// Apply orchestration-settings overrides to executor levers when a store
	// is wired. Defaults come from config.DefaultLevers().
	if levers, ok := o.executorLevers(); ok {
		executor.WithLevers(levers)
	}

	// Configure for resumption
	if o.checkpoints != nil {
		executor.WithCheckpointRepository(o.checkpoints)
	}
	// Configure executor with broadcaster for real-time WebSocket updates
	if o.broadcaster != nil {
		executor.WithBroadcaster(o.broadcaster)
	}
	if o.workspaceSandbox != nil {
		executor.WithWorkspaceSandboxEnsurer(o.workspaceSandbox)
	}
	if len(o.identitySecret) > 0 {
		executor.WithIdentitySecret(o.identitySecret)
	}
	executor.WithResumeFrom(checkpoint)
	executor.WithOnRunning(started)

	executor.Execute(ctx)
	// Completion-driven advance for a workflow run recovered/resumed after a
	// restart between run-terminal and the original nudge.
	if !executor.parked {
		o.nudgeWorkflowForRun(run.ID)
	}
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
// Model Policy Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) GetModelHealthSnapshot(ctx context.Context) (health.Snapshot, error) {
	if o.healthStore == nil {
		return health.Snapshot{
			Models:  map[string]map[string]health.ModelEntry{},
			Runners: map[string]health.RunnerEntry{},
		}, nil
	}
	return o.healthStore.Snapshot(ctx)
}

// ExplainProfilePolicy resolves the profile against the active catalog without
// creating a run. The returned snapshot includes the same availability
// preflight and precedence explanation run creation would persist.
func (o *Orchestrator) ExplainProfilePolicy(ctx context.Context, profileID uuid.UUID) (*domain.ExecutionPolicySnapshot, error) {
	cfg, _, err := o.resolveRunConfig(ctx, CreateRunRequest{AgentProfileID: &profileID})
	if err != nil {
		return nil, err
	}
	if cfg == nil || cfg.PolicySnapshot == nil {
		return nil, domain.NewValidationError("rolePolicyCatalog", "profile resolution produced no policy snapshot")
	}
	return cfg.PolicySnapshot, nil
}

// ExplainRunPolicy returns only the immutable snapshot stored with the run. It
// never reconstructs historical provenance from the current catalog.
func (o *Orchestrator) ExplainRunPolicy(ctx context.Context, runID uuid.UUID) (*domain.ExecutionPolicySnapshot, error) {
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	if run.ResolvedConfig == nil || run.ResolvedConfig.PolicySnapshot == nil {
		return nil, nil
	}
	return run.ResolvedConfig.PolicySnapshot, nil
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

	// A valid signature alone is insufficient. The token must still be the
	// active token recorded for the same run and must not have been revoked at
	// run completion. This turns a signed, time-limited bearer token into a
	// live run credential rather than a 24-hour post-completion capability.
	tokenHash := identity.HashToken(token)
	run, err := o.runs.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return &IdentityVerifyResult{Valid: false, Error: "identity token is not active"}, nil
	}
	if run.ID != claims.RunID {
		return &IdentityVerifyResult{Valid: false, Error: "identity token does not match its active run"}, nil
	}
	if run.IdentityTokenRevokedAt != nil {
		return &IdentityVerifyResult{Valid: false, Error: "identity token has been revoked"}, nil
	}

	return &IdentityVerifyResult{
		Valid:     true,
		Claims:    claims,
		RunStatus: run.Status,
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
	if o.profiles != nil && o.workflows != nil && o.tasks != nil && o.runs != nil {
		status.Dependencies.Database = &DependencyStatus{Connected: true, Storage: o.storageLabel}
	} else {
		msg := "core or workflow repository not configured"
		status.Dependencies.Database = &DependencyStatus{
			Connected: false,
			Error:     &msg,
			Storage:   o.storageLabel,
		}
	}
	if o.workflows != nil && o.workflowExecutions != nil && o.workflowEngine != nil {
		status.Dependencies.WorkflowRuntime = &DependencyStatus{Connected: true, Storage: o.storageLabel}
	} else {
		msg := "workflow catalog, execution repository, or interpreter is not configured"
		status.Dependencies.WorkflowRuntime = &DependencyStatus{Connected: false, Error: &msg, Storage: o.storageLabel}
		status.Readiness = false
		status.Status = "degraded"
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
	case domain.RunnerTypeGrok:
		cmdName = "grok"
		// Headless single-turn; plain output is enough for a one-word probe
		// and one turn cannot reach a tool that would need approval.
		probeCmd = exec.CommandContext(probeCtx, cmdName, "-p", probePrompt, "--output-format", "plain", "--max-turns", "1")
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
//
// The canonical definition lives in the phases package so per-phase
// functions can reference it without an import cycle. The alias here keeps
// existing orchestration call sites compiling without per-site rewrites.
type EventBroadcaster = phases.EventBroadcaster

func appendAndBroadcastEvents(ctx context.Context, store event.Store, broadcaster EventBroadcaster, runID uuid.UUID, events ...*domain.RunEvent) error {
	persistable := make([]*domain.RunEvent, 0, len(events))
	for _, evt := range events {
		if evt != nil {
			persistable = append(persistable, evt)
		}
	}
	if len(persistable) == 0 {
		return nil
	}
	if store == nil {
		return fmt.Errorf("event store is required before broadcasting run events")
	}
	if err := store.Append(ctx, runID, persistable...); err != nil {
		return err
	}
	if broadcaster != nil {
		for _, evt := range persistable {
			broadcaster.BroadcastEvent(evt)
		}
	}
	return nil
}

func (o *Orchestrator) appendAndBroadcastEvents(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error {
	return appendAndBroadcastEvents(ctx, o.events, o.broadcaster, runID, events...)
}

// eventStoreAdapter adapts event.Store to runner.EventSink
type eventStoreAdapter struct {
	store        event.Store
	runID        uuid.UUID
	lastSequence int64
}

func (e *eventStoreAdapter) Emit(evt *domain.RunEvent) error {
	if err := e.store.Append(context.Background(), e.runID, evt); err != nil {
		return err
	}
	e.lastSequence = evt.Sequence
	return nil
}

func (e *eventStoreAdapter) Close() error {
	return nil
}

func (e *eventStoreAdapter) LastSequence() int64 {
	return e.lastSequence
}

// broadcastingEventSink stores events AND broadcasts them via WebSocket.
type broadcastingEventSink struct {
	store        event.Store
	runID        uuid.UUID
	broadcaster  EventBroadcaster
	lastSequence int64
}

func (b *broadcastingEventSink) Emit(evt *domain.RunEvent) error {
	// Validate event and log warnings for missing data
	domain.ValidateEvent(evt)

	if err := appendAndBroadcastEvents(context.Background(), b.store, b.broadcaster, b.runID, evt); err != nil {
		obs.Component("broadcast-sink").Warn("event store append failed",
			obs.KeyRunID, b.runID.String(),
			obs.KeyError, err.Error(),
		)
		return err
	}
	b.lastSequence = evt.Sequence

	if b.broadcaster != nil {
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

func (b *broadcastingEventSink) LastSequence() int64 {
	return b.lastSequence
}

func (o *Orchestrator) runEventSink(runID uuid.UUID) runner.EventSink {
	switch {
	case o.events != nil && o.broadcaster != nil:
		return &broadcastingEventSink{
			store:       o.events,
			runID:       runID,
			broadcaster: o.broadcaster,
		}
	case o.events != nil:
		return &eventStoreAdapter{store: o.events, runID: runID}
	default:
		return &noOpEventSink{}
	}
}

// runnerTypeOrEmpty returns the runner type from a run's resolved
// config, or "" when no resolved config is set yet (e.g. during
// pre-spawn validation). Used for lifecycle event tagging.
func runnerTypeOrEmpty(run *domain.Run) domain.RunnerType {
	if run == nil || run.ResolvedConfig == nil {
		return ""
	}
	return run.ResolvedConfig.RunnerType
}

// dispatcherSink returns an obs.Sink for emitting lifecycle events
// (spawn-enqueued, spawn-started) from the spawn dispatcher path. It
// uses the same store + broadcaster as the per-run gate, so the
// timeline shows a continuous lifecycle from "queued" through "exited"
// regardless of where in the run-executor stack the event originated.
//
// Returned sink is non-nil even when the orchestrator has no event
// store wired (defaults to the noOp sink so dispatcher.Enqueue still
// emits its log line).
func (o *Orchestrator) dispatcherSink(runID uuid.UUID) obs.Sink {
	return o.runEventSink(runID)
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
