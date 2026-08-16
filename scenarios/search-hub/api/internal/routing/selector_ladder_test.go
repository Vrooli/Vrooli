package routing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	aisearch "github.com/vrooli/ai-go/search"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
	"search-hub/internal/routing"
)

type selectorLadderReranker struct {
	crossErr error
	llmErr   error
	active   string
}

func (r *selectorLadderReranker) Name() string { return "selector-ladder-test" }

func (r *selectorLadderReranker) Available(context.Context) bool {
	return r.crossErr == nil || r.llmErr == nil
}

func (r *selectorLadderReranker) Rerank(_ context.Context, _ string, candidates []*routingv1.SearchHit) ([]*routingv1.SearchHit, error) {
	if r.llmErr != nil {
		return nil, r.llmErr
	}
	r.active = "llm:test"
	return candidates, nil
}

func (r *selectorLadderReranker) RerankWithPreference(_ context.Context, _ string, candidates []*routingv1.SearchHit, preference string) ([]*routingv1.SearchHit, error) {
	switch preference {
	case aisearch.RerankPreferenceCrossEncoderRequired:
		if r.crossErr != nil {
			return nil, r.crossErr
		}
		r.active = "cross-encoder:test"
	default:
		if r.llmErr != nil {
			return nil, r.llmErr
		}
		r.active = "llm:test"
	}
	return candidates, nil
}

func (r *selectorLadderReranker) ActiveName(context.Context) string { return r.active }

func TestAutomaticSelectionUsesLLMLegWithoutDegrading(t *testing.T) {
	r := routing.NewRouter(routing.Deps{
		Lister: threeProviderLister(), Resolver: threeProviderResolver(), Doer: threeProviderDoer(),
		Reranker: &selectorLadderReranker{crossErr: errors.New("cross-encoder down")},
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "where is the provider type declared", Explain: true})
	require.NoError(t, err)
	require.Equal(t, "llm", resp.GetSelectorLeg())
	require.False(t, resp.GetDegraded(), "a named LLM selection leg is not degradation")
	require.Contains(t, resp.GetRoutingExplanation(), "cross-encoder provider pick unavailable; LLM provider pick selected the leading lexical candidate")
}

func TestAutomaticSelectionUsesCrossEncoderLeg(t *testing.T) {
	r := routing.NewRouter(routing.Deps{
		Lister: threeProviderLister(), Resolver: threeProviderResolver(), Doer: threeProviderDoer(),
		Reranker: &selectorLadderReranker{},
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "where is the provider type declared"})
	require.NoError(t, err)
	require.Equal(t, "cross_encoder", resp.GetSelectorLeg())
	require.False(t, resp.GetDegraded())
}

func TestAutomaticSelectionUsesLexicalLegOnlyWhenBothModelsFail(t *testing.T) {
	r := routing.NewRouter(routing.Deps{
		Lister: threeProviderLister(), Resolver: threeProviderResolver(), Doer: threeProviderDoer(),
		Reranker: &selectorLadderReranker{
			crossErr: errors.New("cross-encoder down"),
			llmErr:   errors.New("ollama down"),
		},
	})

	resp, err := r.Query(context.Background(), &routingv1.QueryRequest{Query: "where is the provider type declared", Explain: true})
	require.NoError(t, err)
	require.Equal(t, "lexical", resp.GetSelectorLeg())
	require.True(t, resp.GetDegraded())
	require.Contains(t, resp.GetRoutingExplanation(), "cross-encoder and LLM provider picks unavailable (cross-encoder: cross-encoder down; llm: ollama down); lexical provider pick selected (reranker_down)")
}
