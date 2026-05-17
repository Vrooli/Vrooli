package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
)

func TestFakeBYOK_Smoke(t *testing.T) {
	f := &FakeBYOK{IDStr: "x", Available: true, Streaming: true}
	require.Equal(t, "x", f.ID())
	require.True(t, f.IsAvailable(context.Background(), "k"))
	require.Equal(t, "fake-model", f.Model())
	require.True(t, f.StreamingCapability())
	res, _ := f.Transcribe(context.Background(), "k", sttchain.Request{})
	require.Equal(t, "byok", res.Text)
	f.Err = errors.New("e")
	_, err := f.Transcribe(context.Background(), "k", sttchain.Request{})
	require.Error(t, err)
	f.Err = nil
	f.TranscribeFn = func(context.Context, string, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "fn"}, nil
	}
	res, _ = f.Transcribe(context.Background(), "k", sttchain.Request{})
	require.Equal(t, "fn", res.Text)
	ch, err := f.TranscribeStreaming(context.Background(), "k", sttchain.StreamStart{}, nil)
	require.Nil(t, ch)
	require.NoError(t, err)
	f.StreamFn = func(context.Context, string, sttchain.StreamStart, <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
		out := make(chan sttchain.StreamEvent)
		close(out)
		return out, nil
	}
	ch, _ = f.TranscribeStreaming(context.Background(), "k", sttchain.StreamStart{}, nil)
	require.NotNil(t, ch)
}

func TestFakeProvider_Smoke(t *testing.T) {
	p := NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{})
	require.Equal(t, sttchain.TierLocal, p.Type())
	require.True(t, p.IsAvailable(context.Background()))
	require.Equal(t, "fake", p.Model())
	require.False(t, p.Traits().Stream)
	res, _ := p.Transcribe(context.Background(), sttchain.Request{})
	require.Equal(t, "x", res.Text)
	ch, _ := p.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, nil)
	require.Nil(t, ch)
	p.Result = &sttchain.Result{Text: "explicit"}
	res, _ = p.Transcribe(context.Background(), sttchain.Request{})
	require.Equal(t, "explicit", res.Text)
}

func TestFakeBatchExecutor_Smoke(t *testing.T) {
	e := &FakeBatchExecutor{}
	res, err := e.Execute(context.Background(), sttchain.Request{})
	require.NoError(t, err)
	require.NotNil(t, res)
	e.Result = &sttchain.Result{Text: "r"}
	res, _ = e.Execute(context.Background(), sttchain.Request{})
	require.Equal(t, "r", res.Text)
}

func TestFakeVrooliClient_Smoke(t *testing.T) {
	c := &FakeVrooliClient{Available: true}
	require.True(t, c.IsAvailable(context.Background()))
	require.Equal(t, "lpbs-model", c.Model())
	res, _ := c.Transcribe(context.Background(), "t", "u", sttchain.Request{})
	require.Equal(t, "vrooli", res.Text)
	c.Err = errors.New("e")
	_, err := c.Transcribe(context.Background(), "t", "u", sttchain.Request{})
	require.Error(t, err)
	c.Err = nil
	c.TranscribeFn = func(context.Context, string, string, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "fn"}, nil
	}
	res, _ = c.Transcribe(context.Background(), "t", "u", sttchain.Request{})
	require.Equal(t, "fn", res.Text)
}
