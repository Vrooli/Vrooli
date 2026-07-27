// Package domain contains the central concepts that agent-manager operates on:
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
	Effort   Effort        `json:"effort,omitempty" db:"effort"`

	// Tool permissions
	AllowedTools          []string              `json:"allowedTools,omitempty" db:"allowed_tools"`
	DeniedTools           []string              `json:"deniedTools,omitempty" db:"denied_tools"`
	ToolRestrictionPolicy ToolRestrictionPolicy `json:"toolRestrictionPolicy,omitempty" db:"tool_restriction_policy"`

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

// CanonicalTool is the runner-neutral vocabulary used by profile tool
// restrictions. Codecs translate these coarse capabilities to their native
// command names at launch time.
type CanonicalTool string

const (
	CanonicalToolRead      CanonicalTool = "read"
	CanonicalToolWrite     CanonicalTool = "write"
	CanonicalToolEdit      CanonicalTool = "edit"
	CanonicalToolGlob      CanonicalTool = "glob"
	CanonicalToolGrep      CanonicalTool = "grep"
	CanonicalToolShell     CanonicalTool = "shell"
	CanonicalToolWebSearch CanonicalTool = "web_search"
	CanonicalToolWebFetch  CanonicalTool = "web_fetch"
)

// CanonicalTools returns the complete, stable profile tool vocabulary.
func CanonicalTools() []CanonicalTool {
	return []CanonicalTool{
		CanonicalToolRead, CanonicalToolWrite, CanonicalToolEdit, CanonicalToolGlob,
		CanonicalToolGrep, CanonicalToolShell, CanonicalToolWebSearch, CanonicalToolWebFetch,
	}
}

// ToolRestrictionPolicy determines what happens when the selected runner
// cannot enforce a non-empty allowedTools restriction.
type ToolRestrictionPolicy string

const (
	ToolRestrictionPolicyEnforced ToolRestrictionPolicy = "enforced"
	ToolRestrictionPolicyAdvisory ToolRestrictionPolicy = "advisory"
)

func (p ToolRestrictionPolicy) IsValid() bool {
	return p == "" || p == ToolRestrictionPolicyEnforced || p == ToolRestrictionPolicyAdvisory
}

// Effort is the canonical reasoning-effort scale shared by runner codecs.
// Empty leaves the runner default unchanged.
type Effort string

const (
	EffortLow    Effort = "low"
	EffortMedium Effort = "medium"
	EffortHigh   Effort = "high"
	EffortXHigh  Effort = "xhigh"
	EffortMax    Effort = "max"
)

func (e Effort) IsValid() bool {
	switch e {
	case "", EffortLow, EffortMedium, EffortHigh, EffortXHigh, EffortMax:
		return true
	default:
		return false
	}
}

func (p ToolRestrictionPolicy) Effective() ToolRestrictionPolicy {
	if p == "" {
		return ToolRestrictionPolicyEnforced
	}
	return p
}

