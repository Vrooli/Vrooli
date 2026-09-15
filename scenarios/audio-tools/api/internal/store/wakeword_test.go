package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestWakeWordStore_UpsertGet(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	ww := store.NewWakeWordStore(d)

	require.NoError(t, ww.Upsert(ctx, store.WakeWordTemplate{
		ID: "hello", Phrase: "hello vrooli", Embedding: []byte{4, 5},
	}))
	tmpl, ok, err := ww.Get(ctx, "hello")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "hello vrooli", tmpl.Phrase)
}

func TestWakeWordStore_Delete(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	ww := store.NewWakeWordStore(d)
	require.NoError(t, ww.Upsert(ctx, store.WakeWordTemplate{ID: "k", Phrase: "p"}))
	deleted, err := ww.Delete(ctx, "k")
	require.NoError(t, err)
	require.True(t, deleted)
	_, ok, err := ww.Get(ctx, "k")
	require.NoError(t, err)
	require.False(t, ok)
}
