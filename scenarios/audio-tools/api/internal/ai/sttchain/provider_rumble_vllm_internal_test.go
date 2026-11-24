package sttchain

import (
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/audioformat"
)

func TestParseRumbleTranscription(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantText   string
		wantLang   string
		languageDB map[string]string
	}{
		{name: "plain transcript", input: "  just words  ", wantText: "just words"},
		{name: "empty transcript", input: "  ", wantText: ""},
		{name: "known detected language", input: "language English<asr_text> hello there ", wantText: "hello there", wantLang: "en", languageDB: map[string]string{"en": "English"}},
		{name: "unknown detected label", input: "language klingon<asr_text> qapla ", wantText: "qapla", wantLang: "klingon", languageDB: map[string]string{}},
		{name: "language prefix without marker is preserved", input: "language English hello", wantText: "language English hello", languageDB: map[string]string{"en": "English"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotLang := parseRumbleTranscription(tt.input, tt.languageDB)
			require.Equal(t, tt.wantText, gotText)
			require.Equal(t, tt.wantLang, gotLang)
		})
	}
}

func TestRumbleMIMEAndAudioPreparation(t *testing.T) {
	tests := []struct {
		codec audioformat.Codec
		want  string
	}{
		{audioformat.CodecPCMS16LE, "audio/wav"},
		{audioformat.CodecWAV, "audio/wav"},
		{audioformat.CodecMP3, "audio/mpeg"},
		{audioformat.CodecFLAC, "audio/flac"},
		{audioformat.CodecOGG, "audio/ogg"},
		{audioformat.CodecOpus, "audio/ogg"},
		{audioformat.CodecWebM, "audio/webm"},
		{audioformat.CodecAAC, "audio/aac"},
	}
	for _, tt := range tests {
		got, err := rumbleMIME(tt.codec)
		require.NoError(t, err)
		require.Equal(t, tt.want, got)
	}
	_, err := rumbleMIME(audioformat.CodecUnknown)
	require.ErrorContains(t, err, "unsupported audio codec")

	backend := NewRumbleVLLMBackend("http://localhost:8000/v1", nil)
	payload, contentType, err := backend.audioData([]byte{1, 0, 2, 0}, "pcm_s16le")
	require.NoError(t, err)
	require.Equal(t, "audio/wav", contentType)
	require.True(t, len(payload) > 4 && string(payload[:4]) == "RIFF")
	_, _, err = backend.audioData([]byte("not audio"), "")
	require.Error(t, err, "unknown undeclared formats fail closed")
}

func TestRumbleEndpointURLValidationAndV1Normalization(t *testing.T) {
	backend := NewRumbleVLLMBackend("http://localhost:8000/v1", nil)
	require.Equal(t, "http://localhost:8000/health", backend.apiURLRoot()+"/health")
	require.Equal(t, "http://localhost:8000/v1/models", backend.apiURL("/v1/models"))
	rootBase := NewRumbleVLLMBackend("http://localhost:8000", nil)
	require.Equal(t, "http://localhost:8000/v1/chat/completions", rootBase.apiURL("/v1/chat/completions"))

	for _, baseURL := range []string{
		"", "relative/path", "file:///tmp/vllm", "http://user:pass@localhost:8000",
		"http://localhost:8000?token=hidden", "http://localhost:8000#fragment",
	} {
		t.Run(baseURL, func(t *testing.T) {
			bad := NewRumbleVLLMBackend(baseURL, nil)
			require.Empty(t, bad.apiBaseURL())
			require.Empty(t, bad.apiURLRoot())
			require.Empty(t, bad.apiURL("/v1/models"))
		})
	}
	var nilBackend *RumbleVLLMBackend
	require.Empty(t, nilBackend.apiBaseURL())
	require.Empty(t, nilBackend.apiURLRoot())
	require.Empty(t, nilBackend.apiURL("/v1/models"))
}
