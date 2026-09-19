package sweep

import (
	"testing"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

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
