// Package focus is the gaps-registry + prioritization domain. It aggregates the
// known gaps — the non-NOW denominator cells across every projection (read live
// via the owner space verbs, behind a seam) plus the cross-cutting/global gaps
// and explored-but-unbuilt ideas held in the owned SQLite registry — and ranks
// them by impact × importance so the team's "what do I do next?" question gets a
// ranked answer with qualitative context, not just a percentage.
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → {GapSource, Repository}
//	             ↑           ↑ (faked in tests)  ↑
//	          (proto edge)  live space reads   owned registry (SQLite)
//
// The proto wire types live one floor up and never import this package; the
// handler is the only translation point (api-steer §7). See
// docs/concepts/DOMAINS.md (focus) and docs/concepts/COVERAGE-MODEL.md.
package focus

import (
	"context"

	"github.com/vrooli/api-core/spacedoc"
)

// Axis identifies whether a gap is a declared coverage deficit or an observed
// empirical problem. Empirical evidence has no enumerable denominator, so it is
// kept on a separate axis and described by recurrence plus traceability.
type Axis string

const (
	AxisCoverage  Axis = "coverage"
	AxisEmpirical Axis = "empirical"
)

// Projection re-exports spacedoc.Projection so callers of this package work in
// one vocabulary. An empty Projection denotes a cross-cutting / convergence gap
// that is not projection-scoped.
type Projection = spacedoc.Projection

const (
	ProjectionAnswer   = spacedoc.ProjectionAnswer
	ProjectionValidate = spacedoc.ProjectionValidate
	ProjectionGuide    = spacedoc.ProjectionGuide
	ProjectionAct      = spacedoc.ProjectionAct
)

// OwnerFor maps a projection to the scenario that owns its denominator, or ""
// for an unknown projection (used to validate a projection filter).
func OwnerFor(p Projection) string {
	switch p {
	case ProjectionAnswer:
		return "search-hub"
	case ProjectionValidate:
		return "test-genie"
	case ProjectionGuide:
		return "prompt-manager"
	case ProjectionAct:
		return "program-runtime"
	default:
		return ""
	}
}

// Gap is one known gap with its qualitative context. A gap derived from a
// denominator cell carries the cell's projection, status, and source id; a
// registry-only (global) gap carries Global=true and may have an empty
// projection. Notes/Approaches/FollowUps accumulate the team's thinking.
type Gap struct {
	ID                 string
	Axis               Axis
	Projection         Projection
	Title              string
	Status             spacedoc.CellStatus
	SourceCellID       string
	ProviderIDs        []string
	Global             bool
	EvidenceSource     string
	EvidenceLocator    string
	Recurrence         int
	AvailabilityReason string
	ConditionStatus    string
	CauseKey           string
	AffectedCellIDs    []string
	AffectedCellCount  int
	MaturityFindings   []MaturityFinding
	Notes              []string
	Approaches         []string
	FollowUps          []string
}

// MaturityFinding is the actionable evidence returned by Search Hub for one
// blocking maturity rule. Keeping the finding as a structured value prevents
// the focus board from reducing a repairable defect to an opaque code.
type MaturityFinding struct {
	Code          string
	Message       string
	Location      string
	Remediation   string
	FixClass      string
	RepairCommand string
}

type ConditionInstrumentation struct {
	Healthy        int
	Degraded       int
	Dormant        int
	Uninstrumented int
	Unavailable    int
	Instrumented   int
	Total          int
	FilteredOut    int
}

type ConditionReport struct {
	Gaps            []Gap
	Instrumentation ConditionInstrumentation
}

// RouterQualityFinding is the small cross-domain read model for the coverage
// self-honesty signal. Coverage owns validation; focus only groups its
// router-quality findings into an actionable shared cause.
type RouterQualityFinding struct {
	Projection Projection
	CellID     string
	Owner      string
	Message    string
	Locator    string
}

// SubstrateObservation is an operator-facing health fact for a shared search
// dependency. Focus reports unhealthy observations; it does not remediate
// resources or own their lifecycle.
type SubstrateObservation struct {
	Name    string
	Healthy bool
	Reason  string
	Locator string
}

// FocusItem is a ranked next-best gap: the gap plus its impact × importance
// decomposition and a human rationale. Scoring lives only here, never on Gap.
type FocusItem struct {
	Gap        Gap
	Impact     float64
	Importance float64
	Priority   float64 // impact × importance
	Rationale  string
}

// ProviderInsight is the small, provider-agnostic slice of Search Hub
// telemetry needed to prioritize coverage gaps. The focus domain deliberately
// does not import Search Hub's transport or storage types.
type ProviderInsight struct {
	ProviderID      string
	ProviderGroup   string
	TimesRouted     int64
	DegradationRate float64
}

// ProviderInsights is the read seam for the optional third ranking factor.
// Implementations may be remote; a failure must degrade the focus response,
// never prevent the board from returning its structural ranking.
type ProviderInsights interface {
	Insights(ctx context.Context) ([]ProviderInsight, error)
}

// MaturityObservation is the small, read-only slice of Search Hub's
// validation result needed by the condition axis. Findings remain owned by
// Search Hub; focus only turns blocking evidence into a rankable item.
type MaturityObservation struct {
	Scenario      string
	BlockingCodes []string
	Findings      []MaturityFinding
}

// MaturityReader reads Search Hub's per-scenario maturity evidence. It is
// optional and independently degradable, like ProviderInsights.
type MaturityReader interface {
	Maturity(ctx context.Context) ([]MaturityObservation, error)
}

// FocusResult is the honest focus read model. Items remain useful when a live
// join or telemetry read fails, but callers can distinguish that fallback from
// a fully live computation.
type FocusResult struct {
	Items          []FocusItem
	Degraded       bool
	DegradedReason string
}

// GapFilter narrows ListGaps. A zero value matches everything.
type GapFilter struct {
	Projection Projection
	CellID     string
	Status     spacedoc.CellStatus
}
