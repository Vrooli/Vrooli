package metrics_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	"search-hub/internal/aigateway"
	localdb "search-hub/internal/database"
	"search-hub/internal/metrics"
)

type failingEmbeddingTelemetryDB struct{}

func (failingEmbeddingTelemetryDB) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, errors.New("write failure")
}
func (failingEmbeddingTelemetryDB) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("query failure")
}
func (failingEmbeddingTelemetryDB) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

func TestEmbeddingTelemetryStoresBoundedMetadataOnlyWindow(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(metrics.Schema),
	))
	clock := scheduletest.New(time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC))
	store := metrics.NewEmbeddingTelemetryStore(db, clock, nil)
	defer store.Close(context.Background())
	store.Observe(aigateway.EmbeddingObservation{
		Kind: "query", Role: "embedding.default", Model: "nomic-embed-text:latest",
		Duration: 12*time.Millisecond + 345*time.Microsecond, VectorCount: 1,
	})
	require.NoError(t, store.Flush(ctx))
	rows, err := store.RecentEmbeddings(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "query", rows[0].Kind)
	require.Equal(t, "embedding.default", rows[0].Role)
	require.Equal(t, "nomic-embed-text:latest", rows[0].Model)
	require.EqualValues(t, 12345, rows[0].DurationUS)
	require.Equal(t, 1, rows[0].VectorCount)
	require.False(t, rows[0].Failed)

	_, err = db.ExecContext(ctx, `WITH RECURSIVE seed(n) AS (SELECT 1 UNION ALL SELECT n+1 FROM seed WHERE n < ?) INSERT INTO embedding_telemetry (operation_kind, role, model, duration_us, vector_count, failed, created_at) SELECT 'document', 'embedding.default', 'nomic-embed-text:latest', n, 1, 0, ? FROM seed`, metrics.EmbeddingTelemetryLimit+1, clock.Now().UTC().Format(time.RFC3339Nano))
	require.NoError(t, err)
	store.Observe(aigateway.EmbeddingObservation{Kind: "private text must not persist", Role: "embedding.default", Model: "model with spaces", Duration: time.Second, Failed: true})
	require.NoError(t, store.Flush(ctx))
	rows, err = store.RecentEmbeddings(ctx, metrics.EmbeddingTelemetryLimit+50)
	require.NoError(t, err)
	require.Len(t, rows, metrics.EmbeddingTelemetryLimit)
	require.Equal(t, "other", rows[0].Kind)
	require.Equal(t, "model_with_spaces", rows[0].Model)
	require.True(t, rows[0].Failed)
	var leaked int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM embedding_telemetry WHERE operation_kind LIKE '%private%' OR model LIKE '% %'`).Scan(&leaked))
	require.Equal(t, 0, leaked)
}

func TestEmbeddingTelemetryReadWindowIsBoundedAndReportsRetention(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(metrics.Schema),
	))
	store := metrics.NewEmbeddingTelemetryStore(db, scheduletest.New(time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC)), nil)
	defer store.Close(context.Background())

	base := time.Date(2026, 11, 20, 19, 0, 0, 0, time.UTC)
	for i, stamp := range []time.Time{base, base.Add(time.Second), base.Add(2 * time.Second)} {
		_, err := db.ExecContext(ctx, `INSERT INTO embedding_telemetry (operation_kind, role, model, duration_us, vector_count, failed, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			"query", "embedding.default", "test-model", 100+i, 1, 0, stamp.Format(time.RFC3339Nano))
		require.NoError(t, err)
	}

	page, err := store.ReadWindow(ctx, base, base.Add(3*time.Second), 2)
	require.NoError(t, err)
	require.Len(t, page.Rows, 2)
	require.True(t, page.Truncated)
	require.True(t, page.HasRetainedWindow)
	require.Equal(t, base, page.RetainedFrom)
	require.Equal(t, base.Add(2*time.Second), page.RetainedTo)
	require.Equal(t, base, page.Rows[0].CreatedAt)
	require.Equal(t, base.Add(time.Second), page.Rows[1].CreatedAt)

	// The lower bound is inclusive and upper bound exclusive.
	page, err = store.ReadWindow(ctx, base.Add(time.Second), base.Add(2*time.Second), 10)
	require.NoError(t, err)
	require.Len(t, page.Rows, 1)
	require.Equal(t, base.Add(time.Second), page.Rows[0].CreatedAt)
	_, err = store.ReadWindow(ctx, base.Add(time.Second), base, 10)
	require.Error(t, err)
}

