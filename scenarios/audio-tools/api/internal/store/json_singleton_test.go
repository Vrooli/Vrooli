package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestJSONSingletonStores(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	stts := store.NewSTTStreamConfigStore(d)
	speaker := store.NewSTTSpeakerConfigStore(d)
	require.NoError(t, speaker.Set(ctx, `{"profile":"voice-1"}`))
	got, ok, err := speaker.Get(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `{"profile":"voice-1"}`, got)

	_, ok, err = stts.Get(ctx)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, stts.Set(ctx, `{"sample_rate":16000}`))
	got, ok, err = stts.Get(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `{"sample_rate":16000}`, got)
	require.NoError(t, stts.Set(ctx, `{"sample_rate":24000}`))
	got, ok, err = stts.Get(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `{"sample_rate":24000}`, got)

	ttss := store.NewTTSConfigStore(d)
	require.NoError(t, ttss.Set(ctx, `{"voice":"warm"}`, `{"level":"moderate"}`))
	cfg, summ, ok, err := ttss.Get(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `{"voice":"warm"}`, cfg)
	require.Equal(t, `{"level":"moderate"}`, summ)
}
