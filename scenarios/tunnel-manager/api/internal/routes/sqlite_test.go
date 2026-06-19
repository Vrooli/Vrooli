package routes_test

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/internal/routes"
	"tunnel-manager/internal/testutil/db"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "tunnel-manager/internal/database"
)

func newRepo(t *testing.T) (routes.Repository, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(routes.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return routes.NewSQLiteRepository(d, clk), clk
}

func sampleRoute() routes.Route {
	return routes.Route{
		Subdomain: "agent-manager", Scenario: "agent-manager",
		Domain: "itsagitime.com", LocalPort: 21100,
		Tier: routes.TierLeased, Enabled: true, HealthPath: "/health",
	}
}

func TestSQLite_CreateAndGet(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, sampleRoute())
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.False(t, created.CreatedAt.IsZero())

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "agent-manager", got.Subdomain)
	require.Equal(t, routes.TierLeased, got.Tier)
	require.True(t, got.Enabled)
	require.Equal(t, 21100, got.LocalPort)
}

func TestSQLite_GetNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	_, err := repo.Get(context.Background(), "missing")
	var nf routes.ErrRouteNotFound
	require.ErrorAs(t, err, &nf)
}

func TestSQLite_DuplicateSubdomainConflict(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, sampleRoute())
	require.NoError(t, err)

	_, err = repo.Create(ctx, sampleRoute())
	var conflict routes.ErrRouteConflict
	require.ErrorAs(t, err, &conflict)
	require.Equal(t, "agent-manager", conflict.Subdomain)
}

func TestSQLite_GetBySubdomain(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, sampleRoute())
	require.NoError(t, err)

	got, err := repo.GetBySubdomain(ctx, "agent-manager")
	require.NoError(t, err)
	require.Equal(t, "agent-manager", got.Scenario)
}

func TestSQLite_ListAndTierFilter(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, sampleRoute())
	require.NoError(t, err)
	core := sampleRoute()
	core.Subdomain = "web-console"
	core.Tier = routes.TierCore
	_, err = repo.Create(ctx, core)
	require.NoError(t, err)

	all, err := repo.List(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, "agent-manager", all[0].Subdomain, "ordered by subdomain ASC")

	onlyCore, err := repo.List(ctx, routes.TierCore)
	require.NoError(t, err)
	require.Len(t, onlyCore, 1)
	require.Equal(t, "web-console", onlyCore[0].Subdomain)
}

func TestSQLite_UpdateRefreshesTimestamp(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()
	created, err := repo.Create(ctx, sampleRoute())
	require.NoError(t, err)

	clk.Advance(time.Hour)
	created.LocalPort = 22200
	updated, err := repo.Update(ctx, created)
	require.NoError(t, err)
	require.Equal(t, 22200, updated.LocalPort)
	require.True(t, updated.UpdatedAt.After(created.CreatedAt))
}

func TestSQLite_UpdateNotFound(t *testing.T) {
	repo, _ := newRepo(t)
	r := sampleRoute()
	r.ID = "missing"
	_, err := repo.Update(context.Background(), r)
	var nf routes.ErrRouteNotFound
	require.ErrorAs(t, err, &nf)
}

func TestSQLite_Delete(t *testing.T) {
	repo, _ := newRepo(t)
	ctx := context.Background()
	created, err := repo.Create(ctx, sampleRoute())
	require.NoError(t, err)

	ok, err := repo.Delete(ctx, created.ID)
	require.NoError(t, err)
	require.True(t, ok)

	ok, err = repo.Delete(ctx, created.ID)
	require.NoError(t, err)
	require.False(t, ok, "deleting a missing id reports false, not error")
}
