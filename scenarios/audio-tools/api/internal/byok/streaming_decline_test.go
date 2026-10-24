package byok

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	"audio-tools/internal/ai/ttschain"
)

func TestElevenLabs_StreamingDeclines(t *testing.T) {
	a := NewElevenLabsTTS()
	require.False(t, a.StreamingCapability())
	ch, err := a.SynthesizeStreaming(context.Background(), "k", ttschain.Request{})
	require.Nil(t, ch)
	require.NoError(t, err)
}

func TestOpenAITTS_StreamingDeclines(t *testing.T) {
	a := NewOpenAITTS()
	ch, err := a.SynthesizeStreaming(context.Background(), "k", ttschain.Request{})
	require.Nil(t, ch)
	require.NoError(t, err)
}

func TestOpenAIWhisper_StreamingDeclines(t *testing.T) {
	a := NewOpenAIWhisperSTT()
	ch, err := a.TranscribeStreaming(context.Background(), "k", sttchain.StreamStart{}, nil)
	require.Nil(t, ch)
	require.NoError(t, err)
}

func TestDeepgram_StreamingCapability(t *testing.T) {
	a := NewDeepgramSTT()
	require.True(t, a.StreamingCapability())
	// Missing key path.
	ch, err := a.TranscribeStreaming(context.Background(), "", sttchain.StreamStart{}, nil)
	require.Nil(t, ch)
	require.Error(t, err)

	// URL builder paths.
	require.NotEmpty(t, deepgramStreamURL("en"))
	require.NotEmpty(t, deepgramStreamURL(""))
}

func TestVoiceMapping_All(t *testing.T) {
	cases := []string{
		"voice.feminine.warm", "voice.feminine.neutral",
		"voice.masculine.warm", "voice.masculine.neutral",
		"voice.neutral.default", "unknown",
	}
	for _, c := range cases {
		require.NotEmpty(t, canonicalToElevenVoiceID(c, nil))
	}
	// Override path
	require.Equal(t, "x", canonicalToElevenVoiceID("voice.feminine.warm", map[string]string{"byok:elevenlabs": "x"}))
}

func TestContentTypeFor_All(t *testing.T) {
	require.NotEmpty(t, contentTypeFor("wav"))
	require.NotEmpty(t, contentTypeFor("mp3"))
	require.NotEmpty(t, contentTypeFor("flac"))
	require.NotEmpty(t, contentTypeFor("ogg"))
	require.NotEmpty(t, contentTypeFor("unknown"))
}

func TestTTSContentType_All(t *testing.T) {
	require.NotEmpty(t, ttsContentType("wav"))
	require.NotEmpty(t, ttsContentType("mp3"))
	require.NotEmpty(t, ttsContentType("flac"))
	require.NotEmpty(t, ttsContentType("opus"))
	require.NotEmpty(t, ttsContentType("unknown"))
}

func TestTruncate(t *testing.T) {
	require.Equal(t, "abc", truncate("abc", 5))
	require.Contains(t, truncate("abcdef", 5), "...")
}
