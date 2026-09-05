// Package planmodel contains the neutral structured-plan kernel shared by the
// plan-manager domains. It has no transport, persistence, or domain-service
// dependencies.
package planmodel

import (
	"reflect"
	"strings"
)

// PlanStatus is the lifecycle state of a plan. COMPUTED from the phase-status
// set plus lifecycle actions; never free-text edited.
type PlanStatus string

const (
	PlanStatusDraft    PlanStatus = "draft"
	PlanStatusActive   PlanStatus = "active"
	PlanStatusComplete PlanStatus = "complete"
	PlanStatusArchived PlanStatus = "archived"
)

// PhaseStatus is the state of a single first-class phase.
type PhaseStatus string

const (
	PhaseStatusTodo    PhaseStatus = "todo"
	PhaseStatusActive  PhaseStatus = "active"
	PhaseStatusDone    PhaseStatus = "done"
	PhaseStatusBlocked PhaseStatus = "blocked"
)

// StalenessTier is the computed validity of referenced code since authoring.
type StalenessTier string

const (
	StalenessFresh           StalenessTier = "fresh"
	StalenessLightlyStale    StalenessTier = "lightly_stale"
	StalenessDefinitelyStale StalenessTier = "definitely_stale"
	StalenessUnknown         StalenessTier = "" // unset / degraded
)

// ReferenceKind is the machine-readable locator family.
type ReferenceKind string

const (
	ReferenceCode ReferenceKind = "code"
	ReferenceReq  ReferenceKind = "req"
	ReferenceDoc  ReferenceKind = "doc"
)

// ReferenceResolution is whether a reference resolves against code-facts.
type ReferenceResolution string

const (
	ResolutionUnspecified ReferenceResolution = ""
	ResolutionResolved    ReferenceResolution = "resolved"
	ResolutionUnresolved  ReferenceResolution = "unresolved"
	ResolutionFuture      ReferenceResolution = "future"
	ResolutionMissing     ReferenceResolution = "missing"
)

// RelevantContextKind classifies one setup item a fresh/resumed agent should
// load, inspect, or run before acting on a plan or phase.
type RelevantContextKind string

const (
	RelevantContextSkill   RelevantContextKind = "skill"
	RelevantContextDoc     RelevantContextKind = "doc"
	RelevantContextCommand RelevantContextKind = "command"
	RelevantContextSearch  RelevantContextKind = "search"
	RelevantContextCodeRef RelevantContextKind = "code_ref"
	RelevantContextReqRef  RelevantContextKind = "req_ref"
	RelevantContextNote    RelevantContextKind = "note"
)

// RelevantContextScope says whether an item applies plan-wide or to one phase.
type RelevantContextScope string

const (
	RelevantContextScopeGlobal RelevantContextScope = "global"
	RelevantContextScopePhase  RelevantContextScope = "phase"
)

// RelevantContextRepeatPolicy says when execution should re-emit the item.
type RelevantContextRepeatPolicy string

const (
	RelevantContextOncePerExecution RelevantContextRepeatPolicy = "once_per_execution"
	RelevantContextOnResume         RelevantContextRepeatPolicy = "on_resume"
	RelevantContextEveryPhase       RelevantContextRepeatPolicy = "every_phase"
	RelevantContextPhaseEntry       RelevantContextRepeatPolicy = "phase_entry"
	RelevantContextAsNeeded         RelevantContextRepeatPolicy = "as_needed"
)

// RelevantContextSource captures how the context item entered the plan.
type RelevantContextSource string

const (
	RelevantContextSourceAuthored   RelevantContextSource = "authored"
	RelevantContextSourceDiscovered RelevantContextSource = "discovered"
	RelevantContextSourceMigrated   RelevantContextSource = "migrated"
	RelevantContextSourceAutofilled RelevantContextSource = "autofilled"
)

// RelevantContextStatus records whether the item is usable or degraded.
type RelevantContextStatus string

