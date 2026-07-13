// Package domain defines the core domain entities for agent-manager.
//
// This package contains the central concepts that agent-manager operates on:
// - AgentProfile: defines HOW an agent runs (portable role, permissions)
// - Task: defines WHAT needs to be done (scope, context, requirements)
// - Run: a concrete execution linking Task to AgentProfile within a sandbox
// - RunEvent: append-only event stream capturing all agent activity
// - Policy: rules governing execution, approval, and resource access
package domain

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// AgentProfile - Defines HOW an agent runs
// -----------------------------------------------------------------------------

// AgentProfile defines the configuration for running an agent.
// This is a reusable definition that can be applied to many tasks.
type AgentProfile struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	ProfileKey  string    `json:"profileKey" db:"profile_key"`
	Description string    `json:"description,omitempty" db:"description"`

	// RoleRef is portable desired intent. Concrete runner/model selections are
	// captured only in a run's immutable PolicySnapshot.
	RoleRef  string        `json:"roleRef,omitempty" db:"role_ref"`
	MaxTurns int           `json:"maxTurns,omitempty" db:"max_turns"`
	Timeout  time.Duration `json:"timeout,omitempty" db:"timeout_ms"`

	// Tool permissions
	AllowedTools []string `json:"allowedTools,omitempty" db:"allowed_tools"`
	DeniedTools  []string `json:"deniedTools,omitempty" db:"denied_tools"`

	// Execution flags
	SkipPermissionPrompt bool `json:"skipPermissionPrompt,omitempty" db:"skip_permission_prompt"`

	// Feature flags (typed, discoverable capabilities)
	Features FeatureFlags `json:"features,omitempty" db:"features"`

	// Extra CLI flags per runner type (validated escape hatch)
	ExtraFlags RunnerExtraFlags `json:"extraFlags,omitempty" db:"extra_flags"`

	// Default policies (can be overridden per task)
	NetworkAccess NetworkAccess `json:"networkAccess" db:"network_access"`

	// Sandbox behavior settings
	SandboxConfig *SandboxConfig `json:"sandboxConfig,omitempty" db:"sandbox_config"`

	// Path restrictions
	AllowedPaths []string `json:"allowedPaths,omitempty" db:"allowed_paths"`
	DeniedPaths  []string `json:"deniedPaths,omitempty" db:"denied_paths"`

	// Metadata
	CreatedBy       string    `json:"createdBy,omitempty" db:"created_by"`
	OwnerScenario   string    `json:"ownerScenario,omitempty" db:"owner_scenario"`
	SourcePath      string    `json:"sourcePath,omitempty" db:"source_path"`
	SourceHash      string    `json:"sourceHash,omitempty" db:"source_hash"`
	LastAppliedHash string    `json:"lastAppliedHash,omitempty" db:"last_applied_hash"`
	SourceUpdatedAt time.Time `json:"sourceUpdatedAt,omitempty" db:"source_updated_at"`
	LocalOverride   bool      `json:"localOverride,omitempty" db:"local_override"`
	CreatedAt       time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt       time.Time `json:"updatedAt" db:"updated_at"`
}

// RunnerType identifies which agent runner to use.
type RunnerType string

const (
	RunnerTypeClaudeCode RunnerType = "claude-code"
	RunnerTypeCodex      RunnerType = "codex"
	RunnerTypeOpenCode   RunnerType = "opencode"
	RunnerTypeGrok       RunnerType = "grok"
)

// ValidRunnerTypes returns all valid runner types.
func ValidRunnerTypes() []RunnerType {
	return []RunnerType{
		RunnerTypeClaudeCode,
		RunnerTypeCodex,
		RunnerTypeOpenCode,
		RunnerTypeGrok,
	}
}

// IsValid checks if the runner type is valid.
func (r RunnerType) IsValid() bool {
	for _, valid := range ValidRunnerTypes() {
		if r == valid {
			return true
		}
	}
	return false
}

// ModelSelectionType makes runner-default selection explicit in persisted run
// snapshots. Empty model strings are not sentinels in this contract.
type ModelSelectionType string

const (
	ModelSelectionTypeModel         ModelSelectionType = "model"
	ModelSelectionTypeRunnerDefault ModelSelectionType = "runner_default"
)

// ExecutionCandidate is one immutable runner/model attempt in resolved order.
type ExecutionCandidate struct {
	RunnerType    RunnerType            `json:"runnerType"`
	SelectionType ModelSelectionType    `json:"selectionType"`
	Model         string                `json:"model,omitempty"`
	ResourceRole  string                `json:"resourceRole,omitempty"`
	Fallbacks     []string              `json:"fallbacks,omitempty"`
	Available     bool                  `json:"available"`
	FailureCode   string                `json:"failureCode,omitempty"`
	Failure       string                `json:"failure,omitempty"`
	Provenance    ResourceProvenance    `json:"provenance,omitempty"`
	Enforcement   PermissionEnforcement `json:"enforcement,omitempty"`
	PolicyPath    string                `json:"policyPath,omitempty"`
	PolicyDigest  string                `json:"policyDigest,omitempty"`
}

// ResourceProvenance pins where a resource-owned role decision came from.
type ResourceProvenance struct {
	Source     string `json:"source,omitempty"`
	ObservedAt string `json:"observedAt,omitempty"`
}

// PermissionEnforcement reports the resource's actual enforcement posture.
type PermissionEnforcement struct {
	Permissions string   `json:"permissions,omitempty"`
	Caveats     []string `json:"caveats,omitempty"`
}

// CandidatePreflight records the creation-time availability evidence used to
// select the initial candidate. Runtime failures remain separate attempt events.
type CandidatePreflight struct {
	Index     int                `json:"index"`
	Candidate ExecutionCandidate `json:"candidate"`
	Available bool               `json:"available"`
	Reason    string             `json:"reason,omitempty"`
}

// PolicyResolutionExplanation records why a run received its candidate
// sequence. It is persisted with the run so operators never need to reconstruct
// precedence from the current profile or catalog.
type PolicyResolutionExplanation struct {
	Source           string               `json:"source"`
	Summary          string               `json:"summary"`
	RequestedRoleRef string               `json:"requestedRoleRef,omitempty"`
	Preflight        []CandidatePreflight `json:"preflight,omitempty"`
}

// ExecutionPolicySnapshot is the immutable model/runner decision attached to a
// run at creation. Execution must consume Candidates from this snapshot rather
// than rereading the active role-policy catalog.
type ExecutionPolicySnapshot struct {
	CatalogDigest     string                      `json:"catalogDigest"`
	RoleRef           string                      `json:"roleRef,omitempty"`
	Candidates        []ExecutionCandidate        `json:"candidates"`
	SelectedIndex     int                         `json:"selectedIndex"`
	SelectedCandidate ExecutionCandidate          `json:"selectedCandidate"`
	Explanation       PolicyResolutionExplanation `json:"explanation"`
}

// NetworkAccess controls the level of network access granted to an agent during execution.
type NetworkAccess string

const (
	// NetworkAccessNone blocks all network access.
	// Codex: maps to --sandbox workspace-write.
	NetworkAccessNone NetworkAccess = "none"

	// NetworkAccessLocalhost allows access to localhost only (local scenario APIs).
	// Codex: maps to --dangerously-bypass-approvals-and-sandbox.
	NetworkAccessLocalhost NetworkAccess = "localhost"

	// NetworkAccessFull allows unrestricted network access.
	// Codex: maps to --dangerously-bypass-approvals-and-sandbox.
	NetworkAccessFull NetworkAccess = "full"
)

// IsValid reports whether the network access level is a supported value.
// Empty string is valid and treated as NetworkAccessLocalhost at runtime.
func (n NetworkAccess) IsValid() bool {
	switch n {
	case "", NetworkAccessNone, NetworkAccessLocalhost, NetworkAccessFull:
		return true
	default:
		return false
	}
}

// Effective returns the network access level, defaulting empty to NetworkAccessLocalhost.
func (n NetworkAccess) Effective() NetworkAccess {
	if n == "" {
		return NetworkAccessLocalhost
	}
	return n
}

// SandboxLifecycleEvent describes lifecycle triggers for sandbox cleanup.
type SandboxLifecycleEvent string

const (
	SandboxLifecycleTurnCompleted SandboxLifecycleEvent = "turn_completed"
	SandboxLifecycleTurnFailed    SandboxLifecycleEvent = "turn_failed"
	SandboxLifecycleTurnCancelled SandboxLifecycleEvent = "turn_cancelled"
	SandboxLifecycleRunFinalized  SandboxLifecycleEvent = "run_finalized"
	SandboxLifecycleRunCompleted  SandboxLifecycleEvent = "run_completed"
	SandboxLifecycleRunFailed     SandboxLifecycleEvent = "run_failed"
	SandboxLifecycleRunCancelled  SandboxLifecycleEvent = "run_cancelled"
	SandboxLifecycleApproved      SandboxLifecycleEvent = "approved"
	SandboxLifecycleRejected      SandboxLifecycleEvent = "rejected"
	SandboxLifecycleTerminal      SandboxLifecycleEvent = "terminal"
)

// SandboxLifecycleConfig controls sandbox stop/delete behavior.
type SandboxLifecycleConfig struct {
	CheckpointOn []SandboxLifecycleEvent `json:"checkpointOn,omitempty"`
	StopOn       []SandboxLifecycleEvent `json:"stopOn,omitempty"`
	DeleteOn     []SandboxLifecycleEvent `json:"deleteOn,omitempty"`
	TTL          time.Duration           `json:"ttl,omitempty"`
	IdleTimeout  time.Duration           `json:"idleTimeout,omitempty"`
}

// SandboxFileCriteria defines allow/deny matchers for acceptance filtering.
// Both PathGlobs and Extensions are AND-ed: a file must match at least one
// glob AND have a matching extension (if both are specified) to match.
type SandboxFileCriteria struct {
	PathGlobs  []string `json:"pathGlobs,omitempty"`  // e.g. ["ui/**", "src/components/**"]
	Extensions []string `json:"extensions,omitempty"` // e.g. [".tsx", ".css"]
}

// SandboxAcceptanceConfig controls which file changes are eligible for approval
// after the agent finishes its run.
//
// IMPORTANT: Acceptance is about which changes survive the approval process,
// NOT about restricting what the agent can write. The overlay allows writes to
// any file within the scope. Acceptance filtering happens later, when the diff
// is reviewed.
//
// This separation is intentional. It means:
//   - ScopePath on the Task controls the overlay's filesystem coverage (what the
//     lifecycle system can see when restarting scenarios — see SandboxEnvVars in
//     run_executor.go).
//   - AcceptanceConfig controls the blast radius of approved changes (what
//     actually gets applied to the real repo).
//
// Example: An agent tasked with UI styling changes should have:
//   - ScopePath = "scenarios/my-app"        (full scenario, so restarts work)
//   - Allow     = {PathGlobs: ["ui/**"]}    (only UI changes get approved)
//   - Deny      = {PathGlobs: ["api/**"]}   (API changes are always rejected)
//
// This way the agent can restart the scenario to see its UI changes rendered,
// but any accidental API modifications are caught during approval review.
type SandboxAcceptanceConfig struct {
	Mode         string              `json:"mode,omitempty"` // "allowlist" (default)
	Allow        SandboxFileCriteria `json:"allow,omitempty"`
	Deny         SandboxFileCriteria `json:"deny,omitempty"`
	IgnoreBinary bool                `json:"ignoreBinary,omitempty"`
}

