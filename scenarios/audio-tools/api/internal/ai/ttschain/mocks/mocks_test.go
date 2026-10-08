package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/ttschain"
)

func TestFakeBYOK_Smoke(t *testing.T) {
	f := &FakeBYOK{IDStr: "x", Available: true, Streaming: true}
	require.Equal(t, "x", f.ID())
	require.True(t, f.IsAvailable(context.Background(), "k"))
	require.Equal(t, "fake-model", f.Model())
	require.True(t, f.StreamingCapability())
	res, _ := f.Synthesize(context.Background(), "k", ttschain.Request{})
	require.Equal(t, "byok-audio", string(res.Audio))
	f.Err = errors.New("e")
	_, err := f.Synthesize(context.Background(), "k", ttschain.Request{})
	require.Error(t, err)
	f.Err = nil
	f.SynthesizeFn = func(context.Context, string, ttschain.Request) (*ttschain.Result, error) {
		return &ttschain.Result{Audio: []byte("fn")}, nil
	}
	res, _ = f.Synthesize(context.Background(), "k", ttschain.Request{})
	require.Equal(t, "fn", string(res.Audio))
	ch, err := f.SynthesizeStreaming(context.Background(), "k", ttschain.Request{})
	require.Nil(t, ch)
	require.NoError(t, err)
}

func TestFakeVrooliClient_Smoke(t *testing.T) {
	c := &FakeVrooliClient{Available: true}
	require.True(t, c.IsAvailable(context.Background()))
	require.Equal(t, "lpbs-tts", c.Model())
	res, _ := c.Synthesize(context.Background(), "t", "u", ttschain.Request{})
	require.Equal(t, "vrooli-audio", string(res.Audio))
	c.Err = errors.New("e")
	_, err := c.Synthesize(context.Background(), "t", "u", ttschain.Request{})
	require.Error(t, err)
	c.Err = nil
	c.SynthesizeFn = func(context.Context, string, string, ttschain.Request) (*ttschain.Result, error) {
		return &ttschain.Result{Audio: []byte("fn")}, nil
	}
	res, _ = c.Synthesize(context.Background(), "t", "u", ttschain.Request{})
	require.Equal(t, "fn", string(res.Audio))
}
