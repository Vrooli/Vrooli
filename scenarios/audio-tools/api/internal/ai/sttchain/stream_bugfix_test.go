package sttchain_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/chains/tiered"
	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
)

// TestChain_Stream_CreditsExhaustionIsTerminal is the regression for a
// fallback path that swallowed ErrInsufficientCredits: a streaming BYOK
// provider that hit a billing failure used to fall through to the buffered
// unary tier, which then succeeded and hid the failure.
func TestChain_Stream_CreditsExhaustionIsTerminal(t *testing.T) {
	adapter := &sttmocks.FakeBYOK{
		IDStr: "x", Available: true, Streaming: true,
		TranscribeFn: func(context.Context, string, sttchain.Request) (*sttchain.Result, error) {
			return &sttchain.Result{Text: "unary success that must not run"}, nil
		},
		StreamFn: func(context.Context, string, sttchain.StreamStart, <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
			return nil, fmt.Errorf("billing: %w", sttchain.ErrInsufficientCredits)
		},
	}
	c := sttchain.NewChain(sttchain.Options{
		EnableBYOK: true,
		BYOK:       sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": adapter}),
	})
	chunks := make(chan sttchain.AudioChunk)
	close(chunks)

	events, err := c.Stream(context.Background(), sttchain.StreamStart{BYOKProvider: "x", BYOKKey: "k"}, chunks)
	require.Nil(t, events)
	require.ErrorIs(t, err, sttchain.ErrInsufficientCredits)
}

func TestChain_Stream_RequireStreamingStillFailsClosedOnCredits(t *testing.T) {
	adapter := &sttmocks.FakeBYOK{
		IDStr: "x", Available: true, Streaming: true,
		StreamFn: func(context.Context, string, sttchain.StreamStart, <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
			return nil, sttchain.ErrInsufficientCredits
		},
	}
	c := sttchain.NewChain(sttchain.Options{
		EnableBYOK: true,
		BYOK:       sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"x": adapter}),
	})
	chunks := make(chan sttchain.AudioChunk)
	close(chunks)

	_, err := c.Stream(context.Background(), sttchain.StreamStart{
		BYOKProvider: "x", BYOKKey: "k", RequireStreaming: true,
	}, chunks)
	require.ErrorIs(t, err, sttchain.ErrStreamingUnavailable)
}

// TestChain_FallbackReasonRedactsCredentials covers the wire-visible redaction
// path: fallback reasons are logged and forwarded in x-audio-tools-fallback.
func TestChain_FallbackReasonRedactsCredentials(t *testing.T) {
	const key = "sk-stt-secret-value"
	byok := sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{
		"x": &sttmocks.FakeBYOK{
			IDStr: "x", Available: true,
			Err: fmt.Errorf("adapter rejected api key %s", key),
		},
	})
	vrooli := sttchain.NewVrooliProvider(&sttmocks.FakeVrooliClient{
		Available: true,
		Result:    &sttchain.Result{Text: "vrooli-text"},
	})
	c := sttchain.NewChain(sttchain.Options{
		EnableBYOK: true, EnableVrooli: true, BYOK: byok, Vrooli: vrooli,
	})

	var event tiered.FallbackEvent
	ctx := tiered.WithOnFallback(context.Background(), func(ev tiered.FallbackEvent) { event = ev })
	res, err := c.Execute(ctx, sttchain.Request{BYOKProvider: "x", BYOKKey: key, LPBSToken: "lpbs-token"})
	require.NoError(t, err)
	require.Equal(t, "vrooli-text", res.Text)
	require.Equal(t, tiered.SlotBYOK, event.From)
	require.Equal(t, tiered.SlotVrooli, event.To)
	require.NotContains(t, event.Reason, key)
	require.Contains(t, event.Reason, "[redacted]")
}

func TestChain_BYOKTerminalErrorsAreNotRedactedAway(t *testing.T) {
	byok := sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{
		"x": &sttmocks.FakeBYOK{IDStr: "x", Available: true},
	})
	c := sttchain.NewChain(sttchain.Options{EnableBYOK: true, BYOK: byok})
	_, err := c.Execute(context.Background(), sttchain.Request{BYOKProvider: "missing", BYOKKey: "k"})
	require.ErrorIs(t, err, sttchain.ErrUnknownBYOKProvider)
}
