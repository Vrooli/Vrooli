// Package authoring is the guided-composer domain. It walks a plan's sections in
// order, runs a structure-validation gate as it goes (rejecting empty mandatory
// sections and an empty regression anchor), captures mechanical context behind
// seams, and drives relevant-context candidate discovery so a small local model
// supplies only genuine prose and final context judgment (OT-P0-002). The
// produced plan is written THROUGH the plans domain; this service does not own
// the record.
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → {SessionStore, PlanWriter, AnchorIntentDeriver,
//	            ↑            ↑ owned store    ↑ plans domain   ContextDiscoverer,
//	        (proto edge)   (faked in tests)                   ReferenceSuggester}
//	                                                          ↑ all behind seams,
//	                                                            all degrade gracefully
//
// Every autofill source degrades honestly: a nil seam or an error leaves that
// section for the author and marks the result degraded — NEVER a false fill. The
// structured Plan/Phase/Reference Go types live in the neutral planmodel kernel;
// authoring imports that kernel as the shared model. See
// docs/concepts/DOMAINS.md (authoring), docs/concepts/FLOWS.md, and
// docs/concepts/PLAN-MODEL.md.
package authoring

import (
	"context"

	planmodel "plan-manager/internal/planmodel"
)

// SectionKey is the stable identifier for one section of a plan-in-progress.
type SectionKey string

const (
	SectionPurpose              SectionKey = "purpose"
	SectionProblemStatement     SectionKey = "problem_statement"
	SectionTargetOutcome        SectionKey = "target_outcome"
	SectionScope                SectionKey = "scope"
	SectionNonGoals             SectionKey = "non_goals"
	SectionAssumptions          SectionKey = "assumptions"
	SectionWorkPosture          SectionKey = "work_posture"
	SectionTechnicalApproach    SectionKey = "technical_approach"
	SectionConstraints          SectionKey = "constraints"
	SectionProhibitedApproaches SectionKey = "prohibited_approaches"
	SectionAcceptanceBoundary   SectionKey = "acceptance_boundary"
	SectionReferences           SectionKey = "references"
	SectionRegressionAnchor     SectionKey = "regression_anchor"
	SectionRequiredReading      SectionKey = "required_reading"
	SectionRelevantContext      SectionKey = "relevant_context"
	SectionValidationStrategy   SectionKey = "validation_strategy"
	SectionDefinitionOfDone     SectionKey = "definition_of_done"
	SectionPhases               SectionKey = "phases"
)

// PhaseField is one structured phase-draft field the authoring wizard can ask
// for independently. This is the phase-native path that replaces asking a small
// model to submit one large markdown "phases" blob.
type PhaseField string

const (
	PhaseFieldTitle            PhaseField = "title"
	PhaseFieldIntent           PhaseField = "intent"
	PhaseFieldAffectedAreas    PhaseField = "affected_areas"
	PhaseFieldReferences       PhaseField = "references"
	PhaseFieldSteps            PhaseField = "steps"
	PhaseFieldExpectedOutputs  PhaseField = "expected_outputs"
	PhaseFieldValidation       PhaseField = "validation"
	PhaseFieldAcceptance       PhaseField = "acceptance"
	PhaseFieldRisksHazards     PhaseField = "risks_hazards"
	PhaseFieldHandoffNotes     PhaseField = "handoff_notes"
	PhaseFieldRequiredReading  PhaseField = "required_reading"
	PhaseFieldReminders        PhaseField = "reminders"
	PhaseFieldNoCodeRefsReason PhaseField = "no_code_refs_reason"
	PhaseFieldRelevantContext  PhaseField = "relevant_context"
)

// Section is one authored or auto-filled section of a plan-in-progress. key/
// label/mandatory are the skeleton (seeded by StartSession); content/filled/
// autofilled accrue as the author or an autofill source touches the section.
type Section struct {
	Key        SectionKey
	Label      string
	Content    string
	Mandatory  bool
	Filled     bool
	Autofilled bool
}

// NextActionKind classifies how strongly the wizard recommends an action.
type NextActionKind string

const (
	NextActionRecommended NextActionKind = "recommended"
	NextActionAlternative NextActionKind = "alternative"
	NextActionOptional    NextActionKind = "optional"
	NextActionRecovery    NextActionKind = "recovery"
)

// NextAction is one API-owned concrete action for the current guided step. Argv
// is canonical; CLI/UI may format it, but they must not decide workflow order.
type NextAction struct {
	ID                 string
	Kind               NextActionKind
	Label              string
	Reason             string
	Argv               []string
	ContentPlaceholder string
	BlockedBy          []string
}

// GuidedStep is deterministic just-in-time wizard steering for the current
// authoring step. It moves the old prose skill reminders and next-command
// decisions into the runtime, so a smaller model receives the relevant rule and
// concrete command exactly when it is authoring a field.
type GuidedStep struct {
	StepKind       string
	Title          string
	Summary        string
	Instructions   []string
	RequiredInputs []string
	Examples       []string
	CommonMistakes []string
	NextActions    []NextAction
}

// PhaseDraft is a structured phase being authored before Finalize maps it into
// the plans SSOT. References are parsed into structured locators immediately so
// phase-specific validation, staleness and execution context can be computed
// without re-parsing a prose blob later.
type PhaseDraft struct {
	ID               string
	Order            int
	Title            string
	Intent           string
	AffectedAreas    []string
	References       []planmodel.Reference
	Steps            []string
	ExpectedOutputs  []string
	Validation       string
	Acceptance       string
	RisksHazards     []string
	HandoffNotes     string
	RequiredReading  []string
	Reminders        []string
	NoCodeRefsReason string
	RelevantContext  []planmodel.RelevantContextItem
}

// Session is the transient state of a guided authoring flow. It persists across
// CLI calls via the SessionStore. The plan it produces is owned by the plans
// domain (PlanID is set after Finalize).
type Session struct {
	ID                  string
	Title               string
	Slug                string
	Sections            []Section
	CurrentSectionKey   SectionKey
	PhaseDrafts         []PhaseDraft
	CurrentPhaseID      string
	RelevantContext     []planmodel.RelevantContextItem
	ContextCandidates   []ContextCandidate
	ReferenceCandidates []ReferenceCandidate
	Finalized           bool
	PlanID              string
	CreatedAt           string
	UpdatedAt           string
}

// StructureViolation is one structure/authoring-gate failure (an empty mandatory
// section, an empty regression anchor, or an invalid current command reference).
type StructureViolation struct {
	SectionKey SectionKey
	Message    string
}

type CommandReferenceValidator interface {
	ValidateCommandReference(context.Context, CommandReferenceRequest) (CommandReferenceResult, error)
}

type CommandReferenceRequest struct {
	CommandText string
	Qualifiers  []string
}

type CommandReferenceResult struct {
	Verdict         string
	ValidationLevel string
	Issues          []CommandIssue
	Suggestions     []string
	Guidance        []string
}

type CommandIssue struct {
	Code    string
	Message string
}

// AutofillSource names one mechanical-section autofill source. The regression
// anchor is the only mechanical autofill: references are discovered as reviewable
// candidates (SuggestReferences) and setup context flows through context
// discovery/acceptance.
type AutofillSource string

const (
	AutofillRegressionAnchor AutofillSource = "regression_anchor"
)

// AutofillResult reports the outcome of one autofill source. Degraded=true means
// the backing dependency was down and the section was left for the author — never
// a false "filled".
type AutofillResult struct {
	Source     AutofillSource
	SectionKey SectionKey
	Filled     bool
	Degraded   bool
	Detail     string
}

type ContextCandidateStatus string

const (
	ContextCandidatePending  ContextCandidateStatus = "pending"
	ContextCandidateAccepted ContextCandidateStatus = "accepted"
	ContextCandidateRejected ContextCandidateStatus = "rejected"
)

// ContextCandidate is a discovered setup item awaiting author judgment.
type ContextCandidate struct {
	ID              string
	Item            planmodel.RelevantContextItem
	Concept         string
	Source          string
	Degraded        bool
	Detail          string
	Status          ContextCandidateStatus
	RejectionReason string
}

// ReferenceCandidateStatus mirrors ContextCandidateStatus for the reference
// review lifecycle.
type ReferenceCandidateStatus string

const (
	ReferenceCandidatePending  ReferenceCandidateStatus = "pending"
	ReferenceCandidateAccepted ReferenceCandidateStatus = "accepted"
	ReferenceCandidateRejected ReferenceCandidateStatus = "rejected"
)

// ReferenceCandidate is a discovered code/doc/req locator awaiting author
// judgment. The suggester proposes it from search-hub's Answer projection
// (routed by locator shape); only accepted candidates finalize into the
// references section. Mirrors ContextCandidate so the two discovery sources share
// one curate-pattern.
type ReferenceCandidate struct {
	ID              string
	Reference       planmodel.Reference
	Source          string
	Confidence      float64
	Degraded        bool
	Detail          string
	Status          ReferenceCandidateStatus
	RejectionReason string
}

// sectionSpec is the seed shape for one section in the default skeleton.
type sectionSpec struct {
	Key       SectionKey
	Label     string
	Mandatory bool
}

// defaultSkeleton is the ordered section skeleton StartSession seeds. The
// mandatory sections gate Finalize; the optional ones (constraints, non_goals,
// relevant_context) may be left empty. The regression-anchor section carries
// typed intent only when the anchor autofill supplies it; references carry only
// reviewed (accepted) search-hub suggestions or a NO_CODE_REFS reason. phases is
// mandatory so a plan always has work.
var defaultSkeleton = []sectionSpec{
	{Key: SectionPurpose, Label: "Purpose", Mandatory: true},
	{Key: SectionProblemStatement, Label: "Problem / Need", Mandatory: true},
	{Key: SectionTargetOutcome, Label: "Target Outcome", Mandatory: true},
	{Key: SectionScope, Label: "Scope", Mandatory: true},
	{Key: SectionNonGoals, Label: "Non-goals", Mandatory: false},
	{Key: SectionAssumptions, Label: "Assumptions", Mandatory: false},
	{Key: SectionWorkPosture, Label: "Work Posture", Mandatory: false},
	{Key: SectionTechnicalApproach, Label: "Technical Approach", Mandatory: true},
	{Key: SectionConstraints, Label: "Constraints", Mandatory: false},
	{Key: SectionProhibitedApproaches, Label: "Prohibited Approaches", Mandatory: false},
	{Key: SectionAcceptanceBoundary, Label: "Change Boundary", Mandatory: true},
	{Key: SectionReferences, Label: "References", Mandatory: true},
	{Key: SectionRegressionAnchor, Label: "Regression anchor", Mandatory: true},
	{Key: SectionRelevantContext, Label: "Relevant context", Mandatory: false},
	{Key: SectionValidationStrategy, Label: "Validation Strategy", Mandatory: true},
	{Key: SectionDefinitionOfDone, Label: "Definition of Done", Mandatory: true},
	{Key: SectionPhases, Label: "Phases", Mandatory: true},
}

// newSkeleton returns a fresh ordered section list from the default skeleton.
func newSkeleton() []Section {
	out := make([]Section, 0, len(defaultSkeleton))
	for _, spec := range defaultSkeleton {
		out = append(out, Section{Key: spec.Key, Label: spec.Label, Mandatory: spec.Mandatory})
	}
	return out
}

// PlanForSession is the resolved subset of the plans model authoring writes
// through. It is the shared plans.Plan; aliased here so the seam signature reads
// in authoring's vocabulary without re-importing at every call site.
type PlanForSession = planmodel.Plan
