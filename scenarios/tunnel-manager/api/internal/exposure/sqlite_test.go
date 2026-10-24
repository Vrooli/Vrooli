package exposure_test

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/internal/exposure"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "tunnel-manager/internal/database"
)

func newRepo(t *testing.T) (exposure.Repository, *scheduletest.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(exposure.Schema),
	))
	clk := scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return exposure.NewSQLiteRepository(d, clk), clk
}

func TestSQLite_CreateAndGet(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()
	created, err := repo.Create(ctx, exposure.Lease{Scenario: "web-console", RequestedBy: "tester", ExpiresAt: clk.Now().Add(time.Hour)})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, exposure.LeaseActive, created.Status)

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "web-console", got.Scenario)
	require.Equal(t, clk.Now().UTC(), got.CreatedAt)
}

func TestSQLite_ActiveForScenario(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, exposure.Lease{Scenario: "svc", ExpiresAt: clk.Now().Add(time.Hour), Status: exposure.LeaseActive})
	require.NoError(t, err)

	got, err := repo.ActiveForScenario(ctx, "svc")
	require.NoError(t, err)
	require.Equal(t, "svc", got.Scenario)

	_, err = repo.ActiveForScenario(ctx, "missing")
	var nf exposure.ErrLeaseNotFound
	require.ErrorAs(t, err, &nf)
}

func TestSQLite_UpdateAndListByStatus(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()
	l, err := repo.Create(ctx, exposure.Lease{Scenario: "svc", ExpiresAt: clk.Now().Add(time.Hour), Status: exposure.LeaseActive})
	require.NoError(t, err)

	l.Status = exposure.LeaseRevoked
	_, err = repo.Update(ctx, l)
	require.NoError(t, err)

	active, err := repo.List(ctx, exposure.LeaseActive)
	require.NoError(t, err)
	require.Empty(t, active)
	revoked, err := repo.List(ctx, exposure.LeaseRevoked)
	require.NoError(t, err)
	require.Len(t, revoked, 1)
}

func TestSQLite_UpdateNotFound(t *testing.T) {
	repo, clk := newRepo(t)
	_, err := repo.Update(context.Background(), exposure.Lease{ID: "ghost", ExpiresAt: clk.Now()})
	var nf exposure.ErrLeaseNotFound
	require.ErrorAs(t, err, &nf)
}
