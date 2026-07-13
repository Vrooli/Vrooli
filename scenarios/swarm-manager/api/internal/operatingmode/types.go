package operatingmode

import (
	"strings"

	"swarm-manager/internal/evidence"
)

type (
	Mode                  string
	TargetKind            string
	Phase                 string
	PhaseKind             string
	InputValueType        string
	InputSensitivity      string
	InputRetention        string
	InputSourceKind       string
	RunStrategyKind       string
	BacklogSyncCapability string
	BacklogSyncApplyMode  string
	ResultBindingKind     string
)

const (
	InputTypeString  InputValueType = "string"
	InputTypeInteger InputValueType = "integer"
	InputTypeNumber  InputValueType = "number"
	InputTypeBoolean InputValueType = "boolean"
	InputTypeObject  InputValueType = "object"
	InputTypeArray   InputValueType = "array"
)

const (
	InputSensitivityPublic    InputSensitivity = "public"
	InputSensitivityInternal  InputSensitivity = "internal"
	InputSensitivitySensitive InputSensitivity = "sensitive"
)

const (
	InputRetentionValue  InputRetention = "value"
	InputRetentionDigest InputRetention = "digest"
	InputRetentionOmit   InputRetention = "omit"
)

const (
	InputSourceGenericProvider InputSourceKind = "generic_provider"
	InputSourceTargetAdapter   InputSourceKind = "target_adapter"
	InputSourceCaller          InputSourceKind = "caller"
	InputSourceDerived         InputSourceKind = "derived"
	InputSourceDefault         InputSourceKind = "default"
)

const (
	ModeItemLevel       Mode = "item-level"
	ModeHolisticLoop    Mode = "holistic-loop"
	ModePhasedPlanDrain Mode = "phased-plan-drain"
)

// TargetKind is the mode's declared unit of work — the thing one run of the
// loop operates on. Each kind has a target adapter that supplies the
// target-specific reads, ownership key, and resolution; the initiative is one
// adapter among several, not the substrate everything else is bolted onto.
const (
	// TargetPlanManagerPlan targets a canonical plan-manager plan
	// (execution id / slug).
	TargetPlanManagerPlan TargetKind = "plan-manager-plan"
	// TargetPlanRef targets a plan file or reference not imported into
	// swarm-manager (e.g. a repo-relative plan path).
	TargetPlanRef TargetKind = "plan-ref"
	// TargetInitiative targets a swarm-manager initiative and its member items.
	TargetInitiative TargetKind = "initiative"
)

// IsValidTargetKind reports whether the given kind is one of the three
// registered target kinds. Empty is invalid; unknown values are invalid.
func IsValidTargetKind(kind TargetKind) bool {
	switch kind {
	case TargetPlanManagerPlan, TargetPlanRef, TargetInitiative:
		return true
	default:
		return false
	}
}

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
	Target                 TargetPolicy
	InputContract          InputContractDefinition
	PhaseGraph             PhaseGraph
	RunStrategy            RunStrategyPolicy
	Artifact               ArtifactPolicy
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

// InputContractDefinition is the mode-authored input SSOT. Logical specs own
// value and retention semantics; sources own acquisition policy; aliases bind
// prompt variable names to logical inputs. Keeping the three sets separate
// prevents a logical input identity from becoming coupled to one provider.
type InputContractDefinition struct {
	Specs   []InputSpec          `json:"specs,omitempty"`
	Sources []InputSourceBinding `json:"sources,omitempty"`
	Aliases []InputAlias         `json:"aliases,omitempty"`
}

// InputSpec describes one stable logical input independently of how its value
// is obtained. IDs are dotted, namespaced identities such as
// "execution.operator_note" or "plan.context".
type InputSpec struct {
	ID          string           `json:"id"`
	Type        InputValueType   `json:"type"`
	Format      string           `json:"format,omitempty"`
	Required    bool             `json:"required,omitempty"`
	Minimum     *float64         `json:"minimum,omitempty"`
	Maximum     *float64         `json:"maximum,omitempty"`
	MinLength   *int             `json:"min_length,omitempty"`
	MaxLength   *int             `json:"max_length,omitempty"`
	MinItems    *int             `json:"min_items,omitempty"`
	MaxItems    *int             `json:"max_items,omitempty"`
	Sensitivity InputSensitivity `json:"sensitivity"`
	Retention   InputRetention   `json:"retention"`
	Description string           `json:"description"`
}

// InputSourceBinding assigns exactly one acquisition policy to one logical
// input. Provider and derived bindings name a typed capability; derived
// bindings also declare their logical dependencies so the compiler can reject
// cycles. Default values remain JSON-native data.
type InputSourceBinding struct {
	InputID    string          `json:"input_id"`
	Kind       InputSourceKind `json:"kind"`
	Capability string          `json:"capability,omitempty"`
	DependsOn  []string        `json:"depends_on,omitempty"`
	Default    any             `json:"default,omitempty"`
}

// InputAlias binds a prompt-facing SCREAMING_SNAKE variable name to one
// logical input. A phase's Reads list selects aliases from this mode-level map,
// so variable spelling and logical identity remain explicit data.
type InputAlias struct {
	Name    string `json:"name"`
	InputID string `json:"input_id"`
}

// TargetPolicy is the mode's declared unit of work plus adapter-specific
// configuration. It subsumes the former top-level plan_ref policy: a bound
// plan-manager plan is initiative-adapter configuration (the initiative
// carries a plan_ref the run consumes), which is distinct from the plan-ref
// *target kind* (the plan itself is the unit of work).
type TargetPolicy struct {
	Kind TargetKind
	// PlanRef configures the initiative adapter's bound-plan contract: when
	// Required, non-start phases refuse to run until the initiative binds a
	// canonical plan-manager reference, and the adapter supplies its resolved
	// execution context through the PLAN_CONTEXT_JSON read. Only meaningful
	// for initiative-target modes; the loader rejects it elsewhere.
	PlanRef PlanRefPolicy
}

