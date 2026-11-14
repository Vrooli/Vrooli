package aisearch

import (
	"context"
	"errors"
	"testing"
)

type fakeSuiteSearcher map[string][]SearchResult

func (f fakeSuiteSearcher) Search(_ context.Context, q SearchQuery, _ ...SearchOption) (SearchResponse, error) {
	if q.Query == "error" {
		return SearchResponse{}, errors.New("boom")
	}
	results := append([]SearchResult(nil), f[q.Query]...)
	if q.Limit > 0 && len(results) > q.Limit {
		results = results[:q.Limit]
	}
	return SearchResponse{Results: results, Query: q.Query, Total: len(results)}, nil
}

func TestGradeCasePositiveAndNegativeOutcomes(t *testing.T) {
	policy := DefaultScoringPolicy
	results := []SearchResult{
		{ID: "wrong", Score: 0.9},
		{ID: "want", Score: 0.8},
	}
	if got := GradeCase(results, TestCase{ExpectIDs: []string{"want"}}, policy); got != CaseOutcomeHit {
		t.Fatalf("positive hit = %s", got)
	}
	if got := GradeCase(results, TestCase{ExpectIDs: []string{"missing"}}, policy); got != CaseOutcomeMiss {
		t.Fatalf("positive miss = %s", got)
	}
	if got := GradeCase(results, TestCase{ExpectNoStrongHit: true, ExpectMaxScore: 0.95}, policy); got != CaseOutcomeJunkRejected {
		t.Fatalf("negative rejected = %s", got)
	}
	if got := GradeCase(results, TestCase{ExpectNoStrongHit: true, ExpectMaxScore: 0.5}, policy); got != CaseOutcomeJunkLeaked {
		t.Fatalf("negative leaked = %s", got)
	}
}

