package livesearch_test

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	livesearchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/livesearch"

	handler "web-search/handlers/livesearch"
	internallivesearch "web-search/internal/livesearch"
)

// fakeClient is a SearxngClient seam for the handler test.
type fakeClient struct {
	results []internallivesearch.RawResult
}

func (f *fakeClient) Search(_ context.Context, _ string, _ int) ([]internallivesearch.RawResult, error) {
	return f.results, nil
}

// fakeSynthesizer returns a canned synthesis.
type fakeSynthesizer struct {
	out *internallivesearch.Synthesis
}

func (f *fakeSynthesizer) Synthesize(_ context.Context, _ string, _ []internallivesearch.Result) (*internallivesearch.Synthesis, error) {
	return f.out, nil
}

func newHandler(client internallivesearch.SearxngClient, syn internallivesearch.Synthesizer) *handler.Deps {
	svc := internallivesearch.NewService(internallivesearch.Deps{Client: client, Synthesizer: syn})
	return &handler.Deps{Service: svc}
}

func TestSearchProjectsResultsToProto(t *testing.T) {
	client := &fakeClient{results: []internallivesearch.RawResult{
		{URL: "https://anthropic.com", Title: "Anthropic", Content: "Claude maker", Engine: "google", Score: 0.9, Category: "general"},
	}}
	h := handler.NewConnectHandler(*newHandler(client, nil))

	resp, err := h.Search(context.Background(), connect.NewRequest(&livesearchv1.SearchRequest{Query: "anthropic", Limit: 5}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Results, 1)
	r := resp.Msg.Results[0]
	require.Equal(t, "https://anthropic.com", r.Url)
	require.Equal(t, "Claude maker", r.Snippet) // content -> snippet
	require.Equal(t, "google", r.Engine)
	require.InDelta(t, 0.9, r.Score, 1e-9)
	require.False(t, resp.Msg.Degraded)
	require.Nil(t, resp.Msg.Synthesis, "synthesis off by default")
}

func TestSearchSurfacesCitedSynthesis(t *testing.T) {
	client := &fakeClient{results: []internallivesearch.RawResult{
		{URL: "https://anthropic.com", Title: "Anthropic", Content: "Claude maker"},
	}}
	syn := &fakeSynthesizer{out: &internallivesearch.Synthesis{
		Text:      "Anthropic makes Claude.",
		Citations: []internallivesearch.Citation{{ResultIndex: 0, URL: "https://anthropic.com", Title: "Anthropic"}},
	}}
	h := handler.NewConnectHandler(*newHandler(client, syn))

	resp, err := h.Search(context.Background(), connect.NewRequest(&livesearchv1.SearchRequest{Query: "claude", Synthesize: true}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.Synthesis)
	require.False(t, resp.Msg.Synthesis.Abstained)
	require.Equal(t, "Anthropic makes Claude.", resp.Msg.Synthesis.Text)
	require.Len(t, resp.Msg.Synthesis.Citations, 1)
	require.Equal(t, int32(0), resp.Msg.Synthesis.Citations[0].ResultIndex)
}

func TestSearchDegradedOnBudgetExhaustion(t *testing.T) {
	// Drain a capacity-1 governor so the service's next call degrades without
	// hitting the client; the handler returns a successful (non-error) degraded
	// response rather than a Connect error.
	gov := internallivesearch.NewGovernor(1, 0, nil)
	require.True(t, gov.Allow())
	require.False(t, gov.Allow())

	drained := internallivesearch.NewService(internallivesearch.Deps{Client: &fakeClient{}, Governor: gov})
	h := handler.NewConnectHandler(handler.Deps{Service: drained})
	resp, err := h.Search(context.Background(), connect.NewRequest(&livesearchv1.SearchRequest{Query: "q"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Degraded)
	require.NotEmpty(t, resp.Msg.DegradedReason)
	require.Empty(t, resp.Msg.Results)
}
