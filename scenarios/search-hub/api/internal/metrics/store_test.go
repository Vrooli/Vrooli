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
		QueryHash:   "h1",
		RoutedTypes: []string{"command"},
		ProviderResults: map[string]metrics.ProviderResult{
			"cli-health.commands": {LatencyMs: 100, Degraded: true, DegradeReason: "timeout"},
		},
		ResultCount: 0,
		Degraded:    true,
		LatencyMs:   100,
	}))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash:   "h2",
		RoutedTypes: []string{"command", "record"},
		ProviderResults: map[string]metrics.ProviderResult{
			"cli-health.commands":   {HitCount: 3, LatencyMs: 200},
			"swarm-manager.records": {HitCount: 2, LatencyMs: 500},
		},
		ResultCount:        5,
		Reranked:           true,
		AutoRoutedExternal: true,
		LatencyMs:          200,
	}))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash:   "h3",
		RoutedTypes: []string{"command"},
		ProviderResults: map[string]metrics.ProviderResult{
			"cli-health.commands": {HitCount: 1, LatencyMs: 300},
		},
		ResultCount: 1,
		Escalated:   true,
		LatencyMs:   300,
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
	require.Equal(t, int64(200), usage["cli-health.commands"].LatencyP50Ms)
	require.Equal(t, int64(300), usage["cli-health.commands"].LatencyP95Ms)
	require.Equal(t, int64(1), usage["cli-health.commands"].DegradedCount)
	require.InDelta(t, 1.0/3.0, usage["cli-health.commands"].DegradationRate, 1e-9)
	require.Equal(t, []metrics.ProviderDegradationReason{{Reason: "timeout", Count: 1}}, usage["cli-health.commands"].DegradationReasons)
	require.Equal(t, int64(1), usage["swarm-manager.records"].TimesRouted)
	require.Equal(t, int64(2), usage["swarm-manager.records"].TotalHits)
	require.Equal(t, int64(500), usage["swarm-manager.records"].LatencyP95Ms)
}

func TestInsightsWindowFiltersOldRows(t *testing.T) {
	store, clk := newStore(t)
	ctx := context.Background()

	// An old query (10 days ago) and a recent one (now).
	clk.SetNow(time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash: "old",
		ProviderResults: map[string]metrics.ProviderResult{
			"cli-health.commands": {HitCount: 1, LatencyMs: 50},
		},
		ResultCount: 1,
		LatencyMs:   50,
	}))
	clk.SetNow(time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash: "new",
		ProviderResults: map[string]metrics.ProviderResult{
			"ui-health.surfaces": {HitCount: 2, LatencyMs: 150},
		},
		ResultCount: 2,
		LatencyMs:   150,
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

func TestInsightsRangeFiltersExactWindow(t *testing.T) {
	store, clk := newStore(t)
	ctx := context.Background()

	clk.SetNow(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash: "before",
		ProviderResults: map[string]metrics.ProviderResult{
			"slow.provider": {LatencyMs: 100, Degraded: true, DegradeReason: "timeout"},
		},
		ResultCount: 0,
		Degraded:    true,
		LatencyMs:   100,
	}))
	clk.SetNow(time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash: "inside",
		ProviderResults: map[string]metrics.ProviderResult{
			"slow.provider": {LatencyMs: 400, Degraded: true, DegradeReason: "timeout"},
		},
		ResultCount: 1,
		Degraded:    true,
		LatencyMs:   400,
	}))
	clk.SetNow(time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC))
	require.NoError(t, store.Record(ctx, metrics.Sample{
		QueryHash: "after",
		ProviderResults: map[string]metrics.ProviderResult{
			"slow.provider": {LatencyMs: 900},
		},
		ResultCount: 1,
		LatencyMs:   900,
	}))

	got, err := store.InsightsRange(ctx,
		time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), got.TotalQueries)
	require.Equal(t, int64(400), got.LatencyP95Ms)
	require.Len(t, got.ProviderUsage, 1)
	require.Equal(t, int64(1), got.ProviderUsage[0].DegradedCount)
	require.Equal(t, []metrics.ProviderDegradationReason{{Reason: "timeout", Count: 1}}, got.ProviderUsage[0].DegradationReasons)
}

func TestMigrateAddsProviderTelemetryColumns(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(func() string {
			return `
CREATE TABLE IF NOT EXISTS query_telemetry (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  query_hash TEXT NOT NULL,
  routed_types TEXT NOT NULL DEFAULT '',
  result_count INTEGER NOT NULL DEFAULT 0,
  zero_result INTEGER NOT NULL DEFAULT 0,
  degraded INTEGER NOT NULL DEFAULT 0,
  reranked INTEGER NOT NULL DEFAULT 0,
  auto_routed_external INTEGER NOT NULL DEFAULT 0,
  escalated INTEGER NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS query_telemetry_provider (
  query_id INTEGER NOT NULL REFERENCES query_telemetry(id) ON DELETE CASCADE,
  provider_id TEXT NOT NULL,
  hit_count INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (query_id, provider_id)
);`
		}),
	))

	require.NoError(t, metrics.Migrate(ctx, d))
	require.NoError(t, metrics.Migrate(ctx, d), "migration is idempotent")
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(metrics.Schema),
	))
}
