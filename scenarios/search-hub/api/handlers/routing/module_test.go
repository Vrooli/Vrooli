package routing

import (
	"testing"
	"time"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

func TestHasRecentPassingDirectRun(t *testing.T) {
	now := time.Date(2026, 8, 14, 4, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		runs []*evalv1.EvalRun
		want bool
	}{
		{
			name: "newest unavailable does not hide recent direct pass",
			runs: []*evalv1.EvalRun{
				directRun("newest-unavailable", now.Add(-time.Hour), 0, 0, true),
				directRun("recent-pass", now.Add(-2*time.Hour), 8, 8, false),
			},
			want: true,
		},
		{
			name: "federated pass is not provider evidence",
			runs: []*evalv1.EvalRun{
				{RunId: "federated-pass", Tier: "federated", CreatedAt: now.Add(-time.Hour).Format(time.RFC3339Nano), Aggregate: &evalv1.EvalAggregate{GradedCases: 8, PassRate: 1}},
			},
			want: false,
		},
		{
			name: "degraded direct run does not pass",
			runs: []*evalv1.EvalRun{directRun("degraded", now.Add(-time.Hour), 8, 1, true)},
			want: false,
		},
		{
			name: "stale direct pass does not pass",
			runs: []*evalv1.EvalRun{directRun("stale", now.Add(-(evalQualityFreshnessWindow + time.Hour)), 8, 1, false)},
			want: false,
		},
		{
			name: "below threshold direct run does not pass",
			runs: []*evalv1.EvalRun{directRun("below", now.Add(-time.Hour), 8, 3, false)},
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := hasRecentPassingDirectRun(test.runs, now); got != test.want {
				t.Fatalf("hasRecentPassingDirectRun() = %v, want %v", got, test.want)
			}
		})
	}
}

func directRun(id string, created time.Time, graded, met int32, degraded bool) *evalv1.EvalRun {
	return &evalv1.EvalRun{
		RunId:     id,
		Tier:      "provider_direct",
		CreatedAt: created.Format(time.RFC3339Nano),
		Degraded:  degraded,
		Aggregate: &evalv1.EvalAggregate{GradedCases: graded, Met: met, PassRate: float64(met) / float64(graded)},
	}
}
