package ttschain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveLocalVoice(t *testing.T) {
	require.Equal(t, "af_bella", resolveLocalVoice("voice.feminine.warm", nil))
	require.Equal(t, "af_sarah", resolveLocalVoice("voice.feminine.neutral", nil))
	require.Equal(t, "am_adam", resolveLocalVoice("voice.masculine.warm", nil))
	require.Equal(t, "am_michael", resolveLocalVoice("voice.masculine.neutral", nil))
	require.Equal(t, "af_nicole", resolveLocalVoice("voice.neutral.default", nil))
	require.Equal(t, "custom", resolveLocalVoice("custom", nil))
	require.Equal(t, "override", resolveLocalVoice("voice.feminine.warm", map[string]string{"local:kokoro-local": "override"}))
}
