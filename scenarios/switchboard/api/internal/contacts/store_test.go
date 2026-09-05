package contacts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	"switchboard/internal/channels"
	"switchboard/internal/threads"
	"switchboard/internal/trust"
)

func newStore(t *testing.T) (*Store, *threads.Store) {
	database := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), database, apidb.SchemaProviderFunc(threads.Schema), apidb.SchemaProviderFunc(Schema)))
	return NewStore(database), threads.NewStore(database)
}

// [REQ:SWBD-P0-009] [REQ:SWBD-P0-010]
func TestSeenStartsAtDescriptorDefaultAndNeverWidens(t *testing.T) {
	store, _ := newStore(t)
	ctx := context.Background()
	c, err := store.Seen(ctx, "telegram", "555", "stranger")
	require.NoError(t, err)
	require.Equal(t, "stranger", c.Tier)
	require.EqualValues(t, 1, c.MessageCount)
	// A later message on a channel with a wider default must not widen an existing contact.
	c, err = store.Seen(ctx, "telegram", "555", "owner")
	require.NoError(t, err)
	require.Equal(t, "stranger", c.Tier)
	require.EqualValues(t, 2, c.MessageCount)
	_, err = store.Seen(ctx, "telegram", "556", "vip")
	require.Error(t, err)
}

// [REQ:SWBD-P0-010] [REQ:SWBD-P1-001]
func TestCeilingIsMinimumAcrossRosterAndTierChangeReportsRooms(t *testing.T) {
	store, threadStore := newStore(t)
	ctx := context.Background()
	thread, err := threadStore.Upsert(ctx, channels.Envelope{ChannelID: "telegram", ThreadKey: "room"}, true)
	require.NoError(t, err)
	owner, _ := store.Seen(ctx, "telegram", "1", "owner")
	guest, _ := store.Seen(ctx, "telegram", "2", "stranger")
	require.NoError(t, store.Join(ctx, thread.ID, owner.ID))
	require.NoError(t, store.Join(ctx, thread.ID, guest.ID))
	require.NoError(t, store.Join(ctx, thread.ID, guest.ID))
	ceiling, err := store.Ceiling(ctx, thread.ID)
	require.NoError(t, err)
	require.Equal(t, trust.Stranger, ceiling)

	known := "known"
	updated, changes, err := store.Update(ctx, guest.ID, &known, nil)
	require.NoError(t, err)
	require.Equal(t, "known", updated.Tier)
	require.Len(t, changes, 1)
	require.Equal(t, "stranger", changes[0].PreviousCeiling)
	require.Equal(t, "known", changes[0].NewCeiling)

	bad := "vip"
	_, _, err = store.Update(ctx, guest.ID, &bad, nil)
	require.Error(t, err)
	_, err = store.Get(ctx, "missing")
	require.ErrorIs(t, err, ErrNotFound)
	list, err := store.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
}
