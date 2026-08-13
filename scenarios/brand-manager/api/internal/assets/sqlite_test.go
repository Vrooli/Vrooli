package assets_test

import (
	"context"
	"testing"
	"time"

	"brand-manager/internal/assets"
	"brand-manager/internal/testutil/db"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "brand-manager/internal/database"
)

func newSchemaRepo(t *testing.T) (assets.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(assets.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return assets.NewSQLiteRepository(d, clk), clk
}

func TestSQLite_UpsertGetRoundTrip(t *testing.T) {
	repo, clk := newSchemaRepo(t)
	ctx := context.Background()

	stored, err := repo.Upsert(ctx, assets.Asset{
		BrandID:  "b1",
		Filename: "logo.png",
		MimeType: "image/png",
		FilePath: "/data/assets/b1/logo.png",
		Size:     128,
	})
	require.NoError(t, err)
	require.NotEmpty(t, stored.ID)
	require.Equal(t, clk.Now(), stored.CreatedAt)

	got, err := repo.Get(ctx, stored.ID)
	require.NoError(t, err)
	require.Equal(t, "logo.png", got.Filename)
	require.Equal(t, "image/png", got.MimeType)
	require.Equal(t, int64(128), got.Size)
	require.Equal(t, "/data/assets/b1/logo.png", got.FilePath)
}

func TestSQLite_UpsertReplacesByBrandAndFilename(t *testing.T) {
	repo, _ := newSchemaRepo(t)
	ctx := context.Background()

	first, err := repo.Upsert(ctx, assets.Asset{BrandID: "b1", Filename: "logo.png", MimeType: "image/png", FilePath: "/p1", Size: 1})
	require.NoError(t, err)

	second, err := repo.Upsert(ctx, assets.Asset{BrandID: "b1", Filename: "logo.png", MimeType: "image/png", FilePath: "/p1", Size: 999})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "same (brand_id, filename) preserves the id")
	require.Equal(t, first.CreatedAt, second.CreatedAt, "created_at is preserved across re-upload")
	require.Equal(t, int64(999), second.Size, "mutable columns are replaced")

	all, err := repo.ListByBrand(ctx, "b1")
	require.NoError(t, err)
	require.Len(t, all, 1, "the unique key prevents a duplicate row")
}

func TestSQLite_ListByBrandFiltersAndOrders(t *testing.T) {
	repo, clk := newSchemaRepo(t)
	ctx := context.Background()

	_, err := repo.Upsert(ctx, assets.Asset{BrandID: "b1", Filename: "a.png", MimeType: "image/png", FilePath: "/a"})
	require.NoError(t, err)
	clk.Advance(time.Minute)
	_, err = repo.Upsert(ctx, assets.Asset{BrandID: "b1", Filename: "b.png", MimeType: "image/png", FilePath: "/b"})
	require.NoError(t, err)
	clk.Advance(time.Minute)
	_, err = repo.Upsert(ctx, assets.Asset{BrandID: "b2", Filename: "c.png", MimeType: "image/png", FilePath: "/c"})
	require.NoError(t, err)

	b1, err := repo.ListByBrand(ctx, "b1")
	require.NoError(t, err)
	require.Len(t, b1, 2)
	require.Equal(t, "b.png", b1[0].Filename, "newest-uploaded first")
	require.Equal(t, "a.png", b1[1].Filename)

	all, err := repo.ListByBrand(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 3, "empty brand id returns every asset")
}

func TestSQLite_GetNotFound(t *testing.T) {
	repo, _ := newSchemaRepo(t)
	_, err := repo.Get(context.Background(), "ghost")
	require.ErrorAs(t, err, &assets.ErrAssetNotFound{})
}

func TestSQLite_DeleteRemovesAndReportsMissing(t *testing.T) {
	repo, _ := newSchemaRepo(t)
	ctx := context.Background()

	stored, err := repo.Upsert(ctx, assets.Asset{BrandID: "b1", Filename: "logo.png", MimeType: "image/png", FilePath: "/p"})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, stored.ID))
	err = repo.Delete(ctx, stored.ID)
	require.ErrorAs(t, err, &assets.ErrAssetNotFound{}, "deleting a missing row reports not-found (the service makes it idempotent)")
}
