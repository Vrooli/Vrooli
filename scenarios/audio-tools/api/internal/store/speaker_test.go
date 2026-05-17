package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestSpeakerStore_UpsertGetClearBinding(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	sp := store.NewSpeakerStore(d)

	require.NoError(t, sp.Upsert(ctx, store.SpeakerProfile{
		ID: "sp1", Name: "alice", Embedding: []byte{1, 2, 3}, BoundUserIdentity: "user-1",
	}))
	got, ok, err := sp.Get(ctx, "sp1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "user-1", got.BoundUserIdentity)
	require.Equal(t, "alice", got.Name)

	require.NoError(t, sp.ClearBinding(ctx, "sp1"))
	got, _, _ = sp.Get(ctx, "sp1")
	require.Empty(t, got.BoundUserIdentity)
}

func TestSpeakerStore_GetMissing(t *testing.T) {
	d := newTestDB(t)
	sp := store.NewSpeakerStore(d)
	_, ok, err := sp.Get(context.Background(), "missing")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestSpeakerStore_List(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	sp := store.NewSpeakerStore(d)
	require.NoError(t, sp.Upsert(ctx, store.SpeakerProfile{ID: "a", Name: "A", Embedding: []byte{1}}))
	require.NoError(t, sp.Upsert(ctx, store.SpeakerProfile{ID: "b", Name: "B", Embedding: []byte{2}}))
	rows, err := sp.List(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}
