package metrics_test

import (
	"context"
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
