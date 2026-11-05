package routing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/scheduletest"
	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"

	db "github.com/vrooli/api-core/databasetest"
	localdb "search-hub/internal/database"
	internaleval "search-hub/internal/eval"
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

func TestLatestProviderEvalAggregatesDeterministicAndLiveSuites(t *testing.T) {
	ctx := context.Background()
	database := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, database,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internaleval.Schema),
	))
	now := time.Date(2026, 9, 4, 23, 0, 0, 0, time.UTC)
	clock := scheduletest.New(now)
	store := internaleval.NewSQLiteStore(database, clock)
	for _, suite := range []*evalv1.EvalSuite{
		{SuiteId: "agent.runs.primary", ProviderId: "agent.runs", Name: "deterministic", Cases: []*evalv1.EvalCase{{CaseId: "fixture", Query: "fixture", ExpectIds: []string{"fixture-run"}}}},
		{SuiteId: "agent.runs.live-overlay", ProviderId: "agent.runs", Name: "live", Cases: []*evalv1.EvalCase{{CaseId: "live", Query: "live", ExpectIds: []string{"retained-run"}}}},
	} {
		_, err := store.UpsertSuite(ctx, suite)
		require.NoError(t, err)
	}
	require.NoError(t, store.AppendRun(ctx, directRun("fixture-fail", now.Add(-2*time.Hour), 8, 0, false)))
	// AppendRun requires the owning suite identity in addition to the run shape.
	livePass := directRun("live-pass", now.Add(-time.Hour), 2, 2, false)
	livePass.SuiteId = "agent.runs.live-overlay"
	require.NoError(t, store.AppendRun(ctx, livePass))

	writer := store.(interface {
		AppendCorpusValidation(context.Context, string, *evalv1.ValidateCorpusResponse, time.Time) error
	})
	require.NoError(t, writer.AppendCorpusValidation(ctx, "agent.runs.primary", &evalv1.ValidateCorpusResponse{Rollup: &evalv1.CorpusValidationRollup{Positives: 1, Stale: 1}}, now.Add(-2*time.Hour)))
	require.NoError(t, writer.AppendCorpusValidation(ctx, "agent.runs.live-overlay", &evalv1.ValidateCorpusResponse{Rollup: &evalv1.CorpusValidationRollup{Positives: 1, Live: 1}}, now.Add(-time.Hour)))

	reader := &evalQualityReader{store: store, now: clock.Now}
	evidence, err := reader.LatestProviderEval(ctx, "agent.runs")
	require.NoError(t, err)
	require.True(t, evidence.LiveReviewedPositive)
	require.True(t, evidence.RecentPassingRun)
	require.False(t, evidence.CorpusAllStale, "one live overlay must prevent a stale deterministic fixture corpus from withholding the provider")
	require.Equal(t, "live-pass", evidence.RunID)
}

func directRun(id string, created time.Time, graded, met int32, degraded bool) *evalv1.EvalRun {
	return &evalv1.EvalRun{
		RunId:     id,
		SuiteId:   "agent.runs.primary",
		Tier:      "provider_direct",
		CreatedAt: created.Format(time.RFC3339Nano),
		Degraded:  degraded,
		Aggregate: &evalv1.EvalAggregate{GradedCases: graded, Met: met, PassRate: float64(met) / float64(graded)},
	}
}
