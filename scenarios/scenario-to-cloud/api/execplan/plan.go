// Package execplan compiles one typed executable plan that preview, policy
// and apply all consume. A plan is an action graph: each action names an
// owner operation, typed inputs, prerequisites, effects, verification, retry
// and recovery. A shell command is never the semantic unit; the shell strings
// a renderer derives for display are a convenience and are never executed.
//
// Plan identity is the semantic digest: sha256 over the canonical JSON of the
// envelope without the presentation block. Changing a title keeps the digest;
// changing the target, the artifact, a privilege need or a destructive effect
// changes it and therefore invalidates any prior review.
package execplan

// SchemaVersion is the wire shape of Plan. Bump when a field changes meaning.
const SchemaVersion = "1"

// DefaultPolicyVersion names the policy a plan is compiled under when the
// caller does not pin one.
const DefaultPolicyVersion = "cloud-launch-v1"

// Outcomes a compile can produce. Outcome is material: a no-op plan and an
// apply plan for the same inputs are different reviewable artifacts.
const (
	OutcomeApply      = "apply"
	OutcomeNoOp       = "no_op"
	OutcomeNeedsInput = "needs_input"
)

// Scopes select which part of the full action graph a plan carries. The REST
// setup endpoints compile ScopeInstall, the deploy endpoints ScopeRuntime and
// the start/resume pipeline ScopeStart. ScopeFull is install followed by
// runtime in one envelope.
const (
	ScopeFull    = "full"
	ScopeInstall = "install"
	ScopeRuntime = "runtime"
	ScopeStart   = "start"
	// ScopeRetire is the terminal lifecycle: routing, runtime, grants, data
	// and artifacts in owner order with an explicit data disposition.
	ScopeRetire = "retire"
	// ScopeStop is the intentional stop: one scoped lifecycle stop that
	// releases only this deployment's demand on shared resources.
	ScopeStop = "stop"
)

// Maintenance strategies for release.activate.
const (
	StrategyMaintenance = "maintenance"
	StrategySideBySide  = "side_by_side"
)

// Owner operations: the supported action vocabulary. Every action in a plan
// names exactly one of these. See docs/reference/executable-plan.md.
const (
	OpHostPrepare             = "host.prepare"
	OpEdgeFirewallAllow       = "edge.firewall.allow"
	OpReleaseDeliver          = "release.deliver"
	OpReleaseVerify           = "release.verify"
	OpReleaseStage            = "release.stage"
	OpDataInventory           = "data.inventory"
	OpDataBackup              = "data.backup"
	OpConfigApply             = "config.apply"
	OpCredentialsProvision    = "credentials.provision"
	OpRuntimeStartDeps        = "runtime.start_dependencies"
	OpReleaseActivate         = "release.activate"
	OpEdgeRouteApply          = "edge.route.apply"
	OpVerifyReadiness         = "verify.readiness"
	OpReleaseRetainPredecesor = "release.retain_predecessor"
	OpWorkloadStop            = "workload.stop"
	OpWorkloadStart           = "workload.start"
	OpInputResumeHandoff      = "input.resume_handoff"
	OpEdgeRouteRetire         = "edge.route.retire"
	OpGrantsRevoke            = "grants.revoke"
	OpDataRetire              = "data.retire"
	OpArtifactsRetire         = "artifacts.retire"
)

// Effects an action declares. "none" means observe-only.
const (
	EffectNone            = "none"
	EffectHostWrite       = "host_write"
	EffectEdgeWrite       = "edge_write"
	EffectDeploymentWrite = "deployment_write"
	EffectDataWrite       = "data_write"
	EffectCredentialWrite = "credential_write" // #nosec G101 -- public plan effect vocabulary, not credential material.
	EffectRuntimeStop     = "runtime_stop"
	EffectRuntimeStart    = "runtime_start"
)

// Retry semantics: whether an action may simply be replayed, must be observed
// before replay, or needs the declared recovery first.
const (
	RetrySafeReplay        = "safe_replay"
	RetryObserveThenReplay = "observe_then_replay"
	RetryRecover           = "recover"
)

// Precondition kinds. Every kind listed here is material: a change in its
// value after preview makes the plan stale.
const (
	PreconditionDeploymentRevision  = "deployment_revision"
	PreconditionTargetEnrollment    = "target_enrollment"
	PreconditionReleaseDigest       = "release_digest"
	PreconditionClosureDigest       = "closure_digest"
	PreconditionConfigurationDigest = "configuration_digest"
	PreconditionDataSchema          = "data_schema"
	PreconditionPrivilegeSet        = "privilege_set"
	// PreconditionRecoveryPoint binds an update of persistent data to the
	// recovery point that must exist before the release changes the schema
	// (value: <release digest>@<schema version>).
	PreconditionRecoveryPoint = "recovery_point_required"
)

