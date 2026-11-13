package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestCPULexicalRerankerUsesBoundedDeterministicRanking(t *testing.T) {
	rr := NewCPULexicalReranker()
	candidates := []*routingv1.SearchHit{
		{Id: "unrelated", Title: "Audio playback", Snippet: "play music", Score: .99},
		{Id: "relevant", Title: "Restart API service", Snippet: "restart the api with vrooli scenario restart", Score: .1},
		{Id: "tie", Title: "Restart service", Snippet: "restart service", Score: .2},
	}

	ranked, err := rr.Rerank(context.Background(), "restart api", candidates)
	require.NoError(t, err)
	require.Equal(t, []string{"relevant", "tie", "unrelated"}, ids(ranked))
	require.Greater(t, ranked[0].GetRerankScore(), ranked[1].GetRerankScore())
	require.Equal(t, cpuLexicalRerankerName, rr.Name())
	require.True(t, rr.Available(context.Background()))
}

func TestCPULexicalRerankerRejectsUnboundedInputsAndCanceledBudget(t *testing.T) {
	rr := NewCPULexicalReranker()
	tooMany := make([]*routingv1.SearchHit, CPULexicalCandidateLimit+1)
	for i := range tooMany {
		tooMany[i] = &routingv1.SearchHit{Id: "candidate"}
	}
	_, err := rr.Rerank(context.Background(), "query", tooMany)
	require.ErrorContains(t, err, "candidate count")

	_, err = rr.Rerank(context.Background(), "", []*routingv1.SearchHit{{Id: "candidate"}})
	require.ErrorContains(t, err, "query must contain terms")

	longText := make([]byte, maxCPULexicalTextBytes+1)
	for i := range longText {
		longText[i] = 'x'
	}
	_, err = rr.Rerank(context.Background(), "query", []*routingv1.SearchHit{{Id: "candidate", Path: string(longText)}})
	require.ErrorContains(t, err, "exceeds")

	rr.gate <- struct{}{}
	defer func() { <-rr.gate }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = rr.Rerank(ctx, "query", []*routingv1.SearchHit{{Id: "candidate"}})
	require.ErrorIs(t, err, context.Canceled)
}

func TestCPULexicalRerankerInstancesShareProcessBudget(t *testing.T) {
	first, second := NewCPULexicalReranker(), NewCPULexicalReranker()
	require.Equal(t, first.gate, second.gate)
	first.gate <- struct{}{}
	defer func() { <-first.gate }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := second.Rerank(ctx, "query", []*routingv1.SearchHit{{Id: "candidate"}})
	require.ErrorIs(t, err, context.Canceled)
}

func TestSharedRerankerUsesCPUOnlyAfterPreferredModelFailure(t *testing.T) {
	cross := &stubSharedReranker{
		name:      "cross-encoder:test",
		available: true,
		err:       errors.New("CPU backend disabled or model unavailable"),
		onRerank:  func(s *stubSharedReranker) { s.available = false },
	}
	llm := &stubSharedReranker{name: "llm:test", available: true}
	rr := NewSharedReranker(aisearch.NewRerankerChain(cross, llm))
	rr.cpuFallback = NewCPULexicalReranker()

	ranked, err := rr.RerankWithPreference(context.Background(), "restart api", []*routingv1.SearchHit{
		{Id: "irrelevant", Title: "Audio playback"},
		{Id: "api", Title: "Restart API service"},
	}, aisearch.RerankPreferenceCrossEncoderPreferred)
	require.NoError(t, err)
	require.Equal(t, []string{"api", "irrelevant"}, ids(ranked))
	require.False(t, llm.called, "the persisted CPU degraded route must not load/use the generative fallback")
	require.Equal(t, cpuLexicalRerankerName, rr.ActiveNameWithPreference(context.Background(), aisearch.RerankPreferenceCrossEncoderPreferred))
	require.Equal(t, "none", rr.ActiveNameWithPreference(context.Background(), aisearch.RerankPreferenceCrossEncoderRequired))

	// A later request after the cross-encoder has become unavailable must keep
	// following the same persisted CPU route rather than promoting the LLM leg.
	_, err = rr.RerankWithPreference(context.Background(), "restart api", []*routingv1.SearchHit{
		{Id: "irrelevant", Title: "Audio playback"},
		{Id: "api", Title: "Restart API service"},
	}, aisearch.RerankPreferenceCrossEncoderPreferred)
	require.NoError(t, err)
	require.False(t, llm.called, "an unavailable cross-encoder must fall directly to CPU when policy names CPU")
}

func TestSharedRerankerDoesNotViolateRequiredCrossEncoderPolicy(t *testing.T) {
	cross := &stubSharedReranker{
		name:      "cross-encoder:test",
		available: true,
		err:       errors.New("cross encoder failed"),
		onRerank:  func(s *stubSharedReranker) { s.available = false },
	}
	rr := NewSharedReranker(aisearch.NewRerankerChain(cross))
	rr.cpuFallback = NewCPULexicalReranker()

	_, err := rr.RerankWithPreference(context.Background(), "restart api", []*routingv1.SearchHit{
		{Id: "api", Title: "Restart API service"},
		{Id: "other", Title: "Audio playback"},
	}, aisearch.RerankPreferenceCrossEncoderRequired)
	require.ErrorContains(t, err, "cross encoder failed")
}

func TestDefaultRerankerCPUFallbackFollowsPersistedStrategy(t *testing.T) {
	active, _, err := LoadActiveStrategy()
	require.NoError(t, err)
	require.True(t, cpuLexicalFallbackEnabled(active))
	require.NotNil(t, NewDefaultRerankerChain(active).cpuFallback)

	catalog, err := LoadStrategyCatalog()
	require.NoError(t, err)
	var lexicalFallback *RetrievalStrategy
	for i := range catalog {
		if catalog[i].Name == "lexical-fallback" {
			lexicalFallback = &catalog[i]
			break
		}
	}
	require.NotNil(t, lexicalFallback)
	require.False(t, cpuLexicalFallbackEnabled(*lexicalFallback))
	require.Nil(t, NewDefaultRerankerChain(*lexicalFallback).cpuFallback)
}

func TestRouterReportsCPURerankerAsExplicitDegradation(t *testing.T) {
	cross := &stubSharedReranker{name: "cross-encoder:test", available: false, err: errors.New("TEI CPU runtime unavailable")}
	rr := NewSharedReranker(aisearch.NewRerankerChain(cross))
	rr.cpuFallback = NewCPULexicalReranker()
	router := NewRouter(Deps{Reranker: rr})
	groups := []*routingv1.ProviderResultGroup{
		{Hits: []*routingv1.SearchHit{{Id: "other", Title: "Audio playback"}}},
		{Hits: []*routingv1.SearchHit{{Id: "match", Title: "Restart API service"}}},
	}

	ranked, reranked, degraded, reason, leg, explanation := router.maybeRerank(context.Background(), "restart api", groups, nil)
	require.True(t, reranked)
	require.True(t, degraded)
	require.Equal(t, "reranker_degraded_to_cpu_lexical", reason)
	require.Equal(t, cpuLexicalRerankerName, leg)
	require.Equal(t, []string{"match", "other"}, ids(ranked))
	require.Contains(t, explanation[0], "via cpu-lexical:v1")
}
