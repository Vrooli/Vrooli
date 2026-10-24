package ttschain_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/ttschain"
	ttsmocks "audio-tools/internal/ai/ttschain/mocks"
)

func TestChain_WiresCredentialProviders(t *testing.T) {
	chain := ttschain.NewChain(ttschain.Options{
		EnableBYOK: true, EnableVrooli: true,
		BYOK:   ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"el": &ttsmocks.FakeBYOK{IDStr: "el", Available: true}}),
		Vrooli: ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{Available: true}),
	})

	byok, err := chain.Execute(context.Background(), ttschain.Request{Text: "hello", BYOKProvider: "el", BYOKKey: "sk", LPBSToken: "tok"})
	require.NoError(t, err)
	require.Equal(t, "byok-audio", string(byok.Audio))

	vrooli, err := chain.Execute(context.Background(), ttschain.Request{Text: "hello", LPBSToken: "tok"})
	require.NoError(t, err)
	require.Equal(t, "vrooli-audio", string(vrooli.Audio))
}

func TestChain_Stream_BufferedFallbackEmitsFinalFrame(t *testing.T) {
	// No tier declares streaming; fallback path runs Execute and emits
	// one is_final=true frame carrying the buffered audio.
	c := ttschain.NewChain(ttschain.Options{
		EnableBYOK: true,
		BYOK: ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{
			"el": &ttsmocks.FakeBYOK{IDStr: "el", Available: true},
		}),
	})
	frames, err := c.Stream(context.Background(), ttschain.Request{Text: "hi", BYOKProvider: "el", BYOKKey: "sk"})
	require.NoError(t, err)

	var got []ttschain.AudioFrame
	for f := range frames {
		got = append(got, f)
	}
	require.Len(t, got, 1)
	require.True(t, got[0].IsFinal)
	require.Equal(t, "byok-audio", string(got[0].Audio))
}
