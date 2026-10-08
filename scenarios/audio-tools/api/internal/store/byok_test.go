package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestBYOKStore_RoundTrip(t *testing.T) {
	d := newTestDB(t)
	s := store.NewBYOKStore(d)
	ctx := context.Background()

	require.NoError(t, s.Upsert(ctx, store.BYOKCredential{
		ProviderID: "openai-whisper", Capability: "stt",
		Cipher: []byte("ciphertext"), Fingerprint: "sk-***abcd",
	}))
	list, err := s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "openai-whisper", list[0].ProviderID)

	// Upsert again with new fingerprint replaces in place.
	require.NoError(t, s.Upsert(ctx, store.BYOKCredential{
		ProviderID: "openai-whisper", Capability: "stt",
		Cipher: []byte("ciphertext2"), Fingerprint: "sk-***efgh",
	}))
	list, err = s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "sk-***efgh", list[0].Fingerprint)

	deleted, err := s.Delete(ctx, "openai-whisper", "stt")
	require.NoError(t, err)
	require.True(t, deleted)

	list, err = s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 0)
}

func TestBYOKStore_DeleteMissing(t *testing.T) {
	d := newTestDB(t)
	s := store.NewBYOKStore(d)
	deleted, err := s.Delete(context.Background(), "nope", "stt")
	require.NoError(t, err)
	require.False(t, deleted)
}
