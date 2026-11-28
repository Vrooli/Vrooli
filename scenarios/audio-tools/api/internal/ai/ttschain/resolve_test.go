package ttschain

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/tts"
)

func TestResolveLocalVoice(t *testing.T) {
	require.Equal(t, "af_bella", resolveLocalVoice("voice.feminine.warm", nil))
	require.Equal(t, "af_sarah", resolveLocalVoice("voice.feminine.neutral", nil))
	require.Equal(t, "am_adam", resolveLocalVoice("voice.masculine.warm", nil))
	require.Equal(t, "am_michael", resolveLocalVoice("voice.masculine.neutral", nil))
	require.Equal(t, "af_nicole", resolveLocalVoice("voice.neutral.default", nil))
	require.Equal(t, "custom", resolveLocalVoice("custom", nil))
	// The catalog adapter id is the persisted key the UI writes.
	require.Equal(t, "override", resolveLocalVoice("voice.feminine.warm", map[string]string{tts.AdapterLocalKokoro: "override"}))
	// The pre-catalog key still applies so stored overrides survive the rename.
	require.Equal(t, "legacy", resolveLocalVoice("voice.feminine.warm", map[string]string{tts.LegacyAdapterLocalKokoro: "legacy"}))
	// The current key wins when both exist.
	require.Equal(t, "current", resolveLocalVoice("voice.feminine.warm", map[string]string{
		tts.AdapterLocalKokoro:       "current",
		tts.LegacyAdapterLocalKokoro: "legacy",
	}))
}