const (
	RelevantContextStatusReady      RelevantContextStatus = "ready"
	RelevantContextStatusDegraded   RelevantContextStatus = "degraded"
	RelevantContextStatusUnresolved RelevantContextStatus = "unresolved"
)

// WorkPosture is the Greenfield/Brownfield stance of a plan. AUTOFILLED from the
// associated scenario's maturity (default greenfield); never agent-authored.
type WorkPosture string

const (
	WorkPostureUnspecified WorkPosture = ""
	WorkPostureGreenfield  WorkPosture = "greenfield"
	WorkPostureBrownfield  WorkPosture = "brownfield"
)

// WorkPostureSource records how the posture was decided (audit/explainability).
type WorkPostureSource string

const (
	WorkPostureSourceUnspecified      WorkPostureSource = ""
	WorkPostureSourceDefault          WorkPostureSource = "default"
	WorkPostureSourceServiceMaturity  WorkPostureSource = "service_maturity"
	WorkPostureSourceExplicitOverride WorkPostureSource = "explicit_override"
	WorkPostureSourceImportLegacy     WorkPostureSource = "import_legacy"
)

// RenderedMirrorStatus describes the freshness/repair state of a plan's durable
// rendered markdown projection. The mirror is derived from the structured plan.
type RenderedMirrorStatus string

const (
	RenderedMirrorStatusUnspecified RenderedMirrorStatus = ""
	RenderedMirrorStatusFresh       RenderedMirrorStatus = "fresh"
	RenderedMirrorStatusMissing     RenderedMirrorStatus = "missing"
	RenderedMirrorStatusStale       RenderedMirrorStatus = "stale"
	RenderedMirrorStatusWriteFailed RenderedMirrorStatus = "write_failed"
	RenderedMirrorStatusUnknown     RenderedMirrorStatus = "unknown"
)

// RenderedPlanMirror is the file-addressable markdown projection metadata for a
// canonical structured plan. The file is durable and repairable, but not truth.
type RenderedPlanMirror struct {
	Path          string
	RelativePath  string
	ContentHash   string
	RenderVersion string
	RenderedAt    string
	Status        RenderedMirrorStatus
	LastError     string
}

// LegacySection is one imported markdown section preserved as provenance because
// it could not be mapped to a canonical field — never silently dropped.
type LegacySection struct {
	Heading            string
	Content            string
	MappedTo           string
	PreservationReason string
}

// PreservationReasonUnmapped is the canonical reason for an unmapped legacy
// section preserved verbatim during import.
const PreservationReasonUnmapped = "unmapped_legacy_section"

// ImportProvenance records that a plan was adopted from a legacy markdown source
// rather than authored fresh. Import is non-destructive.
type ImportProvenance struct {
	SourcePath     string
	ImportedAt     string
	OriginalFormat string
	Note           string
	WorkspaceID    string
	WorkspaceRoot  string
}

// OriginalFormatLegacyMarkdown is the original_format tag for legacy 13-section
// markdown plans adopted through import.
const OriginalFormatLegacyMarkdown = "legacy_markdown"

// Reference is one connected-code locator on a plan or phase. kind/target/future
// are AUTHORED; resolution/staleness/change_factor are filled by validation.
type Reference struct {
	ID           string
	Kind         ReferenceKind
	Target       string
	Future       bool
	Resolution   ReferenceResolution
	Staleness    StalenessTier
	ChangeFactor float64
	Note         string
}

// PlanDecision is one pinned plan-time contract decision (rendered D1..Dn under
// Approach & Decisions). AUTHORED and optional; distinct from execution-time
// log decisions, which live in the log ledger.
type PlanDecision struct {
	Title     string
	Statement string
}

// PlanAssumption is one structured assumption with its "if wrong -> then"
// mitigation, rendered as the Assumptions & Risks table. AUTHORED and optional;
// prose assumptions without mitigations stay in Plan.Assumptions.
type PlanAssumption struct {
	Statement  string
	Mitigation string
}