// SandboxMode names the per-run sandbox execution mode from the
// auditability contract. The default produced by [DefaultSandboxConfig]
// is [SandboxModeProtected]: the agent process tree itself runs inside
// workspace-sandbox (bwrap isolation, NetworkMode translation, git
// allowlist enforcement on /processes and /exec). Runs that request
// Protected without a configured SandboxLauncherFactory fall back to host
// execution with an explicit warn event so misconfigured environments
// are visible rather than silent. [SandboxModeTracking] is the
// documented operator opt-out for runs that legitimately need full host
// capability — set explicitly per-spawn; nothing defaults to it.
// [SandboxModeOff] is the explicit "no sandbox at all" choice — used
// only for runs that legitimately have no auditability requirement
// (e.g. agent-manager developing itself). It is the single switch that
// controls whether the orchestrator allocates a sandbox for the run;
// see [DeriveRunMode] in package orchestration.
type SandboxMode string

const (
	// SandboxModeUnspecified means the SandboxConfig did not pick a mode
	// explicitly. Treated as SandboxModeTracking by [SandboxMode.Effective]
	// — the conservative routing target for code paths that construct a
	// zero-valued SandboxConfig directly. Spawn surfaces should clone
	// [DefaultSandboxConfig] (which sets Mode=Protected) instead of
	// zero-initialising, so the unspecified→tracking fallback only fires
	// for legacy or test code paths.
	SandboxModeUnspecified SandboxMode = ""

	// SandboxModeOff disables sandboxing for the run entirely. The
	// orchestrator skips workspace-sandbox allocation, the runner edits
	// the canonical repo directly, and no provenance record is written.
	// Reserved for runs where auditability is genuinely irrelevant
	// (e.g. agent-manager developing itself, in-place tests). This is
	// the *only* value that produces RunModeInPlace; every other Mode
	// produces RunModeSandboxed.
	SandboxModeOff SandboxMode = "off"

	// SandboxModeTracking is the host-tracked auditability mode: the
	// agent runs on the host and the sandbox merely tracks file changes
	// for accountability/provenance. Used as the explicit operator
	// opt-out for runs that need full host capability (e.g. git push
	// after review, scraping a remote URL). Locked defaults when chosen:
	// ManualReview=false, AutoApply=true, ApplyOnFailure=true,
	// NoLock=true (lock=false), NetworkMode=localhost.
	SandboxModeTracking SandboxMode = "tracking"

	// SandboxModeProtected runs the agent process tree itself inside the
	// workspace-sandbox container — bwrap isolation, network mode, and
	// git allowlist are enforced on the agent process, not just on its
	// merged-overlay output. This is the production default. Whether a
	// protected-mode request actually launches in the sandbox or falls
	// back to host execution depends on the runner having a
	// SandboxLauncherFactory wired at runtime; in production main.go
	// always wires this.
	//
	// See execute/protected-sandbox-agent-launch and
	// scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md.
	SandboxModeProtected SandboxMode = "protected"
)

// IsValid reports whether m is a recognised mode name (not whether it is
// currently implemented — see Validate for the runtime gate).
func (m SandboxMode) IsValid() bool {
	switch m {
	case SandboxModeUnspecified, SandboxModeOff, SandboxModeTracking, SandboxModeProtected:
		return true
	default:
		return false
	}
}

// Effective returns the mode value, defaulting empty to SandboxModeTracking.
// SandboxModeOff is preserved as-is (it is an explicit, intentional choice).
func (m SandboxMode) Effective() SandboxMode {
	if m == SandboxModeUnspecified {
		return SandboxModeTracking
	}
	return m
}

// strictnessRank orders sandbox modes from least to most strict, so a
// "minimum-mode" policy can be expressed as a numeric ≥ comparison.
// SandboxModeUnspecified is treated as Tracking (matches Effective).
//
//	Off (0) < Tracking (1) < Protected (2)
//
// The values are an internal implementation detail of [SandboxMode.AtLeast];
// callers should not depend on the integers.
func (m SandboxMode) strictnessRank() int {
	switch m.Effective() {
	case SandboxModeOff:
		return 0
	case SandboxModeTracking:
		return 1
	case SandboxModeProtected:
		return 2
	default:
		// Unknown values fall back to "off" so an invalid mode never
		// silently satisfies a strictness requirement.
		return 0
	}
}

// AtLeast reports whether m is at least as strict as required. It is the
// canonical comparison used by the orchestrator when validating that a
// resolved SandboxConfig.Mode satisfies a policy-declared minimum.
// SandboxModeUnspecified on either side is normalised via Effective().
func (m SandboxMode) AtLeast(required SandboxMode) bool {
	return m.strictnessRank() >= required.strictnessRank()
}

// SandboxConfig holds lifecycle + acceptance settings for a sandbox.
//
// Design note: SandboxConfig controls sandbox BEHAVIOR (when to clean up,
// which files to accept, when to apply). It does NOT control the sandbox's
// filesystem SCOPE — that comes from Task.ScopePath, which determines what
// directory the overlay covers. See the ScopePath vs Acceptance distinction
// documented on SandboxAcceptanceConfig.
//
// The Mode / ManualReview / AutoApply / ApplyOnFailure / NetworkMode fields
// encode the auditability contract — see
// scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md.
// DefaultSandboxConfig returns the locked defaults; spawn surfaces should
// compose against those rather than zero-initialising.
type SandboxConfig struct {
	Lifecycle  SandboxLifecycleConfig  `json:"lifecycle,omitempty"`
	Acceptance SandboxAcceptanceConfig `json:"acceptance,omitempty"`

	// Mode selects the auditability mode. Empty defaults to "tracking".
	Mode SandboxMode `json:"mode,omitempty"`

	// ManualReview defers apply at run end until an operator approves via
	// one of the three viewing surfaces (git-control-tower, agent-manager,
	// workspace-sandbox). When true, the sandbox persists past run end.
	// Default: false.
	ManualReview bool `json:"manualReview,omitempty"`

	// AutoApply controls whether in-acceptance changes apply to the canonical
	// repo at run end. Stored as a pointer so the zero-value of an unset
	// SandboxConfig is unambiguous; nil is treated as the contract default
	// (true). Use GetAutoApply for the resolved value.
	AutoApply *bool `json:"autoApply,omitempty"`

	// ApplyOnFailure controls whether apply runs identically when the run
	// outcome is failure / cancelled / timeout. nil ↔ contract default true.
	// Run outcome is recorded as metadata on the resulting provenance record
	// but does not gate apply behaviour.
	ApplyOnFailure *bool `json:"applyOnFailure,omitempty"`

	// NetworkMode mirrors NetworkAccess for sandboxed execution. Empty
	// defaults to NetworkAccessLocalhost.
	NetworkMode NetworkAccess `json:"networkMode,omitempty"`

	// NoLock disables mutual exclusion locking. The contract makes locking
	// and acceptance orthogonal: NoLock does not bypass acceptance.
	// (Contract framing names this "lock"; lock=false ↔ NoLock=true.)
	NoLock bool `json:"noLock,omitempty"`
}

// GetAutoApply resolves AutoApply, defaulting to the contract value (true)
// when the pointer is nil. Safe to call on a nil receiver.
func (c *SandboxConfig) GetAutoApply() bool {
	if c == nil || c.AutoApply == nil {
		return true
	}
	return *c.AutoApply
}

// GetApplyOnFailure resolves ApplyOnFailure, defaulting to the contract
// value (true) when the pointer is nil. Safe to call on a nil receiver.
func (c *SandboxConfig) GetApplyOnFailure() bool {
	if c == nil || c.ApplyOnFailure == nil {
		return true
	}
	return *c.ApplyOnFailure
}

// DefaultSandboxConfig returns the auditability-contract defaults. Spawn
// surfaces should clone this and apply overrides on top, rather than
// zero-initialising.
//
// Mode defaults to SandboxModeProtected: the agent process tree itself
// runs inside the workspace-sandbox (bwrap isolation, NetworkMode
// translation, git allowlist enforcement on /processes and /exec). Slices
// 1–3 of execute/protected-sandbox-agent-launch wired all three runners
// (claude_code, codex, opencode) and both Execute and Continue paths
// through the launcher seam, so this default is now safe.
//
// Tracking mode (SandboxModeTracking) remains as the documented operator
// opt-out for runs that legitimately need full host capability — e.g.,
// a `git push` after review, scraping a remote URL, or self-modifying
// scenarios. Operators set it explicitly per-spawn; nothing defaults to
// it. RunMode.IN_PLACE remains the full-bypass mode for the rare cases
// where even tracking-mode auditability is wrong (e.g., agent-manager
// developing itself).
func DefaultSandboxConfig() *SandboxConfig {
	autoApply := true
	applyOnFailure := true
	return &SandboxConfig{
		Mode:           SandboxModeProtected,
		ManualReview:   false,
		AutoApply:      &autoApply,
		ApplyOnFailure: &applyOnFailure,
		NetworkMode:    NetworkAccessLocalhost,
		NoLock:         true,
	}
}

// FeatureFlags contains well-known typed feature flags.
// Each flag maps to runner-specific CLI args at execution time.
// Runners that don't support a feature silently ignore it.
type FeatureFlags struct {
	// EnableBrowser enables browser automation tools.
	// Claude Code: maps to --chrome flag.
	// Other runners: silently ignored (not supported).
	EnableBrowser bool `json:"enableBrowser,omitempty"`
}

// IsZero reports whether all feature flags are at their zero values.
func (f FeatureFlags) IsZero() bool {
	return !f.EnableBrowser
}

// RunnerExtraFlags maps runner types to validated extra CLI flags.
type RunnerExtraFlags map[RunnerType][]string

// -----------------------------------------------------------------------------
// Task - Defines WHAT needs to be done
// -----------------------------------------------------------------------------