// IsValid reports whether the tool belongs to the canonical profile vocabulary.
func (t CanonicalTool) IsValid() bool {
	for _, valid := range CanonicalTools() {
		if t == valid {
			return true
		}
	}
	return false
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
	// explicitly. Treated as SandboxModeProtected by [SandboxMode.Effective]
	// — the safe routing target for code paths that construct a
	// zero-valued SandboxConfig directly. Spawn surfaces should clone
	// [DefaultSandboxConfig] (which sets Mode=Protected) instead of
	// zero-initialising, so the unspecified→protected fallback only fires
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

// Effective returns the mode value, defaulting empty to SandboxModeProtected.
// SandboxModeOff is preserved as-is (it is an explicit, intentional choice).
func (m SandboxMode) Effective() SandboxMode {
	if m == SandboxModeUnspecified {
		return SandboxModeProtected
	}
	return m
}

// strictnessRank orders sandbox modes from least to most strict, so a
// "minimum-mode" policy can be expressed as a numeric ≥ comparison.
// SandboxModeUnspecified is treated as Protected (matches Effective).
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
	// Result is the canonical, provenance-bearing terminal output projection.
	// Summary is a compatibility view derived from Result and must never be
	// independently authored for new runner completions.
	Result   *RunResult  `json:"result,omitempty" db:"run_result"`
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
	CommitHash     string `json:"commitHash,omitempty" db:"commit_hash"`

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

// FinalOutputSelectionStatus describes whether terminal evidence identifies a
// unique final assistant handoff. Historical runs may have no RunResult at all;
// a present result always carries one of these explicit outcomes.
type FinalOutputSelectionStatus string

const (
	FinalOutputSelectionSelected    FinalOutputSelectionStatus = "selected"
	FinalOutputSelectionAmbiguous   FinalOutputSelectionStatus = "ambiguous"
	FinalOutputSelectionUnavailable FinalOutputSelectionStatus = "unavailable"
)

// FinalOutputCandidate is an immutable projection of one assistant message
// considered by the final-output resolver.
type FinalOutputCandidate struct {
	ID                string `json:"id"`
	EventID           string `json:"eventId,omitempty"`
	Sequence          int64  `json:"sequence,omitempty"`
	Content           string `json:"content"`
	MessageID         string `json:"messageId,omitempty"`
	ConversationID    string `json:"conversationId,omitempty"`
	TurnID            string `json:"turnId,omitempty"`
	ProviderOrigin    string `json:"providerOrigin,omitempty"`
	CompletionReason  string `json:"completionReason,omitempty"`
	Terminal          bool   `json:"terminal,omitempty"`
	ParentMessageID   string `json:"parentMessageId,omitempty"`
	ProviderEventType string `json:"providerEventType,omitempty"`
	RawEvidenceRef    string `json:"rawEvidenceRef,omitempty"`
	EvidenceTier      int    `json:"evidenceTier"`
}

// FinalOutputSelection records the deterministic resolver decision and the
// exact rule/version needed to explain or reproduce it.
type FinalOutputSelection struct {
	Status              FinalOutputSelectionStatus `json:"status"`
	SelectedCandidateID string                     `json:"selectedCandidateId,omitempty"`
	Rule                string                     `json:"rule"`
	AlgorithmVersion    string                     `json:"algorithmVersion"`
	Evidence            []string                   `json:"evidence,omitempty"`
}

// RunResult is the canonical terminal result for one execute or continue turn.
// It intentionally remains useful when selection is ambiguous/unavailable.
type RunResult struct {
	FinalOutput    string                 `json:"finalOutput,omitempty"`
	Selection      FinalOutputSelection   `json:"selection"`
	Candidates     []FinalOutputCandidate `json:"candidates,omitempty"`
	Success        bool                   `json:"success"`
	ExitCode       int                    `json:"exitCode"`
	TerminalReason string                 `json:"terminalReason,omitempty"`
	Structured     *StructuredResult      `json:"structured,omitempty"`
}

// ResultSpecKind selects the one canonical typed-result contract. Enum
// classification is represented as a schema-shaped ResultSpec rather than a
// separate classifier persistence model.
type ResultSpecKind string

const (
	ResultSpecKindNone           ResultSpecKind = "none"
	ResultSpecKindJSONSchema     ResultSpecKind = "json_schema"
	ResultSpecKindClassification ResultSpecKind = "classification"
)

// StructuredExtractionMode controls whether deterministic parsing may fall
// back to the portable extraction seam. The fallback is never trusted without
// the same local schema validation as deterministic candidates.
type StructuredExtractionMode string

const (
	StructuredExtractionDeterministic StructuredExtractionMode = "deterministic_only"
	StructuredExtractionConstrained   StructuredExtractionMode = "constrained_fallback"
)

// ResultSpec is the versioned request for a typed result. Schema contains
// canonical JSON bytes after creation-time normalization. ClassificationValues
// is a create-surface convenience that is compiled into Schema and then
// cleared, keeping Schema as the sole persisted validation authority.
type ResultSpec struct {
	Version              string                   `json:"version"`
	Kind                 ResultSpecKind           `json:"kind"`
	Schema               json.RawMessage          `json:"schema,omitempty"`
	SchemaDigest         string                   `json:"schemaDigest,omitempty"`
	ClassificationValues []string                 `json:"classificationValues,omitempty"`
	ExtractionMode       StructuredExtractionMode `json:"extractionMode,omitempty"`
	ExtractionRole       string                   `json:"extractionRole,omitempty"`
	// SchemaRepairAttempts is nil for the safe workflow default of one repair,
	// zero to disable repair, and one to request the single bounded correction.
	SchemaRepairAttempts *int `json:"schemaRepairAttempts,omitempty"`
}

// StructuredResultStatus separates all honest terminal outcomes. Only
// StructuredResultSuccess may carry Value.
type StructuredResultStatus string

const (
	StructuredResultSuccess     StructuredResultStatus = "success"
	StructuredResultUnavailable StructuredResultStatus = "unavailable"
	StructuredResultInvalid     StructuredResultStatus = "invalid"
	StructuredResultAmbiguous   StructuredResultStatus = "ambiguous"
	StructuredResultAbstained   StructuredResultStatus = "abstained"
)

// StructuredDiagnostic is bounded, normalized, and safe to expose. It never
// includes source output or schema fragments, which prevents secret-bearing
// agent text from leaking through validation errors.
type StructuredDiagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path,omitempty"`
	Message string `json:"message"`
}