// Handoff owner and kind for operator input.
const (
	HandoffOwner = "vrooli-onboarding"
	HandoffKind  = "resume_handoff"
)

// Target is the identity part of a target binding. The locator (host, port,
// user) is reachability metadata and is deliberately not in the envelope.
type Target struct {
	MachineID            string `json:"machine_id,omitempty"`
	NodeID               string `json:"node_id,omitempty"`
	EnrollmentGeneration uint64 `json:"enrollment_generation"`
	Transport            string `json:"transport"`
}

// Precondition is one material fact the plan was compiled against.
type Precondition struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Downtime declares the expected unavailability an action causes.
type Downtime struct {
	ExpectedSeconds int    `json:"expected_seconds"`
	Reason          string `json:"reason"`
}

// Action is one node of the plan graph. Inputs are typed key/value pairs;
// they never carry a shell string.
type Action struct {
	ID                 string            `json:"id"`
	OwnerOperation     string            `json:"owner_operation"`
	Effect             string            `json:"effect"`
	RequiredCapability string            `json:"required_capability"`
	Inputs             map[string]string `json:"inputs"`
	DependsOn          []string          `json:"depends_on"`
	Verification       string            `json:"verification"`
	Recovery           string            `json:"recovery"`
	Retry              string            `json:"retry"`
	CancelPoint        bool              `json:"cancel_point"`
	Downtime           *Downtime         `json:"downtime,omitempty"`
}

// Handoff is the single durable reference an operator resumes from when the
// plan needs input the closure declares but nothing has satisfied.
type Handoff struct {
	Owner           string   `json:"owner"`
	Kind            string   `json:"kind"`
	Reference       string   `json:"reference"`
	Missing         []string `json:"missing"`
	DeploymentID    string   `json:"deployment_id"`
	Target          Target   `json:"target"`
	DesiredRevision uint64   `json:"desired_revision"`
	SelectionDigest string   `json:"selection_digest"`
}

// Presentation is human text. It is excluded from the semantic digest.
type Presentation struct {
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	DowntimeNote string `json:"downtime_note,omitempty"`
	RecoveryNote string `json:"recovery_note,omitempty"`
}

// Plan is the executable plan envelope.
type Plan struct {
	SchemaVersion       string         `json:"schema_version"`
	DeploymentID        string         `json:"deployment_id"`
	ScenarioID          string         `json:"scenario_id"`
	Environment         string         `json:"environment"`
	Target              Target         `json:"target"`
	Scope               string         `json:"scope"`
	Outcome             string         `json:"outcome"`
	DesiredRevision     uint64         `json:"desired_revision"`
	ReleaseDigest       string         `json:"release_digest"`
	ConfigurationDigest string         `json:"configuration_digest"`
	ClosureDigest       string         `json:"closure_digest"`
	PolicyVersion       string         `json:"policy_version"`
	Preconditions       []Precondition `json:"preconditions"`
	Actions             []Action       `json:"actions"`
	Handoff             *Handoff       `json:"handoff,omitempty"`
	Presentation        Presentation   `json:"presentation"`
}

// Action returns the action with the given id, or nil.
func (p *Plan) Action(id string) *Action {
	if p == nil {
		return nil
	}
	for i := range p.Actions {
		if p.Actions[i].ID == id {
			return &p.Actions[i]
		}
	}
	return nil
}

// ActionIDs returns the action ids in plan order.
func (p *Plan) ActionIDs() []string {
	if p == nil {
		return nil
	}
	ids := make([]string, 0, len(p.Actions))
	for _, action := range p.Actions {
		ids = append(ids, action.ID)
	}
	return ids
}

// Precondition returns the value recorded for a precondition kind.
func (p *Plan) Precondition(kind string) (string, bool) {
	if p == nil {
		return "", false
	}
	for _, pre := range p.Preconditions {
		if pre.Kind == kind {
			return pre.Value, true
		}
	}
	return "", false
}

// Ops lists every supported owner operation in vocabulary order.
func Ops() []string {
	return []string{
		OpHostPrepare, OpEdgeFirewallAllow, OpReleaseDeliver, OpReleaseVerify, OpReleaseStage,
		OpDataInventory, OpDataBackup, OpConfigApply, OpCredentialsProvision, OpRuntimeStartDeps,
		OpReleaseActivate, OpEdgeRouteApply, OpVerifyReadiness, OpReleaseRetainPredecesor,
		OpWorkloadStop, OpWorkloadStart, OpInputResumeHandoff,
		OpEdgeRouteRetire, OpGrantsRevoke, OpDataRetire, OpArtifactsRetire,
	}
}

// KnownOp reports whether op is in the vocabulary.
func KnownOp(op string) bool {
	for _, known := range Ops() {
		if known == op {
			return true
		}
	}
	return false
}
