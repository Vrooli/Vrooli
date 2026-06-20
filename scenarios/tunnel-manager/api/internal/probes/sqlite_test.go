package probes_test

import (
	"context"
	"testing"
	"time"

	"tunnel-manager/internal/probes"
	"tunnel-manager/internal/testutil/db"
	"tunnel-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "tunnel-manager/internal/database"
)

func newRepo(t *testing.T) (probes.Repository, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(probes.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC))
	return probes.NewSQLiteRepository(d, clk), clk
}

func TestSQLite_PersistAndList(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	first, err := repo.Persist(ctx, probes.ProbeResult{
		Subdomain: "agent-manager", Kind: probes.ProbeKindInternal,
		Status: probes.ProbeStatusUp, LatencyMS: 12, StatusCode: 200,
	})
	require.NoError(t, err)
	require.NotEmpty(t, first.ID)
	require.False(t, first.CreatedAt.IsZero())

	clk.Advance(time.Minute)
	_, err = repo.Persist(ctx, probes.ProbeResult{
		Subdomain: "agent-manager", Kind: probes.ProbeKindExternal,
		Status: probes.ProbeStatusDown, StatusCode: 502, ErrorMsg: "bad gateway",
	})
	require.NoError(t, err)

	all, err := repo.List(ctx, "", 0)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, probes.ProbeStatusDown, all[0].Status, "newest-first ordering")

	require.Equal(t, "bad gateway", all[0].ErrorMsg)
	require.Equal(t, 502, all[0].StatusCode)
}

func TestSQLite_ListFilterAndLimit(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		clk.Advance(time.Second)
		_, err := repo.Persist(ctx, probes.ProbeResult{Subdomain: "a", Kind: probes.ProbeKindInternal, Status: probes.ProbeStatusUp})
		require.NoError(t, err)
	}
	clk.Advance(time.Second)
	_, err := repo.Persist(ctx, probes.ProbeResult{Subdomain: "b", Kind: probes.ProbeKindInternal, Status: probes.ProbeStatusUp})
	require.NoError(t, err)

	onlyA, err := repo.List(ctx, "a", 0)
	require.NoError(t, err)
	require.Len(t, onlyA, 3)
	for _, r := range onlyA {
		require.Equal(t, "a", r.Subdomain)
	}

	limited, err := repo.List(ctx, "a", 2)
	require.NoError(t, err)
	require.Len(t, limited, 2, "limit caps rows")
}

func TestSQLite_LatestPerRoute(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	// Two routes; the older internal probe for "a" must be superseded.
	_, err := repo.Persist(ctx, probes.ProbeResult{Subdomain: "a", Kind: probes.ProbeKindInternal, Status: probes.ProbeStatusDown})
	require.NoError(t, err)
	clk.Advance(time.Minute)
	_, err = repo.Persist(ctx, probes.ProbeResult{Subdomain: "a", Kind: probes.ProbeKindInternal, Status: probes.ProbeStatusUp})
	require.NoError(t, err)
	_, err = repo.Persist(ctx, probes.ProbeResult{Subdomain: "a", Kind: probes.ProbeKindExternal, Status: probes.ProbeStatusDown})
	require.NoError(t, err)
	_, err = repo.Persist(ctx, probes.ProbeResult{Subdomain: "b", Kind: probes.ProbeKindInternal, Status: probes.ProbeStatusUp})
	require.NoError(t, err)

	pairs, err := repo.LatestPerRoute(ctx)
	require.NoError(t, err)
	require.Len(t, pairs, 2)

	bySub := map[string]probes.LatestPair{}
	for _, p := range pairs {
		bySub[p.Subdomain] = p
	}
	require.NotNil(t, bySub["a"].Internal)
	require.Equal(t, probes.ProbeStatusUp, bySub["a"].Internal.Status, "latest internal wins")
	require.NotNil(t, bySub["a"].External)
	require.Equal(t, probes.ProbeStatusDown, bySub["a"].External.Status)
	require.NotNil(t, bySub["b"].Internal)
	require.Nil(t, bySub["b"].External, "no external probe for b yet")
}

func TestSQLite_PersistPrunesExpiredHistory(t *testing.T) {
	repo, clk := newRepo(t)
	ctx := context.Background()

	_, err := repo.Persist(ctx, probes.ProbeResult{Subdomain: "a", Kind: probes.ProbeKindInternal, Status: probes.ProbeStatusDown})
	require.NoError(t, err)

	clk.Advance(probes.HistoryRetentionWindow - time.Second)
	_, err = repo.Persist(ctx, probes.ProbeResult{Subdomain: "a", Kind: probes.ProbeKindExternal, Status: probes.ProbeStatusUp})
	require.NoError(t, err)
	all, err := repo.List(ctx, "", 0)
	require.NoError(t, err)
	require.Len(t, all, 2)

	clk.Advance(2 * time.Second)
	_, err = repo.Persist(ctx, probes.ProbeResult{Subdomain: "b", Kind: probes.ProbeKindInternal, Status: probes.ProbeStatusUp})
	require.NoError(t, err)

	all, err = repo.List(ctx, "", 0)
	require.NoError(t, err)
	require.Len(t, all, 2, "oldest probe was outside retention and pruned")
	for _, got := range all {
		require.NotEqual(t, probes.ProbeStatusDown, got.Status)
	}
}
