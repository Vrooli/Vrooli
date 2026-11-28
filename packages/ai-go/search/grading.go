package aisearch

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
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

// FixedDenominator is an owner-reviewed snapshot of the positive cases used
// for comparisons. The audit fields identify the independent review that
// established the labels; they are required so callers cannot silently turn a
// smaller live suite into a more favorable denominator.
type FixedDenominator struct {
	SuiteID        string     `json:"suite_id"`
	Cases          []TestCase `json:"cases"`
	Reviewer       string     `json:"reviewer"`
	ReviewEvidence string     `json:"review_evidence"`
	SHA256         string     `json:"sha256"`
}

// NewFixedDenominator snapshots the exact reviewed cases and seals their
// membership with a digest. The caller must supply the independent review
// receipt; the digest makes later changes or omissions detectable.
func NewFixedDenominator(suiteID string, cases []TestCase, reviewer, reviewEvidence string) (FixedDenominator, error) {
	if strings.TrimSpace(reviewer) == "" || strings.TrimSpace(reviewEvidence) == "" {
		return FixedDenominator{}, fmt.Errorf("fixed denominator requires reviewer and independent review evidence")
	}
	d := FixedDenominator{SuiteID: suiteID, Cases: cloneTestCases(cases), Reviewer: reviewer, ReviewEvidence: reviewEvidence}
	if err := d.validateCases(); err != nil {
		return FixedDenominator{}, err
	}
	d.SHA256 = d.digest()
	return d, nil
}

// FreezeReviewedPositiveDenominator snapshots the active suite's owner-reviewed
// positives. Candidates, negatives, and generated proposals are deliberately
// omitted; retrieval misses are never omitted because search output lacks them.
func FreezeReviewedPositiveDenominator(suite TestSuite, reviewer, reviewEvidence string) (FixedDenominator, error) {
	if suite.SuiteID == "" {
		return FixedDenominator{}, fmt.Errorf("cannot freeze denominator for a suite without suite_id")
	}
	positive := make([]TestCase, 0, len(suite.Cases))
	for _, c := range suite.Cases {
		if !isPositiveCase(c) || c.ResolvedStatus() != CaseStatusReviewed || (c.HasTag(TagGenerated) && !c.HasTag(TagHeldout)) {
			continue
		}
		positive = append(positive, c)
	}
	return NewFixedDenominator(suite.SuiteID, positive, reviewer, reviewEvidence)
}

// Validate rejects incomplete or mutable-by-omission acceptance snapshots.
// Each case must be a positive, non-candidate reviewed label with a unique ID.
func (d FixedDenominator) Validate() error {
	if d.SuiteID == "" || d.Reviewer == "" || d.ReviewEvidence == "" {
		return fmt.Errorf("fixed denominator requires suite id, reviewer, and review evidence")
	}
	if err := d.validateCases(); err != nil {
		return err
	}
	if d.SHA256 == "" || d.digest() != d.SHA256 {
		return fmt.Errorf("fixed denominator digest is missing or does not match its reviewed cases")
	}
	return nil
}

func (d FixedDenominator) validateCases() error {
	if d.SuiteID == "" {
		return fmt.Errorf("fixed denominator suite id is required")
	}
	if err := (TestSuite{SuiteID: d.SuiteID, Cases: d.Cases}).Validate(); err != nil {
		return fmt.Errorf("fixed denominator suite is invalid: %w", err)
	}
	seen := make(map[string]struct{}, len(d.Cases))
	for _, c := range d.Cases {
		if c.ID == "" || c.Query == "" || !isPositiveCase(c) || c.ResolvedStatus() != CaseStatusReviewed || (c.HasTag(TagGenerated) && !c.HasTag(TagHeldout)) {
			return fmt.Errorf("fixed denominator case %q must be a reviewed positive", c.ID)
		}
		if _, ok := seen[c.ID]; ok {
			return fmt.Errorf("fixed denominator contains duplicate case %q", c.ID)
		}
		seen[c.ID] = struct{}{}
	}
	if len(d.Cases) == 0 {
		return fmt.Errorf("fixed denominator has no positive cases")
	}
	return nil
}

func (d FixedDenominator) digest() string {
	payload, _ := json.Marshal(struct {
		SuiteID        string
		Cases          []TestCase
		Reviewer       string
		ReviewEvidence string
	}{d.SuiteID, d.Cases, d.Reviewer, d.ReviewEvidence})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func cloneTestCases(cases []TestCase) []TestCase {
	cloned := append([]TestCase(nil), cases...)
	for i := range cloned {
		cloned[i].Tags = append([]string(nil), cloned[i].Tags...)
		cloned[i].ExpectIDs = append([]string(nil), cloned[i].ExpectIDs...)
	}
	return cloned
}

// GradeSuiteOnFixedDenominator grades the frozen reviewed cases even if a
// caller's current suite has dropped one. This makes omission a retrieval miss
// (and keeps the denominator stable) rather than silently shrinking the exam.
// The live suite is used only to validate provider/suite identity.
func GradeSuiteOnFixedDenominator(ctx context.Context, searcher SuiteSearcher, suite TestSuite, denominator FixedDenominator, policy ScoringPolicy) (SuiteReport, error) {
	if err := denominator.Validate(); err != nil {
		return SuiteReport{}, err
	}
	if err := suite.Validate(); err != nil {
		return SuiteReport{}, fmt.Errorf("live suite invalid: %w", err)
	}
	if suite.SuiteID != denominator.SuiteID {
		return SuiteReport{}, fmt.Errorf("fixed denominator suite %q does not match live suite %q", denominator.SuiteID, suite.SuiteID)
	}
	frozen := TestSuite{SuiteID: denominator.SuiteID, Cases: cloneTestCases(denominator.Cases)}
	return GradeSuite(ctx, searcher, frozen, policy)
}

func (r SuiteReport) MeetsTarget() bool {
	return r.Recall+1e-9 >= r.Policy.RecallTarget
}

// GradeSuite runs every suite case at policy.DeepK and applies the fixed
// acceptance denominator: every reviewed positive label is scored in every
// arm. Retrieval absence is a miss, not evidence that the label is stale. Only
// provider-owned corpus review may remove a label from a comparison.
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

// ClassifyExpectID reports ranking evidence for a positive label. A miss at the
// queried depth is inconclusive about corpus membership: retrieval output is
// candidate-dependent and cannot prove a label stale.
func ClassifyExpectID(deepResults []SearchResult, c TestCase, policy ScoringPolicy) Referential {
	if len(c.ExpectIDs) == 0 || c.ExpectNoStrongHit {
		return ReferentialUnknown
	}
	rank := ExpectedRank(deepResults, c.ExpectIDs)
	if rank == 0 {
		return ReferentialUnknown
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
