package ttschain_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/ttschain"
	ttsmocks "audio-tools/internal/ai/ttschain/mocks"
)

// TestChain_Stream_CreditsExhaustionIsTerminal is the regression for the TTS
// streaming path, which ignored every non-BYOK streaming error and let the
// buffered fallback succeed instead of surfacing the billing failure.
func TestChain_Stream_CreditsExhaustionIsTerminal(t *testing.T) {
	adapter := &ttsmocks.FakeBYOK{
		IDStr: "x", Available: true, Streaming: true,
		SynthesizeFn: func(context.Context, string, ttschain.Request) (*ttschain.Result, error) {
			return &ttschain.Result{Audio: []byte("unary success that must not run")}, nil
		},
		StreamFn: func(context.Context, string, ttschain.Request) (<-chan ttschain.AudioFrame, error) {
			return nil, fmt.Errorf("billing: %w", ttschain.ErrInsufficientCredits)
		},
	}
	c := ttschain.NewChain(ttschain.Options{
		EnableBYOK: true,
		BYOK:       ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"x": adapter}),
	})

	frames, err := c.Stream(context.Background(), ttschain.Request{BYOKProvider: "x", BYOKKey: "k", Text: "hi"})
	require.Nil(t, frames)
	require.ErrorIs(t, err, ttschain.ErrInsufficientCredits)
}

func TestChain_Stream_NonTerminalErrorFallsBack(t *testing.T) {
	adapter := &ttsmocks.FakeBYOK{
		IDStr: "x", Available: true, Streaming: true,
		SynthesizeFn: func(context.Context, string, ttschain.Request) (*ttschain.Result, error) {
			return &ttschain.Result{Audio: []byte("buffered")}, nil
		},
		StreamFn: func(context.Context, string, ttschain.Request) (<-chan ttschain.AudioFrame, error) {
			return nil, errors.New("transport reset")
		},
	}
	c := ttschain.NewChain(ttschain.Options{
		EnableBYOK: true,
		BYOK:       ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"x": adapter}),
	})

	frames, err := c.Stream(context.Background(), ttschain.Request{BYOKProvider: "x", BYOKKey: "k", Text: "hi"})
	require.NoError(t, err)
	var got []ttschain.AudioFrame
	for frame := range frames {
		got = append(got, frame)
	}
	require.Len(t, got, 1)
	require.Equal(t, "buffered", string(got[0].Audio))
}
