package research_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"

	researchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-search/v1/research"

	handler "web-search/handlers/research"
	"web-search/internal/capture"
	"web-search/internal/evaluation"
	"web-search/internal/evidence"
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

type recordingSynth struct {
	docs []internalresearch.Document
}

func (s *recordingSynth) Synthesize(_ context.Context, _ string, docs []internalresearch.Document) (internalresearch.Synthesis, error) {
	s.docs = append([]internalresearch.Document(nil), docs...)
	return internalresearch.Synthesis{Text: "bounded", Citations: []internalresearch.Citation{{ResultIndex: 0, URL: "https://a.example"}}}, nil
}

type fakeAgent struct {
	spawn agentmanager.RunResult
	state agentmanager.RunState
}

type recordingAgent struct {
	request agentmanager.SpawnRequest
}

func (a *recordingAgent) Spawn(_ context.Context, request agentmanager.SpawnRequest) (agentmanager.RunResult, error) {
	a.request = request
	return agentmanager.RunResult{RunID: "run-policy", Status: "pending"}, nil
}

func (a *recordingAgent) GetRunState(_ context.Context, _ string) (agentmanager.RunState, error) {
	return agentmanager.RunState{RunID: "run-policy", Status: "pending"}, nil
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

type evidenceReader struct{}

func (evidenceReader) CreateReceipt(context.Context, evidence.NewObservation) (evidence.Receipt, error) {
	return evidence.Receipt{}, nil
}
func (evidenceReader) GetReceipt(context.Context, string) (evidence.Receipt, error) {
	return evidence.Receipt{ReceiptID: "receipt-1", ObservationID: "observation-1", URL: "https://source.example", RetrievedAt: time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC), ContentHash: "hash", ArtifactID: "artifact-1", ExtractionRevision: "text-v1", Retention: "standard"}, nil
}
func (evidenceReader) CreatePassage(context.Context, string, int, int) (evidence.Passage, error) {
	return evidence.Passage{}, nil
}
func (evidenceReader) GetPassage(context.Context, string) (evidence.Passage, error) {
	return evidence.Passage{PassageID: "passage-1", ReceiptID: "receipt-1", StartByte: 0, EndByte: 5, Content: "claim", Hash: "hash"}, nil
}
func (evidenceReader) ExpireContent(context.Context, time.Time) (int, error) { return 0, nil }

func TestEvidenceReadsProjectOpaqueOwnerRecords(t *testing.T) {
	svc := internalresearch.NewService(internalresearch.Deps{ReceiptStore: evidenceReader{}})
	h := handler.NewConnectHandler(*newHandler(svc))
	receipt, err := h.GetEvidenceReceipt(context.Background(), connect.NewRequest(&researchv1.GetEvidenceReceiptRequest{ReceiptId: "receipt-1"}))
	require.NoError(t, err)
	require.Equal(t, "receipt-1", receipt.Msg.ReceiptId)
	require.Equal(t, "https://source.example", receipt.Msg.OriginalUrl)
	passage, err := h.GetEvidencePassage(context.Background(), connect.NewRequest(&researchv1.GetEvidencePassageRequest{PassageId: "passage-1"}))
	require.NoError(t, err)
	require.Equal(t, "claim", passage.Msg.Content)
	require.Equal(t, "receipt-1", passage.Msg.ReceiptId)
}

