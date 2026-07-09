package operatingmode

type (
	Mode                  string
	ScopeKind             string
	Phase                 string
	PhaseKind             string
	RunStrategyKind       string
	BacklogSyncCapability string
	BacklogSyncApplyMode  string
	ResultBindingKind     string
)

const (
	ModeItemLevel       Mode = "item-level"
	ModeHolisticLoop    Mode = "holistic-loop"
	ModePhasedPlanDrain Mode = "phased-plan-drain"
)

const (
	ScopeBacklogItem ScopeKind = "backlog_item"
	ScopeInitiative  ScopeKind = "initiative"
)

// PhaseKind classifies a phase by the kind of work it performs. Lane
// assignment, Operations Center column placement, and per-lane metric
// grouping all key off this axis. PhaseKind is descriptive, not restrictive:
// modes are free to define any phases they want — each phase declares its
// kind. Adding a fifth kind is a substrate change (lane plumbing, UI columns,
// authoring contract all move together) and is intentionally out of scope.
const (
	PhaseKindInvestigate PhaseKind = "investigate"
	PhaseKindExecute     PhaseKind = "execute"
	PhaseKindReview      PhaseKind = "review"
	PhaseKindReconcile   PhaseKind = "reconcile"
)

const (
	RunStrategyExistingItemFlow  RunStrategyKind = "existing_item_flow"
	RunStrategySinglePhaseRun    RunStrategyKind = "single_phase_run"
	RunStrategySequentialHandoff RunStrategyKind = "sequential_handoff"
	RunStrategyOperatorGatedLoop RunStrategyKind = "operator_gated_loop"
)

const (
	BacklogSyncReadOnly         BacklogSyncCapability = "read_only"
	BacklogSyncProposeMutations BacklogSyncCapability = "propose_mutations"
	BacklogSyncMarkComplete     BacklogSyncCapability = "mark_complete"
	BacklogSyncCreateFollowups  BacklogSyncCapability = "create_followups"
	BacklogSyncUpdateScope      BacklogSyncCapability = "update_scope"
)

// BacklogSyncApplyMode controls whether reconcile-phase proposals require an
// operator decision before mutating the backlog, or whether the system applies
// some/all proposed mutations automatically. Only operator-gated is
// implemented in v1; the auto-apply variants land as a typed
// ErrApplyModeNotImplemented at the apply seam so loud failures, not silent
// backlog edits, surface any future mode misconfiguration.
const (
	BacklogSyncApplyOperatorGated BacklogSyncApplyMode = "operator-gated"
	BacklogSyncApplyAutoSafe      BacklogSyncApplyMode = "auto-apply-safe"
	BacklogSyncApplyAutoAll       BacklogSyncApplyMode = "auto-apply-all"
)

// IsValidBacklogSyncApplyMode reports whether the given mode is one of the
// three registered apply modes. Empty is invalid; unknown values are invalid.
func IsValidBacklogSyncApplyMode(mode BacklogSyncApplyMode) bool {
	switch mode {
	case BacklogSyncApplyOperatorGated, BacklogSyncApplyAutoSafe, BacklogSyncApplyAutoAll:
		return true
	default:
		return false
	}
}

const (
	ResultBindingProgressArtifact ResultBindingKind = "progress_artifact"
)

const (
	ProfileDefault  = "swarm-manager/default"
	ProfileDeepWork = "swarm-manager/deep-work"
	ProfileAnalysis = "swarm-manager/analysis"
)

type Definition struct {
	Mode        Mode
	Label       string
	Description string
	// BestFor enumerates the work shapes this mode is the right pick for.
	// Surfaced in the picker and details page as decision support; populated
	// alongside the mode definition. Validator requires ≥1 entry per mode.
	BestFor []string
	// NotFor enumerates the work shapes this mode handles poorly. Validator
	// requires ≥1 entry per mode.
	NotFor []string
	// Tradeoffs enumerates the structural tradeoffs an operator accepts when
	// choosing this mode. Validator requires ≥1 entry per mode.
	Tradeoffs []string
	// WhenInDoubtPickInstead names a registered mode the operator should pick
	// if they're unsure between this one and another. Empty means this mode is
	// itself the safe default. Validator requires this to reference a
	// registered mode and not be self.
	WhenInDoubtPickInstead Mode
	Scope                  ScopePolicy
	PhaseGraph             PhaseGraph
	RunStrategy            RunStrategyPolicy
	Artifact               ArtifactPolicy
	PlanRef                PlanRefPolicy
	Prompt                 PromptPolicy
	Profile                ProfilePolicy
	BacklogSync            BacklogSyncPolicy
	Metrics                MetricsPolicy
	Lock                   LockPolicy
	UI                     UIPolicy
	// ExampleRuns are the mode-owned simulation fixtures loaded from
	// modes/<id>/example-runs/*.json. Each seeds phase outputs and asserts the
	// phase path the real generic guard evaluator produces; they are the data
	// behind the simulator's presets. Populated by LoadModesFromDir; empty for
	// modes with no example-runs directory (the simulator then synthesizes a
	// generic happy-path). Ordered happy-path-first.
	ExampleRuns []ExampleRun
}

type PlanRefPolicy struct {
	Required bool
	Role     string
}

// ExampleRun looks up a loaded example-run by id.
func (d Definition) ExampleRun(id string) (ExampleRun, bool) {
	for _, run := range d.ExampleRuns {
		if run.ID == id {
			return run, true
		}
	}
	return ExampleRun{}, false
}

type ScopePolicy struct {
	Kind ScopeKind
}

