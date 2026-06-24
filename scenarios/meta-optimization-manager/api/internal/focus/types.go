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
	"github.com/vrooli/api-core/spacedoc"
)

// Projection re-exports spacedoc.Projection so callers of this package work in
// one vocabulary. An empty Projection denotes a cross-cutting / convergence gap
// that is not projection-scoped.
type Projection = spacedoc.Projection

const (
	ProjectionAnswer   = spacedoc.ProjectionAnswer
	ProjectionValidate = spacedoc.ProjectionValidate
	ProjectionGuide    = spacedoc.ProjectionGuide
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
	default:
		return ""
	}
}

// Gap is one known gap with its qualitative context. A gap derived from a
// denominator cell carries the cell's projection, status, and source id; a
// registry-only (global) gap carries Global=true and may have an empty
// projection. Notes/Approaches/FollowUps accumulate the team's thinking.
type Gap struct {
	ID           string
	Projection   Projection
	Title        string
	Status       spacedoc.CellStatus
	SourceCellID string
	Global       bool
	Notes        []string
	Approaches   []string
	FollowUps    []string
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

// GapFilter narrows ListGaps. A zero value matches everything.
type GapFilter struct {
	Projection Projection
	CellID     string
	Status     spacedoc.CellStatus
}