// PlanDefinition records a term coined or narrowed by a plan. Shared ecosystem
// vocabulary belongs in docs/concepts/GLOSSARY.md and is referenced instead.
type PlanDefinition struct {
	Term    string
	Meaning string
}

// RelevantContextItem is the execution-facing setup contract.
type RelevantContextItem struct {
	ID           string
	Kind         RelevantContextKind
	Scope        RelevantContextScope
	PhaseID      string
	Label        string
	Reason       string
	Instruction  string
	Command      string
	Argv         []string
	Target       string
	Required     bool
	RepeatPolicy RelevantContextRepeatPolicy
	Source       RelevantContextSource
	Status       RelevantContextStatus
	StatusDetail string
}

// RegressionAnchor is the "before" anchor captured prior to changes.
//
// Strategy is one of:
//   - AnchorStrategyChangeBoundary ("change_boundary"): the boundary-native
//     anchor new plans author. Affected scenarios and the tiered baseline/diff
//     commands are DERIVED from the plan's ChangeBoundary; Scenario/AllowlistPaths
//     are not hand-authored.
//   - "scenario_baseline" / "head_sha_allowlist": legacy strategies, retained for
//     import/read of pre-cutover plans only — never produced for new authored plans.
//   - "legacy_prose": an unstructured imported anchor preserved as provenance.
type RegressionAnchor struct {
	Strategy       string
	Scenario       string
	BaselineName   string
	HeadSha        string
	AllowlistPaths []string
	Commands       []string
	CapturedAt     string
	CaptureStatus  string
	CaptureReason  string
	Fallback       string
	Unavailable    bool
}

// BaselineSetIntent is the plan-time, immutable request for one comprehensive
// before-state.  It deliberately records policy rather than shell commands:
// Git Control Tower owns the collection mechanics while Plan Manager owns which
// members and source paths a plan needs.  Targets are resolved from the change
// boundary when a plan is finalized, then persisted here so execution never has
// to scrape rendered markdown or infer policy from a later worktree.
type BaselineSetIntent struct {
	// Name is the stable collection/baseline identity used for every selected
	// scenario. It is intentionally shared with the existing anchor name so a
	// legacy per-scenario baseline can be translated without a second name.
	Name string
	// ScenarioTargets is the sorted, deduplicated inventory derived from the
	// plan change boundary at finalization time.
	ScenarioTargets []string
	// RepoPaths is the sorted, deduplicated set of non-scenario boundary globs
	// for informational source snapshots. They never become behavioral oracles.
	RepoPaths []string
	// CapturePolicy is currently execution_start. Keeping it explicit makes a
	// later policy evolution additive and auditably distinct from legacy plans.
	CapturePolicy string
	// Compatibility is baseline_set for newly authored plans and legacy_anchor
	// for imported/pre-cutover anchor records.
	Compatibility string
}

const (
	BaselineCapturePolicyExecutionStart = "execution_start"
	BaselineSetCompatibilityCurrent     = "baseline_set"
	BaselineSetCompatibilityLegacy      = "legacy_anchor"
)

// BaselineSetFromBoundary derives the persisted plan intent from the one
// authoritative blast-radius contract. It is deterministic so repairing an old
// plan cannot silently expand its coverage.
func BaselineSetFromBoundary(boundary ChangeBoundary, name string) BaselineSetIntent {
	return BaselineSetIntent{
		Name:            strings.TrimSpace(name),
		ScenarioTargets: boundary.AffectedScenarios(),
		RepoPaths:       boundary.RepoPaths(),
		CapturePolicy:   BaselineCapturePolicyExecutionStart,
		Compatibility:   BaselineSetCompatibilityCurrent,
	}
}

