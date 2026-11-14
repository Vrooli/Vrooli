package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestVoiceOverridesStore(t *testing.T) {
	d := newTestDB(t)
	s := store.NewVoiceOverrideStore(d)
	ctx := context.Background()
	require.NoError(t, s.Set(ctx, store.VoiceOverride{CanonicalVoice: "voice.feminine.warm", TierProvider: "byok:elevenlabs", AdapterVoice: "Rachel"}))
	list, err := s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	// Empty adapter = delete
	require.NoError(t, s.Set(ctx, store.VoiceOverride{CanonicalVoice: "voice.feminine.warm", TierProvider: "byok:elevenlabs"}))
	list, err = s.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 0)
}