type PlanRefPolicy struct {
	Required bool
	Role     string
}

// RunsModeRounds reports whether the mode executes durable operating-mode
// rounds through a phase graph. Item-level work owned by the existing backlog
// execution flow (run strategy existing_item_flow) does not.
func (d Definition) RunsModeRounds() bool {
	return d.RunStrategy.Kind != RunStrategyExistingItemFlow
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

// TransitionClassification is a transition-owned classification contract
// (classification-on-transition): when the routing field a guard needs was not
// directly emitted by the completed round, the transition declares how to
// derive it from the round's resolved output — the same resolution-ladder
// mechanics applied at the edge instead of at the phase result. The loader
// expands a classified transition's routes into ordinary eq-guards over Field,
// so guard evaluation stays exactly what it is; classification only changes
// where the field's value can come from. At most one classification contract
// exists per phase (loader-enforced).
//
// Derivation precedence (documented contract, tested):
//   - Short-circuit (not_required): the round already emitted Field (top-level
//     payload or inside the structured-result envelope). The emitted value must
//     satisfy Enum; an out-of-enum emitted value is a contract violation and an
//     honest abstain, never overridden by the classifier.
//   - L1 (deterministic): the value is carried inline on the source object at
//     `<From>.<Field>` (e.g. handoff.progress). No model call.
//   - L2 (classifier fallback): the JSON of the From field's value (or the whole
//     envelope when From is empty) is classified against Enum; the classifier
//     abstains rather than guessing.
//   - Abstain: the round routes to needs_attention — a routing decision is never
//     fabricated and never crashes the loop.
type TransitionClassification struct {
	// Field is the dotted path of the routing field to derive (e.g.
	// progress.decision). The expanded route guards compare this field.
	Field string
	// Enum is the closed set of permitted routing values.
	Enum []string
	// From names the declared output field to derive from (e.g. handoff).
	// Empty means the whole structured result envelope.
	From string
	// Description explains what the routing decision means; steers L2.
	Description string
}

type PhaseDefinition struct {
	Phase Phase
	// Kind is the phase classification (investigate / execute / review /
	// reconcile). Lane assignment, Operations Center column placement, and
	// per-lane metric grouping all key off this axis. Must be set on every
	// initiative-scoped phase; the validator rejects empty values.
	Kind PhaseKind
	// ExecutedBy names the sub-mode that executes this phase (phase
	// delegation, EXECUTION-MODES.md D3). Empty for regular phases. When set,
	// the engine runs the sub-mode's loop as this phase — sub-rounds execute
	// under the parent run with the sub-mode's prompts, reads, and classified
	// edges — and the sub-mode's terminal outcome becomes this phase's
	// resolved output for the parent's guards to route on. Exactly one level
	// deep: the loader rejects nesting, self-delegation, unknown sub-modes,
	// and target-incompatible delegation. A delegated phase declares no
	// prompt/reads/declared_output of its own.
	ExecutedBy Mode
	// AutoStartAfter lists phases whose successful completion should
	// automatically start this phase. Constrained to length ≤ 1 in v1; the
	// validator rejects multi-predecessor declarations to avoid
	// race-condition design pressure on the round refresher hook. The
	// auto-start path fires only on RoundStatusCompleted (not Failed /
	// Cancelled) and only after the initiative lock is released.
	AutoStartAfter []Phase
	// Reads is the phase's declared input contract: the named variables its
	// prompt template may reference, symmetric with DeclaredOutput on the emit
	// side. The composed provider set (generic base ∪ target adapter) must
	// satisfy every declared read — the loader validates this at load time —
	// and the renderer substitutes exactly the declared reads, so an
	// undeclared template slot fails loudly instead of rendering empty.
	Reads           []string
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
	DeclaredOutput *DeclaredOutput
	// TransitionClassification is the phase's transition-owned classification
	// contract, when one of its transitions is declared as a classified edge
	// (classify/routes). Nil for phases whose transitions are all plain guarded
	// edges. Populated by the data loader from the classified transition.
	TransitionClassification *TransitionClassification
	// EvidenceRequirements are normalized, data-defined facts that must be
	// present before this round may complete. They are deliberately independent
	// of a mode's domain vocabulary: the evidence ledger owns fact lookup and
	// producer-completeness semantics.
	EvidenceRequirements []EvidenceRequirement
	RequiresCriteria     bool
}

// EvidenceRequirement is the operating-mode data representation of one
// canonical-ledger predicate. A match is accepted only at or above
// MinConfidence; missing facts remain pending until the named producer has
// published terminal coverage for the run and subject kind.
type EvidenceRequirement struct {
	SubjectKind   string              `json:"subject_kind"`
	Action        string              `json:"action"`
	ProducerID    string              `json:"producer,omitempty"`
	MinConfidence evidence.Confidence `json:"min_confidence"`
	MinCount      int                 `json:"min_count,omitempty"`
	MatchFields   map[string]string   `json:"match_fields,omitempty"`
}

func (r EvidenceRequirement) LedgerRequirement() evidence.Requirement {
	return evidence.Requirement{
		SubjectKind:   r.SubjectKind,
		Action:        r.Action,
		ProducerID:    r.ProducerID,
		MinConfidence: r.MinConfidence,
		MinCount:      r.MinCount,
		MatchFields:   r.MatchFields,
	}
}

// Delegated reports whether the phase is executed by a sub-mode
// (executed_by) rather than by its own prompt/reads/emits contract.
func (p PhaseDefinition) Delegated() bool {
	return strings.TrimSpace(string(p.ExecutedBy)) != ""
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