type PhaseGraph struct {
	StartPhase  Phase
	Terminal    []Phase
	Transitions map[Phase][]Phase
	// Guards is the generic, data-driven branching model: an ordered list of
	// guarded edges per phase, evaluated by the generic guard evaluator
	// (guard.go). It is the sole branching representation — the runtime routes,
	// the simulation walks, and the UI renders transitions from it. Populated by
	// the data loader (LoadModeDefinition). Transitions above is the derived
	// static adjacency (guard targets flattened) used for ordering and
	// reachability; Guards carries the conditions.
	Guards map[Phase][]GuardedTransition
	Phases map[Phase]PhaseDefinition
}

// GuardedTransition is one edge out of a phase: when the guard matches the
// completed round's structured output, the loop routes to To. An empty To is a
// guarded stop (e.g. a blocked progress decision). The generic guard replaces
// the closed always/payload_bool/progress_decision TransitionCondition kinds.
type GuardedTransition struct {
	When Guard
	To   []Phase
}

type PhaseDefinition struct {
	Phase Phase
	// Kind is the phase classification (investigate / execute / review /
	// reconcile). Lane assignment, Operations Center column placement, and
	// per-lane metric grouping all key off this axis. Must be set on every
	// initiative-scoped phase; the validator rejects empty values.
	Kind PhaseKind
	// AutoStartAfter lists phases whose successful completion should
	// automatically start this phase. Constrained to length ≤ 1 in v1; the
	// validator rejects multi-predecessor declarations to avoid
	// race-condition design pressure on the round refresher hook. The
	// auto-start path fires only on RoundStatusCompleted (not Failed /
	// Cancelled) and only after the initiative lock is released.
	AutoStartAfter  []Phase
	ActivityPurpose string
	LockPurpose     string
	CatalogID       string
	SkillID         string
	PromptCatalog   PromptCatalogMetadata
	ProfileKey      string
	WritesRepo      bool
	OutputArtifacts []ArtifactDefinition
	ResultBindings  []ResultBinding
	OutputContract  PhaseOutputContract
	// DeclaredOutput is the phase's declared structured-output schema (field
	// name/type/required/enum/bounds) plus resolution-ladder tuning. It is the
	// single artifact that both validates a round's result and steers the
	// resolution ladder; guards reference its fields by path. Populated by the
	// data loader; the hardcoded Go definitions leave it nil.
	DeclaredOutput   *DeclaredOutput
	RequiresCriteria bool
}

// DeclaredOutput is the per-phase contract for what a round is supposed to emit.
type DeclaredOutput struct {
	EnvelopeKey              string
	RequiresStructuredResult bool
	Fields                   []OutputField
	Resolution               ResolutionPolicy
}

// OutputField is one declared output field. Object-typed fields may nest
// further fields; guards and the resolution ladder reference fields by path.
type OutputField struct {
	Name        string
	Type        string
	Required    bool
	Enum        []any
	Minimum     *float64
	Maximum     *float64
	MinLength   *int
	MaxLength   *int
	Description string
	Fields      []OutputField
}

// ResolutionPolicy is the per-phase resolution-ladder tuning. The engine
// applies these knobs when resolving imperfect model output (Phase 5); the
// loader fills the schema defaults so consumers never see zero-values that mean
// "off".
type ResolutionPolicy struct {
	DetectTrueFinalMessage bool
	ScanLastNMessages      int
	AllowClassifier        bool
}

// IsValidPhaseKind reports whether the given kind is one of the four
// registered classifications. Empty is invalid; unknown values are invalid.
func IsValidPhaseKind(kind PhaseKind) bool {
	switch kind {
	case PhaseKindInvestigate, PhaseKindExecute, PhaseKindReview, PhaseKindReconcile:
		return true
	default:
		return false
	}
}

type PromptCatalogMetadata struct {
	Title   string
	Trigger string
	Purpose string
}

type ResultBinding struct {
	Kind     ResultBindingKind  `json:"kind"`
	Artifact ArtifactDefinition `json:"artifact"`
}

type PhaseOutputContract struct {
	RequiresStructuredResult bool
	RequiredArtifacts        []ArtifactDefinition
	RequiresPlanRef          bool
	RequiresProgress         bool
	RequiresVerdict          bool
	RequiresHandoff          bool
	// RequiresBacklogSync demands that the phase emit a non-nil BacklogSync
	// plan in its structured result envelope. Reconcile phases set this so a
	// misbehaving agent cannot complete the phase without producing a
	// proposal — the operator never lands in a state where an
	// auto-dispatched reconcile round leaves no backlog plan to apply.
	RequiresBacklogSync bool
}

type RunStrategyPolicy struct {
	Kind RunStrategyKind
}

type ArtifactPolicy struct {
	Root      string
	RoundRoot string
}

type ArtifactDefinition struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type PromptPolicy struct {
	CatalogPrefix string
}

type ProfilePolicy struct {
	DefaultProfileKey string
	PhaseProfiles     map[Phase]string
}

type BacklogSyncPolicy struct {
	Capabilities       []BacklogSyncCapability
	RequiresRunID      bool
	RequiresMembership bool
	EventSource        string
	// ApplyMode declares how reconcile-phase proposals are committed to the
	// backlog. Required on every initiative-scoped mode that emits proposals;
	// only BacklogSyncApplyOperatorGated is implemented in v1. The validator
	// rejects empty or unknown values for initiative-scoped modes.
	ApplyMode BacklogSyncApplyMode
}

type MetricsPolicy struct {
	EventSource            string
	ReplanSamplePhases     []Phase
	AcceptanceSamplePhases []Phase
	AcceptedVerdicts       []string
}

type LockPolicy struct {
	InitiativeExclusive bool
}

type UIPolicy struct {
	WorkspaceTabID string
}
