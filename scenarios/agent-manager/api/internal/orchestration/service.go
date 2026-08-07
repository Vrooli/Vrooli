// This file defines the orchestration service composition and shared dependencies.
package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"agent-manager/internal/adapters/artifact"
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/durability"
	"agent-manager/internal/findings"
	"agent-manager/internal/health"
	"agent-manager/internal/identity"
	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/orchestration/phases"
	"agent-manager/internal/orchestration/spawn"
	"agent-manager/internal/policy"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/runreport"
	"agent-manager/internal/runstate"
	"agent-manager/internal/storage"
	"agent-manager/internal/structuredresult"
	"agent-manager/internal/workflowruntime"

	agentconfig "agent-manager/internal/config"

	"github.com/google/uuid"
)

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
	Tag              string              `json:"tag,omitempty"`
	WorkloadKind     domain.WorkloadKind `json:"workloadKind,omitempty"`
	WorkloadKey      string              `json:"workloadKey,omitempty"`
	WorkloadInstance string              `json:"workloadInstance,omitempty"`

	// Investigation lineage metadata.
	SourceRunIDs             []uuid.UUID `json:"sourceRunIds,omitempty"`
	SourceInvestigationRunID *uuid.UUID  `json:"sourceInvestigationRunId,omitempty"`

	// Inline config (optional - used if no profile, or overrides profile)
	RoleRef              *string                 `json:"roleRef,omitempty"`
	MaxTurns             *int                    `json:"maxTurns,omitempty"`
	Timeout              *time.Duration          `json:"timeout,omitempty"`
	Model                *string                 `json:"model,omitempty"`
	Effort               *domain.Effort          `json:"effort,omitempty"`
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
	ProfileKey  string                      `json:"profileKey,omitempty"`
	SourcePath  string                      `json:"sourcePath,omitempty"`
	SourceHash  string                      `json:"sourceHash,omitempty"`
	ProfileID   string                      `json:"profileId,omitempty"`
	Status      ProfileReconcileStatus      `json:"status"`
	Message     string                      `json:"message,omitempty"`
	Diagnostics []domain.WorkflowDiagnostic `json:"diagnostics,omitempty"`
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
	// Initiator and IdentityToken are server-boundary facts supplied by the
	// handler. StartWorkflowExecution remains the only policy authority.
	Initiator     domain.WorkflowInitiator `json:"-"`
	IdentityToken string                   `json:"-"`
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
	// Selection records a reproducible cohort or goal predicate resolved by the
	// API. It is carried into the workflow context so truncation is visible to
	// the investigator instead of silently turning into a partial cohort.
	Selection *InvestigationSelection `json:"selection,omitempty"`
}

