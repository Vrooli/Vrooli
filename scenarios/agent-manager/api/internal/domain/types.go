// Package domain contains the central concepts that agent-manager operates on:
// - AgentProfile: defines HOW an agent runs (portable role, permissions)
// - Task: defines WHAT needs to be done (scope, context, requirements)
// - Run: a concrete execution linking Task to AgentProfile within a sandbox
// - RunEvent: append-only event stream capturing all agent activity
// - Policy: rules governing execution, approval, and resource access
package domain

import (
	"github.com/google/uuid"
	"strings"
	"time"
)

// AgentProfile defines the configuration for running an agent.
// This is a reusable definition that can be applied to many tasks.
type AgentProfile struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	ProfileKey  string    `json:"profileKey" db:"profile_key"`
	Description string    `json:"description,omitempty" db:"description"`

	// RoleRef is portable desired intent. Concrete runner/model selections are
	// captured only in a run's immutable PolicySnapshot.
	RoleRef string `json:"roleRef,omitempty" db:"role_ref"`
	// RoleReason is declaration-only context explaining why the flagship role
	// is appropriate. It is intentionally not part of the persisted execution
	// contract; policy snapshots remain the source of run provenance.
	RoleReason string        `json:"roleReason,omitempty" db:"-"`
	MaxTurns   int           `json:"maxTurns,omitempty" db:"max_turns"`
	Timeout    time.Duration `json:"timeout,omitempty" db:"timeout_ms"`
	Effort     Effort        `json:"effort,omitempty" db:"effort"`

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
	SpawnPolicy   *SpawnPolicy   `json:"spawnPolicy,omitempty" db:"-"`

	// Path restrictions
	AllowedPaths []string `json:"allowedPaths,omitempty" db:"allowed_paths"`
	DeniedPaths  []string `json:"deniedPaths,omitempty" db:"denied_paths"`
	// DeclaredScopes is the profile ceiling for delegated identity tokens.
	// Omitted and empty lists both grant nothing. Use IdentityScopeCeiling
	// when intersecting grants so transport nil/empty differences cannot
	// silently restore the account's full authority.
	DeclaredScopes []string `json:"declaredScopes,omitempty" db:"declared_scopes"`
	// SkillPack names the prompt-manager skills projected into this profile's
	// private run scope. It is an allow-list of identifiers, never a path.
	SkillPack         []string `json:"skillPack,omitempty" db:"skill_pack"`
	SkillExperimentID string   `json:"skillExperimentId,omitempty" db:"skill_experiment_id"`

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

// IdentityScopeCeiling distinguishes an absent profile (no profile layer) from
// an existing profile with no declared API capability (an empty ceiling).
// Return a copy so narrowing a request cannot mutate the profile's policy.
func (p *AgentProfile) IdentityScopeCeiling() []string {
	if p == nil {
		return nil
	}
	return append([]string{}, p.DeclaredScopes...)
}

// SpawnPolicy is declaration-only preference data. Runtime selection must
// intersect it with runner-published capabilities before choosing a substrate.
type SpawnPolicy struct {
	AxisOrder     []string           `json:"axisOrder,omitempty"`
	ExecutionMode PreferenceAxis     `json:"executionMode"`
	SandboxMode   PreferenceAxis     `json:"sandboxMode"`
	Require       []SpawnCombination `json:"require,omitempty"`
}

type PreferenceAxis struct {
	Prefer []string `json:"prefer"`
}
type SpawnCombination struct {
	ExecutionMode string `json:"executionMode"`
	SandboxMode   string `json:"sandboxMode"`
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
	RunnerTypeClaudeCode  RunnerType = "claude-code"
	RunnerTypeCodex       RunnerType = "codex"
	RunnerTypeOpenCode    RunnerType = "opencode"
	RunnerTypeGrok        RunnerType = "grok"
	RunnerTypeAntigravity RunnerType = "antigravity"
)

