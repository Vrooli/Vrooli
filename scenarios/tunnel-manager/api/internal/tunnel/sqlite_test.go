package tunnel_test

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/internal/tunnel"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "tunnel-manager/internal/database"
)

func newRepo(t *testing.T) (tunnel.MetricsRepository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(tunnel.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return tunnel.NewSQLiteRepository(d, clk), clk
}

func sampleMetrics() tunnel.MetricsSample {
	return tunnel.MetricsSample{
		HAConnections: 4,
		RequestErrors: 2.5,
		ActiveStreams: 3,
		SmoothedRTTMS: 11.0,
	}
}

func TestSQLite_StoreAndLatest(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	stored, err := repo.Store(ctx, sampleMetrics())
	require.NoError(t, err)
	require.NotEmpty(t, stored.ID)
	require.False(t, stored.ScrapedAt.IsZero())

	clk.Advance(time.Minute)
	newer := sampleMetrics()
	newer.HAConnections = 8
	_, err = repo.Store(ctx, newer)
	require.NoError(t, err)

	latest, err := repo.Latest(ctx)
	require.NoError(t, err)
	require.Equal(t, 8, latest.HAConnections, "latest is the most recently scraped sample")
}

func TestSQLite_LatestEmpty(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Latest(context.Background())
	var noMetrics tunnel.ErrNoMetrics
	require.ErrorAs(t, err, &noMetrics)
}

func TestSQLite_QueryWindow(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	first := clk.Now()
	_, err := repo.Store(ctx, sampleMetrics())
	require.NoError(t, err)

	clk.Advance(time.Hour)
	mid := clk.Now()
	second := sampleMetrics()
	second.HAConnections = 6
	_, err = repo.Store(ctx, second)
	require.NoError(t, err)

	// Unbounded query returns both, ordered ascending.
	all, err := repo.Query(ctx, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, 4, all[0].HAConnections, "ordered by scraped_at ASC")
	require.Equal(t, 6, all[1].HAConnections)

	// Window starting at mid excludes the first sample.
	windowed, err := repo.Query(ctx, mid, time.Time{})
	require.NoError(t, err)
	require.Len(t, windowed, 1)
	require.Equal(t, 6, windowed[0].HAConnections)

	// Round-trip the persisted timestamp survives.
	require.True(t, all[0].ScrapedAt.Equal(first.UTC()))
}

func TestSQLite_StorePrunesExpiredMetrics(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	_, err := repo.Store(ctx, sampleMetrics())
	require.NoError(t, err)

	clk.Advance(tunnel.MetricsRetentionWindow - time.Second)
	_, err = repo.Store(ctx, sampleMetrics())
	require.NoError(t, err)
	all, err := repo.Query(ctx, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, all, 2, "cutoff is exclusive; still inside retention")

	clk.Advance(2 * time.Second)
	latest := sampleMetrics()
	latest.HAConnections = 9
	_, err = repo.Store(ctx, latest)
	require.NoError(t, err)

	all, err = repo.Query(ctx, time.Time{}, time.Time{})
	require.NoError(t, err)
	require.Len(t, all, 2, "oldest sample pruned once outside retention")
	require.Equal(t, 9, all[1].HAConnections)
}
