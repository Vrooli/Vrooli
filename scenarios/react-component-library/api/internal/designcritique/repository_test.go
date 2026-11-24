package designcritique

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/storage"
)

func TestCritiqueRepositoryUsesRequestRoutedDBPool(t *testing.T) {
	ctx := context.Background()
	primaryDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "primary.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	routed, err := apidb.Open(ctx, apidb.Config{Driver: apidb.DriverSQLite, DSN: primaryDSN, MaxOpenConns: 1, MaxIdleConns: 1})
	require.NoError(t, err)
	t.Cleanup(func() { _ = routed.Close() })
	ensureSchema := func(ctx context.Context, pool *sql.DB) error {
		return apidb.EnsureSchemas(ctx, pool, apidb.SchemaProviderFunc(Schema))
	}
	require.NoError(t, ensureSchema(ctx, routed.Primary()))
	routed.SetTestPoolInitializer(ensureSchema)
	testDSN, err := storage.SQLiteDSNAt(filepath.Join(t.TempDir(), "test.db"), storage.SQLiteTuning{})
	require.NoError(t, err)
	require.NoError(t, routed.InstallTestPool(ctx, testDSN, "design-critique-routing", time.Minute))

	repo := NewSQLiteRepository(routed)
	review := reviewFixture()
	primary, err := repo.Create(ctx, "primary-attempt", review, aggregate(review))
	require.NoError(t, err)
	_, err = repo.Get(apidb.WithTestMode(ctx), primary.ID)
	require.ErrorIs(t, err, sql.ErrNoRows, "test-mode request must read the isolated routed pool")

	testRecord, err := repo.Create(apidb.WithTestMode(ctx), "test-attempt", review, aggregate(review))
	require.NoError(t, err)
	_, err = repo.Get(ctx, testRecord.ID)
	require.ErrorIs(t, err, sql.ErrNoRows, "primary request must not see the test pool critique")
	records, _, err := repo.List(apidb.WithTestMode(ctx), Target{Scenario: "demo", DesignID: "home", Revision: strings.Repeat("a", 64), RenderHash: strings.Repeat("b", 64)}, "")
	require.NoError(t, err)
	require.Equal(t, []Record{testRecord}, records, "list query must use the request-selected pool")
}