// ValidRunnerTypes returns all valid runner types.
func ValidRunnerTypes() []RunnerType {
	return []RunnerType{
		RunnerTypeClaudeCode, RunnerTypeCodex, RunnerTypeOpenCode,
		RunnerTypeGrok, RunnerTypeAntigravity,
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

// ChallengerConfig describes an optional sampled model comparison. It is
// copied into the immutable execution snapshot; mutable catalog state is
// never consulted while a run executes or resumes.
type ChallengerConfig struct {
	Model      string  `json:"model"`
	SampleRate float64 `json:"sample_rate"`
}

// ExecutionCandidate is one immutable runner/model attempt in resolved order.
type ExecutionCandidate struct {
	DeclaredEffort       Effort                `json:"declaredEffort,omitempty"`
	RunnerType           RunnerType            `json:"runnerType"`
	SelectionType        ModelSelectionType    `json:"selectionType"`
	Model                string                `json:"model,omitempty"`
	CanonicalModel       string                `json:"canonicalModel,omitempty"`
	ResourceRole         string                `json:"resourceRole,omitempty"`
	Fallbacks            []string              `json:"fallbacks,omitempty"`
	ExcludedModels       []string              `json:"excludedModels,omitempty"`
	Available            bool                  `json:"available"`
	FailureCode          string                `json:"failureCode,omitempty"`
	Failure              string                `json:"failure,omitempty"`
	Provenance           ResourceProvenance    `json:"provenance,omitempty"`
	Enforcement          PermissionEnforcement `json:"enforcement,omitempty"`
	PolicyPath           string                `json:"policyPath,omitempty"`
	PolicyDigest         string                `json:"policyDigest,omitempty"`
	Billing              BillingSnapshot       `json:"billing,omitempty"`
	ChallengerModel      string                `json:"challengerModel,omitempty"`
	ChallengerSampleRate float64               `json:"challengerSampleRate,omitempty"`
	CanaryArm            string                `json:"canaryArm,omitempty"`
}

type BillingMode string

const (
	BillingModeMetered      BillingMode = "metered"
	BillingModeSubscription BillingMode = "subscription"
	BillingModeLocal        BillingMode = "local"
	BillingModeUnknown      BillingMode = "unknown"
)

// BillingSnapshot is stamped at run creation and never re-read from mutable
// resource policy. It preserves the accounting context used by a run.
type BillingSnapshot struct {
	Basis              ChargeBasis `json:"basis,omitempty"`
	Mode               BillingMode `json:"mode"`
	Provider           string      `json:"provider,omitempty"`
	AccountRef         string      `json:"account_ref,omitempty"`
	PlanRef            string      `json:"plan_ref,omitempty"`
	PlanID             string      `json:"plan_id,omitempty"`
	PlanLabel          string      `json:"plan_label,omitempty"`
	QuotaWindow        string      `json:"quota_window,omitempty"`
	SubscriptionPeriod string      `json:"subscription_period,omitempty"`
	ObservedAt         time.Time   `json:"observed_at,omitempty"`
	Source             string      `json:"source,omitempty"`
	PolicyDigest       string      `json:"policy_digest,omitempty"`
}

func (b BillingSnapshot) EffectiveBasis() ChargeBasis {
	if b.Basis != "" {
		return b.Basis
	}
	switch b.Mode {
	case BillingModeMetered:
		return ChargeBasisMetered
	case BillingModeSubscription:
		return ChargeBasisSubscription
	case BillingModeLocal:
		return ChargeBasisLocal
	default:
		return ChargeBasisUnknown
	}
}

type SubscriptionPeriod struct {
	ID             string    `json:"id"`
	Provider       string    `json:"provider"`
	PlanRef        string    `json:"planRef"`
	StartsAt       time.Time `json:"startsAt"`
	EndsAt         time.Time `json:"endsAt"`
	AmountMicroUSD int64     `json:"amountMicroUsd"`
	QuotaTokens    int64     `json:"quotaTokens,omitempty"`
}

type WorkloadRef struct {
	Kind     WorkloadKind `json:"kind"`
	Key      string       `json:"key"`
	Instance string       `json:"instance,omitempty"`
}

type WorkloadKind string

const (
	WorkloadKindWorkflowNode WorkloadKind = "workflow_node"
	WorkloadKindScheduled    WorkloadKind = "scheduled"
	WorkloadKindInteractive  WorkloadKind = "interactive"
	WorkloadKindAdhoc        WorkloadKind = "adhoc"
	WorkloadKindImported     WorkloadKind = "imported"
)

func (k WorkloadKind) IsValid() bool {
	switch k {
	case WorkloadKindWorkflowNode, WorkloadKindScheduled, WorkloadKindInteractive, WorkloadKindAdhoc, WorkloadKindImported:
		return true
	default:
		return false
	}
}

// WorkloadFromHistoricalTag recovers only the unambiguous workflow tag shape
// used before workload identity was persisted separately.
func WorkloadFromHistoricalTag(tag string) (WorkloadRef, bool) {
	const prefix = "workflow-"
	if strings.HasPrefix(tag, "agent-manager-imported-") {
		return WorkloadRef{Kind: WorkloadKindImported, Key: strings.TrimPrefix(tag, "agent-manager-imported-")}, true
	}
	if !strings.HasPrefix(tag, prefix) {
		return WorkloadRef{}, false
	}
	remainder := strings.TrimPrefix(tag, prefix)
	if len(remainder) < 36 || remainder[36] != '-' {
		return WorkloadRef{}, false
	}
	instance, err := uuid.Parse(remainder[:36])
	if err != nil || remainder[37:] == "" {
		return WorkloadRef{}, false
	}
	return WorkloadRef{Kind: WorkloadKindWorkflowNode, Key: remainder[37:], Instance: instance.String()}, true
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
	SelectionReason   string                      `json:"selectionReason,omitempty"`
	CanaryArm         string                      `json:"canaryArm,omitempty"`
}

type ExecutionPreferences struct {
	PreferredRunner string `json:"preferredRunner,omitempty"`
	Model           string `json:"model,omitempty"`
	Effort          string `json:"effort,omitempty"`
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
	// merged-overlay output. This is the production default. Launch requires
	// a bound SandboxLauncherFactory and a current containment report;
	// missing wiring or enforcement refuses launch, never falls back to host.
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
	// WritePolicy is a runtime workspace grant, separate from apply acceptance.
	// Nil preserves the full workspace; an explicit empty policy is read-only.
	WritePolicy *WorkspaceWritePolicy `json:"writePolicy,omitempty"`

	// Mode selects the auditability mode. Empty defaults to "protected".
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

// WorkspaceWritePolicy grants existing literal paths relative to the sandbox's
// merged root. Directories include descendants. The sandbox owner rejects root,
// traversal, globs, overlapping grants, missing paths and symlink components.
type WorkspaceWritePolicy struct {
	Paths []string `json:"paths"`
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
