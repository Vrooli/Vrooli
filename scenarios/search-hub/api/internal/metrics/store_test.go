package metrics_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "search-hub/internal/database"
	"search-hub/internal/metrics"
	"search-hub/internal/testutil/db"
	"search-hub/internal/testutil/mocks"
)

// newStore returns a SQLite-backed metrics Store with the production schema
// applied — the canonical compose pattern (db.NewSQLite + apidb.EnsureSchemas
// over system + metrics), so tests exercise the same shape main.go ships.
func newStore(t *testing.T) (metrics.Store, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(metrics.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC))
	return metrics.NewSQLiteStore(d, clk), clk
}

func TestInsightsEmpty(t *testing.T) {
	store, _ := newStore(t)
	got, err := store.Insights(context.Background(), 0)
	require.NoError(t, err)
	require.Equal(t, int64(0), got.TotalQueries)
	require.Equal(t, int64(0), got.LatencyP50Ms)
	require.Empty(t, got.ProviderUsage)
}

func TestRecordAndAggregate(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()

	// Three queries: one zero-result+degraded, two with hits (one reranked).
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash:    "h1",
		RoutedTypes:  []string{"command"},
		ProviderHits: map[string]int{"cli-health.commands": 0},
		ResultCount:  0,
		Degraded:     true,
		LatencyMs:    100,
	}))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash:          "h2",
		RoutedTypes:        []string{"command", "record"},
		ProviderHits:       map[string]int{"cli-health.commands": 3, "swarm-manager.records": 2},
		ResultCount:        5,
		Reranked:           true,
		AutoRoutedExternal: true,
		LatencyMs:          200,
	}))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash:    "h3",
		RoutedTypes:  []string{"command"},
		ProviderHits: map[string]int{"cli-health.commands": 1},
		ResultCount:  1,
		Escalated:    true,
		LatencyMs:    300,
	}))

	got, err := store.Insights(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, int64(3), got.TotalQueries)
	require.Equal(t, int64(1), got.ZeroResultQueries)
	require.Equal(t, int64(1), got.DegradedQueries)
	require.Equal(t, int64(1), got.RerankedQueries)
	require.Equal(t, int64(1), got.AutoRoutedExternalQueries)
	require.Equal(t, int64(1), got.EscalatedQueries)

	// Nearest-rank over [100,200,300]: p50 → idx ceil(1.5)-1=1 → 200; p95 → idx 2 → 300.
	require.Equal(t, int64(200), got.LatencyP50Ms)
	require.Equal(t, int64(300), got.LatencyP95Ms)

	// Provider usage: cli-health routed 3× with 4 hits; swarm-manager routed 1× with 2 hits.
	usage := map[string]metrics.ProviderUsage{}
	for _, u := range got.ProviderUsage {
		usage[u.ProviderID] = u
	}
	require.Equal(t, int64(3), usage["cli-health.commands"].TimesRouted)
	require.Equal(t, int64(4), usage["cli-health.commands"].TotalHits)
	require.Equal(t, int64(1), usage["swarm-manager.records"].TimesRouted)
	require.Equal(t, int64(2), usage["swarm-manager.records"].TotalHits)
}

func TestInsightsWindowFiltersOldRows(t *testing.T) {
	store, clk := newStore(t)
	ctx := context.Background()

	// An old query (10 days ago) and a recent one (now).
	clk.SetNow(time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash:    "old",
		ProviderHits: map[string]int{"cli-health.commands": 1},
		ResultCount:  1,
		LatencyMs:    50,
	}))
	clk.SetNow(time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash:    "new",
		ProviderHits: map[string]int{"ui-health.surfaces": 2},
		ResultCount:  2,
		LatencyMs:    150,
	}))

	// 7-day window excludes the 10-day-old row.
	got, err := store.Insights(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.TotalQueries)
	require.Len(t, got.ProviderUsage, 1)
	require.Equal(t, "ui-health.surfaces", got.ProviderUsage[0].ProviderID)

	// All-time sees both.
	all, err := store.Insights(ctx, 0)
	require.NoError(t, err)
	require.Equal(t, int64(2), all.TotalQueries)
	require.Len(t, all.ProviderUsage, 2)
}