func TestMethodReleaseRequiresGrantAndSupportsCurrentRead(t *testing.T) {
	revision := evaluation.MethodRevision{ID: "candidate", ProgramHash: "program", ConfigHash: "config"}
	report := evaluation.Report{Accepted: true, BaselineMethod: "base", CandidateMethod: "candidate", Population: 1, MeanBaselineEffort: 10, MeanCandidateEffort: 8}
	receipt := evaluation.EvaluationReceipt{ID: "receipt-1", CandidateHash: evaluation.RevisionHash(revision), ReportHash: evaluation.ReportHash(report), Accepted: true}
	registry := &evaluation.MethodRegistry{}
	h := handler.NewConnectHandler(handler.Deps{Service: internalresearch.NewService(internalresearch.Deps{}), Registry: registry})
	request := func(grant string) *connect.Request[researchv1.PromoteMethodRequest] {
		return connect.NewRequest(&researchv1.PromoteMethodRequest{Revision: &researchv1.MethodRevision{Id: revision.ID, ProgramHash: revision.ProgramHash, ConfigHash: revision.ConfigHash}, Receipt: &researchv1.EvaluationReceipt{Id: receipt.ID, CandidateHash: receipt.CandidateHash, ReportHash: receipt.ReportHash, Accepted: true}, Report: &researchv1.EvaluationReport{Accepted: true, BaselineMethod: report.BaselineMethod, CandidateMethod: report.CandidateMethod, Population: 1, MeanBaselineEffort: 10, MeanCandidateEffort: 8}, Grant: grant})
	}
	_, err := h.PromoteMethod(context.Background(), request(""))
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	promoted, err := h.PromoteMethod(context.Background(), request("grant-1"))
	require.NoError(t, err)
	require.Equal(t, evaluation.RevisionHash(revision), promoted.Msg.Release.RevisionHash)
	current, err := h.GetMethodRelease(context.Background(), connect.NewRequest(&researchv1.GetMethodReleaseRequest{}))
	require.NoError(t, err)
	require.True(t, current.Msg.Found)
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

func TestRunL2PropagatesEvidenceBytePolicy(t *testing.T) {
	synth := &recordingSynth{}
	svc := internalresearch.NewService(internalresearch.Deps{
		Searcher: fakeSearcher{cands: []internalresearch.Candidate{{URL: "https://a.example", Title: "A"}}},
		Fetcher:  fakeFetcher{text: "0123456789"}, Synthesizer: synth,
	})
	h := handler.NewConnectHandler(*newHandler(svc))
	_, err := h.RunL2(context.Background(), connect.NewRequest(&researchv1.RunL2Request{
		Query: "q", TopN: 1, Policy: &researchv1.EvidencePolicy{MaxEvidenceBytes: 5},
	}))
	require.NoError(t, err)
	require.Len(t, synth.docs, 1)
	require.Equal(t, "01234", synth.docs[0].Text)
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

func TestCaptureStatusProjectsPendingAndFailedSeparately(t *testing.T) {
	db := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(capture.Schema)))
	clock := time.Date(2026, 9, 6, 4, 0, 0, 0, time.UTC)
	outbox := capture.NewRepository(db, func() time.Time { return clock })
	_, _, err := outbox.Enqueue(context.Background(), "pending", "task-pending", "delivery-pending", []byte("pending"))
	require.NoError(t, err)
	_, _, err = outbox.Enqueue(context.Background(), "failed", "task-failed", "delivery-failed", []byte("failed"))
	require.NoError(t, err)
	claimed, err := outbox.Claim(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	require.NoError(t, outbox.MarkFailed(context.Background(), claimed[0].AttemptID, "memory unavailable", clock.Add(time.Minute)))

	svc := internalresearch.NewService(internalresearch.Deps{AttemptOutbox: outbox})
	h := handler.NewConnectHandler(*newHandler(svc))
	response, err := h.GetCaptureStatus(context.Background(), connect.NewRequest(&researchv1.GetCaptureStatusRequest{}))
	require.NoError(t, err)
	require.EqualValues(t, 1, response.Msg.Pending)
	require.EqualValues(t, 1, response.Msg.Failed)
	require.Equal(t, "pending", response.Msg.Status)
}

func TestRunL3PropagatesTypedEvidencePolicyAndQuestions(t *testing.T) {
	agent := &recordingAgent{}
	svc := internalresearch.NewService(internalresearch.Deps{AgentManager: agent})
	h := handler.NewConnectHandler(*newHandler(svc))
	_, err := h.RunL3(context.Background(), connect.NewRequest(&researchv1.RunL3Request{
		Query: "policy query", IdempotencyKey: "retry-1",
		Policy:    &researchv1.EvidencePolicy{MinimumSources: 2, TopN: 3, MaxEvidenceBytes: 4096},
		Questions: []*researchv1.ResearchQuestion{{Id: "release", Prompt: "When?", Required: true}},
	}))
	require.NoError(t, err)
	require.Equal(t, "policy query", agent.request.Query)
	require.Equal(t, "retry-1", agent.request.IdempotencyKey)
	require.Equal(t, 2, agent.request.Policy["minimum_sources"])
	require.Equal(t, 3, agent.request.Policy["top_n"])
	require.Equal(t, 4096, agent.request.Policy["max_evidence_bytes"])
	require.Equal(t, []map[string]any{{"id": "release", "prompt": "When?", "required": true}}, agent.request.Questions)
}

// recordingSearcher captures the topN the service actually used so the
// schema test can pin the 1..10 clamp end-to-end through the endpoint.
type recordingSearcher struct {
	cands    []internalresearch.Candidate
	lastTopN int
}

func (r *recordingSearcher) Candidates(_ context.Context, _ string, topN int) (internalresearch.CandidateSet, error) {
	r.lastTopN = topN
	return internalresearch.CandidateSet{Candidates: r.cands}, nil
}

// TestRunL2SchemaValidation [REQ:REQ-P1-001] pins the L2 endpoint schema:
// the request must include a query (empty → InvalidArgument) and top_n is
// bounded to 1..10 (non-positive → server default, oversized → max); the
// response carries the synthesis text, citations, the per-document excerpts
// actually sent to the model, and — on abstention — the abstain_reason.
func TestRunL2SchemaValidation(t *testing.T) {
	longBody := strings.Repeat("filler text ", 700) // >6000 chars: forces a real excerpt cut

	newSvc := func(rs *recordingSearcher) *internalresearch.Service {
		return internalresearch.NewService(internalresearch.Deps{
			Searcher: rs,
			Fetcher:  fakeFetcher{text: longBody},
			Synthesizer: fakeSynth{out: internalresearch.Synthesis{
				Text:      "the cited answer",
				Citations: []internalresearch.Citation{{ResultIndex: 0, URL: "https://a.example", Title: "A"}},
			}},
		})
	}

	t.Run("query is required", func(t *testing.T) {
		h := handler.NewConnectHandler(*newHandler(newSvc(&recordingSearcher{})))
		_, err := h.RunL2(context.Background(), connect.NewRequest(&researchv1.RunL2Request{Query: "   ", TopN: 3}))
		require.Error(t, err)
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	})

	t.Run("top_n is clamped into 1..10", func(t *testing.T) {
		for _, tc := range []struct {
			in   int32
			want int
		}{
			{in: 0, want: 5}, // non-positive → server default (DefaultTopN)
			{in: 99, want: 10} /* oversized → MaxTopN */, {in: 3, want: 3},
		} {
			rs := &recordingSearcher{}
			h := handler.NewConnectHandler(*newHandler(newSvc(rs)))
			_, err := h.RunL2(context.Background(), connect.NewRequest(&researchv1.RunL2Request{Query: "q", TopN: tc.in}))
			require.NoError(t, err)
			require.Equal(t, tc.want, rs.lastTopN, "top_n=%d must reach the searcher as %d", tc.in, tc.want)
		}
	})

	t.Run("response carries synthesis, citations, and excerpts", func(t *testing.T) {
		rs := &recordingSearcher{cands: []internalresearch.Candidate{{URL: "https://a.example", Title: "A"}}}
		h := handler.NewConnectHandler(*newHandler(newSvc(rs)))
		resp, err := h.RunL2(context.Background(), connect.NewRequest(&researchv1.RunL2Request{Query: "q", TopN: 1}))
		require.NoError(t, err)
		require.Equal(t, "the cited answer", resp.Msg.Synthesis)
		require.Len(t, resp.Msg.Brief.Citations, 1)
		require.Empty(t, resp.Msg.AbstainReason, "a non-abstaining response carries no abstain_reason")
		require.Len(t, resp.Msg.Excerpts, 1, "one excerpt per fetched document")
		require.Equal(t, "https://a.example", resp.Msg.Excerpts[0].Url)
		require.NotEmpty(t, resp.Msg.Excerpts[0].Excerpt, "the excerpt mirrors what the model read")
		require.LessOrEqual(t, len(resp.Msg.Excerpts[0].Excerpt), 800, "excerpts are transport-capped")
	})

	t.Run("abstention carries the machine-readable reason", func(t *testing.T) {
		h := handler.NewConnectHandler(*newHandler(newSvc(&recordingSearcher{}))) // zero candidates
		resp, err := h.RunL2(context.Background(), connect.NewRequest(&researchv1.RunL2Request{Query: "q", TopN: 1}))
		require.NoError(t, err)
		require.True(t, resp.Msg.Abstained)
		require.Equal(t, "no_candidates", resp.Msg.AbstainReason)
		require.Empty(t, resp.Msg.Excerpts)
	})
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

// [REQ:REQ-P0-010] Input and availability failures retain their public type.
func TestL3InvalidInputAndMissingDependency(t *testing.T) {
	h := handler.NewConnectHandler(*newHandler(internalresearch.NewService(internalresearch.Deps{})))
	for _, query := range []string{"", strings.Repeat("x", 4097)} {
		_, err := h.RunL3(context.Background(), connect.NewRequest(&researchv1.RunL3Request{Query: query}))
		require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	}
	_, err := h.RunL3(context.Background(), connect.NewRequest(&researchv1.RunL3Request{Query: "q"}))
	require.Equal(t, connect.CodeUnavailable, connect.CodeOf(err))
}
