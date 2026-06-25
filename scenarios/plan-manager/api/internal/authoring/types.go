// Package authoring is the guided-composer domain. It walks a plan's sections in
// order, runs a structure-validation gate as it goes (rejecting empty mandatory
// sections and an empty regression anchor), and auto-fills the mechanical
// sections behind seams — the regression anchor (via git-control-tower),
// required-reading (via prompt-manager plan-skill-discovery) and code references
// (via code-facts) — so a small local model supplies only genuine prose
// (OT-P0-002). The produced plan is written THROUGH the plans domain; this
// service does not own the record.
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → {SessionStore, PlanWriter, AnchorAutofiller,
//	            ↑            ↑ owned store    ↑ plans domain   RequiredReadingSource,
//	        (proto edge)   (faked in tests)                   ReferenceExtractor}
//	                                                          ↑ all behind seams,
//	                                                            all degrade gracefully
//
// Every autofill source degrades honestly: a nil seam or an error leaves that
// section for the author and marks the result degraded — NEVER a false fill. The
// structured Plan/Phase/Reference Go types are owned by the plans domain;
// authoring imports them (internal/plans) as the shared model. See
// docs/concepts/DOMAINS.md (authoring), docs/concepts/FLOWS.md, and
// docs/concepts/PLAN-MODEL.md.
package authoring

import internalplans "plan-manager/internal/plans"

// SectionKey is the stable identifier for one section of a plan-in-progress.
type SectionKey string

const (
	SectionPurpose          SectionKey = "purpose"
	SectionScope            SectionKey = "scope"
	SectionConstraints      SectionKey = "constraints"
	SectionNonGoals         SectionKey = "non_goals"
	SectionReferences       SectionKey = "references"
	SectionRegressionAnchor SectionKey = "regression_anchor"
	SectionRequiredReading  SectionKey = "required_reading"
	SectionDefinitionOfDone SectionKey = "definition_of_done"
	SectionPhases           SectionKey = "phases"
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

// Session is the transient state of a guided authoring flow. It persists across
// CLI calls via the SessionStore. The plan it produces is owned by the plans
// domain (PlanID is set after Finalize).
type Session struct {
	ID                string
	Title             string
	Slug              string
	Sections          []Section
	CurrentSectionKey SectionKey
	Finalized         bool
	PlanID            string
	CreatedAt         string
	UpdatedAt         string
}

// StructureViolation is one structure-gate failure (an empty mandatory section
// or an empty regression anchor).
type StructureViolation struct {
	SectionKey SectionKey
	Message    string
}

// AutofillSource names one mechanical-section autofill source.
type AutofillSource string

const (
	AutofillRegressionAnchor AutofillSource = "regression_anchor"
	AutofillRequiredReading  AutofillSource = "required_reading"
	AutofillReferences       AutofillSource = "references"
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

// sectionSpec is the seed shape for one section in the default skeleton.
type sectionSpec struct {
	Key       SectionKey
	Label     string
	Mandatory bool
}

// defaultSkeleton is the ordered section skeleton StartSession seeds. The
// mandatory sections gate Finalize; the optional ones (constraints, non_goals,
// references) may be left empty. The autofillable mechanical sections
// (regression_anchor, required_reading, references) carry prose only when an
// autofill source supplies it. phases is mandatory so a plan always has work.
var defaultSkeleton = []sectionSpec{
	{Key: SectionPurpose, Label: "Purpose", Mandatory: true},
	{Key: SectionScope, Label: "Scope", Mandatory: true},
	{Key: SectionConstraints, Label: "Constraints", Mandatory: false},
	{Key: SectionNonGoals, Label: "Non-goals", Mandatory: false},
	{Key: SectionReferences, Label: "References", Mandatory: false},
	{Key: SectionRegressionAnchor, Label: "Regression anchor", Mandatory: true},
	{Key: SectionRequiredReading, Label: "Required reading", Mandatory: false},
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
type PlanForSession = internalplans.Plan
