package sweep

import (
	"reflect"
	"strings"
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestCompletePositiveRunKeepsFailureCasesInFixedDenominator(t *testing.T) {
	suite := &evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "one", ExpectIds: []string{"1"}},
		{CaseId: "two", ExpectIds: []string{"2"}},
		{CaseId: "three", ExpectIds: []string{"3"}},
	}}

	failed := &evalv1.EvalRun{Results: []*evalv1.CaseResult{
		{CaseId: "one", Outcome: "met"},
		{CaseId: "two", Outcome: "below_expectation"},
	}}
	if reason := completePositiveRun(suite, failed); reason != "missing fixed-denominator case three" {
		t.Fatalf("failed/missing case must remain in denominator, got %q", reason)
	}

	unavailable := &evalv1.EvalRun{Results: []*evalv1.CaseResult{
		{CaseId: "one", Outcome: "met"},
		{CaseId: "two", Outcome: "below_expectation"},
		{CaseId: "three", Outcome: "unavailable"},
	}}
	if reason := completePositiveRun(suite, unavailable); !strings.Contains(reason, "non-gradeable outcome") {
		t.Fatalf("unavailable case must not shrink denominator, got %q", reason)
	}

	degraded := &evalv1.EvalRun{Degraded: true, DegradedReason: "provider transport failure", Results: []*evalv1.CaseResult{
		{CaseId: "one", Outcome: "met"},
		{CaseId: "two", Outcome: "below_expectation"},
		{CaseId: "three", Outcome: "below_expectation"},
	}}
	if reason := completePositiveRun(suite, degraded); !strings.Contains(reason, "run is degraded") {
		t.Fatalf("degraded run cannot qualify even with synthetic result rows, got %q", reason)
	}
}

func TestGeneratedCasesRequireOwnerReviewBeforeDenominatorAndScoring(t *testing.T) {
	suite := &evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "ordinary", ExpectIds: []string{"1"}},
		{CaseId: "generated-unreviewed", Tags: []string{"generated"}, ExpectIds: []string{"2"}},
		{CaseId: "generated-reviewed", Tags: []string{"generated", "heldout"}, ExpectIds: []string{"3"}},
	}}
	run := &evalv1.EvalRun{Results: []*evalv1.CaseResult{
		{CaseId: "ordinary", Outcome: "below_expectation"},
		{CaseId: "generated-unreviewed", Outcome: "met"},
		{CaseId: "generated-reviewed", Outcome: "below_expectation"},
	}}
	denominator := positiveCaseIDs(suite)
	if !reflect.DeepEqual(denominator, []string{"generated-reviewed", "ordinary"}) {
		t.Fatalf("fixed denominator contains unreviewed generated case or omits reviewed case: %v", denominator)
	}
	if reason := completePositiveRun(suite, run); reason != "" {
		t.Fatalf("unreviewed generated result must not change completeness: %s", reason)
	}
	recall := recallByCase(suite, run)
	if _, included := recall["generated-unreviewed"]; included {
		t.Fatalf("unreviewed generated gain entered promotion score: %v", recall)
	}
	if recall["generated-reviewed"] != 0 {
		t.Fatalf("owner-reviewed generated miss must remain in denominator: %v", recall)
	}
}

func TestIsPositiveCase(t *testing.T) {
	tests := []struct {
		name string
		c    *evalv1.EvalCase
		want bool
	}{
		{"expect ids", &evalv1.EvalCase{ExpectIds: []string{"x"}}, true},
		{"top-k only", &evalv1.EvalCase{ExpectWithinTopK: 3}, true},
		{"gibberish flag", &evalv1.EvalCase{ExpectNoStrongHit: true, ExpectIds: []string{"x"}}, false},
		{"gibberish tag", &evalv1.EvalCase{Tags: []string{"gibberish"}, ExpectIds: []string{"x"}}, false},
		{"no expectation", &evalv1.EvalCase{Query: "q"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPositiveCase(tc.c); got != tc.want {
				t.Fatalf("isPositiveCase = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRecallByCaseAndMeans(t *testing.T) {
	suite := &evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "a", ExpectIds: []string{"1"}},
		{CaseId: "b", ExpectIds: []string{"2"}},
		{CaseId: "c", ExpectIds: []string{"3"}},
		{CaseId: "g", Tags: []string{"gibberish"}, ExpectNoStrongHit: true}, // not positive
		{CaseId: "n", Query: "no expectation"},                              // ignored
	}}
	run := &evalv1.EvalRun{Results: []*evalv1.CaseResult{
		{CaseId: "a", Outcome: "met"},
		{CaseId: "b", Outcome: "below_expectation"},
		{CaseId: "c", Outcome: "met"},
		{CaseId: "g", Outcome: "met"}, // gibberish met — excluded from recall
		{CaseId: "n", Outcome: "n/a"},
	}}

	recall := recallByCase(suite, run)
	if len(recall) != 3 {
		t.Fatalf("recall map size = %d, want 3 (positive cases only)", len(recall))
	}
	if recall["a"] != 1 || recall["b"] != 0 || recall["c"] != 1 {
		t.Fatalf("unexpected recall map: %v", recall)
	}
	if _, ok := recall["g"]; ok {
		t.Fatalf("gibberish case must not appear in recall map")
	}

	pos := positiveCaseIDs(suite)
	if got := meanOver(recall, pos); got != 2.0/3.0 {
		t.Fatalf("mean recall = %v, want 2/3", got)
	}
	if got := meanOver(recall, nil); got != 0 {
		t.Fatalf("mean over empty = %v, want 0", got)
	}
	vec := vectorOver(recall, []string{"a", "b", "c"})
	if len(vec) != 3 || vec[0] != 1 || vec[1] != 0 || vec[2] != 1 {
		t.Fatalf("vectorOver = %v", vec)
	}
}

func TestGeneratedCaseIDs(t *testing.T) {
	suite := &evalv1.EvalSuite{Cases: []*evalv1.EvalCase{
		{CaseId: "a", ExpectIds: []string{"1"}},
		{CaseId: "b", Tags: []string{"generated"}, ExpectIds: []string{"2"}},
	}}
	gen := generatedCaseIDs(suite)
	if !gen["b"] || gen["a"] {
		t.Fatalf("generated set = %v, want only b", gen)
	}
}
