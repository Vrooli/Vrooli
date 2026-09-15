package main

import "testing"

func TestCalculatePriority_OrdersDistinctCandidatesWithPercentageDuplication(t *testing.T) {
	rr := &RefactorRecommender{}
	duplication := func(value float64) *float64 { return &value }
	candidates := []RefactorRecommendation{
		{FilePath: "api/low.go", StalenessScore: 10, DuplicationPct: duplication(6), HasTestFile: true, CommentRatio: 0.2},
		{FilePath: "api/medium.go", StalenessScore: 10, DuplicationPct: duplication(45), HasTestFile: true, CommentRatio: 0.2},
		{FilePath: "api/high.go", StalenessScore: 10, DuplicationPct: duplication(100), HasTestFile: true, CommentRatio: 0.2},
	}
	for i := range candidates {
		candidates[i].RefactorPriority = rr.calculatePriority(candidates[i])
	}
	rr.sortRecommendations(candidates, "priority")
	if !(candidates[0].RefactorPriority > candidates[1].RefactorPriority && candidates[1].RefactorPriority > candidates[2].RefactorPriority) {
		t.Fatalf("priorities = %#v, want strictly descending distinct order", candidates)
	}
	if candidates[0].FilePath != "api/high.go" || candidates[2].FilePath != "api/low.go" {
		t.Fatalf("ordered candidates = %#v, want high to low duplication", candidates)
	}
}
