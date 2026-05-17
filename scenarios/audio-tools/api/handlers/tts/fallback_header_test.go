package tts_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	ttsH "audio-tools/handlers/tts"
	"audio-tools/internal/ai/ttschain"
	ttsmocks "audio-tools/internal/ai/ttschain/mocks"
	"audio-tools/internal/byok/envelope"

	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// TestTTS_Synthesize_FallbackHeader_SetWhenLowerTierServes asserts the
// x-audio-tools-fallback header is emitted when BYOK fails and Vrooli
// serves the response.
func TestTTS_Synthesize_FallbackHeader_SetWhenLowerTierServes(t *testing.T) {
	chain := ttschain.NewChain(ttschain.Options{
		EnableBYOK:   true,
		EnableVrooli: true,
		BYOK: ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{
			"openrouter": &ttsmocks.FakeBYOK{IDStr: "openrouter", Available: true, Err: errors.New("upstream timeout")},
		}),
		Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{
			Available: true,
			Result:    &ttschain.Result{Audio: []byte("PCM"), ContentType: "audio/mpeg"},
		}),
	})
	c := newServer(t, ttsH.Deps{Chain: chain})
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{Text: "hi", Voice: "voice.feminine.warm"})
	req.Header().Set(envelope.HeaderKey, "sk-1")
	req.Header().Set(envelope.HeaderProvider, "openrouter")
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	res, err := c.Synthesize(context.Background(), req)
	require.NoError(t, err)
	got := res.Header().Get("x-audio-tools-fallback")
	require.Contains(t, got, "from=byok")
	require.Contains(t, got, "to=vrooli")
	require.Contains(t, got, "reason=")
}

// TestTTS_Synthesize_FallbackHeader_AbsentWhenFirstTierServes asserts the
// header is not set when the first-priority tier serves the response.
func TestTTS_Synthesize_FallbackHeader_AbsentWhenFirstTierServes(t *testing.T) {
	chain := ttschain.NewChain(ttschain.Options{
		EnableVrooli: true,
		Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{
			Available: true,
			Result:    &ttschain.Result{Audio: []byte("PCM"), ContentType: "audio/mpeg"},
		}),
	})
	c := newServer(t, ttsH.Deps{Chain: chain})
	req := connect.NewRequest(&ttsv1.SynthesizeRequest{Text: "hi"})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	res, err := c.Synthesize(context.Background(), req)
	require.NoError(t, err)
	require.Empty(t, res.Header().Get("x-audio-tools-fallback"))
}