// StructuredExtractionProvenance explains how a fallback candidate was
// produced without making provider output authoritative.
type StructuredExtractionProvenance struct {
	RoleRef        string                   `json:"roleRef,omitempty"`
	Provider       string                   `json:"provider,omitempty"`
	Model          string                   `json:"model,omitempty"`
	PolicySnapshot *ExecutionPolicySnapshot `json:"policySnapshot,omitempty"`
}

// StructuredResult is the locally validated typed projection attached to the
// canonical RunResult. Requested schema digest, method, and diagnostics remain
// durable even when resolution abstains or fails.
type StructuredResult struct {
	Status            StructuredResultStatus          `json:"status"`
	SpecKind          ResultSpecKind                  `json:"specKind"`
	SchemaDigest      string                          `json:"schemaDigest"`
	Value             json.RawMessage                 `json:"value,omitempty"`
	Method            string                          `json:"method,omitempty"`
	SourceCandidateID string                          `json:"sourceCandidateId,omitempty"`
	Extractor         *StructuredExtractionProvenance `json:"extractor,omitempty"`
	Diagnostics       []StructuredDiagnostic          `json:"diagnostics,omitempty"`
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
	Effort     Effort        `json:"effort,omitempty"`

	// PolicySnapshot pins the exact active catalog revision and ordered
	// candidate sequence selected before this run was persisted.
	PolicySnapshot *ExecutionPolicySnapshot `json:"policySnapshot,omitempty"`

	// ResultSpec is normalized before the run is persisted. Nil/none preserves
	// the historical unstructured behavior.
	ResultSpec *ResultSpec `json:"resultSpec,omitempty"`

	// Tool permissions
	AllowedTools          []string              `json:"allowedTools,omitempty"`
	DeniedTools           []string              `json:"deniedTools,omitempty"`
	ToolRestrictionPolicy ToolRestrictionPolicy `json:"toolRestrictionPolicy,omitempty"`

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
	c.Effort = profile.Effort
	c.AllowedTools = profile.AllowedTools
	c.DeniedTools = profile.DeniedTools
	c.ToolRestrictionPolicy = profile.ToolRestrictionPolicy.Effective()
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
		RunnerType:            RunnerTypeClaudeCode,
		MaxTurns:              30,
		Timeout:               60 * time.Minute,
		NetworkAccess:         NetworkAccessLocalhost,
		ToolRestrictionPolicy: ToolRestrictionPolicyEnforced,
		SandboxConfig:         DefaultSandboxConfig(),
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
