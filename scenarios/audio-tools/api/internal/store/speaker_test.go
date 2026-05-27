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

func TestSpeakerStore_PersistsEnrollmentMetadata(t *testing.T) {
	d := newTestDB(t)
	ctx := context.Background()
	sp := store.NewSpeakerStore(d)

	require.NoError(t, sp.Upsert(ctx, store.SpeakerProfile{
		ID:                     "sp-meta",
		Name:                   "Laptop",
		EnrollmentAudioSeconds: 3.42,
		SampleRate:             16000,
		EmbeddingDim:           192,
		ModelName:              "speechbrain/spkrec-ecapa-voxceleb",
	}))

	// Get round-trips every metadata field.
	got, ok, err := sp.Get(ctx, "sp-meta")
	require.NoError(t, err)
	require.True(t, ok)
	require.InDelta(t, 3.42, got.EnrollmentAudioSeconds, 1e-9)
	require.Equal(t, 16000, got.SampleRate)
	require.Equal(t, 192, got.EmbeddingDim)
	require.Equal(t, "speechbrain/spkrec-ecapa-voxceleb", got.ModelName)

	// List surfaces it too.
	rows, err := sp.List(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.InDelta(t, 3.42, rows[0].EnrollmentAudioSeconds, 1e-9)
	require.Equal(t, 16000, rows[0].SampleRate)

	// Re-enroll (Upsert on conflict) overwrites the metadata, not just name.
	require.NoError(t, sp.Upsert(ctx, store.SpeakerProfile{
		ID:                     "sp-meta",
		Name:                   "Laptop",
		EnrollmentAudioSeconds: 5.0,
		SampleRate:             16000,
		EmbeddingDim:           192,
		ModelName:              "speechbrain/spkrec-ecapa-voxceleb",
	}))
	got, _, err = sp.Get(ctx, "sp-meta")
	require.NoError(t, err)
	require.InDelta(t, 5.0, got.EnrollmentAudioSeconds, 1e-9)
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