// IsLegacy reports whether this plan explicitly uses the historical baseline
// adoption path. A zero-value intent is deliberately not legacy: treating an
// omitted intent as legacy would let newly created plans bypass collection
// capture.
func (i BaselineSetIntent) IsLegacy() bool {
	return strings.TrimSpace(i.Compatibility) == BaselineSetCompatibilityLegacy
}

// IsCurrent reports whether this is a current collection-baseline intent.
func (i BaselineSetIntent) IsCurrent() bool {
	return strings.TrimSpace(i.Compatibility) == BaselineSetCompatibilityCurrent
}

// IsLegacyBaselinePlan identifies the two explicit historical compatibility
// forms. A missing intent is never legacy merely because it is absent.
func IsLegacyBaselinePlan(p Plan) bool {
	if p.BaselineSet.IsCurrent() {
		return false
	}
	if p.BaselineSet.IsLegacy() {
		return true
	}
	switch strings.TrimSpace(p.RegressionAnchor.Strategy) {
	case AnchorStrategyScenarioBaseline, AnchorStrategyHeadShaAllowlist, AnchorStrategyLegacyProse:
		return true
	default:
		return false
	}
}

// EnsureCurrentBaselineSet derives the collection policy for every current
// change-boundary plan, regardless of whether it came from guided authoring,
// a direct API write, or the CLI. Call it only after the slug is stable.
func EnsureCurrentBaselineSet(p *Plan) {
	if p == nil || p.BaselineSet.IsLegacy() || strings.TrimSpace(p.RegressionAnchor.Strategy) != AnchorStrategyChangeBoundary {
		return
	}
	name := strings.TrimSpace(p.RegressionAnchor.BaselineName)
	if name == "" && strings.TrimSpace(p.Slug) != "" {
		name = strings.TrimSpace(p.Slug) + "-baseline"
		p.RegressionAnchor.BaselineName = name
	}
	p.BaselineSet = BaselineSetFromBoundary(p.ChangeBoundary, name)
}

// CurrentBaselineSetValid proves the stored intent is exactly the deterministic
// policy derived from the plan's anchor and change boundary. It prevents stale
// or hand-widened collection membership from becoming execution evidence.
func CurrentBaselineSetValid(p Plan) bool {
	if !p.BaselineSet.IsCurrent() || strings.TrimSpace(p.RegressionAnchor.Strategy) != AnchorStrategyChangeBoundary {
		return false
	}
	expected := BaselineSetFromBoundary(p.ChangeBoundary, p.RegressionAnchor.BaselineName)
	return p.BaselineSet.Name == expected.Name &&
		p.BaselineSet.CapturePolicy == expected.CapturePolicy &&
		containsAllStrings(p.BaselineSet.ScenarioTargets, expected.ScenarioTargets) &&
		containsAllStrings(p.BaselineSet.RepoPaths, expected.RepoPaths)
}

func containsAllStrings(have, required []string) bool {
	set := make(map[string]struct{}, len(have))
	for _, value := range have {
		set[value] = struct{}{}
	}
	for _, value := range required {
		if _, ok := set[value]; !ok {
			return false
		}
	}
	return true
}

// Regression-anchor strategy identifiers.
const (
	// AnchorStrategyChangeBoundary is the boundary-native anchor strategy used by
	// all newly authored plans. Its commands are derived from the plan boundary.
	AnchorStrategyChangeBoundary = "change_boundary"
	// AnchorStrategyScenarioBaseline is the legacy single-scenario baseline anchor
	// (import/read only).
	AnchorStrategyScenarioBaseline = "scenario_baseline"
	// AnchorStrategyHeadShaAllowlist is the legacy HEAD-sha + file-allowlist anchor
	// (import/read only).
	AnchorStrategyHeadShaAllowlist = "head_sha_allowlist"
	// AnchorStrategyLegacyProse marks an unstructured imported anchor preserved as
	// provenance; it is never a verdict oracle.
	AnchorStrategyLegacyProse = "legacy_prose"
)

