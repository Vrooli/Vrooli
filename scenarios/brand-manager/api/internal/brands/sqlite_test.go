package brands_test

import (
	"context"
	"testing"
	"time"

	"brand-manager/internal/brands"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "brand-manager/internal/database"
)

type sqliteRepos struct {
	repo     brands.Repository
	versions brands.VersionRepository
	clock    *scheduletest.FakeClock
}

func newSchemaDB(t *testing.T) *sqliteRepos {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(brands.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return &sqliteRepos{
		repo:     brands.NewSQLiteRepository(d, clk),
		versions: brands.NewSQLiteVersionRepository(d, clk),
		clock:    clk,
	}
}

func TestSQLite_CreateGetRoundTrip(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()

	created, err := tr.repo.Create(ctx, brands.Brand{
		Name:       "Acme",
		Identity:   brands.Identity{DisplayName: "Acme Inc", Tagline: "We build"},
		Colors:     brands.Colors{Primary: "#112233", Text: "#000000"},
		Typography: brands.Typography{HeadingFont: "Inter"},
		Voice:      brands.Voice{Tone: "bold", Keywords: []string{"fast", "clean"}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, 1, created.Version)
	require.Equal(t, tr.clock.Now(), created.CreatedAt)

	got, err := tr.repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "Acme", got.Name)
	require.Equal(t, "Acme Inc", got.Identity.DisplayName)
	require.Equal(t, "#112233", got.Colors.Primary)
	require.Equal(t, "Inter", got.Typography.HeadingFont)
	require.Equal(t, []string{"fast", "clean"}, got.Voice.Keywords, "slice facet round-trips through JSON")
}

func TestSQLite_GetNotFound(t *testing.T) {
	tr := newSchemaDB(t)
	_, err := tr.repo.Get(context.Background(), "ghost")
	require.ErrorAs(t, err, &brands.ErrBrandNotFound{})
}

func TestSQLite_UpdateIncrementsVersionAndPersists(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()

	created, err := tr.repo.Create(ctx, brands.Brand{Name: "Acme"})
	require.NoError(t, err)

	created.Description = "updated"
	updated, err := tr.repo.Update(ctx, created)
	require.NoError(t, err)
	require.Equal(t, 2, updated.Version, "Update increments the stored version")

	got, err := tr.repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "updated", got.Description)
	require.Equal(t, 2, got.Version)
}

func TestSQLite_UpdateNotFound(t *testing.T) {
	tr := newSchemaDB(t)
	_, err := tr.repo.Update(context.Background(), brands.Brand{ID: "ghost", Name: "x"})
	require.ErrorAs(t, err, &brands.ErrBrandNotFound{})
}

func TestSQLite_DeleteThenGetMissing(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()

	created, err := tr.repo.Create(ctx, brands.Brand{Name: "Acme"})
	require.NoError(t, err)
	require.NoError(t, tr.repo.Delete(ctx, created.ID))

	_, err = tr.repo.Get(ctx, created.ID)
	require.ErrorAs(t, err, &brands.ErrBrandNotFound{})

	require.ErrorAs(t, tr.repo.Delete(ctx, created.ID), &brands.ErrBrandNotFound{},
		"deleting a missing row reports not-found at the repository layer")
}

func TestSQLite_ListFilterAndOrder(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()

	// Insert three brands with advancing clock so updated_at ordering is stable.
	for _, name := range []string{"Alpha", "Beta", "Alphabet"} {
		_, err := tr.repo.Create(ctx, brands.Brand{Name: name})
		require.NoError(t, err)
		tr.clock.Advance(time.Minute)
	}

	all, err := tr.repo.List(ctx, brands.ListFilter{Limit: 100})
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, "Alphabet", all[0].Name, "newest-updated first")

	filtered, err := tr.repo.List(ctx, brands.ListFilter{NameContains: "alpha", Limit: 100})
	require.NoError(t, err)
	require.Len(t, filtered, 2, "case-insensitive substring matches Alpha + Alphabet")
}

func TestSQLite_VersionsRoundTrip(t *testing.T) {
	tr := newSchemaDB(t)
	ctx := context.Background()

	created, err := tr.repo.Create(ctx, brands.Brand{Name: "Acme"})
	require.NoError(t, err)

	_, err = tr.versions.CreateVersion(ctx, brands.BrandVersion{BrandID: created.ID, Version: 1, Snapshot: `{"v":1}`})
	require.NoError(t, err)
	tr.clock.Advance(time.Minute)
	_, err = tr.versions.CreateVersion(ctx, brands.BrandVersion{BrandID: created.ID, Version: 2, Snapshot: `{"v":2}`})
	require.NoError(t, err)

	list, err := tr.versions.ListVersions(ctx, created.ID)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, 2, list[0].Version, "versions are newest-first")
	require.Equal(t, 1, list[1].Version)
}
