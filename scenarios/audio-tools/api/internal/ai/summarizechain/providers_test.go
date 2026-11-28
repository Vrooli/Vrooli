package summarizechain_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/summarizechain"
	summocks "audio-tools/internal/ai/summarizechain/mocks"
	"audio-tools/internal/summarize"
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

func TestLocalProvider_PreservesExplicitSourceAssignment(t *testing.T) {
	s := summarize.NewSummarizerWithRunner("unused", func(context.Context, []string, string) ([]byte, error) {
		return []byte(`{"model":"fixture","response":"Atlas opens in Denver. Priya manages captions.","done_reason":"stop"}`), nil
	})
	p := summarizechain.NewLocalProvider(s, "model")
	res, err := p.Summarize(context.Background(), summarizechain.Request{
		Text:  "Atlas opens in Denver. Priya owns caption follow-up. Marcus owns legal approval.",
		Level: "moderate",
	})
	require.NoError(t, err)
	require.Contains(t, res.Text, "Marcus owns legal approval.")
	require.Contains(t, res.Text, "Priya manages captions.")
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

func TestLocalRoleNeverOverridesBYOKModelSelection(t *testing.T) {
	var received summarizechain.Request
	adapter := &summocks.FakeBYOK{IDStr: "openrouter", Available: true, SummarizeFn: func(_ context.Context, _ string, req summarizechain.Request) (*summarizechain.Result, error) {
		received = req
		return &summarizechain.Result{Text: "remote"}, nil
	}}
	p := summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{"openrouter": adapter})
	_, err := p.Summarize(context.Background(), summarizechain.Request{BYOKProvider: "openrouter", BYOKKey: "secret", Model: "vendor-model-id", LocalModelRole: "summarize.audio_tools.quality"})
	require.NoError(t, err)
	require.Equal(t, "vendor-model-id", received.Model, "provider-specific override is preserved")
	require.Empty(t, received.LocalModelRole, "local role is never forwarded to a remote adapter")
}

func TestLocalRoleNeverOverridesLPBSModelSelection(t *testing.T) {
	var received summarizechain.Request
	client := &summocks.FakeVrooliClient{Available: true, SummarizeFn: func(_ context.Context, _, _ string, req summarizechain.Request) (*summarizechain.Result, error) {
		received = req
		return &summarizechain.Result{Text: "remote"}, nil
	}}
	p := summarizechain.NewVrooliProvider(client)
	_, err := p.Summarize(context.Background(), summarizechain.Request{LPBSToken: "token", Model: "lpbs-model-override", LocalModelRole: "summarize.audio_tools.quality"})
	require.NoError(t, err)
	require.Equal(t, "lpbs-model-override", received.Model, "existing LPBS-specific model override remains intact")
	require.Empty(t, received.LocalModelRole, "local role is never forwarded to LPBS")
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

func TestChain_Probe_RespectsEnableFlagsAndMissingLocal(t *testing.T) {
	byok := summarizechain.NewBYOKProvider(map[string]summarizechain.BYOKAdapter{
		"openrouter": &summocks.FakeBYOK{IDStr: "openrouter", Available: true},
	})
	vrooli := summarizechain.NewVrooliProvider(&summocks.FakeVrooliClient{Available: true})
	c := summarizechain.NewChain(summarizechain.Options{
		EnableBYOK: true, EnableVrooli: true, EnableLocal: true,
		BYOK: byok, Vrooli: vrooli,
	})
	r := c.Probe(context.Background())
	require.True(t, r.BYOK)
	require.True(t, r.Vrooli)
	require.False(t, r.Local, "local is enabled but not configured")

	c.Reconfigure(false, true, true, time.Minute, time.Second)
	r = c.Probe(context.Background())
	require.False(t, r.BYOK, "disabled tiers are not probed")
	require.True(t, r.Vrooli)
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