func TestClassifyExpectID(t *testing.T) {
	policy := ScoringPolicy{GateK: 2, DeepK: 4}
	results := []SearchResult{
		{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"},
	}
	cases := []struct {
		name string
		ids  []string
		want Referential
	}{
		{"live", []string{"b"}, ReferentialLive},
		{"hard", []string{"d"}, ReferentialHard},
		{"absent is not proof of stale membership", []string{"z"}, ReferentialUnknown},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyExpectID(results, TestCase{ExpectIDs: tc.ids}, policy)
			if got != tc.want {
				t.Fatalf("ClassifyExpectID = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestGradeSuiteUsesFixedReviewedPositiveDenominator(t *testing.T) {
	suite := TestSuite{Cases: []TestCase{
		{ID: "hit", Query: "hit", ExpectIDs: []string{"a"}},
		{ID: "hard", Query: "hard", ExpectIDs: []string{"a"}, ExpectWithinTopK: 1},
		{ID: "stale", Query: "stale", ExpectIDs: []string{"missing"}},
		{ID: "candidate", Query: "candidate", Status: CaseStatusCandidate, ExpectIDs: []string{"a"}},
		{ID: "junk", Query: "junk", ExpectNoStrongHit: true, ExpectMaxScore: 0.5},
	}}
	searcher := fakeSuiteSearcher{
		"hit":       {{ID: "a", Score: 0.9}},
		"hard":      {{ID: "x", Score: 0.9}, {ID: "a", Score: 0.8}},
		"stale":     {{ID: "x", Score: 0.9}},
		"candidate": {{ID: "a", Score: 0.9}},
		"junk":      {{ID: "x", Score: 0.2}},
	}
	report, err := GradeSuite(context.Background(), searcher, suite, ScoringPolicy{GateK: 1, DeepK: 5, RecallTarget: 0.8})
	if err != nil {
		t.Fatalf("GradeSuite: %v", err)
	}
	if report.GradeablePositives != 3 {
		t.Fatalf("gradeable positives = %d, want 3; missing expected IDs remain misses", report.GradeablePositives)
	}
	if report.Hits != 1 || report.Recall != 1.0/3.0 {
		t.Fatalf("recall = %.3f hits=%d, want 1/3 and 1", report.Recall, report.Hits)
	}
	if len(report.Stale) != 0 {
		t.Fatalf("candidate retrieval results cannot establish stale corpus labels: %+v", report.Stale)
	}
	if len(report.ExcludedCandidate) != 1 || report.ExcludedCandidate[0].ID != "candidate" {
		t.Fatalf("candidate cases = %+v", report.ExcludedCandidate)
	}
	if len(report.Misses) != 2 || report.Misses[0].CaseID != "hard" || report.Misses[1].CaseID != "stale" {
		t.Fatalf("misses = %+v", report.Misses)
	}
}

func TestWeakArmCannotImproveByOmittingKnownPositive(t *testing.T) {
	suite := TestSuite{Cases: []TestCase{
		{ID: "ordinary", Query: "ordinary", ExpectIDs: []string{"ordinary-id"}},
		{ID: "difficult", Query: "difficult", ExpectIDs: []string{"difficult-id"}},
	}}
	strong, err := GradeSuite(context.Background(), fakeSuiteSearcher{
		"ordinary":  {{ID: "ordinary-id", Score: 0.9}},
		"difficult": {{ID: "difficult-id", Score: 0.8}},
	}, suite, ScoringPolicy{GateK: 1, DeepK: 5})
	if err != nil {
		t.Fatal(err)
	}
	weak, err := GradeSuite(context.Background(), fakeSuiteSearcher{
		"ordinary":  {{ID: "ordinary-id", Score: 0.9}},
		"difficult": {{ID: "unrelated", Score: 0.99}},
	}, suite, ScoringPolicy{GateK: 1, DeepK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if strong.GradeablePositives != 2 || weak.GradeablePositives != 2 {
		t.Fatalf("arm denominators differ: strong=%d weak=%d", strong.GradeablePositives, weak.GradeablePositives)
	}
	if weak.Recall >= strong.Recall {
		t.Fatalf("weak arm improved by omitting a known positive: strong=%.3f weak=%.3f", strong.Recall, weak.Recall)
	}
}

func TestGradeSuitePassesCaseScope(t *testing.T) {
	suite := TestSuite{Cases: []TestCase{
		{ID: "scoped", Query: "architecture", Scope: "scenario:cli-health", ExpectIDs: []string{"scoped-hit"}},
	}}
	report, err := GradeSuite(context.Background(), scopedSuiteSearcher{}, suite, ScoringPolicy{GateK: 1, DeepK: 5, RecallTarget: 1})
	if err != nil {
		t.Fatalf("GradeSuite: %v", err)
	}
	if report.Recall != 1 {
		t.Fatalf("recall = %.3f, want 1", report.Recall)
	}
}

func TestScoringDefaultsAndValidation(t *testing.T) {
	p := ScoringConfig{RecallAt: 7}.WithDefaults()
	if p.GateK != 7 || p.DeepK != DefaultScoringPolicy.DeepK || p.MRRAt != DefaultScoringPolicy.MRRAt {
		t.Fatalf("defaults not filled: %+v", p)
	}
	p = ScoringConfig{RecallAt: 10, DeepK: 2}.WithDefaults()
	if p.DeepK != 10 {
		t.Fatalf("deep_k should floor to gate k: %+v", p)
	}
	if err := (ScoringConfig{RecallTarget: 1.1}).Validate(); err == nil {
		t.Fatal("expected invalid recall target")
	}
}

type scopedSuiteSearcher struct{}

func (scopedSuiteSearcher) Search(_ context.Context, q SearchQuery, _ ...SearchOption) (SearchResponse, error) {
	if q.Scope.Kind == ScopeScenario && q.Scope.Value == "cli-health" {
		return SearchResponse{Results: []SearchResult{{ID: "scoped-hit", Score: 1}}, Query: q.Query, Total: 1}, nil
	}
	return SearchResponse{Query: q.Query}, nil
}