// InvestigationSelection describes a non-manual investigation scope after it
// has been resolved to run IDs. Explicit run IDs leave it nil.
type InvestigationSelection struct {
	Kind        string                     `json:"kind"`
	Filter      invocationreadmodel.Filter `json:"filter"`
	MatchedRuns int                        `json:"matchedRuns"`
	DroppedRuns int                        `json:"droppedRuns"`
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
	runStateRoot     string
	runStateResolver runstate.RootResolver

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

	// clock is the wall-clock seam for orchestration state transitions. It is
	// injected by deterministic tests and defaults to time.Now in production.
	clock func() time.Time

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

	// receipts is an optional, read-only Vrooli Events seam. It must never
	// influence run execution or terminal state; reports surface its status.
	receipts            ReceiptSummaryReader
	findings            findings.Repository
	receiptEvidence     runreport.ReceiptJoinStore
	investigationLedger runreport.LedgerStore
	invocationReadModel invocationreadmodel.Store
	durabilityEvidence  DurabilityEvidenceReader
	durabilityBoundary  durability.BoundaryStore

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
	labelGenerator    structuredresult.Extractor
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

// SetDurabilityEvidenceReader installs the swarm-owned read seam. It is a
// setter because scenario discovery is resolved after the orchestrator is
// constructed. A missing reader is represented as unlinked evidence.
func (o *Orchestrator) SetDurabilityEvidenceReader(reader DurabilityEvidenceReader) {
	o.durabilityEvidence = reader
}

// WithDurabilityBoundary installs the durable analysis-epoch store. Without it
// Durability refuses to grade rather than inventing an epoch.
func WithDurabilityBoundary(store durability.BoundaryStore) Option {
	return func(o *Orchestrator) {
		o.durabilityBoundary = store
	}
}

// Durability projects a run's durable friction and optional swarm evidence.
// It deliberately never reads Run.Result, Run.Status, or any other
// self-authored completion field.
func (o *Orchestrator) Durability(ctx context.Context, id uuid.UUID) (durability.Verdict, error) {
	run, err := o.GetRun(ctx, id)
	if err != nil {
		return durability.Verdict{}, err
	}
	started := time.Time{}
	if run.StartedAt != nil {
		started = run.StartedAt.UTC()
	} else if run.ImportedAt != nil {
		started = run.ImportedAt.UTC()
	}
	lane := durability.LaneUnlinked
	if run.IdentityTokenHash != "" {
		lane = durability.LaneVerified
	} else if run.ImportSourceSessionID != "" {
		lane = durability.LaneObserved
	}
	evidence := make([]durability.Evidence, 0)
	episodes, err := o.Episodes(ctx, id)
	if err != nil {
		return durability.Verdict{}, err
	}
	for _, episode := range episodes {
		evidence = append(evidence, durability.Evidence{Kind: "friction", Reference: "agent-manager://runs/" + id.String() + "/episodes/" + episode.EpisodeID, At: started, Lane: lane})
	}
	if o.durabilityEvidence != nil {
		observed, readErr := o.durabilityEvidence.ReadDurabilityEvidence(ctx, run)
		if readErr != nil {
			evidence = append(evidence, durability.Evidence{Kind: "swarm-evidence-unavailable", Reference: "swarm-manager://durability/evidence", At: started, Lane: durability.LaneUnlinked, Degraded: true})
		} else {
			evidence = append(evidence, observed...)
		}
	}
	boundary, err := durability.ResolveBoundary(ctx, o.durabilityBoundary, systemNow())
	if err != nil {
		return durability.Verdict{}, err
	}
	return durability.Project(boundary, durability.Work{ID: id.String(), Subject: append([]string(nil), run.Subject...), StartedAt: started, Lane: lane}, evidence), nil
}

// systemNow is the production clock behind injected orchestration clocks.
var systemNow = time.Now

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

// ReceiptSummaryReader provides only the bounded receipt discriminator needed
// by RunReport. Receipt payloads remain on the dedicated inspection endpoint.
type ReceiptSummaryReader interface {
	ReadReceiptSummary(context.Context, uuid.UUID) (runreport.ReceiptSummary, error)
}

type ReceiptSummaryReaderFunc func(context.Context, uuid.UUID) (runreport.ReceiptSummary, error)

func (f ReceiptSummaryReaderFunc) ReadReceiptSummary(ctx context.Context, id uuid.UUID) (runreport.ReceiptSummary, error) {
	return f(ctx, id)
}

// WithConfig sets the configuration.
func WithConfig(cfg OrchestratorConfig) Option {
	return func(o *Orchestrator) {
		o.config = cfg
	}
}

// WithClock injects the wall clock used for durable orchestration timestamps.
// A nil clock retains the production default.
func WithClock(clock func() time.Time) Option {
	return func(o *Orchestrator) {
		if clock != nil {
			o.clock = clock
		}
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

func WithReceiptSummaryReader(reader ReceiptSummaryReader) Option {
	return func(o *Orchestrator) { o.receipts = reader }
}

func WithFindings(repo findings.Repository) Option {
	return func(o *Orchestrator) { o.findings = repo }
}

func WithReceiptEvidenceStore(store runreport.ReceiptJoinStore) Option {
	return func(o *Orchestrator) { o.receiptEvidence = store }
}

func WithInvestigationLedgerStore(store runreport.LedgerStore) Option {
	return func(o *Orchestrator) { o.investigationLedger = store }
}

// WithInvocationReadModel wires the independent durable analytics projection.
// It is deliberately distinct from the legacy report cache.
func WithInvocationReadModel(store invocationreadmodel.Store) Option {
	return func(o *Orchestrator) { o.invocationReadModel = store }
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

// WithLabelGenerator wires the constrained ai-gateway seam used only when an
// imported transcript has neither a harness title nor a user prompt. It is
// deliberately separate from structured result projection so title generation
// cannot affect run outcomes.
func WithLabelGenerator(generator structuredresult.Extractor) Option {
	return func(o *Orchestrator) {
		o.labelGenerator = generator
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

// WithRunStateRoot supplies the explicit root for durable per-run artifacts.
func WithRunStateRoot(root string) Option {
	return func(o *Orchestrator) { o.runStateRoot = root }
}

// WithRunStateRootResolver selects a state root from the operation context.
// Production wiring uses this so HTTP test-mode leases survive dispatch.
func WithRunStateRootResolver(resolver runstate.RootResolver) Option {
	return func(o *Orchestrator) { o.runStateResolver = resolver }
}

func (o *Orchestrator) resolveRunStateRoot(ctx context.Context) (string, error) {
	if o.runStateResolver != nil {
		return o.runStateResolver.Resolve(ctx)
	}
	if o.runStateRoot == "" {
		return "", fmt.Errorf("run state root is required")
	}
	return o.runStateRoot, nil
}

func (o *Orchestrator) recordRunStateWrite(ctx context.Context) {
	if o.runStateResolver != nil {
		o.runStateResolver.RecordWrite(ctx)
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
		clock:              time.Now,
		interactiveDrivers: newInteractiveDriverRegistry(),
		structuredResults:  structuredresult.Resolver{},
		workflowWaiters:    newWorkflowWaitRegistry(),
	}

	for _, opt := range opts {
		opt(o)
	}
	if o.workflowExecutions != nil && o.workflows != nil {
		expressions, _ := workflowruntime.NewExpressionEvaluator()
		o.workflowEngine = &workflowruntime.Engine{Store: o.workflowExecutions, Catalog: o.workflows, Children: workflowChildLauncher{o: o}, Subworkflows: workflowSubworkflowLauncher{o: o}, Expressions: expressions, Now: o.now}
		if source, ok := o.promptClient.(promptmanager.AssignmentClient); ok && source != nil {
			o.workflowEngine.PromptResolver = workflowPromptResolver{source: source}
		}
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

func (o *Orchestrator) now() time.Time {
	if o != nil && o.clock != nil {
		return o.clock()
	}
	return systemNow()
}

// -----------------------------------------------------------------------------
// AgentProfile Operations
// -----------------------------------------------------------------------------

func (o *Orchestrator) CreateProfile(ctx context.Context, profile *domain.AgentProfile) (*domain.AgentProfile, error) {
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}
	profile.CreatedAt = o.now()
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
	profile.UpdatedAt = o.now()
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

	now := o.now()
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
	task.CreatedAt = o.now()
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
	updated.UpdatedAt = o.now()

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
	task.UpdatedAt = o.now()
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
