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

// SkillFitnessVerdict is DTV's derived cross-golden fitness classification for a
// skill — "is this skill currently fit to run unattended?" It is NOT efficacy
// (how well it fixes a broken target); DTV cannot observe that on pristine
// goldens.
type SkillFitnessVerdict int

const (
	SkillFitnessVerdictUnspecified SkillFitnessVerdict = 0
	// SkillFitnessVerdictUnknown: no validation records exist for the skill.
	SkillFitnessVerdictUnknown SkillFitnessVerdict = 1
	// SkillFitnessVerdictGreen: the latest record on every golden is PASS.
	SkillFitnessVerdictGreen SkillFitnessVerdict = 2
	// SkillFitnessVerdictYellow: a latest record is UNEXPECTED_MUTATION (or
	// otherwise non-PASS but runnable); no latest record is a run failure.
	SkillFitnessVerdictYellow SkillFitnessVerdict = 3
	// SkillFitnessVerdictRed: a latest record is RUN_FAILURE (intrinsic
	// non-convergence / crash — Flavor-1 thrashing).
	SkillFitnessVerdictRed SkillFitnessVerdict = 4
)

// GoldenSkillSnapshot is the per-golden slice of a skill's fitness.
type GoldenSkillSnapshot struct {
	GoldenSlug    string
	LatestVerdict vr.Verdict
	Stale         bool
	RunCount      int
}

// SkillFitness is the cross-golden trust/cost/convergence aggregate for one
// skill. Raw counts are always populated so consumers can re-policy without a
// DTV change.
type SkillFitness struct {
	SkillID string

	PassCount               int64
	UnexpectedMutationCount int64
	RunFailureCount         int64
	ToolFailureCount        int64
	TotalRuns               int64
	PassRate                float64

	TotalTokens       int64
	AvgTokens         float64
	TotalCostUSDMicro int64
	AvgCostUSDMicro   float64
	TotalDurationMS   int64
	AvgDurationMS     float64

	UniqueDiffHashes int
	ConvergenceRatio float64

	LatestVerdict vr.Verdict
	AnyStale      bool

	Verdict  SkillFitnessVerdict
	ByGolden map[string]GoldenSkillSnapshot
}

// ErrInvalidReport is the typed sentinel for input validation.
type ErrInvalidReport struct {
	Field  string
	Reason string
}

func (e ErrInvalidReport) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