func TestEmbeddingTelemetryReadWindowEmptyStoreHasNoRetentionBounds(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(metrics.Schema),
	))
	store := metrics.NewEmbeddingTelemetryStore(db, nil, nil)
	defer store.Close(context.Background())
	page, err := store.ReadWindow(ctx, time.Date(2026, 11, 20, 19, 0, 0, 0, time.UTC), time.Date(2026, 11, 20, 20, 0, 0, 0, time.UTC), 0)
	require.NoError(t, err)
	require.Empty(t, page.Rows)
	require.False(t, page.Truncated)
	require.False(t, page.HasRetainedWindow)
}

func TestEmbeddingTelemetryReadWindowComparesSubsecondInstants(t *testing.T) {
	ctx := context.Background()
	db := databasetest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(localdb.SystemSchema),
		database.SchemaProviderFunc(metrics.Schema),
	))
	store := metrics.NewEmbeddingTelemetryStore(db, nil, nil)
	defer store.Close(context.Background())
	for _, stamp := range []string{"2026-11-20T19:00:00.500Z", "2026-11-20T19:00:00Z"} {
		_, err := db.ExecContext(ctx, `INSERT INTO embedding_telemetry (operation_kind, role, model, duration_us, vector_count, failed, created_at) VALUES ('query', 'embedding.default', 'test-model', 1, 1, 0, ?)`, stamp)
		require.NoError(t, err)
	}
	from := time.Date(2026, 11, 20, 19, 0, 0, 250_000_000, time.UTC)
	to := time.Date(2026, 11, 20, 19, 0, 0, 750_000_000, time.UTC)
	page, err := store.ReadWindow(ctx, from, to, 10)
	require.NoError(t, err)
	require.Len(t, page.Rows, 1)
	require.Equal(t, time.Date(2026, 11, 20, 19, 0, 0, 500_000_000, time.UTC), page.Rows[0].CreatedAt)
	require.Equal(t, time.Date(2026, 11, 20, 19, 0, 0, 0, time.UTC), page.RetainedFrom)
	require.Equal(t, time.Date(2026, 11, 20, 19, 0, 0, 500_000_000, time.UTC), page.RetainedTo)
}

func TestProfileEmbeddingTelemetryGroupsByKindRoleModelAndMarksCompleteness(t *testing.T) {
	rows := []metrics.EmbeddingTelemetryRow{
		{Kind: "query", Role: "embedding.default", Model: "model-b", DurationUS: 300},
		{Kind: "query", Role: "embedding.default", Model: "model-b", DurationUS: 100, Failed: true},
		{Kind: "query", Role: "embedding.default", Model: "model-b", DurationUS: 200},
		{Kind: "document", Role: "embedding.default", Model: "model-b", DurationUS: 900},
	}
	profiles := metrics.ProfileEmbeddingTelemetry(rows, false)
	require.Len(t, profiles, 2)
	require.Equal(t, "document", profiles[0].Kind)
	require.EqualValues(t, 1, profiles[0].SampleCount)
	require.EqualValues(t, 900, profiles[0].DurationP50US)
	require.EqualValues(t, 900, profiles[0].DurationP95US)
	require.False(t, profiles[0].Complete)
	require.Equal(t, "query", profiles[1].Kind)
	require.EqualValues(t, 3, profiles[1].SampleCount)
	require.EqualValues(t, 1, profiles[1].FailedCount)
	require.EqualValues(t, 200, profiles[1].DurationP50US)
	require.EqualValues(t, 300, profiles[1].DurationP95US)
	require.False(t, profiles[1].Complete)
}

func TestEmbeddingTelemetryReadCountersExposeWriteFailure(t *testing.T) {
	store := metrics.NewEmbeddingTelemetryStore(failingEmbeddingTelemetryDB{}, nil, nil)
	defer store.Close(context.Background())
	store.Observe(aigateway.EmbeddingObservation{Kind: "query", Role: "embedding.default", Model: "test-model", Duration: time.Millisecond})
	require.NoError(t, store.Flush(context.Background()))
	dropped, failures := store.Counters()
	require.EqualValues(t, 0, dropped)
	require.EqualValues(t, 1, failures)
}
