package aisearch

import (
	"context"
	"fmt"
)

type CaseOutcome string

const (
	CaseOutcomeHit          CaseOutcome = "hit"
	CaseOutcomeMiss         CaseOutcome = "miss"
	CaseOutcomeJunkRejected CaseOutcome = "junk_rejected"
	CaseOutcomeJunkLeaked   CaseOutcome = "junk_leaked"
	CaseOutcomeNotGradeable CaseOutcome = "not_gradeable"
)

type Referential string

const (
	ReferentialUnknown Referential = "unknown"
	ReferentialLive    Referential = "live"
	ReferentialHard    Referential = "hard"
	ReferentialStale   Referential = "stale"
)

// SuiteSearcher is the shared eval seam. *Service satisfies it, and scenarios
// with product-shaped facades can adapt their public hit type at the boundary.
type SuiteSearcher interface {
	Search(ctx context.Context, q SearchQuery, opts ...SearchOption) (SearchResponse, error)
}

type CaseReport struct {
	CaseID           string
	Query            string
	Outcome          CaseOutcome
	Referential      Referential
	ExpectedRank     int
	ObservedTopScore float64
	Error            error
}

type SuiteReport struct {
	Policy             ScoringPolicy
	Recall             float64
	Hits               int
	GradeablePositives int
	Misses             []CaseReport
	Stale              []CaseReport
	ExcludedCandidate  []TestCase
	PerCase            []CaseReport
}

func (r SuiteReport) MeetsTarget() bool {
	return r.Recall+1e-9 >= r.Policy.RecallTarget
}

// GradeSuite runs every suite case at policy.DeepK and applies the shared
// acceptance denominator: reviewed, non-stale positives only.
func GradeSuite(ctx context.Context, searcher SuiteSearcher, suite TestSuite, policy ScoringPolicy) (SuiteReport, error) {
	if searcher == nil {
		return SuiteReport{}, fmt.Errorf("nil searcher")
	}
	if err := suite.Validate(); err != nil {
		return SuiteReport{}, err
	}
	if policy.GateK <= 0 || policy.DeepK <= 0 {
		policy = DefaultScoringPolicy
	} else if policy.DeepK < policy.GateK {
		policy.DeepK = policy.GateK
	}

	report := SuiteReport{Policy: policy}
	for _, c := range suite.Cases {
		if c.IsCandidate() {
			report.ExcludedCandidate = append(report.ExcludedCandidate, c)
			continue
		}

		resp, err := searcher.Search(ctx, SearchQuery{Query: c.Query, Scope: c.ResolvedScope(), Limit: policy.DeepK})
		cr := CaseReport{CaseID: c.ID, Query: c.Query}
		if err != nil {
			cr.Outcome = CaseOutcomeMiss
			cr.Error = err
			report.PerCase = append(report.PerCase, cr)
			if isPositiveCase(c) {
				report.GradeablePositives++
				report.Misses = append(report.Misses, cr)
			}
			continue
		}

		cr.Outcome = GradeCase(resp.Results, c, policy)
		cr.Referential = ClassifyExpectID(resp.Results, c, policy)
		cr.ExpectedRank = ExpectedRank(resp.Results, c.ExpectIDs)
		if len(resp.Results) > 0 {
			cr.ObservedTopScore = resp.Results[0].Score
		}
		report.PerCase = append(report.PerCase, cr)

		if !isPositiveCase(c) {
			continue
		}
		if cr.Referential == ReferentialStale {
			report.Stale = append(report.Stale, cr)
			continue
		}
		report.GradeablePositives++
		if cr.Outcome == CaseOutcomeHit {
			report.Hits++
		} else {
			report.Misses = append(report.Misses, cr)
		}
	}
	if report.GradeablePositives > 0 {
		report.Recall = float64(report.Hits) / float64(report.GradeablePositives)
	}
	return report, nil
}

// GradeCase labels a single case from already-ranked results.
func GradeCase(results []SearchResult, c TestCase, policy ScoringPolicy) CaseOutcome {
	if c.ExpectNoStrongHit {
		for _, r := range results {
			if r.Score > c.ExpectMaxScore {
				return CaseOutcomeJunkLeaked
			}
		}
		return CaseOutcomeJunkRejected
	}
	if !isPositiveCase(c) {
		return CaseOutcomeNotGradeable
	}
	rank := ExpectedRank(results, c.ExpectIDs)
	if rank == 0 || rank > effectiveGateK(c, policy) {
		return CaseOutcomeMiss
	}
	if c.ExpectMinScore > 0 {
		score := resultScoreAtRank(results, rank)
		if score < c.ExpectMinScore {
			return CaseOutcomeMiss
		}
	}
	return CaseOutcomeHit
}

// ClassifyExpectID classifies whether a positive label still points at a result
// discoverable by the provider at the probed depth.
func ClassifyExpectID(deepResults []SearchResult, c TestCase, policy ScoringPolicy) Referential {
	if len(c.ExpectIDs) == 0 || c.ExpectNoStrongHit {
		return ReferentialUnknown
	}
	rank := ExpectedRank(deepResults, c.ExpectIDs)
	if rank == 0 {
		return ReferentialStale
	}
	if rank <= effectiveGateK(c, policy) {
		return ReferentialLive
	}
	if rank <= policy.DeepK {
		return ReferentialHard
	}
	return ReferentialStale
}

func isPositiveCase(c TestCase) bool {
	return !c.ExpectNoStrongHit && len(c.ExpectIDs) > 0
}

func effectiveGateK(c TestCase, policy ScoringPolicy) int {
	if c.ExpectWithinTopK > 0 {
		return c.ExpectWithinTopK
	}
	if policy.GateK > 0 {
		return policy.GateK
	}
	return DefaultScoringPolicy.GateK
}

// ExpectedRank returns the 1-based rank of the first matching expected id, or 0
// when none is present.
func ExpectedRank(results []SearchResult, ids []string) int {
	if len(ids) == 0 {
		return 0
	}
	want := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		want[id] = struct{}{}
	}
	for i, r := range results {
		if _, ok := want[r.ID]; ok {
			return i + 1
		}
	}
	return 0
}

func resultScoreAtRank(results []SearchResult, rank int) float64 {
	if rank <= 0 || rank > len(results) {
		return 0
	}
	return results[rank-1].Score
}