// Task defines a unit of work to be performed by an agent.
type Task struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description,omitempty" db:"description"`

	// ScopePath defines the directory that the overlayfs sandbox covers.
	// It is relative to ProjectRoot (e.g., "scenarios/agent-inbox").
	//
	// IMPORTANT: The overlay's merged directory contains ONLY the contents of
	// this path. The scope determines what the agent sees in the sandbox AND
	// what the Vrooli CLI lifecycle system can build/run when the agent restarts
	// a scenario (via the VROOLI_SANDBOX_* env vars — see run_executor.go).
	//
	// Best practice: Set ScopePath to the full scenario directory (e.g.,
	// "scenarios/my-app"), not a subdirectory. If the scope is too narrow
	// (e.g., "scenarios/my-app/ui"), the lifecycle system won't have the
	// Makefile or service.json needed to restart the scenario, and will fall
	// back to the real repo — making the agent's changes invisible on restart.
	//
	// To restrict WHICH changes get approved (blast radius), use
	// SandboxAcceptanceConfig.Allow/Deny on the run's SandboxConfig instead.
	ScopePath   string `json:"scopePath" db:"scope_path"`
	ProjectRoot string `json:"projectRoot,omitempty" db:"project_root"`

	// Multi-phase execution support
	PhasePromptIDs []uuid.UUID `json:"phasePromptIds,omitempty" db:"phase_prompt_ids"`

	// Context attachments (files, links, notes)
	ContextAttachments []ContextAttachment `json:"contextAttachments,omitempty" db:"context_attachments"`

	// Status tracking
	Status TaskStatus `json:"status" db:"status"`

	// Ownership
	CreatedBy string    `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	TaskStatusQueued      TaskStatus = "queued"
	TaskStatusRunning     TaskStatus = "running"
	TaskStatusNeedsReview TaskStatus = "needs_review"
	TaskStatusApproved    TaskStatus = "approved"
	TaskStatusRejected    TaskStatus = "rejected"
	TaskStatusFailed      TaskStatus = "failed"
	TaskStatusCancelled   TaskStatus = "cancelled"
)

// ContextAttachment represents additional context for a task.
// Each attachment should have a clear summary and appropriate priority
// to help agents quickly understand relevance and focus on important context.
type ContextAttachment struct {
	Type         string   `json:"type"`                    // "file", "link", "note", "image"
	Key          string   `json:"key,omitempty"`           // Unique identifier (e.g., "error-logs", "deployment-manifest")
	Tags         []string `json:"tags,omitempty"`          // Categorization tags for filtering and analytics
	Path         string   `json:"path,omitempty"`          // File path for "file" type
	URL          string   `json:"url,omitempty"`           // URL for "link" type
	Content      string   `json:"content,omitempty"`       // Inline content for "note" type
	Label        string   `json:"label,omitempty"`         // Human-readable title
	Summary      string   `json:"summary,omitempty"`       // One-sentence TL;DR of what this context contains
	Format       string   `json:"format,omitempty"`        // Content format: "text", "json", "markdown", "yaml", "log"
	Priority     string   `json:"priority,omitempty"`      // Importance: "high", "medium", "low"
	AttachmentID string   `json:"attachment_id,omitempty"` // Reference to uploaded Attachment (for "image" type)
}

// -----------------------------------------------------------------------------
// Run - A concrete execution attempt
// -----------------------------------------------------------------------------

// Run represents a single execution attempt of a task using a specific agent profile.
type Run struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	TaskID         uuid.UUID  `json:"taskId" db:"task_id"`
	AgentProfileID *uuid.UUID `json:"agentProfileId,omitempty" db:"agent_profile_id"` // Optional if inline config provided

	// Custom tag for identification (defaults to ID if not set)
	// Used for agent tracking, log filtering, and external process identification
	Tag string `json:"tag,omitempty" db:"tag"`

	// Sandbox integration
	SandboxID     *uuid.UUID     `json:"sandboxId,omitempty" db:"sandbox_id"`
	RunMode       RunMode        `json:"runMode" db:"run_mode"`
	SandboxConfig *SandboxConfig `json:"sandboxConfig,omitempty" db:"sandbox_config"`

	// ExecutionMode selects the CLI-driving substrate for this run
	// (codec-pipe vs interactive web-console session). Empty is treated as
	// ExecutionModeCodecPipe; see [ExecutionMode]. Orthogonal to RunMode.
	ExecutionMode ExecutionMode `json:"executionMode,omitempty" db:"execution_mode"`

	// WebConsoleSessionID is the id of the web-console session hosting the
	// interactive agent CLI, set only for ExecutionModeInteractive runs. It
	// backs the run-detail deep link to the live session and routes the
	// interactive Continue/Stop terminal input + session teardown.
	WebConsoleSessionID string `json:"webConsoleSessionId,omitempty" db:"web_console_session_id"`

	// WebConsoleSessionURL is the resolved deep link to the live web-console
	// session (computed at read time from WebConsoleSessionID + the web-console
	// UI base, not persisted). Empty for non-interactive runs and when the
	// web-console UI base cannot be resolved server-side.
	WebConsoleSessionURL string `json:"webConsoleSessionUrl,omitempty"`

	// Execution state
	Status    RunStatus  `json:"status" db:"status"`
	StartedAt *time.Time `json:"startedAt,omitempty" db:"started_at"`
	EndedAt   *time.Time `json:"endedAt,omitempty" db:"ended_at"`

	// Progress tracking (for resumption and visibility)
	Phase            RunPhase   `json:"phase" db:"phase"`
	LastCheckpointID *uuid.UUID `json:"lastCheckpointId,omitempty" db:"last_checkpoint_id"`
	LastHeartbeat    *time.Time `json:"lastHeartbeat,omitempty" db:"last_heartbeat"`
	ProgressPercent  int        `json:"progressPercent" db:"progress_percent"`

	// Idempotency (for replay safety)
	IdempotencyKey string `json:"idempotencyKey,omitempty" db:"idempotency_key"`

	// Results
	Summary  *RunSummary `json:"summary,omitempty" db:"summary"`
	ErrorMsg string      `json:"errorMsg,omitempty" db:"error_msg"`
	ExitCode *int        `json:"exitCode,omitempty" db:"exit_code"`

	// Approval workflow
	ApprovalState ApprovalState `json:"approvalState" db:"approval_state"`
	ApprovedBy    string        `json:"approvedBy,omitempty" db:"approved_by"`
	ApprovedAt    *time.Time    `json:"approvedAt,omitempty" db:"approved_at"`

	// Post-run sandbox finalization. This tracks apply/checkpoint effects
	// separately from the runner turn status so infrastructure cleanup cannot
	// make a completed turn appear to still be running.
	FinalizationStatus RunFinalizationStatus `json:"finalizationStatus" db:"finalization_status"`
	FinalizationError  string                `json:"finalizationError,omitempty" db:"finalization_error"`
	FinalizedAt        *time.Time            `json:"finalizedAt,omitempty" db:"finalized_at"`

	// Inline config (used when no profile provided, or to store resolved config)
	ResolvedConfig *RunConfig `json:"resolvedConfig,omitempty" db:"resolved_config"`

	// Artifacts
	DiffPath       string `json:"diffPath,omitempty" db:"diff_path"`
	LogPath        string `json:"logPath,omitempty" db:"log_path"`
	ChangedFiles   int    `json:"changedFiles" db:"changed_files"`
	TotalSizeBytes int64  `json:"totalSizeBytes" db:"total_size_bytes"`

	// Session continuation support
	// Stores the runner-specific session identifier for conversation resumption.
	// For Claude Code: session_id from stream events
	// For Codex: thread_id from stream events
	// For OpenCode: sessionID from stream events
	SessionID string `json:"sessionId,omitempty" db:"session_id"`

	// Transcript recovery metadata for restart-safe run reconciliation.
	RunnerPID         int    `json:"runnerPid,omitempty" db:"runner_pid"`
	RunnerPGID        int    `json:"runnerPgid,omitempty" db:"runner_pgid"`
	TranscriptPath    string `json:"transcriptPath,omitempty" db:"transcript_path"`
	TranscriptCursor  int64  `json:"transcriptCursor,omitempty" db:"transcript_cursor"`
	TranscriptLastSeq int64  `json:"transcriptLastSeq,omitempty" db:"transcript_last_seq"`

	// Model provenance — requested is the first concrete entry the preset chain expanded to
	// when the run was created; actual is the model the CLI actually executed with once
	// model-fallback (if any) converged. When they differ the run degraded through the chain.
	RequestedModel string `json:"requestedModel,omitempty" db:"requested_model"`
	ActualModel    string `json:"actualModel,omitempty" db:"actual_model"`

	// Investigation lineage fields
	// SourceRunIDs links investigation runs back to the run(s) being investigated.
	SourceRunIDs []uuid.UUID `json:"sourceRunIds,omitempty" db:"source_run_ids"`
	// SourceInvestigationRunID links apply runs back to the investigation run they apply.
	SourceInvestigationRunID *uuid.UUID `json:"sourceInvestigationRunId,omitempty" db:"source_investigation_run_id"`

	// ParentRunID is the generic "parent run" link for conversation continuity.
	// When a spawner is creating a follow-up run as a continuation of an
	// existing agent thread (e.g. swarm-manager queue resuming a swarm,
	// agent-manager UI "continue conversation"), it sets ParentRunID to the
	// originating run. The run-creation path uses ParentRunID to inherit
	// ConversationID — see ResolveConversationID.
	//
	// ParentRunID is a separate concept from SourceInvestigationRunID
	// (apply-from-investigation linkage) and SourceRunIDs (investigation
	// targets); a run can have any combination of those plus ParentRunID.
	ParentRunID *uuid.UUID `json:"parentRunId,omitempty" db:"parent_run_id"`

	// ConversationID groups runs that belong to the same agent thread for
	// auditability. One ID per agent-thread; child runs inherit from
	// ParentRunID's run when set, otherwise the run-creation path generates
	// a fresh UUID. Spawn surfaces that already know they are continuing a
	// thread (e.g. swarm-manager queue resuming a swarm) populate this value
	// directly; standalone runs get a new ID. See
	// scenarios/workspace-sandbox/docs/AUDITABILITY_CONTRACT.md Finding 2.
	//
	// IMPORTANT: this is NOT the same as Run.SessionID, which is a
	// runner-specific resume token (Claude Code session_id, Codex thread_id,
	// etc.) with an unrelated lifetime.
	ConversationID string `json:"conversationId,omitempty" db:"conversation_id"`

	// Recommendation extraction state (for investigation runs)
	// Recommendations are extracted passively after investigation runs complete.
	RecommendationStatus   RecommendationStatus `json:"recommendationStatus,omitempty" db:"recommendation_status"`
	RecommendationResult   *ExtractionResult    `json:"recommendationResult,omitempty" db:"recommendation_result"`
	RecommendationAttempts int                  `json:"recommendationAttempts,omitempty" db:"recommendation_attempts"`
	RecommendationError    string               `json:"recommendationError,omitempty" db:"recommendation_error"`
	RecommendationQueuedAt *time.Time           `json:"recommendationQueuedAt,omitempty" db:"recommendation_queued_at"`

	// Identity token fields
	IdentityTokenHash      string     `json:"identityTokenHash,omitempty" db:"identity_token_hash"`
	IdentityTokenRevokedAt *time.Time `json:"identityTokenRevokedAt,omitempty" db:"identity_token_revoked_at"`

	// CustomEnv holds the caller-supplied VROOLI_-prefixed environment
	// variables passed at run creation (CreateRunRequest.Environment).
	// Persisted so the continue/wake path can re-inject them: a continued
	// turn that bypassed this would silently drop scenario-injected custom
	// env (the latent bug Phase 0 of the park/resume work fixes). Values are
	// VROOLI_*-validated at the API boundary (≤20 entries / ≤4096 bytes).
	CustomEnv map[string]string `json:"customEnv,omitempty" db:"custom_env"`

	// AwaitHandle describes the externally-owned async work a parked run is
	// waiting on. It is set when the run transitions running→parked and cleared
	// on wake (parked→running) or cancel (parked→cancelled). Persisted (JSON
	// column) so an agent-manager restart can re-spawn the waiter for every
	// parked run (one open handle per run — a second park while parked is
	// rejected). Nil for every non-parked run.
	AwaitHandle *AwaitHandle `json:"awaitHandle,omitempty" db:"await_handle"`

	// LastAwaitKey / LastAwaitResult / LastAwaitResolvedAt record the most
	// recently RESOLVED await: the producer:key, the full result string that was
	// injected into the woken turn, and when it resolved. They are the durable
	// SSOT behind the re-fetch path (GET /runs/{id}/await-result): a woken agent
	// that did not see — or wants to re-read — the result can retrieve it cheaply
	// without re-running the blocking producer. Set on wake (parked→running),
	// retained across subsequent turns until the next await resolves.
	LastAwaitKey        string     `json:"lastAwaitKey,omitempty" db:"last_await_key"`
	LastAwaitResult     string     `json:"lastAwaitResult,omitempty" db:"last_await_result"`
	LastAwaitResolvedAt *time.Time `json:"lastAwaitResolvedAt,omitempty" db:"last_await_resolved_at"`

	// LastWakeSeq snapshots TranscriptLastSeq at the moment of the last wake. The
	// no-progress re-park guard compares it against the live TranscriptLastSeq to
	// best-effort detect whether the agent did any work since being woken before
	// it tries to park again.
	LastWakeSeq int64 `json:"lastWakeSeq,omitempty" db:"last_wake_seq"`

	// SameKeyParkStreak counts how many times in a row this run has tried to park
	// on the SAME await key without forward progress in between. It is the
	// timing-independent backstop the re-park guard uses to refuse a degenerate
	// "wake → immediately re-run the same blocking command → re-park" loop (the
	// coding-agent limitation park exists to absorb). Reset to 0 on a different-key
	// park or on detected progress.
	SameKeyParkStreak int `json:"sameKeyParkStreak,omitempty" db:"same_key_park_streak"`

	// First ~120 chars of the associated task description (computed, not persisted).
	PromptPreview string `json:"promptPreview,omitempty"`

	// Action availability (computed, not persisted)
	Actions *RunActions `json:"actions,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// GetTag returns the tag for this run, defaulting to the run ID if no custom tag is set.
