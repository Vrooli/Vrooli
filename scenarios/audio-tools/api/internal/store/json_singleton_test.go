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
	_, ok, err := stts.Get(ctx)
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, stts.Set(ctx, `{"sample_rate":16000}`))
	v, ok, err := stts.Get(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `{"sample_rate":16000}`, v)

	// Overwrite is idempotent — Set on existing key replaces value.
	require.NoError(t, stts.Set(ctx, `{"sample_rate":24000}`))
	v, _, _ = stts.Get(ctx)
	require.Equal(t, `{"sample_rate":24000}`, v)

	ttss := store.NewTTSConfigStore(d)
	require.NoError(t, ttss.Set(ctx, `{"voice":"warm"}`, `{"level":"moderate"}`))
	cfg, summ, ok, err := ttss.Get(ctx)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, `{"voice":"warm"}`, cfg)
	require.Equal(t, `{"level":"moderate"}`, summ)
}