// Phase is a first-class phase. order/title/intent/required_reading/reminders/
// acceptance/references are AUTHORED; baseline_scope/status are COMPUTED.
type Phase struct {
	ID              string
	Order           int
	Title           string
	Intent          string
	RequiredReading []string
	Reminders       []string
	BaselineScope   []string
	Acceptance      string
	Status          PhaseStatus
	References      []Reference
	RelevantContext []RelevantContextItem
	// Optional per-phase boundary refinement. When set it NARROWS the plan-level
	// boundary for phase-specific checks; it never widens the plan blast radius.
	ChangeBoundary ChangeBoundary
	// ValidationScope is an authored declaration of whether phase validation is
	// deliberately narrow or intentionally covers the full plan boundary.
	// ChangeBoundary remains the legacy narrowing representation and is read as
	// a narrow scope when ValidationScope is absent.
	ValidationScope ValidationScope
	// AUTHORED professional phase fields (see docs/concepts/PLAN-MODEL.md).
	AffectedAreas   []string
	Steps           []string
	ExpectedOutputs []string
	Validation      string
	HandoffNotes    string
	RisksHazards    []string
}

type ValidationScopeMode string

const (
	ValidationScopeUnspecified ValidationScopeMode = ""
	ValidationScopeNarrow      ValidationScopeMode = "narrow"
	ValidationScopeFullPlan    ValidationScopeMode = "full_plan"
)

type ValidationScope struct {
	Mode      ValidationScopeMode
	Boundary  ChangeBoundary
	Rationale string
}

// Plan is the top-level structured record.
type Plan struct {
	ID            string
	Slug          string
	Title         string
	Status        PlanStatus
	ContentHash   string
	CreatedAt     string
	UpdatedAt     string
	WorkspaceID   string
	WorkspaceRoot string
	Purpose       string
	Scope         string
	Constraints   string
	NonGoals      string
	References    []Reference
	// ChangeBoundary is the plan's first-class blast-radius contract
	// (acceptance_allow / acceptance_deny). It is the source of truth for posture,
	// regression-anchor intent, validation scope, and execution reminders.
	ChangeBoundary   ChangeBoundary
	RegressionAnchor RegressionAnchor
	// BaselineSet is the execution-facing collection intent for new plans. It
	// is optional so imported and pre-cutover plans retain their legacy anchor
	// behavior unchanged.
	BaselineSet      BaselineSetIntent
	DefinitionOfDone string
	Phases           []Phase
	Supersedes       []string
	SupersededBy     []string
	RelevantContext  []RelevantContextItem
	// AUTHORED professional plan fields (see docs/concepts/PLAN-MODEL.md).
	ProblemStatement        string
	TargetOutcome           string
	Assumptions             string
	TechnicalApproach       string
	ValidationStrategy      string
	FinalValidationCommands []string
	RisksHazards            string
	ProhibitedApproaches    string
	// AUTHORED optional plan-time contract decisions (D1..Dn) and structured
	// assumptions with mitigations (contract decision D3). Empty renders nothing.
	Decisions       []PlanDecision
	AssumptionRisks []PlanAssumption
	Definitions     []PlanDefinition
	// AUTOFILLED/COMPUTED work posture (never agent-authored).
	WorkPosture       WorkPosture
	WorkPostureSource WorkPostureSource
	WorkPostureDetail string
	// GOVERNANCE: import bookkeeping (only when imported).
	ImportProvenance        *ImportProvenance
	PreservedLegacySections []LegacySection
	// COMPUTED projection metadata for the durable rendered markdown mirror.
	Mirror RenderedPlanMirror
}

// FieldClass describes who owns a top-level Plan field. Keeping this contract
// alongside Plan makes additions deliberate: callers, persistence, hashing,
// and wire tests can all derive their behaviour from the same declaration.
type FieldClass string