func (r *Run) GetTag() string {
	if r.Tag != "" {
		return r.Tag
	}
	return r.ID.String()
}

// IsResumable returns whether this run can be resumed from its current state.
func (r *Run) IsResumable() bool {
	// Can only resume runs that are in a non-terminal state
	switch r.Status {
	case RunStatusComplete, RunStatusFailed, RunStatusCancelled:
		return false
	}
	// Check if the phase supports resumption
	return r.Phase.CanResumeFromPhase()
}

// IsStale returns whether this run appears to have stalled.
func (r *Run) IsStale(staleDuration time.Duration) bool {
	if r.LastHeartbeat == nil {
		// No heartbeat recorded, check based on started time
		if r.StartedAt == nil {
			return false
		}
		return time.Since(*r.StartedAt) > staleDuration
	}
	return time.Since(*r.LastHeartbeat) > staleDuration
}

// UpdateProgress updates the run's progress tracking fields.
func (r *Run) UpdateProgress(phase RunPhase, percent int) {
	r.Phase = phase
	r.ProgressPercent = percent
	now := time.Now()
	r.LastHeartbeat = &now
	r.UpdatedAt = now
}

// IsInvestigationRun returns true if this run is an investigation run (not an apply run).
// Investigation runs have a tag starting with "agent-manager-investigation" but not ending in "-apply".
func (r *Run) IsInvestigationRun() bool {
	return strings.HasPrefix(r.Tag, "agent-manager-investigation") &&
		!strings.HasSuffix(r.Tag, "-apply")
}

// RunMode indicates whether the run uses sandbox isolation.
type RunMode string

const (
	RunModeSandboxed RunMode = "sandboxed"
	RunModeInPlace   RunMode = "in_place"
)

// ExecutionMode indicates how agent-manager drives the agent CLI for a run.
// It is orthogonal to [RunMode] (sandbox isolation): a run picks an execution
// substrate independently of whether it is sandboxed.
//
//   - ExecutionModeCodecPipe (default): agent-manager owns the CLI process and
//     reads events off its stdout pipe via the codec decoders. This is the
//     historical path and the only path for protected (sandboxed) runs.
//   - ExecutionModeInteractive: agent-manager launches the real interactive
//     agent CLI inside a web-console (persistent/tmux) session and reads events
//     by tailing the agent-owned on-disk transcript. Allowed only for
//     non-protected (in-place) runs.
type ExecutionMode string

const (
	ExecutionModeCodecPipe   ExecutionMode = "codec_pipe"
	ExecutionModeInteractive ExecutionMode = "interactive"
)

// Normalized returns the mode with the empty value defaulted to
// ExecutionModeCodecPipe, so rows written before the column existed (and
// callers that leave the field unset) behave as codec-pipe runs.
func (m ExecutionMode) Normalized() ExecutionMode {
	if m == "" {
		return ExecutionModeCodecPipe
	}
	return m
}

// IsValid reports whether the mode is one of the known execution modes.
func (m ExecutionMode) IsValid() bool {
	switch m {
	case ExecutionModeCodecPipe, ExecutionModeInteractive:
		return true
	default:
		return false
	}
}

// RunStatus represents the current state of a run.
type RunStatus string

const (
	RunStatusPending     RunStatus = "pending"
	RunStatusStarting    RunStatus = "starting"
	RunStatusRunning     RunStatus = "running"
	RunStatusNeedsReview RunStatus = "needs_review"
	RunStatusComplete    RunStatus = "complete"
	RunStatusFailed      RunStatus = "failed"
	RunStatusCancelled   RunStatus = "cancelled"
	// RunStatusParked: the run is suspended waiting on externally-owned async
	// work (a test-genie run, a git-control-tower baseline diff). The agent
	// process has exited (zero tokens burned) but the run is NOT terminal — its
	// sandbox is preserved and agent-manager re-spawns ("wakes") the conversation
	// with the awaited result injected once it resolves. Modeled on needs_review
	// (process-exited, non-terminal, resume-with-injected-message) but for
	// infrastructure-driven waiting rather than operator review. See
	// scenarios/agent-manager/docs/internal/SEAMS.md (park/wake) and the
	// LivenessPolicy table in decisions.go (parked is scanned but never
	// heartbeat-reaped).
	RunStatusParked RunStatus = "parked"
)

// AwaitHandle identifies the externally-owned async work a parked run is
// blocked on. agent-manager (which owns the agent process) performs the
// blocking wait on the agent's behalf via a per-producer Waiter seam and wakes
// the run when the handle resolves. The handle is the unit persisted for
// restart recovery and the key the waiter de-duplicates on so wake is
// idempotent (a double-resolve must not double-wake).
type AwaitHandle struct {
	// Producer identifies which Waiter resolves this handle (e.g. "test-genie",
	// "git-control-tower"). The await-handle registry (Phase 3) maps it to a
	// concrete Waiter implementation.
	Producer string `json:"producer"`
	// Key is the producer-scoped identifier of the awaited work (e.g. a
	// test-genie run ID, a baseline diff request key). Producer+Key together
	// uniquely identify the work being awaited.
	Key string `json:"key"`
	// Deadline bounds the wait. When it elapses agent-manager wakes the run with
	// a typed "timed-out / unknown" result rather than hanging forever. Nil ⇒
	// the orchestrator default ParkTTL is applied at park time, so a persisted
	// handle always carries a concrete deadline.
	Deadline *time.Time `json:"deadline,omitempty"`
	// RegisteredAt records when the park happened (for observability / ETA).
	RegisteredAt time.Time `json:"registeredAt"`
}

// RunFinalizationStatus represents post-run sandbox apply/checkpoint state.
type RunFinalizationStatus string

const (
	RunFinalizationStatusNone      RunFinalizationStatus = "none"
	RunFinalizationStatusPending   RunFinalizationStatus = "pending"
	RunFinalizationStatusRunning   RunFinalizationStatus = "running"
	RunFinalizationStatusSucceeded RunFinalizationStatus = "succeeded"
	RunFinalizationStatusFailed    RunFinalizationStatus = "failed"
	RunFinalizationStatusSkipped   RunFinalizationStatus = "skipped"
)

// ApprovalState represents the approval workflow state.
type ApprovalState string

const (
	ApprovalStateNone              ApprovalState = "none"
	ApprovalStatePending           ApprovalState = "pending"
	ApprovalStatePartiallyApproved ApprovalState = "partially_approved"
	ApprovalStateApproved          ApprovalState = "approved"
	ApprovalStateRejected          ApprovalState = "rejected"
)

// RecommendationStatus represents the state of recommendation extraction for investigation runs.
type RecommendationStatus string

const (
	// RecommendationStatusNone - Not applicable (non-investigation run or not yet complete)
	RecommendationStatusNone RecommendationStatus = "none"

	// RecommendationStatusPending - Awaiting extraction (queued for background processing)
	RecommendationStatusPending RecommendationStatus = "pending"

	// RecommendationStatusExtracting - Extraction in progress
	RecommendationStatusExtracting RecommendationStatus = "extracting"

	// RecommendationStatusComplete - Successfully extracted and cached
	RecommendationStatusComplete RecommendationStatus = "complete"

	// RecommendationStatusFailed - Extraction failed after max retries
	RecommendationStatusFailed RecommendationStatus = "failed"
)

// RunSummary contains the structured summary from an agent run.
type RunSummary struct {
	Description   string   `json:"description,omitempty"`
	FilesModified []string `json:"filesModified,omitempty"`
	FilesCreated  []string `json:"filesCreated,omitempty"`
	FilesDeleted  []string `json:"filesDeleted,omitempty"`
	TokensUsed    int      `json:"tokensUsed,omitempty"`
	TurnsUsed     int      `json:"turnsUsed,omitempty"`
	CostEstimate  float64  `json:"costEstimate,omitempty"`
	ContextTokens int      `json:"contextTokens,omitempty"`
}

