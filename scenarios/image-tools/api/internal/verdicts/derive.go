package verdicts

import (
	"fmt"
	"sort"
)

// JobStats is the recent terminal job history for one (model, operation).
// Canceled jobs are not counted: a cancel says nothing about the model.
type JobStats struct {
	Succeeded int
	Failed    int
}

// Terminal is the number of counted jobs.
func (s JobStats) Terminal() int { return s.Succeeded + s.Failed }

// RatingStats summarises the operator's 1-5 ratings for one (model, operation).
type RatingStats struct {
	Count int
	Mean  float64
}

// Signals are the live evidence for one (model, operation).
type Signals struct {
	Jobs    JobStats
	Ratings RatingStats
}

// Policy holds the thresholds Derive applies. They are data so a test (or a
// future operator setting) can state the rule without editing the logic.
type Policy struct {
	// MinRatings is how many ratings are needed before they move a grade. Fewer
	// ratings are still shown as evidence.
	MinRatings int
	// LiftMean and DropMean bound the mean rating: at or above LiftMean the
	// operator is satisfied (recommended); at or below DropMean the operator is
	// not (not recommended); between them ratings leave the grade alone.
	LiftMean float64
	DropMean float64
	// MinJobs is how many terminal jobs make job history meaningful, and
	// WorksRate / FailRate bound the success rate that earns "works" or
	// withdraws a good grade.
	MinJobs   int
	WorksRate float64
	FailRate  float64
	// DemoteMinJobs is the history needed to demote a curated good grade: a
	// curated "recommended" is not overturned by two unlucky jobs.
	DemoteMinJobs int
}

// DefaultPolicy is the rule set the service uses.
var DefaultPolicy = Policy{
	MinRatings: 3, LiftMean: 4.0, DropMean: 2.0,
	MinJobs: 3, WorksRate: 0.8, FailRate: 0.5, DemoteMinJobs: 5,
}

// Derive combines curated claims with live signals into one verdict.
//
// Rule order (each step sees the grade the previous step left):
//
//  1. Claim: the latest curated claim sets the starting grade and reason; with
//     none the model is untested.
//  2. Job history: with a claim of broken, a record of recent success
//     (>= MinJobs terminal jobs, success rate >= WorksRate) lifts it to
//     experimental, since the curated claim is stale; with a good claim
//     (recommended or works) a record of failure (>= DemoteMinJobs terminal,
//     rate < FailRate) demotes it to experimental; with none, success earns
//     experimental (any success) or works (>= MinJobs at >= WorksRate), and
//     repeated failure with no success earns broken.
//  3. Ratings: with >= MinRatings ratings, a mean >= LiftMean lifts a grade to
//     recommended and a mean <= DropMean drops it to not recommended, unless the
//     grade is broken (a low mean still applies after a job-history demotion; a
//     high one does not undo it). A mean in between lifts an untested or
//     experimental grade to works, again unless job history just demoted it.
//
// The reason always names the evidence that decided the grade.
func Derive(claims []Claim, sig Signals, p Policy) Verdict {
	v := Verdict{Grade: Untested, Reason: "no qualification, job history or rating yet"}
	if latest, ok := latestClaim(claims); ok {
		v.Grade, v.Reason = latest.Grade, latest.Reason
	}
	for _, c := range sortedClaims(claims) {
		v.Evidence = append(v.Evidence, Evidence{Kind: c.Kind, Summary: c.Reason, Ref: c.Ref, Date: c.Date})
	}

	demotedByJobs := false
	jobs := sig.Jobs
	if jobs.Terminal() > 0 {
		rate := float64(jobs.Succeeded) / float64(jobs.Terminal())
		summary := fmt.Sprintf("%d of %d recent jobs succeeded", jobs.Succeeded, jobs.Terminal())
		switch {
		case v.Grade == Broken && jobs.Terminal() >= p.MinJobs && rate >= p.WorksRate:
			v.Grade, v.Reason = Experimental, fmt.Sprintf("curated record says broken but %s; re-qualify", summary)
		case (v.Grade == Recommended || v.Grade == Works) && jobs.Terminal() >= p.DemoteMinJobs && rate < p.FailRate:
			v.Grade, v.Reason = Experimental, fmt.Sprintf("curated record says %s but only %s", v.Grade.Label(), summary)
			demotedByJobs = true
		case v.Grade == Untested && jobs.Succeeded == 0 && jobs.Failed >= p.MinJobs:
			v.Grade, v.Reason = Broken, fmt.Sprintf("%s; none succeeded", summary)
		case v.Grade == Untested && jobs.Terminal() >= p.MinJobs && rate >= p.WorksRate:
			v.Grade, v.Reason = Works, summary
		case v.Grade == Untested && jobs.Succeeded > 0:
			v.Grade, v.Reason = Experimental, summary+"; not enough history yet"
		}
		v.Evidence = append(v.Evidence, Evidence{Kind: KindJobHistory, Summary: summary, Ref: "image-tools measures usage"})
	}

	ratings := sig.Ratings
	if ratings.Count > 0 {
		summary := fmt.Sprintf("operator rated %.1f of 5 over %d output(s)", ratings.Mean, ratings.Count)
		if ratings.Count >= p.MinRatings && v.Grade != Broken {
			switch {
			case ratings.Mean <= p.DropMean:
				v.Grade, v.Reason = NotRecommended, summary
			case ratings.Mean >= p.LiftMean && !demotedByJobs:
				v.Grade, v.Reason = Recommended, summary
			case !demotedByJobs && (v.Grade == Untested || v.Grade == Experimental):
				v.Grade, v.Reason = Works, summary
			}
		}
		v.Evidence = append(v.Evidence, Evidence{Kind: KindOperatorRating, Summary: summary, Ref: "image-tools jobs ratings"})
	}
	return v
}

// latestClaim returns the claim that sets the starting grade: the newest by
// date, ties broken by the entry written later in the seed.
func latestClaim(claims []Claim) (Claim, bool) {
	if len(claims) == 0 {
		return Claim{}, false
	}
	sorted := sortedClaims(claims)
	return sorted[0], true
}

// sortedClaims orders claims newest first. Among claims with the same date the
// one written later in the seed comes first, because a later entry is a later
// finding.
func sortedClaims(claims []Claim) []Claim {
	out := make([]Claim, len(claims))
	for i, c := range claims {
		out[len(claims)-1-i] = c
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	return out
}

// Hold is the catalog's own reason a model is switched off (the seed's
// "disabled" record). It is not evidence about quality; it is the answer to "why
// can't I use this" and so shows wherever the model's verdict does.
type Hold struct {
	// Reason is the one-line why.
	Reason string
	// Breaks means the model cannot run on this build (a runtime hold).
	Breaks bool
	// Ref points at where the reason is recorded.
	Ref string
}

// ApplyHold adds a catalog hold to a derived verdict. A model whose evidence has
// not earned a grade (untested) reads as its hold: broken when the hold says it cannot run, otherwise
// untested with the hold as the reason. A model with a graded verdict keeps its
// grade and reason (the evidence is about its output, the hold is why it is not
// offered) and carries the hold as one more evidence entry.
func ApplyHold(v Verdict, h Hold) Verdict {
	if h.Reason == "" {
		return v
	}
	reason := "disabled: " + h.Reason
	if v.Grade == Untested {
		v.Reason = reason
		if h.Breaks {
			v.Grade = Broken
		}
	}
	v.Evidence = append(v.Evidence, Evidence{Kind: KindCatalogHold, Summary: reason, Ref: h.Ref})
	return v
}
