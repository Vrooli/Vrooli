package settings_test

import (
	"context"
	"testing"
	"time"

	"flow-verifier/internal/settings"
	"flow-verifier/internal/testutil/db"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "flow-verifier/internal/database"
)

func newRepo(t *testing.T) (settings.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(settings.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 12, 12, 0, 0, 0, time.UTC))
	return settings.NewSQLiteRepository(d, clk), clk
}

func TestRepository_GetReturnsDefaultsWhenEmpty(t *testing.T) {
	repo, _ := newRepo(t)
	got, err := repo.Get(context.Background(), settings.PrincipalLocal)
	require.NoError(t, err)
	require.Equal(t, settings.PrincipalLocal, got.PrincipalID)
	require.Equal(t, settings.ThemeSystem, got.Theme)
	require.Equal(t, settings.FontScaleMd, got.FontScale)
	require.False(t, got.ReducedMotion)
	require.False(t, got.RTL)
	require.Equal(t, ".", got.DefaultRoot)
	require.Equal(t, settings.DensityComfortable, got.Density)
	require.Equal(t, 320, got.SidebarWidth)
	require.Equal(t, "all", got.InventoryFilters.Language)
	require.Equal(t, "flowId", got.InventoryFilters.Sort.Key)
	require.Equal(t, "asc", got.InventoryFilters.Sort.Dir)
}

func TestRepository_UpsertRoundTrip(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	in := settings.DefaultSettings()
	in.Theme = settings.ThemeDark
	in.FontScale = settings.FontScaleLg
	in.ReducedMotion = true
	in.SidebarWidth = 400
	in.InventoryFilters.Status = []string{"failed", "error"}
	in.InventoryFilters.Sort = settings.InventorySortOrder{Key: "lastVerified", Dir: "desc"}

	got, err := repo.Upsert(ctx, in)
	require.NoError(t, err)
	require.False(t, got.UpdatedAt.IsZero(), "Upsert must stamp UpdatedAt")

	roundTripped, err := repo.Get(ctx, settings.PrincipalLocal)
	require.NoError(t, err)
	require.Equal(t, settings.ThemeDark, roundTripped.Theme)
	require.Equal(t, settings.FontScaleLg, roundTripped.FontScale)
	require.True(t, roundTripped.ReducedMotion)
	require.Equal(t, 400, roundTripped.SidebarWidth)
	require.Equal(t, []string{"failed", "error"}, roundTripped.InventoryFilters.Status)
	require.Equal(t, "lastVerified", roundTripped.InventoryFilters.Sort.Key)
	require.Equal(t, "desc", roundTripped.InventoryFilters.Sort.Dir)
}

// TestRepository_UpsertIdempotent asserts the ON CONFLICT path: two
// Upserts in sequence leave a single row, with the second write's values.
func TestRepository_UpsertIdempotent(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	first := settings.DefaultSettings()
	first.Theme = settings.ThemeDark
	_, err := repo.Upsert(ctx, first)
	require.NoError(t, err)

	second := settings.DefaultSettings()
	second.Theme = settings.ThemeLight
	_, err = repo.Upsert(ctx, second)
	require.NoError(t, err)

	got, err := repo.Get(ctx, settings.PrincipalLocal)
	require.NoError(t, err)
	require.Equal(t, settings.ThemeLight, got.Theme,
		"second Upsert must overwrite the first; ON CONFLICT path")
}

func TestService_PartialMergePreservesUnsetFields(t *testing.T) {
	repo, _ := newRepo(t)
	svc := settings.NewService(repo)
	ctx := context.Background()

	// Seed: dark + lg + width 400.
	full := settings.DefaultSettings()
	full.Theme = settings.ThemeDark
	full.FontScale = settings.FontScaleLg
	full.SidebarWidth = 400
	_, err := svc.Upsert(ctx, settings.Patch{
		Theme:        &full.Theme,
		FontScale:    &full.FontScale,
		SidebarWidth: &full.SidebarWidth,
	})
	require.NoError(t, err)

	// Patch only the theme.
	light := settings.ThemeLight
	_, err = svc.Upsert(ctx, settings.Patch{Theme: &light})
	require.NoError(t, err)

	got, err := svc.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, settings.ThemeLight, got.Theme, "theme patched")
	require.Equal(t, settings.FontScaleLg, got.FontScale, "fontScale preserved")
	require.Equal(t, 400, got.SidebarWidth, "sidebarWidth preserved")
}

func TestService_ValidationRejectsBadEnum(t *testing.T) {
	repo, _ := newRepo(t)
	svc := settings.NewService(repo)

	bogusTheme := settings.Theme("hot-pink")
	_, err := svc.Upsert(context.Background(), settings.Patch{Theme: &bogusTheme})
	require.Error(t, err)
	var ve settings.ValidationError
	require.ErrorAs(t, err, &ve)
	require.Equal(t, "theme", ve.Field)
}

func TestService_GetReturnsDefaultsForEmptyStore(t *testing.T) {
	repo, _ := newRepo(t)
	svc := settings.NewService(repo)
	got, err := svc.Get(context.Background())
	require.NoError(t, err)
	require.Equal(t, settings.DefaultSettings().Theme, got.Theme)
	require.Equal(t, settings.DefaultSettings().Density, got.Density)
}
