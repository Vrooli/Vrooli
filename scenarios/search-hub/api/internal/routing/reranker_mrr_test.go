package routing_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"

	"search-hub/internal/routing"
)

// rerankFixture mirrors testdata/rerank_queries.json.
type rerankFixture struct {
	Cases []struct {
		Query      string `json:"query"`
		Candidates []struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
		} `json:"candidates"`
		ExpectedID string `json:"expected_id"`
	} `json:"cases"`
}

// TestRerankerOrderingMRR is plan §6 #2: it runs the REAL local-Ollama reranker
// (qwen3:4b) over a labeled fixture whose relevant candidate is deliberately
// placed LAST, then asserts the reranked Mean Reciprocal Rank clears a floor and
// reports reranked-vs-baseline MRR so the rerank cost is justified with numbers.
// It is skipped when the Ollama daemon is unavailable, so the deterministic
// applyRerank/parse unit tests remain the always-on guarantee.
func TestRerankerOrderingMRR(t *testing.T) {
	if os.Getenv("SEARCH_HUB_SKIP_OLLAMA") != "" {
		t.Skip("SEARCH_HUB_SKIP_OLLAMA set")
	}
	rr := routing.NewOllamaReranker()
	availCtx, availCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer availCancel()
	if !rr.Available(availCtx) {
		t.Skip("resource-ollama unavailable — rerank MRR gate requires the local Ollama daemon + reranker model")
	}

	fx := loadRerankFixture(t)

	var sumRerankRR, sumBaselineRR float64
	for _, c := range fx.Cases {
		candidates := make([]*routingv1.SearchHit, len(c.Candidates))
		baselineRank := 0
		for i, cand := range c.Candidates {
			candidates[i] = &routingv1.SearchHit{
				Id: cand.ID, Type: cand.Type, Title: cand.Title, Snippet: cand.Snippet,
			}
			if cand.ID == c.ExpectedID {
				baselineRank = i + 1 // pre-rerank order = input order
			}
		}
		require.Positivef(t, baselineRank, "expected_id %q not among candidates", c.ExpectedID)

		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		ranked, err := rr.Rerank(ctx, c.Query, candidates)
		cancel()
		require.NoErrorf(t, err, "rerank %q", c.Query)
		require.Len(t, ranked, len(candidates))

		rerankRank := rankOf(ranked, c.ExpectedID)
		require.Positive(t, rerankRank)
		sumRerankRR += 1.0 / float64(rerankRank)
		sumBaselineRR += 1.0 / float64(baselineRank)
		t.Logf("%-55q expected rank: baseline=%d rerank=%d", c.Query, baselineRank, rerankRank)
	}

	n := float64(len(fx.Cases))
	rerankMRR := sumRerankRR / n
	baselineMRR := sumBaselineRR / n
	t.Logf("MRR: rerank=%.3f  baseline=%.3f  (Δ=%+.3f over %d cases)", rerankMRR, baselineMRR, rerankMRR-baselineMRR, len(fx.Cases))

	// The reranker must beat the baseline ordering (the whole point of the cost)
	// and clear a top-2-ish floor. The fixture deliberately buries the relevant
	// hit last so the baseline MRR is low and a real win is observable.
	require.Greaterf(t, rerankMRR, baselineMRR, "rerank MRR %.3f must beat baseline %.3f", rerankMRR, baselineMRR)
	require.GreaterOrEqualf(t, rerankMRR, 0.6, "rerank MRR %.3f below the 0.6 gate", rerankMRR)
}

func rankOf(hits []*routingv1.SearchHit, id string) int {
	for i, h := range hits {
		if h.GetId() == id {
			return i + 1
		}
	}
	return 0
}

func loadRerankFixture(t *testing.T) rerankFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/rerank_queries.json")
	require.NoError(t, err)
	var fx rerankFixture
	require.NoError(t, json.Unmarshal(raw, &fx))
	require.NotEmpty(t, fx.Cases)
	return fx
}
