package config_test

import (
	"context"
	"testing"
	"time"

	db "github.com/vrooli/api-core/databasetest"
	"tunnel-manager/internal/config"
	internalexposure "tunnel-manager/internal/exposure"
	internaltunnel "tunnel-manager/internal/tunnel"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "tunnel-manager/internal/database"
)

func newLedger(t *testing.T) config.OwnershipLedger {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(internaltunnel.Schema),
		apidb.SchemaProviderFunc(internalexposure.Schema),
	))
	return config.NewSQLiteLedger(d, scheduletest.New(time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)))
}

func TestLedger_GetMissingReturnsNotFound(t *testing.T) {
	l := newLedger(t)
	_, found, err := l.Get(context.Background(), "nope.itsagitime.com")
	require.NoError(t, err)
	require.False(t, found)
}

func TestLedger_PutGetListRoundTrip(t *testing.T) {
	l := newLedger(t)
	ctx := context.Background()

	require.NoError(t, l.Put(ctx, config.LedgerEntry{
		Hostname: "a.itsagitime.com", Owner: config.OwnerManaged, Scenario: "agent-manager",
	}))
	require.NoError(t, l.Put(ctx, config.LedgerEntry{
		Hostname: "b.itsagitime.com", Owner: config.OwnerIgnored, Note: "operator dashboard",
	}))

	got, found, err := l.Get(ctx, "a.itsagitime.com")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, config.OwnerManaged, got.Owner)
	require.Equal(t, "agent-manager", got.Scenario)
	require.False(t, got.AdoptedAt.IsZero(), "Put stamps adopted_at when zero")

	all, err := l.List(ctx)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, "a.itsagitime.com", all[0].Hostname, "ordered by hostname")
	require.Equal(t, "b.itsagitime.com", all[1].Hostname)
}

// TestLedger_PutIsIdempotentUpsert: writing the same hostname twice keeps one
// row and applies the latest values ("twice == once").
func TestLedger_PutIsIdempotentUpsert(t *testing.T) {
	l := newLedger(t)
	ctx := context.Background()

	require.NoError(t, l.Put(ctx, config.LedgerEntry{Hostname: "h.itsagitime.com", Owner: config.OwnerExternal}))
	require.NoError(t, l.Put(ctx, config.LedgerEntry{Hostname: "h.itsagitime.com", Owner: config.OwnerIgnored, Note: "changed mind"}))

	all, err := l.List(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, config.OwnerIgnored, all[0].Owner)
	require.Equal(t, "changed mind", all[0].Note)
}

func TestLedger_Delete(t *testing.T) {
	l := newLedger(t)
	ctx := context.Background()
	require.NoError(t, l.Put(ctx, config.LedgerEntry{Hostname: "h.itsagitime.com", Owner: config.OwnerManaged}))

	removed, err := l.Delete(ctx, "h.itsagitime.com")
	require.NoError(t, err)
	require.True(t, removed)

	removed, err = l.Delete(ctx, "h.itsagitime.com")
	require.NoError(t, err)
	require.False(t, removed, "deleting a missing row reports false")
}
