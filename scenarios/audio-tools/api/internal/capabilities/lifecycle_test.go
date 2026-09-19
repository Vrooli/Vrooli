package capabilities_test

import (
	"context"
	"testing"

	"audio-tools/internal/capabilities"

	"github.com/stretchr/testify/require"
)

func TestResourceSlugForProviderID(t *testing.T) {
	cases := []struct {
		id   string
		slug string
		ok   bool
	}{
		{"whisper-stt", "whisper", true},
		{"kokoro-tts", "kokoro", true},
		{"speaker-verification", "sherpa-onnx", true},
		{"ollama", "ollama", true},
		{"openrouter", "", false},
		{"audio-tools", "", false},
		{"unknown", "", false},
	}
	for _, tc := range cases {
		got, ok := capabilities.ResourceSlugForProviderID(tc.id)
		require.Equal(t, tc.ok, ok, "id=%q", tc.id)
		require.Equal(t, tc.slug, got, "id=%q", tc.id)
	}
}

func TestSupportsPullModel(t *testing.T) {
	require.True(t, capabilities.SupportsPullModel("ollama"))
	require.False(t, capabilities.SupportsPullModel("whisper-stt"))
	require.False(t, capabilities.SupportsPullModel("openrouter"))
}

func TestCLIController_NoBinary_ReturnsUnavailable(t *testing.T) {
	// We can't reliably unset PATH in a unit test that may itself need
	// to spawn helpers; instead instantiate a controller with no
	// resolved binary and assert the sentinel.
	c := &capabilities.CLIController{}
	err := c.Start(context.Background(), "whisper")
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
	err = c.Stop(context.Background(), "whisper")
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
	err = c.Restart(context.Background(), "whisper")
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
	err = c.PullModel(context.Background(), "phi3")
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
	_, err = c.Logs(context.Background(), "whisper", false, 0)
	require.ErrorIs(t, err, capabilities.ErrControllerUnavailable)
}
