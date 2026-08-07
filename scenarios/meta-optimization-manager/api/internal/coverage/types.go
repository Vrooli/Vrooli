// Package coverage is the readiness-scoreboard domain: it reads each
// projection's denominator (via the owner's `space --projection <p> --json`
// verb), joins it against the live numerator registries (search-hub providers /
// test-genie health / prompt-manager graph health) behind seams, and computes
// per-projection coverage + denominator-confidence. It surfaces numbers and
// confidence and never decides; the numerator is computed live, never stored
// (only short-TTL snapshots are cached). See docs/concepts/COVERAGE-MODEL.md.
//
// Layering mirrors the canonical domain pattern:
//
//	handler → Service → {SpaceReader, NumeratorJoiner, SnapshotRepository}
//	             ↑              ↑ (faked in tests)        ↑
//	          (proto edge)   live owner reads         short-TTL cache
//
// The proto wire types live one floor up and never import this package; the
// handler is the only translation point (api-steer §7).
package coverage

import (
	"context"
	"time"

	"github.com/vrooli/api-core/spacedoc"
)

// Projection re-exports spacedoc.Projection so callers of this package work in
// one vocabulary.
type Projection = spacedoc.Projection

const (
	ProjectionAnswer   = spacedoc.ProjectionAnswer
	ProjectionValidate = spacedoc.ProjectionValidate
	ProjectionGuide    = spacedoc.ProjectionGuide
	ProjectionAct      = spacedoc.ProjectionAct
)

// AllProjections is the canonical iteration order for the scoreboard.
var AllProjections = []Projection{ProjectionAnswer, ProjectionValidate, ProjectionGuide, ProjectionAct}

// OwnerFor maps a projection to the scenario that owns its denominator + live
// numerator. The canonical model (COVERAGE-MODEL.md): Answer→search-hub,
// Validate→test-genie, Guide→prompt-manager, Act→program-runtime.
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

// Citation is a provenance pointer behind a cell's status/answer.
type Citation struct {
	Locator string // file:line / url / command
	Kind    string // code | doc | contract | runtime | external
	Note    string
}

// Cell is one denominator grid row enriched with its live-derived status and
// (on ExplainCell) its provenance.
type Cell struct {
	ID                string
	Projection        Projection
	Group             string
	Question          string
	Owner             string
	Status            spacedoc.CellStatus
	Basis             spacedoc.Basis
	Sufficiency       string
	Notes             []string
	Citations         []Citation
	ObservedAdherence ObservedAdherence
}

// ObservedAdherence stays separate from declared capability. An absent event
// source is unavailable evidence, never a zero-success result.
type ObservedAdherence struct {
	State       string  `json:"state"`
	Numerator   int64   `json:"numerator,omitempty"`
	Denominator int64   `json:"denominator,omitempty"`
	Ratio       float64 `json:"ratio,omitempty"`
	Reason      string  `json:"reason,omitempty"`
}

type AdherenceReader interface {
	ReadAdherence(context.Context, Projection, spacedoc.Cell) (ObservedAdherence, error)
}

// ProjectionCoverage is the computed coverage for one projection.
type ProjectionCoverage struct {
	Projection            Projection
	NowCount              int
	InReachCount          int
	MissingCount          int
	TotalCells            int
	CoverageRatio         float64 // now/total in [0,1]; computed live, never persisted
	DenominatorConfidence spacedoc.DenominatorConfidence
	ConfidenceRationale   string
	Available             bool   // false when the owner's space verb / registry was unreachable
	UnavailableReason     string // honest reason when Available is false
}

// EmpiricalTrendPoint is the latest trials trend surfaced on the scoreboard.
type EmpiricalTrendPoint struct {
	SuccessRate      float64
	MedianTokens     int64
	MedianDurationMs int64
	At               time.Time
}

// Status is the whole readiness scoreboard.
type Status struct {
	Projections      []ProjectionCoverage
	LatestTrialTrend *EmpiricalTrendPoint
	ComputedAt       time.Time
}

// Severity grades a base-document-integrity issue.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarn
	SeverityError
)

// BaseDocIssue is a broken/stale reference or shape problem in a space doc.
type BaseDocIssue struct {
	Projection Projection
	Code       string // missing_provider | guide_row_no_skill | guide_row_not_one_skill | ungraduated_pointer | graduation_ref_unresolved | denominator_unavailable | ...
	Message    string
	Location   string
	Severity   Severity
}

// BaseDocReport is the result of ValidateBaseDocs.
type BaseDocReport struct {
	Issues []BaseDocIssue
	OK     bool // true when no ERROR-severity issues exist (the self-honesty gate)
}