// RunConfig contains the resolved configuration for a run.
// This can be loaded from a profile, provided inline, or a combination of both.
type RunConfig struct {
	// Runner configuration
	RunnerType RunnerType    `json:"runnerType"`
	Model      string        `json:"model,omitempty"`
	RoleRef    string        `json:"roleRef,omitempty"`
	MaxTurns   int           `json:"maxTurns,omitempty"`
	Timeout    time.Duration `json:"timeout,omitempty"`

	// PolicySnapshot pins the exact active catalog revision and ordered
	// candidate sequence selected before this run was persisted.
	PolicySnapshot *ExecutionPolicySnapshot `json:"policySnapshot,omitempty"`

	// Tool permissions
	AllowedTools []string `json:"allowedTools,omitempty"`
	DeniedTools  []string `json:"deniedTools,omitempty"`

	// Execution flags
	SkipPermissionPrompt bool `json:"skipPermissionPrompt,omitempty"`

	// Feature flags (typed, discoverable capabilities)
	Features FeatureFlags `json:"features,omitempty"`

	// Extra CLI flags per runner type (validated escape hatch)
	ExtraFlags RunnerExtraFlags `json:"extraFlags,omitempty"`

	// Policy flags
	NetworkAccess NetworkAccess `json:"networkAccess"`

	// Sandbox behavior settings.
	//
	// SandboxConfig.Mode is the single source of truth for whether the
	// run is sandboxed. See [orchestration.DeriveRunMode]. A nil
	// SandboxConfig is treated as "unspecified" — orchestration spawn
	// surfaces clone [DefaultSandboxConfig] before resolving so the
	// nil case only arises in legacy tests.
	SandboxConfig *SandboxConfig `json:"sandboxConfig,omitempty"`

	// Path restrictions
	AllowedPaths []string `json:"allowedPaths,omitempty"`
	DeniedPaths  []string `json:"deniedPaths,omitempty"`
}

// ApplyProfile applies values from an AgentProfile as the base configuration.
func (c *RunConfig) ApplyProfile(profile *AgentProfile) {
	if profile == nil {
		return
	}
	c.RoleRef = profile.RoleRef
	c.MaxTurns = profile.MaxTurns
	c.Timeout = profile.Timeout
	c.AllowedTools = profile.AllowedTools
	c.DeniedTools = profile.DeniedTools
	c.SkipPermissionPrompt = profile.SkipPermissionPrompt
	c.Features = profile.Features
	if len(profile.ExtraFlags) > 0 {
		c.ExtraFlags = make(RunnerExtraFlags, len(profile.ExtraFlags))
		for rt, flags := range profile.ExtraFlags {
			c.ExtraFlags[rt] = append([]string(nil), flags...)
		}
	}
	c.NetworkAccess = profile.NetworkAccess
	// Only overwrite SandboxConfig when the profile actually provides
	// one. A nil profile.SandboxConfig means "use the run-config
	// default"; copying it would silently clobber DefaultSandboxConfig
	// and reintroduce the zero-value-bool bypass class of bug.
	if profile.SandboxConfig != nil {
		c.SandboxConfig = profile.SandboxConfig
	}
	c.AllowedPaths = profile.AllowedPaths
	c.DeniedPaths = profile.DeniedPaths
}

// DefaultRunConfig returns sensible defaults for run configuration.
//
// The auditability-contract apply defaults (ManualReview=false,
// AutoApply=true, ApplyOnFailure=true, Mode=Protected) live on
// SandboxConfig — see DefaultSandboxConfig. Embedding the default
// SandboxConfig here is what makes the "sandbox by default" invariant
// hold even when callers ApplyProfile a profile with no SandboxConfig
// of its own.
func DefaultRunConfig() *RunConfig {
	return &RunConfig{
		RunnerType:    RunnerTypeClaudeCode,
		MaxTurns:      30,
		Timeout:       60 * time.Minute,
		NetworkAccess: NetworkAccessLocalhost,
		SandboxConfig: DefaultSandboxConfig(),
	}
}

// -----------------------------------------------------------------------------
// RunEvent - Append-only event stream
// -----------------------------------------------------------------------------
//
// TAGGED UNION PATTERN:
// RunEvent uses a tagged union for type-safe event payloads. Each event type
// has a specific payload struct, ensuring you can only set relevant fields.
//
// Usage:
//   event := NewLogEvent(runID, "info", "Starting execution")
//   event := NewToolCallEvent(runID, "Read", "toolu_123", map[string]interface{}{"path": "/foo"})
//
// The Data field contains a type-specific payload that can be type-asserted:
//   if log, ok := event.Data.(*LogEventData); ok { ... }

// RunEvent represents a single event in a run's event stream.
//
// SchemaVersion identifies the on-wire shape of Data for typed events. It is
// recorded as a column on run_events (not embedded in the JSON body) so the
// event-log dispatch table can route old payloads to old payload types
// indefinitely while new payloads use the current types. The default is 1;
// the eventlog package is the source of truth for which versions are
// registered for which event types.
type RunEvent struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	RunID         uuid.UUID    `json:"runId" db:"run_id"`
	Sequence      int64        `json:"sequence" db:"sequence"`
	EventType     RunEventType `json:"eventType" db:"event_type"`
	Timestamp     time.Time    `json:"timestamp" db:"timestamp"`
	SchemaVersion int          `json:"schemaVersion,omitempty" db:"schema_version"`
	Data          EventPayload `json:"data" db:"data"`
}

// RunEventType categorizes the event.
type RunEventType string

const (
	EventTypeLog            RunEventType = "log"
	EventTypeMessage        RunEventType = "message"
	EventTypeMessageDeleted RunEventType = "message_deleted"
	EventTypeToolCall       RunEventType = "tool_call"
	EventTypeToolResult     RunEventType = "tool_result"
	EventTypeStatus         RunEventType = "status"
	EventTypeMetric         RunEventType = "metric"
	EventTypeArtifact       RunEventType = "artifact"
	EventTypeError          RunEventType = "error"
	EventTypeCompaction     RunEventType = "compaction"
	EventTypeLifecycle      RunEventType = "lifecycle"

	// Typed operational events.
	//
	// These replace freeform LogEventData strings for operationally-significant
	// signals (fallback walks, sandbox ops, heartbeat misses, checkpoint
	// failures, model/runner health transitions). Payload structs and emit
	// helpers live in the eventlog package; the dispatch table there is the
	// authoritative (event_type, schema_version) → payload-type registry.
	EventTypeRunnerFallbackAttempted RunEventType = "runner.fallback.attempted"
	EventTypeRunnerFallbackExhausted RunEventType = "runner.fallback.exhausted"
	EventTypeModelFallbackAttempted  RunEventType = "model.fallback.attempted"
	EventTypeModelFallbackExhausted  RunEventType = "model.fallback.exhausted"
	EventTypePolicyCandidateAttempt  RunEventType = "policy.candidate.attempt"
	EventTypeModelHealthTransition   RunEventType = "model.health.transition"
	EventTypeRunnerHealthTransition  RunEventType = "runner.health.transition"
	EventTypeSandboxOperation        RunEventType = "sandbox.operation"
	EventTypeHeartbeatMiss           RunEventType = "heartbeat.miss"
	EventTypeCheckpointFailure       RunEventType = "checkpoint.failure"
	EventTypeRetryAttempt            RunEventType = "retry.attempt"
)

// IsTypedOperationalEvent reports whether t is one of the typed-operational
// event categories whose payload shape is owned by the eventlog package.
//
// The SQLite event store consults this so it can deserialize the payload
// through the eventlog dispatch table instead of trying to decode it as a
// legacy tagged-union shape.
func (t RunEventType) IsTypedOperationalEvent() bool {
	switch t {
	case EventTypeRunnerFallbackAttempted,
		EventTypeRunnerFallbackExhausted,
		EventTypeModelFallbackAttempted,
		EventTypeModelFallbackExhausted,
		EventTypePolicyCandidateAttempt,
		EventTypeModelHealthTransition,
		EventTypeRunnerHealthTransition,
		EventTypeSandboxOperation,
		EventTypeHeartbeatMiss,
		EventTypeCheckpointFailure,
		EventTypeRetryAttempt:
		return true
	}
	return false
}

// LifecyclePhase identifies a stable transition in the spawn-to-finalize
// pipeline. The set is closed; new entries require a contract decision in
// scenarios/agent-manager/docs/internal/SEAMS.md.
type LifecyclePhase string

const (
	LifecyclePhaseSpawnEnqueued     LifecyclePhase = "spawn_enqueued"
	LifecyclePhaseSpawnStarted      LifecyclePhase = "spawn_started"
	LifecyclePhaseRunnerAcquired    LifecyclePhase = "runner_acquired"
	LifecyclePhaseRunnerExited      LifecyclePhase = "runner_exited"
	LifecyclePhaseFinalizeStarted   LifecyclePhase = "finalize_started"
	LifecyclePhaseFinalizeCompleted LifecyclePhase = "finalize_completed"
)

// =============================================================================
// EVENT PAYLOAD INTERFACE (Tagged Union)
// =============================================================================

// EventPayload is the interface for all event-specific data.
// Each event type has a corresponding struct implementing this interface.
type EventPayload interface {
	// EventType returns the type of this payload for serialization.
	EventType() RunEventType

	// isEventPayload is a marker method to prevent external implementations.
	isEventPayload()
}

// =============================================================================
// TYPED OPERATIONAL EVENT
// =============================================================================

// TypedEventData carries a typed-operational payload as raw JSON bytes that
// already match the on-wire shape. It satisfies EventPayload so typed events
// can ride the existing run-event seam (Append, Stream, Get) without the
// store having to know each payload Go type.
//
// The eventlog package owns the payload struct definitions and the
// (event_type, schema_version) → Go-type dispatch; this struct is the
// transport between that package and the run-event plumbing.
type TypedEventData struct {
	// Type is the run-event type this payload belongs to. It must be one of
	// the typed-operational event categories (see RunEventType.IsTypedOperationalEvent).
	Type RunEventType `json:"-"`
	// Body is the already-marshaled JSON payload. MarshalJSON returns it
	// verbatim, and UnmarshalJSON stores the raw bytes back into it, so the
	// JSON round-trip preserves the eventlog-package payload shape exactly.
	Body json.RawMessage `json:"-"`
}

func (d *TypedEventData) EventType() RunEventType {
	if d == nil {
		return ""
	}
	return d.Type
}

func (d *TypedEventData) isEventPayload() {}

// MarshalJSON returns Body verbatim so the on-wire shape is the
// eventlog-package payload, not a wrapper.
func (d *TypedEventData) MarshalJSON() ([]byte, error) {
	if d == nil || len(d.Body) == 0 {
		return []byte("{}"), nil
	}
	return d.Body, nil
}

// UnmarshalJSON captures the raw bytes into Body for later typed decoding
// through the eventlog dispatch table.
func (d *TypedEventData) UnmarshalJSON(b []byte) error {
	d.Body = append(d.Body[:0], b...)
	return nil
}

