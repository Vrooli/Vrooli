package sttchain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
	sttmocks "audio-tools/internal/ai/sttchain/mocks"
	"audio-tools/internal/stt/whisperinfo"
	whisperinfomocks "audio-tools/internal/stt/whisperinfo/mocks"
)

func TestLocalProvider_NilSvc(t *testing.T) {
	infoFake := &whisperinfomocks.FakeClient{Info: whisperinfo.Info{ModelID: "whisper-medium"}}
	p := sttchain.NewLocalProviderWith(nil, nil, infoFake)
	require.Equal(t, sttchain.TierLocal, p.Type())
	require.False(t, p.IsAvailable(context.Background()))
	require.Equal(t, "whisper-medium", p.Model())
	_, err := p.Transcribe(context.Background(), sttchain.Request{})
	require.Error(t, err)
	tr := p.Traits()
	require.True(t, tr.Batch)
	require.False(t, tr.Stream)
	ch, err := p.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, nil)
	require.Nil(t, ch)
	require.NoError(t, err)
	// NewLocalProvider with the default EnvClient and env unset → Unknown.
	pDefault := sttchain.NewLocalProvider(nil)
	require.Equal(t, whisperinfo.ModelUnknown, pDefault.Model())
	// Nil info client safely falls back to EnvClient default.
	p2 := sttchain.NewLocalProviderWith(nil, nil, nil)
	require.NotNil(t, p2)
}

func TestBYOKProvider_TypeModelTraits(t *testing.T) {
	streaming := &sttmocks.FakeBYOK{IDStr: "s", Available: true, Streaming: true}
	nonStreaming := &sttmocks.FakeBYOK{IDStr: "n", Available: true}
	p := sttchain.NewBYOKProvider(map[string]sttchain.BYOKAdapter{"s": streaming, "n": nonStreaming})
	require.Equal(t, sttchain.TierBYOK, p.Type())
	require.Equal(t, "byok-dispatched", p.Model())
	require.True(t, p.IsAvailable(context.Background()))
	tr := p.Traits()
	require.True(t, tr.Stream)

	empty := sttchain.NewBYOKProvider(nil)
	require.False(t, empty.IsAvailable(context.Background()))
	require.False(t, empty.Traits().Stream)

	// Missing key.
	_, err := p.Transcribe(context.Background(), sttchain.Request{BYOKProvider: "s"})
	require.Error(t, err)

	// Streaming dispatch decline (no key).
	_, err = p.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, nil)
	require.Error(t, err)
	_, err = p.TranscribeStreaming(context.Background(), sttchain.StreamStart{BYOKKey: "k"}, nil)
	require.ErrorIs(t, err, sttchain.ErrMissingBYOKProvider)
	_, err = p.TranscribeStreaming(context.Background(), sttchain.StreamStart{BYOKKey: "k", BYOKProvider: "missing"}, nil)
	require.ErrorIs(t, err, sttchain.ErrUnknownBYOKProvider)
	// Adapter that declares no streaming returns (nil, nil).
	ch, err := p.TranscribeStreaming(context.Background(), sttchain.StreamStart{BYOKKey: "k", BYOKProvider: "n"}, nil)
	require.Nil(t, ch)
	require.NoError(t, err)
}

func TestVrooliProvider_TypeModelTraitsAvailability(t *testing.T) {
	client := &sttmocks.FakeVrooliClient{Available: true, Result: &sttchain.Result{Text: "x"}}
	p := sttchain.NewVrooliProvider(client)
	require.Equal(t, sttchain.TierVrooli, p.Type())
	require.Equal(t, "lpbs-model", p.Model())
	require.True(t, p.IsAvailable(context.Background()))
	tr := p.Traits()
	require.True(t, tr.Batch)
	require.False(t, tr.Stream)
	// Missing token.
	_, err := p.Transcribe(context.Background(), sttchain.Request{})
	require.Error(t, err)
	// Streaming declined.
	ch, err := p.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, nil)
	require.Nil(t, ch)
	require.NoError(t, err)

	// Nil client variants.
	var nilP *sttchain.VrooliProvider
	require.False(t, nilP.IsAvailable(context.Background()))
	require.Equal(t, "", nilP.Model())
}

func TestVrooliProvider_TranscribeErrorPath(t *testing.T) {
	client := &sttmocks.FakeVrooliClient{Available: true, Err: errors.New("boom")}
	p := sttchain.NewVrooliProvider(client)
	_, err := p.Transcribe(context.Background(), sttchain.Request{LPBSToken: "t"})
	require.Error(t, err)
}

func TestFakeProvider_Methods(t *testing.T) {
	fp := sttmocks.NewFakeProvider(sttchain.TierLocal, sttchain.ProviderTraits{Batch: true, Stream: true})
	require.Equal(t, sttchain.TierLocal, fp.Type())
	require.True(t, fp.IsAvailable(context.Background()))
	require.Equal(t, "fake", fp.Model())
	require.True(t, fp.Traits().Stream)
	res, err := fp.Transcribe(context.Background(), sttchain.Request{})
	require.NoError(t, err)
	require.Equal(t, "x", res.Text)

	// Err path
	fp.Err = errors.New("x")
	_, err = fp.Transcribe(context.Background(), sttchain.Request{})
	require.Error(t, err)
	fp.Err = nil

	// TranscribeFn override
	fp.TranscribeFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "fn"}, nil
	}
	res, _ = fp.Transcribe(context.Background(), sttchain.Request{})
	require.Equal(t, "fn", res.Text)

	// TranscribeStreaming default
	ch, err := fp.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, nil)
	require.Nil(t, ch)
	require.NoError(t, err)

	// StreamFn override
	out := make(chan sttchain.StreamEvent)
	close(out)
	fp.StreamFn = func(context.Context, sttchain.StreamStart, <-chan sttchain.AudioChunk) (<-chan sttchain.StreamEvent, error) {
		return out, nil
	}
	got, err := fp.TranscribeStreaming(context.Background(), sttchain.StreamStart{}, nil)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestFakeBYOK_IsAvailable(t *testing.T) {
	f := &sttmocks.FakeBYOK{Available: true}
	require.True(t, f.IsAvailable(context.Background(), "k"))
}

func TestFakeBatchExecutor(t *testing.T) {
	e := &sttmocks.FakeBatchExecutor{Result: &sttchain.Result{Text: "r"}}
	res, err := e.Execute(context.Background(), sttchain.Request{})
	require.NoError(t, err)
	require.Equal(t, "r", res.Text)
	e.Err = errors.New("x")
	_, err = e.Execute(context.Background(), sttchain.Request{})
	require.Error(t, err)
	e.Err = nil
	e.ExecuteFn = func(context.Context, sttchain.Request) (*sttchain.Result, error) {
		return &sttchain.Result{Text: "fn"}, nil
	}
	res, _ = e.Execute(context.Background(), sttchain.Request{})
	require.Equal(t, "fn", res.Text)

	// Default-only fallback.
	e2 := &sttmocks.FakeBatchExecutor{}
	res2, err := e2.Execute(context.Background(), sttchain.Request{})
	require.NoError(t, err)
	require.NotNil(t, res2)
}
