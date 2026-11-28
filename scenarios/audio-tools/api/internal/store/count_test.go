package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestOwnerStoresCountCreatedInHalfOpenUTCWindow(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	from := time.Date(2026, 4, 3, 12, 0, 0, 0, time.FixedZone("UTC-4", -4*60*60))
	to := from.Add(time.Hour)
	inside := from.UTC().Add(30 * time.Minute)
	boundary := to.UTC()

	speakers := store.NewSpeakerStore(db)
	for _, p := range []store.SpeakerProfile{
		{ID: "before", CreatedAt: from.Add(-time.Nanosecond)},
		{ID: "inside", CreatedAt: inside},
		{ID: "end", CreatedAt: boundary},
	} {
		require.NoError(t, speakers.Upsert(ctx, p))
	}
	n, err := speakers.CountCreatedBetween(ctx, from, to)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)

	wakewords := store.NewWakeWordStore(db)
	for _, w := range []store.WakeWordTemplate{
		{ID: "before", CreatedAt: from.Add(-time.Second)},
		{ID: "inside", CreatedAt: inside},
		{ID: "end", CreatedAt: boundary},
	} {
		require.NoError(t, wakewords.Upsert(ctx, w))
	}
	n, err = wakewords.CountCreatedBetween(ctx, from, to)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
}