// =============================================================================
// LOG EVENT
// =============================================================================

// LogEventData contains data for log events (debug, info, warn, error messages).
type LogEventData struct {
	Level   string `json:"level"`   // debug, info, warn, error
	Message string `json:"message"` // The log message
}

func (d *LogEventData) EventType() RunEventType { return EventTypeLog }
func (d *LogEventData) isEventPayload()         {}

// NewLogEvent creates a new log event.
func NewLogEvent(runID uuid.UUID, level, message string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeLog,
		Timestamp: time.Now(),
		Data:      &LogEventData{Level: level, Message: message},
	}
}

// =============================================================================
// MESSAGE EVENT
// =============================================================================

// MessageEventData contains data for conversation messages (user, assistant, system).
type MessageEventData struct {
	Role        string                  `json:"role"`                  // user, assistant, system
	Content     string                  `json:"content"`               // Message content
	Attachments []MessageAttachmentInfo `json:"attachments,omitempty"` // Image/file attachments
}

// MessageAttachmentInfo stores metadata about attachments included with a message.
// Used by the UI to render image thumbnails inline.
type MessageAttachmentInfo struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"` // Serving URL relative to API base
}

func (d *MessageEventData) EventType() RunEventType { return EventTypeMessage }
func (d *MessageEventData) isEventPayload()         {}

// NewMessageEvent creates a new message event.
func NewMessageEvent(runID uuid.UUID, role, content string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMessage,
		Timestamp: time.Now(),
		Data:      &MessageEventData{Role: role, Content: content},
	}
}

// NewMessageEventWithAttachments creates a message event that includes attachment metadata.
func NewMessageEventWithAttachments(runID uuid.UUID, role, content string, attachments []MessageAttachmentInfo) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMessage,
		Timestamp: time.Now(),
		Data:      &MessageEventData{Role: role, Content: content, Attachments: attachments},
	}
}

// =============================================================================
// MESSAGE DELETED EVENT
// =============================================================================

// MessageDeletedEventData marks a message event as deleted/redacted.
type MessageDeletedEventData struct {
	TargetEventID string `json:"targetEventId"`
}

func (d *MessageDeletedEventData) EventType() RunEventType { return EventTypeMessageDeleted }
func (d *MessageDeletedEventData) isEventPayload()         {}

// NewMessageDeletedEvent creates a new message deletion event.
func NewMessageDeletedEvent(runID uuid.UUID, targetEventID string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMessageDeleted,
		Timestamp: time.Now(),
		Data:      &MessageDeletedEventData{TargetEventID: targetEventID},
	}
}

// =============================================================================
// TOOL CALL EVENT
// =============================================================================

// ToolCallEventData contains data for tool invocation events.
type ToolCallEventData struct {
	ToolName   string                 `json:"toolName"`             // Name of the tool being called
	ToolCallID string                 `json:"toolCallId,omitempty"` // Correlation ID linking to the tool_result
	Input      map[string]interface{} `json:"input"`                // Tool input parameters
}

func (d *ToolCallEventData) EventType() RunEventType { return EventTypeToolCall }
func (d *ToolCallEventData) isEventPayload()         {}

// NewToolCallEvent creates a new tool call event.
// toolCallID is the correlation ID (e.g. "toolu_01GXZ...") that links this call to its result.
func NewToolCallEvent(runID uuid.UUID, toolName, toolCallID string, input map[string]interface{}) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeToolCall,
		Timestamp: time.Now(),
		Data:      &ToolCallEventData{ToolName: toolName, ToolCallID: toolCallID, Input: input},
	}
}

// =============================================================================
// TOOL RESULT EVENT
// =============================================================================

// ToolResultEventData contains data for tool result events.
type ToolResultEventData struct {
	ToolName   string `json:"toolName"`             // Display name of the tool (e.g., "Write", "bash")
	ToolCallID string `json:"toolCallId,omitempty"` // Tool invocation ID (e.g., "toolu_01GXZ...")
	Output     string `json:"output"`               // Tool output (success)
	Error      string `json:"error,omitempty"`      // Error message (if failed)
	Success    bool   `json:"success"`              // Whether the tool call succeeded
}

func (d *ToolResultEventData) EventType() RunEventType { return EventTypeToolResult }
func (d *ToolResultEventData) isEventPayload()         {}

// NewToolResultEvent creates a new tool result event.
// toolName is the display name (e.g., "Write"), toolCallID is the invocation ID.
func NewToolResultEvent(runID uuid.UUID, toolName, toolCallID, output string, err error) *RunEvent {
	data := &ToolResultEventData{
		ToolName:   toolName,
		ToolCallID: toolCallID,
		Output:     output,
		Success:    err == nil,
	}
	if err != nil {
		data.Error = err.Error()
	}
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeToolResult,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// =============================================================================
// STATUS EVENT
// =============================================================================

// StatusEventData contains data for status transition events.
type StatusEventData struct {
	OldStatus string `json:"oldStatus"`        // Previous status
	NewStatus string `json:"newStatus"`        // New status
	Reason    string `json:"reason,omitempty"` // Why the transition happened
}

func (d *StatusEventData) EventType() RunEventType { return EventTypeStatus }
func (d *StatusEventData) isEventPayload()         {}

// NewStatusEvent creates a new status change event.
func NewStatusEvent(runID uuid.UUID, oldStatus, newStatus, reason string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeStatus,
		Timestamp: time.Now(),
		Data:      &StatusEventData{OldStatus: oldStatus, NewStatus: newStatus, Reason: reason},
	}
}

// =============================================================================
// METRIC EVENT
// =============================================================================

// MetricEventData contains data for metric/telemetry events.
type MetricEventData struct {
	Name  string            `json:"name"`           // Metric name (e.g., "tokens_used")
	Value float64           `json:"value"`          // Metric value
	Unit  string            `json:"unit,omitempty"` // Unit (e.g., "tokens", "ms", "bytes")
	Tags  map[string]string `json:"tags,omitempty"` // Additional tags for grouping
}

func (d *MetricEventData) EventType() RunEventType { return EventTypeMetric }
func (d *MetricEventData) isEventPayload()         {}

// NewMetricEvent creates a new metric event.
func NewMetricEvent(runID uuid.UUID, name string, value float64, unit string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMetric,
		Timestamp: time.Now(),
		Data:      &MetricEventData{Name: name, Value: value, Unit: unit},
	}
}

// =============================================================================
// ARTIFACT EVENT
// =============================================================================

// ArtifactEventData contains data for artifact creation events.
type ArtifactEventData struct {
	Type     string `json:"type"`               // Artifact type (diff, log, screenshot, etc.)
	Path     string `json:"path"`               // Path to the artifact
	Size     int64  `json:"size,omitempty"`     // Size in bytes
	MimeType string `json:"mimeType,omitempty"` // MIME type
}

func (d *ArtifactEventData) EventType() RunEventType { return EventTypeArtifact }
func (d *ArtifactEventData) isEventPayload()         {}

// NewArtifactEvent creates a new artifact event.
func NewArtifactEvent(runID uuid.UUID, artifactType, path string, size int64) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeArtifact,
		Timestamp: time.Now(),
		Data:      &ArtifactEventData{Type: artifactType, Path: path, Size: size},
	}
}

// =============================================================================
// ERROR EVENT
// =============================================================================

// ErrorEventData contains data for error events.
type ErrorEventData struct {
	Code       string                 `json:"code"`                 // Machine-readable error code
	Message    string                 `json:"message"`              // Human-readable error message
	Retryable  bool                   `json:"retryable"`            // Whether the error is retryable
	Recovery   RecoveryAction         `json:"recovery,omitempty"`   // Suggested recovery action
	StackTrace string                 `json:"stackTrace,omitempty"` // Optional stack trace
	Details    map[string]interface{} `json:"details,omitempty"`    // Structured error details (e.g., conflicting sandboxes)
}

// =============================================================================
// RATE LIMIT EVENT
// =============================================================================

// RateLimitEventData contains data for rate limit events.
type RateLimitEventData struct {
	LimitType   string     `json:"limitType"`             // Type of limit: "5_hour", "daily", "weekly", "token"
	ResetTime   *time.Time `json:"resetTime,omitempty"`   // When the limit resets
	RetryAfter  int        `json:"retryAfter,omitempty"`  // Seconds until retry is safe
	CurrentUsed int        `json:"currentUsed,omitempty"` // Current usage count
	Limit       int        `json:"limit,omitempty"`       // The limit that was hit
	Message     string     `json:"message"`               // Human-readable message
}

func (d *RateLimitEventData) EventType() RunEventType { return EventTypeError }
func (d *RateLimitEventData) isEventPayload()         {}

// NewRateLimitEvent creates a new rate limit event.
func NewRateLimitEvent(runID uuid.UUID, limitType, message string, resetTime *time.Time, retryAfter int) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeError,
		Timestamp: time.Now(),
		Data: &RateLimitEventData{
			LimitType:  limitType,
			ResetTime:  resetTime,
			RetryAfter: retryAfter,
			Message:    message,
		},
	}
}

// =============================================================================
// COST EVENT
// =============================================================================

// CostEventData contains data for cost/usage tracking events.
type CostEventData struct {
	InputTokens           int        `json:"inputTokens"`
	OutputTokens          int        `json:"outputTokens"`
	CacheCreationTokens   int        `json:"cacheCreationTokens,omitempty"`
	CacheReadTokens       int        `json:"cacheReadTokens,omitempty"`
	InputCostUSD          float64    `json:"inputCostUsd,omitempty"`
	OutputCostUSD         float64    `json:"outputCostUsd,omitempty"`
	CacheCreationCostUSD  float64    `json:"cacheCreationCostUsd,omitempty"`
	CacheReadCostUSD      float64    `json:"cacheReadCostUsd,omitempty"`
	TotalCostUSD          float64    `json:"totalCostUsd"`
	ServiceTier           string     `json:"serviceTier,omitempty"` // e.g., "standard", "priority"
	Model                 string     `json:"model,omitempty"`
	CostSource            string     `json:"costSource,omitempty"`
	PricingProvider       string     `json:"pricingProvider,omitempty"`
	PricingModel          string     `json:"pricingModel,omitempty"`
	PricingFetchedAt      *time.Time `json:"pricingFetchedAt,omitempty"`
	PricingVersion        string     `json:"pricingVersion,omitempty"`
	WebSearchRequests     int        `json:"webSearchRequests,omitempty"`
	ServerToolUseRequests int        `json:"serverToolUseRequests,omitempty"`
}

func (d *CostEventData) EventType() RunEventType { return EventTypeMetric }
func (d *CostEventData) isEventPayload()         {}

// Cost source identifiers for cost provenance tracking.
const (
	CostSourceRunnerReported       = "runner_reported"
	CostSourceProviderUsageAPI     = "provider_usage_api"
	CostSourcePricingTableEstimate = "pricing_table_estimate"
	CostSourceUnknown              = "unknown"
)

