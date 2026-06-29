// Package planmodel contains the neutral structured-plan kernel shared by the
// plan-manager domains. It has no transport, persistence, or domain-service
// dependencies.
package planmodel

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
	Unavailable    bool
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
	// AUTHORED professional phase fields (see docs/concepts/PLAN-MODEL.md).
	AffectedAreas   []string
	Steps           []string
	ExpectedOutputs []string
	Validation      string
	HandoffNotes    string
	RisksHazards    []string
}

// Plan is the top-level structured record.
type Plan struct {
	ID          string
	Slug        string
	Title       string
	Status      PlanStatus
	ContentHash string
	CreatedAt   string
	UpdatedAt   string
	Purpose     string
	Scope       string
	Constraints string
	NonGoals    string
	References  []Reference
	// ChangeBoundary is the plan's first-class blast-radius contract
	// (acceptance_allow / acceptance_deny). It is the source of truth for posture,
	// regression-anchor intent, validation scope, and execution reminders.
	ChangeBoundary   ChangeBoundary
	RegressionAnchor RegressionAnchor
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
	// AUTOFILLED/COMPUTED work posture (never agent-authored).
	WorkPosture       WorkPosture
	WorkPostureSource WorkPostureSource
	WorkPostureDetail string
	// GOVERNANCE: import bookkeeping (only when imported).
	ImportProvenance        *ImportProvenance
	PreservedLegacySections []LegacySection
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
}
