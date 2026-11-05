package aisearch

import (
	"testing"
	"time"

	"prompt-manager/internal/store"
)

func ptrInt(v int) *int { return &v }

func TestDiscoveryMetricsAggregates(t *testing.T) {
	sink := &fakeCallStore{readResp: []store.DiscoveryCall{
		{
			Type: "skill", Complexity: "moderate", Threshold: 0.5,
			BudgetStatus: "under", ReturnedCount: 3,
			Results: []store.DiscoveryCallResult{
				{ID: "a", Score: 0.80, Chars: 4000, Type: "skill"},
				{ID: "b", Score: 0.60, Chars: 3000, Type: "skill"},
				{ID: "c", Score: 0.52, Chars: 2000, Type: "skill"}, // min 0.52 → within 0.05 of 0.5 → near-threshold
			},
		},
		{
			Type: "skill", Complexity: "moderate", Threshold: 0.5,
			BudgetStatus: "over", ReturnedCount: 2, ClippedBelowThreshold: ptrInt(4),
			Results: []store.DiscoveryCallResult{
				{ID: "big", Score: 0.90, Chars: 50000, Type: "skill"},
				{ID: "a", Score: 0.70, Chars: 4000, Type: "skill"}, // min 0.70 → NOT near threshold
			},
		},
		{
			Type: "skill", Complexity: "minor", Threshold: 0.5,
			BudgetStatus: "under", ReturnedCount: 1, ClippedBelowThreshold: ptrInt(0),
			Results: []store.DiscoveryCallResult{
				{ID: "a", Score: 0.95, Chars: 4000, Type: "skill"},
			},
		},
	}}
	s := serviceWithCallStore(sink)

	report, err := s.DiscoveryMetrics(7*24*time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}

	if report.CallCount != 3 {
		t.Fatalf("expected 3 calls, got %d", report.CallCount)
	}
	if report.ReturnedCount.Count != 3 || report.ReturnedCount.Median != 2 {
		t.Fatalf("unexpected returned-count distribution: %#v", report.ReturnedCount)
	}
	// 1 of 3 budgeted calls was over budget.
	if report.BudgetedCallCount != 3 || report.OverBudgetRate < 0.33 || report.OverBudgetRate > 0.34 {
		t.Fatalf("unexpected over-budget rate: %#v", report)
	}
	// Exactly 1 of 3 calls-with-results had its lowest score on the floor.
	if report.NearThresholdRate < 0.33 || report.NearThresholdRate > 0.34 {
		t.Fatalf("unexpected near-threshold rate: %v", report.NearThresholdRate)
	}
	// 2 calls were probed; 1 of them clipped >=1.
	if report.ProbedCallCount != 2 || report.ThresholdClipRate != 0.5 {
		t.Fatalf("unexpected clip stats: probed=%d rate=%v", report.ProbedCallCount, report.ThresholdClipRate)
	}
	// Budget hog: "big" at 50000 chars should rank first.
	if len(report.BudgetHogs) == 0 || report.BudgetHogs[0].ID != "big" || report.BudgetHogs[0].MaxChars != 50000 {
		t.Fatalf("expected 'big' as top budget hog, got %#v", report.BudgetHogs)
	}
	if report.BudgetHogs[0].OverBudgetSightings != 1 {
		t.Fatalf("expected big to be seen in 1 over-budget call, got %d", report.BudgetHogs[0].OverBudgetSightings)
	}
	// Per-complexity breakdown present for both tiers.
	if report.PerComplexity["moderate"].CallCount != 2 || report.PerComplexity["minor"].CallCount != 1 {
		t.Fatalf("unexpected per-complexity: %#v", report.PerComplexity)
	}
}

func TestDiscoveryMetricsTypeFilter(t *testing.T) {
	sink := &fakeCallStore{readResp: []store.DiscoveryCall{
		{Type: "skill", ReturnedCount: 2},
		{Type: "action", ReturnedCount: 5},
	}}
	s := serviceWithCallStore(sink)

	report, err := s.DiscoveryMetrics(7*24*time.Hour, "action")
	if err != nil {
		t.Fatal(err)
	}
	if report.CallCount != 1 || report.ReturnedCount.Max != 5 {
		t.Fatalf("expected only the action call, got %#v", report)
	}
}

func TestDiscoveryMetricsEmptyWhenNoStore(t *testing.T) {
	s := &Service{threshold: 0.5}
	report, err := s.DiscoveryMetrics(7*24*time.Hour, "")
	if err != nil {
		t.Fatal(err)
	}
	if report.CallCount != 0 || len(report.BudgetHogs) != 0 {
		t.Fatalf("expected empty report with no store, got %#v", report)
	}
}

func TestPercentileInterpolates(t *testing.T) {
	sorted := []float64{1, 2, 3, 4, 5}
	if got := percentile(sorted, 0.5); got != 3 {
		t.Fatalf("median = %v, want 3", got)
	}
	if got := percentile(sorted, 0); got != 1 {
		t.Fatalf("p0 = %v, want 1", got)
	}
	if got := percentile(sorted, 1); got != 5 {
		t.Fatalf("p100 = %v, want 5", got)
	}
}