// NewCostEvent creates a new cost tracking event.
func NewCostEvent(runID uuid.UUID, inputTokens, outputTokens int, costUSD float64) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMetric,
		Timestamp: time.Now(),
		Data: &CostEventData{
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			TotalCostUSD: costUSD,
			CostSource:   CostSourceUnknown,
		},
	}
}

// =============================================================================
// PROGRESS EVENT
// =============================================================================

// ProgressEventData contains data for progress tracking events.
type ProgressEventData struct {
	Phase              RunPhase `json:"phase"`
	PercentComplete    int      `json:"percentComplete"`
	CurrentAction      string   `json:"currentAction,omitempty"`
	TurnsCompleted     int      `json:"turnsCompleted,omitempty"`
	TurnsTotal         int      `json:"turnsTotal,omitempty"` // 0 means unlimited
	TokensUsed         int      `json:"tokensUsed,omitempty"`
	ElapsedSeconds     float64  `json:"elapsedSeconds,omitempty"`
	EstimatedRemaining float64  `json:"estimatedRemaining,omitempty"` // seconds, -1 if unknown
}

func (d *ProgressEventData) EventType() RunEventType { return EventTypeStatus }
func (d *ProgressEventData) isEventPayload()         {}

// NewProgressEvent creates a new progress tracking event.
func NewProgressEvent(runID uuid.UUID, phase RunPhase, percent int, action string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeStatus,
		Timestamp: time.Now(),
		Data: &ProgressEventData{
			Phase:           phase,
			PercentComplete: percent,
			CurrentAction:   action,
		},
	}
}

func (d *ErrorEventData) EventType() RunEventType { return EventTypeError }
func (d *ErrorEventData) isEventPayload()         {}

// NewErrorEvent creates a new error event.
func NewErrorEvent(runID uuid.UUID, code, message string, retryable bool) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeError,
		Timestamp: time.Now(),
		Data:      &ErrorEventData{Code: code, Message: message, Retryable: retryable},
	}
}

// NewErrorEventFromDomainError creates an error event from a DomainError.
func NewErrorEventFromDomainError(runID uuid.UUID, err DomainError) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeError,
		Timestamp: time.Now(),
		Data: &ErrorEventData{
			Code:      string(err.Code()),
			Message:   err.Error(),
			Retryable: err.Retryable(),
			Recovery:  err.Recovery(),
			Details:   err.Details(),
		},
	}
}

// =============================================================================
// COMPACTION EVENT
// =============================================================================

// CompactionEventData represents a context compaction/summarization event.
type CompactionEventData struct {
	Summary           string `json:"summary"`
	Trigger           string `json:"trigger"`         // "manual" or "auto"
	Focus             string `json:"focus,omitempty"` // Optional focus instruction
	MessagesCompacted int64  `json:"messagesCompacted"`
	TokensBefore      int64  `json:"tokensBefore"`
	TokensAfter       int64  `json:"tokensAfter"`
	OriginalCommand   string `json:"originalCommand,omitempty"`
}

func (d *CompactionEventData) EventType() RunEventType { return EventTypeCompaction }
func (d *CompactionEventData) isEventPayload()         {}

// NewCompactionEvent creates a new compaction event.
// trigger should be "manual" or "auto".
// focus is optional (empty string if not specified).
func NewCompactionEvent(
	runID uuid.UUID,
	summary string,
	trigger string,
	focus string,
	messagesCompacted int64,
	tokensBefore int64,
	tokensAfter int64,
	originalCommand string,
) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeCompaction,
		Timestamp: time.Now(),
		Data: &CompactionEventData{
			Summary:           summary,
			Trigger:           trigger,
			Focus:             focus,
			MessagesCompacted: messagesCompacted,
			TokensBefore:      tokensBefore,
			TokensAfter:       tokensAfter,
			OriginalCommand:   originalCommand,
		},
	}
}

// =============================================================================
// LIFECYCLE EVENT
// =============================================================================

// LifecycleEventData carries timing + classification fields for the
// stable spawn → finalize transitions enumerated by [LifecyclePhase].
//
// Lifecycle events are emitted only via the helpers in
// `internal/orchestration/obs/events.go`; ad-hoc construction outside
// that package is a contract violation (see SEAMS.md, contract decision
// "Lifecycle events are emitted through obs/events.go only").
type LifecycleEventData struct {
	Phase        LifecyclePhase `json:"phase"`
	Message      string         `json:"message,omitempty"`
	DurationMS   int64          `json:"durationMs,omitempty"`
	SandboxID    string         `json:"sandboxId,omitempty"`
	RunnerType   string         `json:"runnerType,omitempty"`
	LauncherType string         `json:"launcherType,omitempty"`
	QueueDepth   int            `json:"queueDepth,omitempty"`
	ActiveCount  int            `json:"activeCount,omitempty"`
	ExitCode     *int           `json:"exitCode,omitempty"`
	TerminalCode string         `json:"terminalCode,omitempty"`
}

func (d *LifecycleEventData) EventType() RunEventType { return EventTypeLifecycle }
func (d *LifecycleEventData) isEventPayload()         {}

// NewLifecycleEvent constructs a lifecycle event for the given phase.
// Caller-supplied data is shallow-copied so the event is safe to emit
// without sharing pointer state with the caller.
func NewLifecycleEvent(runID uuid.UUID, data LifecycleEventData) *RunEvent {
	d := data
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeLifecycle,
		Timestamp: time.Now(),
		Data:      &d,
	}
}

// =============================================================================
// LEGACY SUPPORT (RunEventData)
// =============================================================================
// RunEventData is kept for backward compatibility during migration.
// New code should use the specific event data types above.

// RunEventData contains the event-specific payload (DEPRECATED: use specific types).
// This struct is retained for JSON unmarshaling compatibility with existing data.
type RunEventData struct {
	// For log events
	Level   string `json:"level,omitempty"`
	Message string `json:"message,omitempty"`

	// For message events
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`

	// For tool_call and tool_result events
	ToolName   string                 `json:"toolName,omitempty"`
	ToolCallID string                 `json:"toolCallId,omitempty"` // Correlation ID (shared by tool_call and tool_result)
	ToolInput  map[string]interface{} `json:"toolInput,omitempty"`

	// For tool_result events
	ToolOutput string `json:"toolOutput,omitempty"`
	ToolError  string `json:"toolError,omitempty"`

	// For status events
	OldStatus string `json:"oldStatus,omitempty"`
	NewStatus string `json:"newStatus,omitempty"`

	// For metric events
	MetricName  string  `json:"metricName,omitempty"`
	MetricValue float64 `json:"metricValue,omitempty"`

	// For artifact events
	ArtifactType string `json:"artifactType,omitempty"`
	ArtifactPath string `json:"artifactPath,omitempty"`

	// For error events
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// Implement EventPayload interface for backward compatibility
func (d RunEventData) EventType() RunEventType {
	// Infer type from which fields are populated
	if d.Level != "" || (d.Message != "" && d.Role == "") {
		return EventTypeLog
	}
	if d.Role != "" {
		return EventTypeMessage
	}
	if d.ToolName != "" && d.ToolInput != nil {
		return EventTypeToolCall
	}
	if d.ToolOutput != "" || d.ToolError != "" {
		return EventTypeToolResult
	}
	if d.OldStatus != "" || d.NewStatus != "" {
		return EventTypeStatus
	}
	if d.MetricName != "" {
		return EventTypeMetric
	}
	if d.ArtifactType != "" {
		return EventTypeArtifact
	}
	if d.ErrorCode != "" || d.ErrorMessage != "" {
		return EventTypeError
	}
	return EventTypeLog // default fallback
}
func (d RunEventData) isEventPayload() {}

// ToTypedPayload converts legacy RunEventData to the appropriate typed payload.
func (d RunEventData) ToTypedPayload() EventPayload {
	switch d.EventType() {
	case EventTypeLog:
		return &LogEventData{Level: d.Level, Message: d.Message}
	case EventTypeMessage:
		return &MessageEventData{Role: d.Role, Content: d.Content}
	case EventTypeToolCall:
		return &ToolCallEventData{ToolName: d.ToolName, ToolCallID: d.ToolCallID, Input: d.ToolInput}
	case EventTypeToolResult:
		var err string
		if d.ToolError != "" {
			err = d.ToolError
		}
		return &ToolResultEventData{ToolName: d.ToolName, ToolCallID: d.ToolCallID, Output: d.ToolOutput, Error: err, Success: err == ""}
	case EventTypeStatus:
		return &StatusEventData{OldStatus: d.OldStatus, NewStatus: d.NewStatus}
	case EventTypeMetric:
		return &MetricEventData{Name: d.MetricName, Value: d.MetricValue}
	case EventTypeArtifact:
		return &ArtifactEventData{Type: d.ArtifactType, Path: d.ArtifactPath}
	case EventTypeError:
		return &ErrorEventData{Code: d.ErrorCode, Message: d.ErrorMessage}
	default:
		return &LogEventData{Message: d.Message}
	}
}

// -----------------------------------------------------------------------------
// Policy - Rules governing execution
// -----------------------------------------------------------------------------

// Policy defines rules for agent execution, approval, and resource access.
type Policy struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	Priority    int       `json:"priority" db:"priority"` // Higher priority wins

	// Scope matching
	ScopePattern string `json:"scopePattern,omitempty" db:"scope_pattern"` // Glob pattern

	// Execution rules
	Rules PolicyRules `json:"rules" db:"rules"`

	// Metadata
	CreatedBy string    `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
	Enabled   bool      `json:"enabled" db:"enabled"`
}

// PolicyRules contains the actual policy constraints.
type PolicyRules struct {
	// Sandbox requirements
	RequireSandbox *bool `json:"requireSandbox,omitempty"`
	AllowInPlace   *bool `json:"allowInPlace,omitempty"`

	// Approval requirements
	RequireApproval     *bool    `json:"requireApproval,omitempty"`
	AutoApprovePatterns []string `json:"autoApprovePatterns,omitempty"`

	// Concurrency limits
	MaxConcurrentRuns     *int `json:"maxConcurrentRuns,omitempty"`
	MaxConcurrentPerScope *int `json:"maxConcurrentPerScope,omitempty"`

	// Resource limits
	MaxFilesChanged    *int   `json:"maxFilesChanged,omitempty"`
	MaxTotalSizeBytes  *int64 `json:"maxTotalSizeBytes,omitempty"`
	MaxExecutionTimeMs *int64 `json:"maxExecutionTimeMs,omitempty"`

	// Runner restrictions
	AllowedRunners []RunnerType `json:"allowedRunners,omitempty"`
	DeniedRunners  []RunnerType `json:"deniedRunners,omitempty"`
}

// -----------------------------------------------------------------------------
// ScopeLock - Concurrency control
// -----------------------------------------------------------------------------

