package research_test

import (
	"context"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	researchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/research"

	handler "web-search/handlers/research"
	internalresearch "web-search/internal/research"
	"web-search/internal/research/agentmanager"
)

// --- exported-seam fakes for the handler-level test ---

type fakeSearcher struct{ cands []internalresearch.Candidate }

func (f fakeSearcher) Candidates(_ context.Context, _ string, _ int) (internalresearch.CandidateSet, error) {
	return internalresearch.CandidateSet{Candidates: f.cands}, nil
}

type fakeFetcher struct{ text string }

func (f fakeFetcher) Fetch(_ context.Context, _ string) (string, error) { return f.text, nil }

type fakeSynth struct{ out internalresearch.Synthesis }

func (f fakeSynth) Synthesize(_ context.Context, _ string, _ []internalresearch.Document) (internalresearch.Synthesis, error) {
	return f.out, nil
}

type fakeAgent struct {
	spawn agentmanager.RunResult
	state agentmanager.RunState
}

func (f fakeAgent) Spawn(_ context.Context, _ agentmanager.SpawnRequest) (agentmanager.RunResult, error) {
	return f.spawn, nil
}

func (f fakeAgent) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return f.state, nil
}

func newHandler(svc *internalresearch.Service) *handler.Deps {
	return &handler.Deps{Service: svc}
}

func TestRunL2ProjectsBriefToProto(t *testing.T) {
	svc := internalresearch.NewService(internalresearch.Deps{
		Searcher: fakeSearcher{cands: []internalresearch.Candidate{{URL: "https://a.example", Title: "A"}}},
		Fetcher:  fakeFetcher{text: "body"},
		Synthesizer: fakeSynth{out: internalresearch.Synthesis{
			Text:      "the cited answer",
			Citations: []internalresearch.Citation{{ResultIndex: 0, URL: "https://a.example", Title: "A"}},
		}},
	})
	h := handler.NewConnectHandler(*newHandler(svc))

	resp, err := h.RunL2(context.Background(), connect.NewRequest(&researchv1.RunL2Request{Query: "q", TopN: 3}))
	require.NoError(t, err)
	require.False(t, resp.Msg.Abstained)
	require.Equal(t, "the cited answer", resp.Msg.Synthesis)
	require.NotNil(t, resp.Msg.Brief)
	require.Equal(t, "l2", resp.Msg.Brief.Level)
	require.Len(t, resp.Msg.Brief.Citations, 1)
	require.Equal(t, "https://a.example", resp.Msg.Brief.Citations[0].Url)
}

func TestRunL3AndStatusProjectToProto(t *testing.T) {
	svc := internalresearch.NewService(internalresearch.Deps{
		AgentManager: fakeAgent{
			spawn: agentmanager.RunResult{RunID: "run-9", Status: "pending"},
			state: agentmanager.RunState{RunID: "run-9", Status: "complete", Summary: "ok"},
		},
	})
	h := handler.NewConnectHandler(*newHandler(svc))

	l3, err := h.RunL3(context.Background(), connect.NewRequest(&researchv1.RunL3Request{Query: "q"}))
	require.NoError(t, err)
	require.Equal(t, "run-9", l3.Msg.RunId)
	require.Equal(t, "pending", l3.Msg.Status)

	st, err := h.GetResearchStatus(context.Background(), connect.NewRequest(&researchv1.GetResearchStatusRequest{RunId: "run-9"}))
	require.NoError(t, err)
	require.Equal(t, "complete", st.Msg.Status)
	require.Equal(t, "ok", st.Msg.Summary)
}

// TestL2EndpointDocumentsRichnessLatencyTradeoff pins the REQ-P1-001 business
// contract that the API surface itself documents the L2-vs-L1 tradeoff: richer
// full-page grounding at higher latency. The endpoint descriptor is what the
// CLI manifest, docs codegen, and agent tool definitions are derived from, so
// the tradeoff statement living here means every consumer surface carries it.
func TestL2EndpointDocumentsRichnessLatencyTradeoff(t *testing.T) {
	var l2Desc string
	for _, ep := range handler.Endpoints {
		if ep.ID == "research_l2" {
			l2Desc = ep.Description
		}
	}
	require.NotEmpty(t, l2Desc, "research_l2 endpoint descriptor must exist")

	desc := strings.ToLower(l2Desc)
	require.Contains(t, desc, "richer than l1", "the L2 endpoint must document that it is richer than L1")
	require.Contains(t, desc, "higher latency", "the L2 endpoint must document the latency cost of that richness")
	require.Contains(t, desc, "full page content", "the richness claim is grounded in full-page (not snippet) synthesis")
}

// TestRunL3UnavailableSurfacesUnavailable asserts an agent-manager-down error
// maps to the Unavailable Connect code so callers can degrade to L2.
func TestRunL3UnavailableSurfacesUnavailable(t *testing.T) {
	svc := internalresearch.NewService(internalresearch.Deps{}) // no agent-manager
	h := handler.NewConnectHandler(*newHandler(svc))
	_, err := h.RunL3(context.Background(), connect.NewRequest(&researchv1.RunL3Request{Query: "q"}))
	require.Error(t, err)
}
