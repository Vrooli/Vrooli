package snapshot

import (
	"context"
	"testing"
	"time"

	"network-manager/internal/testutil/db"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
)

func TestSQLiteRepositoryCreateListGet(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(d)
	createdAt := time.Date(2026, 6, 23, 16, 30, 0, 0, time.UTC)

	stored, err := repo.Create(context.Background(), Snapshot{
		ID:        "snap-1",
		Status:    "baseline",
		Profile:   "home",
		Summary:   "summary",
		CreatedAt: createdAt,
		Metrics: []Metric{
			{Name: "dns_lookup_latency", Value: "12", Unit: "ms", Status: "healthy"},
			{Name: "throughput_availability", Value: "unavailable", Unit: "status", Status: "unavailable"},
		},
		Findings: []string{"baseline captured"},
	})
	require.NoError(t, err)
	require.Equal(t, "snap-1", stored.ID)

	count, err := repo.Count(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, count)

	got, err := repo.Get(context.Background(), "snap-1")
	require.NoError(t, err)
	require.Equal(t, "baseline", got.Status)
	require.Equal(t, createdAt, got.CreatedAt)
	require.Len(t, got.Metrics, 2)
	require.Equal(t, []string{"baseline captured"}, got.Findings)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 1)
}
