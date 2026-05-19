// Package report is the pure-composition report domain (OT-P0-008).
// Owns no storage; reads from golden, skill_catalog, manifest,
// validation_record, staleness via narrow seam interfaces.
package report

import (
	"fmt"

	vr "development-toolchain-validator/internal/validation_record"
)

// TupleVerdict pairs a subject with its latest verdict against a
// golden.
type TupleVerdict struct {
	TupleKind      vr.TupleKind
	SubjectID      string
	LatestVerdict  vr.Verdict
	LatestRecordID string
	Stale          bool
}

// GoldenSummary is the dashboard's per-golden roll-up.
type GoldenSummary struct {
	GoldenSlug    string
	SkillVerdicts []TupleVerdict
	ToolVerdicts  []TupleVerdict
	StaleCount    int
}

// TupleHistory is the paginated history view for one (tuple_kind,
// subject_id, golden_slug) tuple.
type TupleHistory struct {
	TupleKind     vr.TupleKind
	SubjectID     string
	GoldenSlug    string
	Records       []vr.Record
	NextPageToken string
}

// CoverageRow is one row in the per-golden coverage grid.
type CoverageRow struct {
	TupleKind   vr.TupleKind
	SubjectID   string
	Verdict     vr.Verdict
	Stale       bool
	HasManifest bool
}

// Coverage is the per-golden verdict grid.
type Coverage struct {
	GoldenSlug string
	Rows       []CoverageRow
}

// ErrInvalidReport is the typed sentinel for input validation.
type ErrInvalidReport struct {
	Field  string
	Reason string
}

func (e ErrInvalidReport) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
