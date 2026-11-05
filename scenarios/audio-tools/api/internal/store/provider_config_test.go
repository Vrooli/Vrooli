package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/store"
)

func TestProviderConfigStore_RoundTrip(t *testing.T) {
	d := newTestDB(t)
	s := store.NewProviderConfigStore(d, store.ProviderConfig{
		BYOKEnabled: true, LocalEnabled: true,
		WhisperURL: "http://w", KokoroURL: "http://k", OllamaURL: "http://o",
		AvailTTLBYOKSeconds: 300, AvailTTLVrooliSecs: 30,
	})

	got, err := s.Get(context.Background())
	require.NoError(t, err)
	require.True(t, got.BYOKEnabled)
	require.Equal(t, "http://w", got.WhisperURL)

	want := false
	url := "http://w2"
	got2, err := s.Update(context.Background(), store.ProviderConfigPatch{BYOKEnabled: &want, WhisperURL: &url})
	require.NoError(t, err)
	require.False(t, got2.BYOKEnabled)
	require.Equal(t, "http://w2", got2.WhisperURL)
}
