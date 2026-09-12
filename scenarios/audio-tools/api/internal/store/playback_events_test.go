package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestPlaybackStore(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	ps := store.NewPlaybackStore(d)
	require.NoError(t, ps.Insert(ctx, store.PlaybackEvent{EventID: "e1", Kind: "start", Voice: "warm", ProviderTier: "local", ProviderID: "kokoro"}))
	require.NoError(t, ps.Insert(ctx, store.PlaybackEvent{EventID: "e1", Kind: "start"})) // idempotent
	require.NoError(t, ps.Insert(ctx, store.PlaybackEvent{EventID: "e2", Kind: "finish"}))
	rows, err := ps.List(ctx, 10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
}
