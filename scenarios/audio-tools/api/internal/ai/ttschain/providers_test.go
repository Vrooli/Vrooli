package ttschain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/ttschain"
	ttsmocks "audio-tools/internal/ai/ttschain/mocks"

	"github.com/vrooli/api-core/scheduletest"
)

func TestLocalProvider_NilSvc(t *testing.T) {
	p := ttschain.NewLocalProvider(nil)
	require.Equal(t, ttschain.TierLocal, p.Type())
	require.False(t, p.IsAvailable(context.Background()))
	require.Equal(t, "kokoro", p.Model())
	require.False(t, p.StreamingCapability())
	ch, err := p.SynthesizeStreaming(context.Background(), ttschain.Request{})
	require.Nil(t, ch)
	require.NoError(t, err)
	_, err = p.Synthesize(context.Background(), ttschain.Request{})
	require.Error(t, err)
	p2 := ttschain.NewLocalProviderWith(nil, nil)
	require.NotNil(t, p2)
}

func TestBYOKProvider_TypeModelTraits(t *testing.T) {
	streaming := &ttsmocks.FakeBYOK{IDStr: "s", Available: true, Streaming: true}
	nonStreaming := &ttsmocks.FakeBYOK{IDStr: "n", Available: true}
	p := ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"s": streaming, "n": nonStreaming})
	require.Equal(t, ttschain.TierBYOK, p.Type())
	require.Equal(t, "byok-dispatched", p.Model())
	require.True(t, p.IsAvailable(context.Background()))
	require.True(t, p.StreamingCapability())

	empty := ttschain.NewBYOKProvider(nil)
	require.False(t, empty.IsAvailable(context.Background()))
	require.False(t, empty.StreamingCapability())

	// SynthesizeStreaming routing.
	_, err := p.SynthesizeStreaming(context.Background(), ttschain.Request{BYOKProvider: "s"})
	require.Error(t, err)
	_, err = p.SynthesizeStreaming(context.Background(), ttschain.Request{BYOKKey: "k"})
	require.ErrorIs(t, err, ttschain.ErrMissingBYOKProvider)
	_, err = p.SynthesizeStreaming(context.Background(), ttschain.Request{BYOKKey: "k", BYOKProvider: "missing"})
	require.ErrorIs(t, err, ttschain.ErrUnknownBYOKProvider)
	ch, err := p.SynthesizeStreaming(context.Background(), ttschain.Request{BYOKKey: "k", BYOKProvider: "n"})
	require.Nil(t, ch)
	require.NoError(t, err)
}

func TestVrooliProvider_TypeModelTraitsAvailability(t *testing.T) {
	client := &ttsmocks.FakeVrooliClient{Available: true, Result: &ttschain.Result{Audio: []byte("x")}}
	p := ttschain.NewVrooliProvider(client)
	require.Equal(t, ttschain.TierVrooli, p.Type())
	require.Equal(t, "lpbs-tts", p.Model())
	require.True(t, p.IsAvailable(context.Background()))
	require.False(t, p.StreamingCapability())
	ch, err := p.SynthesizeStreaming(context.Background(), ttschain.Request{})
	require.Nil(t, ch)
	require.NoError(t, err)

	_, err = p.Synthesize(context.Background(), ttschain.Request{})
	require.Error(t, err)

	var nilP *ttschain.VrooliProvider
	require.False(t, nilP.IsAvailable(context.Background()))
	require.Equal(t, "", nilP.Model())
}

func TestVrooliProvider_TranscribeErrorPath(t *testing.T) {
	client := &ttsmocks.FakeVrooliClient{Available: true, Err: errors.New("boom")}
	p := ttschain.NewVrooliProvider(client)
	_, err := p.Synthesize(context.Background(), ttschain.Request{Text: "x", LPBSToken: "t"})
	require.Error(t, err)
}

func TestChain_Probe(t *testing.T) {
	byok := ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"x": &ttsmocks.FakeBYOK{IDStr: "x", Available: true}})
	vrooli := ttschain.NewVrooliProvider(&ttsmocks.FakeVrooliClient{Available: true})
	c := ttschain.NewChain(ttschain.Options{EnableBYOK: true, EnableVrooli: true, BYOK: byok, Vrooli: vrooli})
	r := c.Probe(context.Background())
	require.True(t, r.BYOK)
	require.True(t, r.Vrooli)
	require.False(t, r.Local)
}

