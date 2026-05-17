package summarizechain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/summarizechain"
	summocks "audio-tools/internal/ai/summarizechain/mocks"
)

func TestLocalProvider_NilSummarizer(t *testing.T) {
	p := summarizechain.NewLocalProvider(nil, "model")
	require.Equal(t, summarizechain.TierLocal, p.Type())
	require.False(t, p.IsAvailable(context.Background()))
	require.Equal(t, "model", p.Model())
	_, err := p.Summarize(context.Background(), summarizechain.Request{})
	require.Error(t, err)
	p2 := summarizechain.NewLocalProviderWith(nil, "m", nil)
	require.NotNil(t, p2)
	var nilP *summarizechain.LocalProvider
	require.Equal(t, "", nilP.Model())
}

func TestBYOKProvider_TypeModel(t *testing.T) {
	p := summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{
		"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true},
	})
	require.Equal(t, summarizechain.TierBYOK, p.Type())
	require.Equal(t, "byok-dispatched", p.Model())
	require.True(t, p.IsAvailable(context.Background()))
	empty := summarizechain.NewBYOKProvider(nil)
	require.False(t, empty.IsAvailable(context.Background()))

	// Missing key.
	_, err := p.Summarize(context.Background(), summarizechain.Request{BYOKProvider: "openrouter"})
	require.Error(t, err)
}

func TestVrooliProvider_TypeModelAvailability(t *testing.T) {
	client := &summocks.FakeVrooliClient{Available: true, Result: &summarizechain.Result{Text: "v"}}
	p := summarizechain.NewVrooliProvider(client)
	require.Equal(t, summarizechain.TierVrooli, p.Type())
	require.Equal(t, "lpbs-model", p.Model())
	require.True(t, p.IsAvailable(context.Background()))
	// Missing token error path.
	_, err := p.Summarize(context.Background(), summarizechain.Request{})
	require.Error(t, err)

	var nilP *summarizechain.VrooliProvider
	require.False(t, nilP.IsAvailable(context.Background()))
	require.Equal(t, "", nilP.Model())
}

func TestVrooliProvider_ErrorPath(t *testing.T) {
	client := &summocks.FakeVrooliClient{Available: true, Err: errors.New("boom")}
	p := summarizechain.NewVrooliProvider(client)
	_, err := p.Summarize(context.Background(), summarizechain.Request{LPBSToken: "t"})
	require.Error(t, err)
}

func TestChain_Probe(t *testing.T) {
	byok := summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{
		"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true},
	})
	vrooli := summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{Available: true})
	c := summarizechain.NewChain(summarizechain.Options{EnableBYOK: true, EnableVrooli: true, BYOK: byok, Vrooli: vrooli})
	r := c.Probe(context.Background())
	require.True(t, r.BYOK)
	require.True(t, r.Vrooli)
	require.False(t, r.Local)
}

func TestFakeMocks_Coverage(t *testing.T) {
	bk := &summocks.FakeBYOK{IDStr: "x", Available: true}
	require.Equal(t, "x", bk.ID())
	require.True(t, bk.IsAvailable(context.Background(), "k"))
	require.Equal(t, "fake-model", bk.Model())
	res, _ := bk.Summarize(context.Background(), "k", summarizechain.Request{})
	require.Equal(t, "byok-summary", res.Text)
	bk.Err = errors.New("e")
	_, err := bk.Summarize(context.Background(), "k", summarizechain.Request{})
	require.Error(t, err)
	bk.Err = nil
	bk.SummarizeFn = func(context.Context, string, summarizechain.Request) (*summarizechain.Result, error) {
		return &summarizechain.Result{Text: "fn"}, nil
	}
	res, _ = bk.Summarize(context.Background(), "k", summarizechain.Request{})
	require.Equal(t, "fn", res.Text)

	vc := &summocks.FakeVrooliClient{Available: true}
	require.True(t, vc.IsAvailable(context.Background()))
	require.Equal(t, "lpbs-model", vc.Model())
	res, _ = vc.Summarize(context.Background(), "tok", "u", summarizechain.Request{})
	require.Equal(t, "vrooli-summary", res.Text)
	vc.Err = errors.New("e")
	_, err = vc.Summarize(context.Background(), "tok", "u", summarizechain.Request{})
	require.Error(t, err)
}
