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
	results      []internallivesearch.RawResult
	engineIssues []internallivesearch.EngineIssue
}

func (f *fakeClient) Search(_ context.Context, _ string, _ int) (internallivesearch.SearchPage, error) {
	return internallivesearch.SearchPage{Results: f.results, UnresponsiveEngines: f.engineIssues}, nil
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
	// The synthesis is ADDITIVE: the raw hits ride in the same envelope, in a
	// separate field, never displaced by the optional answer.
	require.Len(t, resp.Msg.Results, 1, "raw hits must accompany the synthesis in the same response envelope")
	require.Equal(t, "https://anthropic.com", resp.Msg.Results[0].Url)
}

// TestSearchSurfacesAbstainSignal pins the conflict contract at the wire: when
// the synthesizer abstains (conflicting/thin sources), the client receives an
// EXPLICIT abstained=true signal — never a misleading answer — while the raw
// hits remain available for the user to judge themselves.
func TestSearchSurfacesAbstainSignal(t *testing.T) {
	client := &fakeClient{results: []internallivesearch.RawResult{
		{URL: "https://a.com", Title: "A", Content: "Claude is made by Anthropic."},
		{URL: "https://b.com", Title: "B", Content: "Claude is made by OpenAI."},
	}}
	syn := &fakeSynthesizer{out: &internallivesearch.Synthesis{
		Text:      "sources insufficient or disagree",
		Abstained: true,
	}}
	h := handler.NewConnectHandler(*newHandler(client, syn))

	resp, err := h.Search(context.Background(), connect.NewRequest(&livesearchv1.SearchRequest{Query: "who makes claude", Synthesize: true}))
	require.NoError(t, err)
	require.NotNil(t, resp.Msg.Synthesis)
	require.True(t, resp.Msg.Synthesis.Abstained, "conflicting sources must surface an explicit abstain signal")
	require.Empty(t, resp.Msg.Synthesis.Citations)
	require.Len(t, resp.Msg.Results, 2, "raw hits still flow when synthesis abstains")
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