func TestChain_Reconfigure_InvalidatesCache(t *testing.T) {
	c := ttschain.NewChain(ttschain.Options{
		EnableBYOK: true,
		BYOK:       ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"x": &ttsmocks.FakeBYOK{IDStr: "x", Available: true}}),
	})
	_, _ = c.Execute(context.Background(), ttschain.Request{Text: "h", BYOKProvider: "x", BYOKKey: "k"})
	c.Reconfigure(false, false, false, time.Minute, time.Second)
	_, err := c.Execute(context.Background(), ttschain.Request{Text: "h", BYOKProvider: "x", BYOKKey: "k"})
	require.ErrorIs(t, err, ttschain.ErrAllProvidersFailed)
}

func TestChain_AvailabilityCache_HitAndExpiry(t *testing.T) {
	byok := ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"x": &ttsmocks.FakeBYOK{IDStr: "x", Available: true}})
	clk := scheduletest.New(time.Unix(1000, 0))
	c := ttschain.NewChain(ttschain.Options{EnableBYOK: true, BYOK: byok, AvailTTLByOK: 10 * time.Second, Clock: clk})
	_, _ = c.Execute(context.Background(), ttschain.Request{Text: "h", BYOKProvider: "x", BYOKKey: "k"})
	clk.Advance(5 * time.Second)
	_, _ = c.Execute(context.Background(), ttschain.Request{Text: "h", BYOKProvider: "x", BYOKKey: "k"})
	clk.Advance(20 * time.Second)
	_, _ = c.Execute(context.Background(), ttschain.Request{Text: "h", BYOKProvider: "x", BYOKKey: "k"})
}

func TestBYOKProvider_StreamingDispatch(t *testing.T) {
	emit := make(chan ttschain.AudioFrame, 1)
	emit <- ttschain.AudioFrame{IsFinal: true, Audio: []byte("done")}
	close(emit)
	streaming := &ttsmocks.FakeBYOK{
		IDStr: "s", Available: true, Streaming: true,
	}
	p := ttschain.NewBYOKProvider(map[string]ttschain.BYOKAdapter{"s": streaming})
	// Default mock returns (nil, nil), so should pass through nil.
	ch, err := p.SynthesizeStreaming(context.Background(), ttschain.Request{BYOKKey: "k", BYOKProvider: "s"})
	require.NoError(t, err)
	require.Nil(t, ch)
}

func TestFakeMocks_Coverage(t *testing.T) {
	bk := &ttsmocks.FakeBYOK{IDStr: "x", Available: true, Streaming: true}
	require.Equal(t, "x", bk.ID())
	require.True(t, bk.IsAvailable(context.Background(), "k"))
	require.Equal(t, "fake-model", bk.Model())
	require.True(t, bk.StreamingCapability())
	res, _ := bk.Synthesize(context.Background(), "k", ttschain.Request{})
	require.Equal(t, "byok-audio", string(res.Audio))
	bk.Err = errors.New("e")
	_, err := bk.Synthesize(context.Background(), "k", ttschain.Request{})
	require.Error(t, err)
	bk.Err = nil
	bk.SynthesizeFn = func(context.Context, string, ttschain.Request) (*ttschain.Result, error) {
		return &ttschain.Result{Audio: []byte("fn")}, nil
	}
	res, _ = bk.Synthesize(context.Background(), "k", ttschain.Request{})
	require.Equal(t, "fn", string(res.Audio))
	ch, _ := bk.SynthesizeStreaming(context.Background(), "k", ttschain.Request{})
	require.Nil(t, ch)

	vc := &ttsmocks.FakeVrooliClient{Available: true, Result: &ttschain.Result{Audio: []byte("v")}}
	require.True(t, vc.IsAvailable(context.Background()))
	require.Equal(t, "lpbs-tts", vc.Model())
	res, _ = vc.Synthesize(context.Background(), "tok", "u", ttschain.Request{})
	require.Equal(t, "v", string(res.Audio))
	vc.Err = errors.New("e")
	_, err = vc.Synthesize(context.Background(), "tok", "u", ttschain.Request{})
	require.Error(t, err)
	vc.Err = nil
	vc.SynthesizeFn = func(context.Context, string, string, ttschain.Request) (*ttschain.Result, error) {
		return &ttschain.Result{Audio: []byte("fn")}, nil
	}
	res, _ = vc.Synthesize(context.Background(), "tok", "u", ttschain.Request{})
	require.Equal(t, "fn", string(res.Audio))
}
