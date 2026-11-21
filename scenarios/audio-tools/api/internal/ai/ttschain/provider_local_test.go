package ttschain_test

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/ttschain"
	"audio-tools/internal/logx"
	"audio-tools/internal/tts"
)

func newLocalTTSService(t *testing.T, capability string, synthesizeErr error) (*tts.Service, *tts.SynthesizeInput) {
	t.Helper()
	cfg := tts.Config{Backend: "kokoro", KokoroVoice: "af_heart", KokoroSpeed: 1}
	seen := &tts.SynthesizeInput{}
	return tts.NewService(tts.Deps{
		GetConfig: func() tts.Config { return cfg },
		SetConfig: func(tts.Config) {},
		KokoroCapability: func(context.Context) (string, string) {
			return capability, "probe"
		},
		SynthesizeAudio: func(_ context.Context, in tts.SynthesizeInput) (io.ReadCloser, string, error) {
			*seen = in
			if synthesizeErr != nil {
				return nil, "", synthesizeErr
			}
			return io.NopCloser(strings.NewReader("audio-bytes")), "audio/mpeg", nil
		},
		Logger: logx.Std{L: log.New(io.Discard, "", 0)},
	}), seen
}

func TestLocalProvider_SynthesizeUsesServiceAndMapsVoice(t *testing.T) {
	svc, seen := newLocalTTSService(t, "available", nil)
	p := ttschain.NewLocalProvider(svc)

	require.True(t, p.IsAvailable(context.Background()))
	out, err := p.Synthesize(context.Background(), ttschain.Request{
		Text: "hello", Voice: "voice.feminine.warm", ResponseFormat: "mp3", Speed: 1.25,
	})
	require.NoError(t, err)
	require.Equal(t, "audio-bytes", string(out.Audio))
	require.Equal(t, "audio/mpeg", out.ContentType)
	require.Equal(t, ttschain.TierLocal, out.Tier)
	require.Equal(t, "kokoro-local", out.ProviderID)
	require.Equal(t, "kokoro", out.ModelID)
	require.Equal(t, "af_bella", out.VoiceUsed)
	require.Equal(t, "af_bella", seen.Voice)
	require.Equal(t, 1.25, seen.Speed)
}

func TestLocalProvider_UsesConfiguredDefaultVoice(t *testing.T) {
	svc, seen := newLocalTTSService(t, "available", nil)
	p := ttschain.NewLocalProvider(svc)

	_, err := p.Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.NoError(t, err)
	require.Equal(t, "af_heart", seen.Voice)
}

func TestLocalProvider_VoiceOverrideWins(t *testing.T) {
	svc, seen := newLocalTTSService(t, "available", nil)
	p := ttschain.NewLocalProvider(svc)

	_, err := p.Synthesize(context.Background(), ttschain.Request{
		Text:           "hello",
		Voice:          "voice.feminine.warm",
		VoiceOverrides: map[string]string{"local:kokoro-local": "am_override"},
	})
	require.NoError(t, err)
	require.Equal(t, "am_override", seen.Voice)
}

func TestLocalProvider_ReportsBackendReadiness(t *testing.T) {
	down, _ := newLocalTTSService(t, "unavailable", nil)
	p := ttschain.NewLocalProvider(down)
	require.False(t, p.IsAvailable(context.Background()))
	_, err := p.Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.ErrorIs(t, err, tts.ErrUnavailable)
}

func TestLocalProvider_PropagatesSynthesisFailure(t *testing.T) {
	svc, _ := newLocalTTSService(t, "available", errors.New("kokoro child exited"))
	p := ttschain.NewLocalProvider(svc)
	_, err := p.Synthesize(context.Background(), ttschain.Request{Text: "hello"})
	require.ErrorIs(t, err, tts.ErrInternal)
}

func TestLocalProvider_NilReceiverContracts(t *testing.T) {
	var p *ttschain.LocalProvider
	require.Equal(t, ttschain.TierLocal, p.Type())
	require.False(t, p.IsAvailable(context.Background()))
	require.Equal(t, "kokoro", p.Model())
	require.False(t, p.StreamingCapability())
	_, err := p.Synthesize(context.Background(), ttschain.Request{})
	require.Error(t, err)
}