// ScopeLock represents an exclusive lock on a path scope.
type ScopeLock struct {
	ID          uuid.UUID `json:"id" db:"id"`
	RunID       uuid.UUID `json:"runId" db:"run_id"`
	ScopePath   string    `json:"scopePath" db:"scope_path"`
	ProjectRoot string    `json:"projectRoot" db:"project_root"`
	AcquiredAt  time.Time `json:"acquiredAt" db:"acquired_at"`
	ExpiresAt   time.Time `json:"expiresAt" db:"expires_at"`
}

// =============================================================================
// IDEMPOTENCY & REPLAY SAFETY
// =============================================================================
// These types enable safe retries, resumption, and replay of operations.
// See: idempotency-replay-safety-hardening.md

// IdempotencyRecord tracks whether an operation has been performed.
// This prevents duplicate work when operations are retried.
type IdempotencyRecord struct {
	// Key uniquely identifies the operation (e.g., "run-create:task-{taskID}:profile-{profileID}:ts-{timestamp}")
	Key string `json:"key" db:"key"`

	// Status indicates the operation outcome
	Status IdempotencyStatus `json:"status" db:"status"`

	// EntityID is the ID of the created/affected entity (if applicable)
	EntityID *uuid.UUID `json:"entityId,omitempty" db:"entity_id"`

	// EntityType identifies what was created (e.g., "Run", "Task")
	EntityType string `json:"entityType,omitempty" db:"entity_type"`

	// CreatedAt is when this record was created
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// ExpiresAt is when this record can be garbage collected
	ExpiresAt time.Time `json:"expiresAt" db:"expires_at"`

	// Response contains the cached response (JSON) for successful operations
	Response []byte `json:"response,omitempty" db:"response"`
}

// IdempotencyStatus indicates the state of an idempotent operation.
type IdempotencyStatus string

const (
	// IdempotencyStatusPending - Operation started but not completed
	IdempotencyStatusPending IdempotencyStatus = "pending"

	// IdempotencyStatusComplete - Operation completed successfully
	IdempotencyStatusComplete IdempotencyStatus = "complete"

	// IdempotencyStatusFailed - Operation failed (may be retried)
	IdempotencyStatusFailed IdempotencyStatus = "failed"
)

// =============================================================================
// PROGRESS & CHECKPOINT TRACKING
// =============================================================================
// These types enable safe interruption and resumption of runs.
// See: progress-continuity-interruption-resilience.md

// RunPhase represents the current phase of run execution.
// This enables resumption from the correct point after interruption.
type RunPhase string

const (
	// RunPhaseQueued - Run created but not started
	RunPhaseQueued RunPhase = "queued"

	// RunPhaseInitializing - Setting up workspace and acquiring resources
	RunPhaseInitializing RunPhase = "initializing"

	// RunPhaseSandboxCreating - Creating sandbox (if sandboxed mode)
	RunPhaseSandboxCreating RunPhase = "sandbox_creating"

	// RunPhaseRunnerAcquiring - Acquiring and validating runner
	RunPhaseRunnerAcquiring RunPhase = "runner_acquiring"

	// RunPhaseExecuting - Agent is actively executing
	RunPhaseExecuting RunPhase = "executing"

	// RunPhaseCollectingResults - Gathering results and artifacts
	RunPhaseCollectingResults RunPhase = "collecting_results"

	// RunPhaseAwaitingReview - Execution complete, awaiting approval
	RunPhaseAwaitingReview RunPhase = "awaiting_review"

	// RunPhaseApplying - Applying approved changes
	RunPhaseApplying RunPhase = "applying"

	// RunPhaseCleaningUp - Releasing resources and cleaning up
	RunPhaseCleaningUp RunPhase = "cleaning_up"

	// RunPhaseCompleted - Run is finished (terminal)
	RunPhaseCompleted RunPhase = "completed"
)

// CanResumeFromPhase returns whether a run can be resumed from this phase.
func (p RunPhase) CanResumeFromPhase() bool {
	switch p {
	case RunPhaseQueued, RunPhaseInitializing, RunPhaseSandboxCreating,
		RunPhaseRunnerAcquiring, RunPhaseExecuting:
		return true
	default:
		return false
	}
}

// IsTerminal returns whether this phase represents a completed run.
func (p RunPhase) IsTerminal() bool {
	return p == RunPhaseCompleted
}

// RunCheckpoint captures the state needed to resume a run.
type RunCheckpoint struct {
	// RunID is the run this checkpoint belongs to
	RunID uuid.UUID `json:"runId" db:"run_id"`

	// Phase is the current execution phase
	Phase RunPhase `json:"phase" db:"phase"`

	// StepWithinPhase tracks progress within a phase (0-indexed)
	StepWithinPhase int `json:"stepWithinPhase" db:"step_within_phase"`

	// SandboxID is set after sandbox creation
	SandboxID *uuid.UUID `json:"sandboxId,omitempty" db:"sandbox_id"`

	// WorkDir is set after workspace setup
	WorkDir string `json:"workDir,omitempty" db:"work_dir"`

	// LockID is set after acquiring scope lock
	LockID *uuid.UUID `json:"lockId,omitempty" db:"lock_id"`

	// LastEventSequence is the last event sequence number persisted
	LastEventSequence int64 `json:"lastEventSequence" db:"last_event_sequence"`

	// LastHeartbeat is when we last confirmed progress
	LastHeartbeat time.Time `json:"lastHeartbeat" db:"last_heartbeat"`

	// RetryCount tracks how many times this phase has been retried
	RetryCount int `json:"retryCount" db:"retry_count"`

	// SavedAt is when this checkpoint was created
	SavedAt time.Time `json:"savedAt" db:"saved_at"`

	// Metadata contains phase-specific state that may be needed for resumption
	Metadata map[string]string `json:"metadata,omitempty" db:"metadata"`
}

// NewCheckpoint creates a checkpoint for the current run state.
func NewCheckpoint(runID uuid.UUID, phase RunPhase) *RunCheckpoint {
	now := time.Now()
	return &RunCheckpoint{
		RunID:         runID,
		Phase:         phase,
		LastHeartbeat: now,
		SavedAt:       now,
		Metadata:      make(map[string]string),
	}
}

// Update creates an updated checkpoint with new phase information.
func (c *RunCheckpoint) Update(phase RunPhase, step int) *RunCheckpoint {
	now := time.Now()
	return &RunCheckpoint{
		RunID:             c.RunID,
		Phase:             phase,
		StepWithinPhase:   step,
		SandboxID:         c.SandboxID,
		WorkDir:           c.WorkDir,
		LockID:            c.LockID,
		LastEventSequence: c.LastEventSequence,
		LastHeartbeat:     now,
		RetryCount:        c.RetryCount,
		SavedAt:           now,
		Metadata:          c.Metadata,
	}
}

// WithSandbox adds sandbox information to the checkpoint.
func (c *RunCheckpoint) WithSandbox(sandboxID uuid.UUID, workDir string) *RunCheckpoint {
	cp := *c
	cp.SandboxID = &sandboxID
	cp.WorkDir = workDir
	cp.SavedAt = time.Now()
	return &cp
}

// WithLock adds lock information to the checkpoint.
func (c *RunCheckpoint) WithLock(lockID uuid.UUID) *RunCheckpoint {
	cp := *c
	cp.LockID = &lockID
	cp.SavedAt = time.Now()
	return &cp
}

// WithEventSequence updates the last persisted event sequence.
func (c *RunCheckpoint) WithEventSequence(seq int64) *RunCheckpoint {
	cp := *c
	cp.LastEventSequence = seq
	cp.SavedAt = time.Now()
	return &cp
}

// IncrementRetry increments the retry count for the current phase.
func (c *RunCheckpoint) IncrementRetry() *RunCheckpoint {
	cp := *c
	cp.RetryCount++
	cp.SavedAt = time.Now()
	return &cp
}

// =============================================================================
// TEMPORAL FLOW & HEARTBEAT
// =============================================================================
// These types support time-based coordination and health monitoring.
// See: temporal-flow-audit.md

// HeartbeatConfig defines heartbeat behavior for long-running operations.
type HeartbeatConfig struct {
	// Interval is how often to send heartbeats
	Interval time.Duration `json:"interval"`

	// Timeout is how long without a heartbeat before considering dead
	Timeout time.Duration `json:"timeout"`

	// MaxMissedBeats is the number of missed heartbeats before termination
	MaxMissedBeats int `json:"maxMissedBeats"`
}

// DefaultHeartbeatConfig returns sensible defaults for heartbeat monitoring.
func DefaultHeartbeatConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Interval:       30 * time.Second,
		Timeout:        2 * time.Minute,
		MaxMissedBeats: 3,
	}
}

// RunProgress represents the current progress of a run for display.
type RunProgress struct {
	// Phase is the current execution phase
	Phase RunPhase `json:"phase"`

	// PhaseDescription is a human-readable description
	PhaseDescription string `json:"phaseDescription"`

	// PercentComplete is an estimate of overall progress (0-100)
	PercentComplete int `json:"percentComplete"`

	// CurrentAction describes what's happening now
	CurrentAction string `json:"currentAction,omitempty"`

	// ElapsedTime is how long the run has been active
	ElapsedTime time.Duration `json:"elapsedTime"`

	// EstimatedRemaining is an estimate of time left (if known)
	EstimatedRemaining *time.Duration `json:"estimatedRemaining,omitempty"`

	// LastUpdate is when progress was last reported
	LastUpdate time.Time `json:"lastUpdate"`
}

// PhaseToProgress converts a phase to approximate progress percentage.
func PhaseToProgress(phase RunPhase) int {
	switch phase {
	case RunPhaseQueued:
		return 0
	case RunPhaseInitializing:
		return 5
	case RunPhaseSandboxCreating:
		return 15
	case RunPhaseRunnerAcquiring:
		return 25
	case RunPhaseExecuting:
		return 50 // This phase takes most of the time
	case RunPhaseCollectingResults:
		return 85
	case RunPhaseAwaitingReview:
		return 90
	case RunPhaseApplying:
		return 95
	case RunPhaseCleaningUp:
		return 98
	case RunPhaseCompleted:
		return 100
	default:
		return 0
	}
}

// PhaseDescription returns a human-readable description of the phase.
func (p RunPhase) Description() string {
	switch p {
	case RunPhaseQueued:
		return "Waiting to start"
	case RunPhaseInitializing:
		return "Initializing execution environment"
	case RunPhaseSandboxCreating:
		return "Creating isolated workspace"
	case RunPhaseRunnerAcquiring:
		return "Acquiring agent runner"
	case RunPhaseExecuting:
		return "Agent is executing"
	case RunPhaseCollectingResults:
		return "Collecting results and artifacts"
	case RunPhaseAwaitingReview:
		return "Awaiting approval"
	case RunPhaseApplying:
		return "Applying approved changes"
	case RunPhaseCleaningUp:
		return "Cleaning up resources"
	case RunPhaseCompleted:
		return "Completed"
	default:
		return "Unknown phase"
	}
}
