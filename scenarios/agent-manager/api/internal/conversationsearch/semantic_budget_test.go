package conversationsearch

import (
	"testing"
	"time"
)

func TestSemanticRebuildBudgetScalesWithPlannedCorpus(t *testing.T) {
	cases := []struct {
		planned uint64
		want    time.Duration
	}{
		{planned: 0, want: semanticRebuildMinimumBudget},
		{planned: 100, want: semanticRebuildMinimumBudget},
		// The measured operator corpus: 73,940 semantic-eligible documents at
		// 250 ms each must get hours, not the 10 seconds that starved it.
		{planned: 73_940, want: 73_940 * semanticRebuildPerDocument},
		{planned: 10_000_000, want: semanticRebuildMaximumBudget},
	}
	for _, tc := range cases {
		if got := semanticRebuildBudget(tc.planned); got != tc.want {
			t.Fatalf("semanticRebuildBudget(%d) = %s, want %s", tc.planned, got, tc.want)
		}
	}
	if semanticRebuildBudget(73_940) < 4*time.Hour {
		t.Fatalf("a 74k-document rebuild must have a multi-hour budget, got %s", semanticRebuildBudget(73_940))
	}
}
