// Package verdicts answers "does this model work for this operation, and should
// I use it?" with one evidence-derived grade per (model, operation).
//
// A verdict is never hand-maintained as a second list. It is derived on read from
// three kinds of evidence:
//
//   - curated claims in evidence.seed.json: what qualification runs and judged
//     comparisons established, each with a one-line reason and a pointer to the
//     artifact that proves it (the seed is observed facts, not a model attribute);
//   - live job history: recent terminal jobs per (model, operation) from the
//     durable job_trace table;
//   - operator ratings: 1-5 scores the operator gave to job outputs.
//
// Derive is the single rule table that combines them. Adding a model needs no
// verdict row: a model with no evidence is honestly "untested", and the first
// successful jobs and ratings move it on their own.
package verdicts

import (
	"fmt"
	"sort"
	"strings"
)

// Grade is a verdict on a model for one operation.
type Grade string

const (
	// Recommended: evidence says this is the best choice for the operation.
	Recommended Grade = "recommended"
	// Works: it produces acceptable results; a better choice may exist.
	Works Grade = "works"
	// Experimental: it ran, but evidence is thin or mixed.
	Experimental Grade = "experimental"
	// Untested: no evidence yet. This is the zero state of every new model.
	Untested Grade = "untested"
	// NotRecommended: it runs but lost comparisons or rates poorly. Avoid.
	NotRecommended Grade = "not_recommended"
	// Broken: it does not run, or fails the task it claims.
	Broken Grade = "broken"
)

// Grades lists every grade from best to worst, the order the roll-up and display
// use.
var Grades = []Grade{Recommended, Works, Experimental, Untested, NotRecommended, Broken}

// Valid reports whether g is a known grade.
func (g Grade) Valid() bool {
	for _, known := range Grades {
		if g == known {
			return true
		}
	}
	return false
}

// rank orders grades best-first (lower is better).
func (g Grade) rank() int {
	for i, known := range Grades {
		if g == known {
			return i
		}
	}
	return len(Grades)
}

// Label is the plain-words form shown to people ("not recommended").
func (g Grade) Label() string { return strings.ReplaceAll(string(g), "_", " ") }

// Evidence kinds. They say where a piece of evidence came from, so a reader can
// tell a curated qualification from a live signal.
const (
	KindQualification      = "qualification"
	KindJudgedComparison   = "judged_comparison"
	KindLiveRun            = "live_run"
	KindProviderEvidence   = "provider_evidence"
	KindJobHistory         = "job_history"
	KindOperatorRating     = "operator_rating"
	KindOperatorAssessment = "operator_assessment"
	// KindCatalogHold is the seed's recorded reason a model is switched off.
	KindCatalogHold = "catalog_hold"
)

// Evidence is one pointer behind a verdict.
type Evidence struct {
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
	// Ref points at the artifact: a path (research: prefix = the Image Tools
	// research topic), a job id, or a command that lists the live data.
	Ref  string `json:"ref"`
	Date string `json:"date,omitempty"`
}

// Verdict is the derived answer for one (model, operation), or a model's roll-up
// when Operation is empty.
type Verdict struct {
	Operation string     `json:"operation,omitempty"`
	Grade     Grade      `json:"grade"`
	Reason    string     `json:"reason"`
	Evidence  []Evidence `json:"evidence,omitempty"`
}

// String renders "grade: reason" for logs and plain CLI lines.
func (v Verdict) String() string { return fmt.Sprintf("%s: %s", v.Grade.Label(), v.Reason) }

// Rollup folds a model's per-operation verdicts into one model-level verdict:
// the best grade among the operations that have evidence, with the other
// operations named in the reason so a mixed record is never hidden behind its
// best row. A model with no evidence on any operation is untested.
func Rollup(perOp []Verdict) Verdict {
	var known []Verdict
	for _, v := range perOp {
		if v.Grade != Untested {
			known = append(known, v)
		}
	}
	if len(known) == 0 {
		return Verdict{Grade: Untested, Reason: "no qualification, job history or rating yet"}
	}
	sort.SliceStable(known, func(i, j int) bool { return known[i].Grade.rank() < known[j].Grade.rank() })
	best := known[0]
	out := Verdict{Grade: best.Grade, Reason: best.Reason, Evidence: best.Evidence}
	if best.Operation != "" {
		out.Reason = best.Operation + ": " + best.Reason
	}
	var others []string
	for _, v := range known[1:] {
		if v.Grade != best.Grade {
			others = append(others, fmt.Sprintf("%s %s", v.Operation, v.Grade.Label()))
		}
	}
	if len(others) > 0 {
		out.Reason += " (also " + strings.Join(others, ", ") + ")"
	}
	return out
}
