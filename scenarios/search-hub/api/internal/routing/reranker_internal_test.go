package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// --- parsing ---------------------------------------------------------------

func TestParseRerankResponse_StrictJSON(t *testing.T) {
	scores, err := parseRerankResponse([]byte(`{"scores":[{"index":0,"score":3},{"index":1,"score":9}]}`), 2)
	require.NoError(t, err)
	require.InDelta(t, 0.3, scores[0], 1e-9)
	require.InDelta(t, 0.9, scores[1], 1e-9)
}

func TestParseRerankResponse_GatewayEnvelopeWithThink(t *testing.T) {
	// The shape resource-ollama gateway generate --json returns for qwen3 with
	// /no_think: an outer {"response":…} wrapping an empty <think> block + JSON.
	raw := []byte(`{"response":"<think>\n\n</think>\n\n{\"scores\":[{\"index\":0,\"score\":10},{\"index\":1,\"score\":2}]}","eval_count":30}`)
	scores, err := parseRerankResponse(raw, 2)
	require.NoError(t, err)
	require.InDelta(t, 1.0, scores[0], 1e-9)
	require.InDelta(t, 0.2, scores[1], 1e-9)
}

func TestParseRerankResponse_OmittedIndexDefaultsToZero(t *testing.T) {
	// The model scored only index 1; index 0 and 2 default to 0 (ranked last).
	scores, err := parseRerankResponse([]byte(`{"scores":[{"index":1,"score":7}]}`), 3)
	require.NoError(t, err)
	require.Equal(t, 0.0, scores[0])
	require.InDelta(t, 0.7, scores[1], 1e-9)
	require.Equal(t, 0.0, scores[2])
}

func TestParseRerankResponse_OutOfRangeIndexDropped(t *testing.T) {
	scores, err := parseRerankResponse([]byte(`{"scores":[{"index":0,"score":5},{"index":9,"score":10}]}`), 1)
	require.NoError(t, err)
	require.Len(t, scores, 1)
	require.InDelta(t, 0.5, scores[0], 1e-9)
}

func TestParseRerankResponse_SalvagesReversedFieldOrder(t *testing.T) {
	// Field order swapped + prose around it: strict decode still works here, so
	// force the salvage path with a trailing comma that breaks strict JSON.
	raw := []byte(`here you go: {"scores":[{"score":8,"index":0,},{"score":2,"index":1,},]}`)
	scores, err := parseRerankResponse(raw, 2)
	require.NoError(t, err)
	require.InDelta(t, 0.8, scores[0], 1e-9, "salvage reads score regardless of field order")
	require.InDelta(t, 0.2, scores[1], 1e-9)
}

func TestParseRerankResponse_NoJSON(t *testing.T) {
	_, err := parseRerankResponse([]byte(`{"response":"I cannot rank these."}`), 3)
	require.Error(t, err)
}

// --- score normalization + ordering ----------------------------------------

func TestNormalizeRerankScore_Clamps(t *testing.T) {
	require.Equal(t, 0.0, normalizeRerankScore(-4))
	require.Equal(t, 1.0, normalizeRerankScore(12))
	require.InDelta(t, 0.55, normalizeRerankScore(5.5), 1e-9)
}

func TestApplyRerank_SortsDescendingStable(t *testing.T) {
	candidates := []*routingv1.SearchHit{
		{Id: "a"}, {Id: "b"}, {Id: "c"}, {Id: "d"},
	}
	// b and d tie at 0.5 — stable sort keeps their input order (b before d).
	ranked := applyRerank(candidates, []float64{0.1, 0.5, 0.9, 0.5})
	require.Equal(t, []string{"c", "b", "d", "a"}, ids(ranked))
	require.InDelta(t, 0.9, ranked[0].GetRerankScore(), 1e-9, "rerank_score is set on each hit")
}

func TestApplyRerank_BreaksTiesByOriginalScore(t *testing.T) {
	// A flat reranker scores every candidate 5/10 (→ 0.5). The tie-break must
	// fall back to the original per-provider score so the providers' retrieval
	// signal is preserved instead of arbitrary fan-out order.
	candidates := []*routingv1.SearchHit{
		{Id: "low", Score: 0.30},
		{Id: "high", Score: 0.90},
		{Id: "mid", Score: 0.60},
	}
	ranked := applyRerank(candidates, []float64{0.5, 0.5, 0.5})
	require.Equal(t, []string{"high", "mid", "low"}, ids(ranked),
		"equal rerank scores break ties by original score, descending")
}

func TestFuseGroups_FlattensAllHits(t *testing.T) {
	groups := []*routingv1.ProviderResultGroup{
		{ProviderId: "p1", Hits: []*routingv1.SearchHit{{Id: "a"}, {Id: "b"}}},
		{ProviderId: "p2", Degraded: true}, // no hits ⇒ contributes nothing
		{ProviderId: "p3", Hits: []*routingv1.SearchHit{{Id: "c"}}},
	}
	require.Equal(t, []string{"a", "b", "c"}, ids(fuseGroups(groups)))
}

// --- OllamaReranker with a seamed runner -----------------------------------

func TestOllamaReranker_Rerank_UsesRunnerOutputAndOrders(t *testing.T) {
	var gotRole, gotPrompt string
	c := &OllamaReranker{
		role:      "rerank.llm_fallback",
		maxTokens: rerankerMaxTokens,
		generate: func(_ context.Context, role, prompt string, _ int) ([]byte, error) {
			gotRole, gotPrompt = role, prompt
			// Score the second candidate highest so it must rise to #1.
			return []byte(`{"response":"{\"scores\":[{\"index\":0,\"score\":2},{\"index\":1,\"score\":9}]}"}`), nil
		},
	}
	candidates := []*routingv1.SearchHit{
		{Id: "first", Title: "Restart help", Type: "doc", Snippet: "how to restart"},
		{Id: "second", Title: "scenario restart", Type: "command"},
	}
	ranked, err := c.Rerank(context.Background(), "restart a scenario", candidates)
	require.NoError(t, err)
	require.Equal(t, []string{"second", "first"}, ids(ranked), "the higher-scored candidate ranks first")
	require.Equal(t, "rerank.llm_fallback", gotRole)
	require.Contains(t, gotPrompt, "restart a scenario")
	require.Contains(t, gotPrompt, "scenario restart", "candidate titles reach the prompt")
	require.Contains(t, gotPrompt, "[0]", "candidates are indexed in the prompt")
}

func TestOllamaReranker_Rerank_RunnerErrorPropagates(t *testing.T) {
	c := &OllamaReranker{
		generate: func(context.Context, string, string, int) ([]byte, error) {
			return nil, errors.New("daemon down")
		},
	}
	_, err := c.Rerank(context.Background(), "q", []*routingv1.SearchHit{{Id: "a"}})
	require.ErrorContains(t, err, "daemon down")
}

func TestOllamaReranker_Rerank_EmptyInputs(t *testing.T) {
	c := &OllamaReranker{generate: func(context.Context, string, string, int) ([]byte, error) { return nil, nil }}
	_, err := c.Rerank(context.Background(), "  ", []*routingv1.SearchHit{{Id: "a"}})
	require.Error(t, err)
	_, err = c.Rerank(context.Background(), "q", nil)
	require.Error(t, err)
}

// --- helpers ---------------------------------------------------------------

func ids(hits []*routingv1.SearchHit) []string {
	out := make([]string, len(hits))
	for i, h := range hits {
		out[i] = h.GetId()
	}
	return out
}
