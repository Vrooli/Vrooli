package stt

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/byok/envelope"

	sttv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/stt"
)

// TestTranscribe_FallbackHeader_SetWhenLowerTierServes asserts that
// x-audio-tools-fallback is emitted on Transcribe when the first-priority
// tier (BYOK) fails and a lower tier (Vrooli) succeeds.
func TestTranscribe_FallbackHeader_SetWhenLowerTierServes(t *testing.T) {
	chain := sttchain.NewChain(sttchain.Options{
		EnableBYOK:   true,
		EnableVrooli: true,
		BYOK: sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{
			"openrouter": &sttmocks.FakeBYOK{IDStr: "openrouter", Available: true, Err: errors.New("upstream timeout")},
		}),
		Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{
			Available: true,
			Result:    &sttchain.Result{Text: "hello", Tier: sttchain.TierVrooli, ProviderID: "lpbs"},
		}),
	})
	c := newSTTRuntimeClient(t, Deps{Chain: chain})
	req := connect.NewRequest(&sttv1.TranscribeRequest{Audio: []byte("x")})
	req.Header().Set(envelope.HeaderKey, "sk-1")
	req.Header().Set(envelope.HeaderProvider, "openrouter")
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	res, err := c.Transcribe(context.Background(), req)
	require.NoError(t, err)
	got := res.Header().Get("x-audio-tools-fallback")
	require.Contains(t, got, "from=byok")
	require.Contains(t, got, "to=vrooli")
	require.Contains(t, got, "reason=")
}

// TestTranscribe_FallbackHeader_AbsentWhenFirstTierServes asserts the
// header is NOT set when the first-priority tier handles the request.
func TestTranscribe_FallbackHeader_AbsentWhenFirstTierServes(t *testing.T) {
	chain := sttchain.NewChain(sttchain.Options{
		EnableVrooli: true,
		Vrooli: sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{
			Available: true,
			Result:    &sttchain.Result{Text: "hello", Tier: sttchain.TierVrooli},
		}),
	})
	c := newSTTRuntimeClient(t, Deps{Chain: chain})
	req := connect.NewRequest(&sttv1.TranscribeRequest{Audio: []byte("x")})
	req.Header().Set(envelope.HeaderLPBSToken, "tok")
	res, err := c.Transcribe(context.Background(), req)
	require.NoError(t, err)
	require.Empty(t, res.Header().Get("x-audio-tools-fallback"))
}
