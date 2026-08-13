package assignments_test

import (
	"context"
	"testing"
	"time"

	"brand-manager/internal/assignments"
	"brand-manager/internal/testutil/db"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "brand-manager/internal/database"
)

func newSQLiteRepo(t *testing.T) (assignments.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(assignments.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return assignments.NewSQLiteRepository(d, clk), clk
}

func TestSQLite_UpsertGetRoundTrip(t *testing.T) {
	repo, clk := newSQLiteRepo(t)
	ctx := context.Background()

	saved, err := repo.Upsert(ctx, assignments.Assignment{
		BrandID:      "b1",
		ScenarioName: "web-console",
		BrandVersion: 3,
		Elements:     []string{"logo", "colors"},
	})
	require.NoError(t, err)
	require.NotEmpty(t, saved.ID)
	require.Equal(t, clk.Now(), saved.AppliedAt)

	got, err := repo.GetByScenario(ctx, "web-console")
	require.NoError(t, err)
	require.Equal(t, saved.ID, got.ID)
	require.Equal(t, "b1", got.BrandID)
	require.Equal(t, 3, got.BrandVersion)
	require.Equal(t, []string{"logo", "colors"}, got.Elements)
}

func TestSQLite_GetMissingReturnsTypedNotFound(t *testing.T) {
	repo, _ := newSQLiteRepo(t)
	_, err := repo.GetByScenario(context.Background(), "ghost")
	var notFound assignments.ErrAssignmentNotFound
	require.ErrorAs(t, err, &notFound)
	require.Equal(t, "ghost", notFound.Scenario)
}

func TestSQLite_UpsertReplacesKeepingStableID(t *testing.T) {
	repo, _ := newSQLiteRepo(t)
	ctx := context.Background()

	first, err := repo.Upsert(ctx, assignments.Assignment{BrandID: "b1", ScenarioName: "web-console", BrandVersion: 1})
	require.NoError(t, err)

	second, err := repo.Upsert(ctx, assignments.Assignment{BrandID: "b2", ScenarioName: "web-console", BrandVersion: 5})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID, "re-upsert preserves the row id")

	got, err := repo.GetByScenario(ctx, "web-console")
	require.NoError(t, err)
	require.Equal(t, "b2", got.BrandID)
	require.Equal(t, 5, got.BrandVersion)

	all, err := repo.ListByBrand(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 1, "scenario_name is unique — upsert never duplicates")
}

func TestSQLite_ListByBrandFilterAndOrder(t *testing.T) {
	repo, clk := newSQLiteRepo(t)
	ctx := context.Background()

	_, err := repo.Upsert(ctx, assignments.Assignment{BrandID: "b1", ScenarioName: "first"})
	require.NoError(t, err)
	clk.Advance(time.Minute)
	_, err = repo.Upsert(ctx, assignments.Assignment{BrandID: "b1", ScenarioName: "second"})
	require.NoError(t, err)
	clk.Advance(time.Minute)
	_, err = repo.Upsert(ctx, assignments.Assignment{BrandID: "b2", ScenarioName: "third"})
	require.NoError(t, err)

	b1, err := repo.ListByBrand(ctx, "b1")
	require.NoError(t, err)
	require.Len(t, b1, 2)
	require.Equal(t, "second", b1[0].ScenarioName, "newest-applied first")
	require.Equal(t, "first", b1[1].ScenarioName)

	all, err := repo.ListByBrand(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 3)
}

func TestSQLite_DeleteByScenario(t *testing.T) {
	repo, _ := newSQLiteRepo(t)
	ctx := context.Background()
	_, err := repo.Upsert(ctx, assignments.Assignment{BrandID: "b1", ScenarioName: "web-console"})
	require.NoError(t, err)

	require.NoError(t, repo.DeleteByScenario(ctx, "web-console"))

	var notFound assignments.ErrAssignmentNotFound
	require.ErrorAs(t, repo.DeleteByScenario(ctx, "web-console"), &notFound, "second delete reports not-found")
}

func TestSQLite_EmptyElementsRoundTrip(t *testing.T) {
	repo, _ := newSQLiteRepo(t)
	ctx := context.Background()
	_, err := repo.Upsert(ctx, assignments.Assignment{BrandID: "b1", ScenarioName: "x"})
	require.NoError(t, err)

	got, err := repo.GetByScenario(ctx, "x")
	require.NoError(t, err)
	require.Empty(t, got.Elements)
}
