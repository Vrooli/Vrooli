// Responsibility: define orchestration request, result and dependency contracts.
package orchestration

import (
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/invocationreadmodel"
	"encoding/json"
	"github.com/google/uuid"
	eventpb "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-events/v1/domain"
	"time"
)

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
	ParentRunID               *uuid.UUID // Direct child runs of one parent run
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
	// Attributable workload declarations only; never authorization claims.
	WorkReferences []*eventpb.WorkReference `json:"workReferences,omitempty"`

	// OwnerToken is supplied only by the HTTP boundary and is never persisted.
	// The verified projection is stored on the run so later token minting does
	// not depend on the authenticator being reachable.
	// RunIdentityToken is transported separately from the human credential.
	RunIdentityToken string `json:"-"`
	EffortProof      string `json:"-"`
	effort           *effortAdmission
	// caller and profileAdmission are derived only inside public admission.
	caller           *domain.CreateRunCaller
	profileAdmission *profileAdmissionPlan
	OwnerToken       string   `json:"-"`
	OwnerSubject     string   `json:"-"`
	OwnerScopes      []string `json:"-"`
	// RequestedScopes is an optional caller narrowing. It is never a source of
	// authority: minting intersects it with the verified owner ceiling.
	RequestedScopes      []string   `json:"scopes,omitempty"`
	ExpectedOwnerSubject string     `json:"-"`
	OwnerExpiresAt       *time.Time `json:"-"`

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
	MaxToolCalls         *int                    `json:"maxToolCalls,omitempty"`
	Timeout              *time.Duration          `json:"timeout,omitempty"`
	Model                *string                 `json:"model,omitempty"`
	PreferredRunner      string                  `json:"preferredRunner,omitempty"`
	Effort               *domain.Effort          `json:"effort,omitempty"`
	AllowedTools         []string                `json:"allowedTools,omitempty"`
	DeniedTools          []string                `json:"deniedTools,omitempty"`
	SkipPermissionPrompt *bool                   `json:"skipPermissionPrompt,omitempty"`
	EnableBrowser        *bool                   `json:"enableBrowser,omitempty"`
	ExtraFlags           domain.RunnerExtraFlags `json:"extraFlags,omitempty"`
	NetworkAccess        *domain.NetworkAccess   `json:"networkAccess,omitempty"`
	AllowedPaths         []string                `json:"allowedPaths,omitempty"`
	DeniedPaths          []string                `json:"deniedPaths,omitempty"`
	// AllowedEffects is inherited from an owner-issued workflow grant. It is
	// validated before a run is persisted and cannot be widened by a child.
	AllowedEffects           []string           `json:"allowedEffects,omitempty"`
	RequireEffectContainment bool               `json:"requireEffectContainment,omitempty"`
	ResultSpec               *domain.ResultSpec `json:"resultSpec,omitempty"`
	Until                    string             `json:"until,omitempty"`

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
	// Interactive mode is selected by the declaration/capability resolver.
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

// AttachRunRequest identifies a harness session that agent-manager did not
// spawn. It intentionally carries only identity/liveness metadata; no
// transcript path or live-output stream is accepted.
type AttachRunRequest struct {
	TaskID         *uuid.UUID
	HarnessKind    string
	HarnessSession string
	ProcessID      int
	HarnessTitle   string
}

// AttachRunResult contains the attached run and the plaintext token that is
// placed in the child environment by the launcher. The token is never stored
// in the database; only its hash is persisted on the run.
type AttachRunResult struct {
	Run       *domain.Run
	Token     string
	ExpiresAt time.Time
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
	// EngagementGrant is an immutable owner-issued aggregate allowance. Agent
	// Manager persists it with the execution and never widens catalog budgets.
	EngagementGrant      *domain.WorkflowEngagementGrant `json:"engagementGrant,omitempty"`
	ApprovalDigest       string                          `json:"approvalDigest,omitempty"`
	GrantDigest          string                          `json:"grantDigest,omitempty"`
	ExecutionPreferences *domain.ExecutionPreferences    `json:"executionPreferences,omitempty"`
}

type ListWorkflowExecutionsRequest struct {
	Owner          string
	WorkflowKey    string
	IdempotencyKey string
	Status         domain.WorkflowExecutionStatus
	Limit          int
	Offset         int
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
	MaxToolCalls   *int               `json:"maxToolCalls,omitempty"`
	Timeout        *time.Duration     `json:"timeout,omitempty"`
	ResultSpec     *domain.ResultSpec `json:"resultSpec,omitempty"`
	// ReinstallGoal re-sends the harness-native goal before the follow-up
	// message. Set by a resume that re-adopts an interrupted interactive run.
	ReinstallGoal bool `json:"reinstallGoal,omitempty"`
}

// ResumeFromFailedRunRequest contains parameters for creating a new run that
// inherits the original task + profile of a failed/cancelled run and is
// seeded with that run's transcript and diff so the agent can complete the
// remaining work instead of starting over.
type ResumeFromFailedRunRequest struct {
	EffortProof   string    `json:"-"`
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