const (
	FieldClassAuthored   FieldClass = "authored"
	FieldClassIdentity   FieldClass = "identity"
	FieldClassComputed   FieldClass = "computed"
	FieldClassGovernance FieldClass = "governance"
	FieldClassGraph      FieldClass = "graph"
)

// PlanFieldClasses classifies every exported field in Plan. It is intentionally
// data, rather than a collection of per-consumer field lists.
var PlanFieldClasses = map[string]FieldClass{
	"ID":                      FieldClassIdentity,
	"Slug":                    FieldClassIdentity,
	"Title":                   FieldClassAuthored,
	"Status":                  FieldClassComputed,
	"ContentHash":             FieldClassComputed,
	"CreatedAt":               FieldClassIdentity,
	"UpdatedAt":               FieldClassIdentity,
	"WorkspaceID":             FieldClassIdentity,
	"WorkspaceRoot":           FieldClassIdentity,
	"Purpose":                 FieldClassAuthored,
	"Scope":                   FieldClassAuthored,
	"Constraints":             FieldClassAuthored,
	"NonGoals":                FieldClassAuthored,
	"References":              FieldClassAuthored,
	"ChangeBoundary":          FieldClassAuthored,
	"RegressionAnchor":        FieldClassAuthored,
	"BaselineSet":             FieldClassAuthored,
	"DefinitionOfDone":        FieldClassAuthored,
	"Phases":                  FieldClassAuthored,
	"Supersedes":              FieldClassGraph,
	"SupersededBy":            FieldClassGraph,
	"RelevantContext":         FieldClassAuthored,
	"ProblemStatement":        FieldClassAuthored,
	"TargetOutcome":           FieldClassAuthored,
	"Assumptions":             FieldClassAuthored,
	"TechnicalApproach":       FieldClassAuthored,
	"ValidationStrategy":      FieldClassAuthored,
	"FinalValidationCommands": FieldClassAuthored,
	"RisksHazards":            FieldClassAuthored,
	"ProhibitedApproaches":    FieldClassAuthored,
	"Decisions":               FieldClassAuthored,
	"AssumptionRisks":         FieldClassAuthored,
	"Definitions":             FieldClassAuthored,
	"WorkPosture":             FieldClassComputed,
	"WorkPostureSource":       FieldClassComputed,
	"WorkPostureDetail":       FieldClassComputed,
	"ImportProvenance":        FieldClassGovernance,
	"PreservedLegacySections": FieldClassGovernance,
	"Mirror":                  FieldClassComputed,
}

// PreserveNonAuthoredPlanFields copies all caller-unowned top-level fields
// from stored into incoming. Lifecycle rules that intentionally derive a new
// value (for example UpdatedAt or Status) run after this operation.
func PreserveNonAuthoredPlanFields(incoming, stored *Plan) {
	if incoming == nil || stored == nil {
		return
	}
	in := reflect.ValueOf(incoming).Elem()
	old := reflect.ValueOf(stored).Elem()
	typ := in.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" || PlanFieldClasses[field.Name] == FieldClassAuthored {
			continue
		}
		in.Field(i).Set(old.Field(i))
	}
}

// PlanEdge is one supersession/dependency edge between two plans.
type PlanEdge struct {
	FromPlanID string
	ToPlanID   string
	Kind       string // "supersedes" | "depends_on"
}

const (
	EdgeKindSupersedes = "supersedes"
	EdgeKindDependsOn  = "depends_on"
)

// PlanTemplate is a per-surface starter plan.
type PlanTemplate struct {
	ID          string
	Name        string
	Description string
	Surface     string // "cli" | "proto" | "ui" | "generic"
}

// ListFilter narrows ListPlans. A zero value matches all non-archived plans.
type ListFilter struct {
	Status          PlanStatus
	IncludeArchived bool
	WorkspaceID     string
	WorkspaceRoot   string
}
