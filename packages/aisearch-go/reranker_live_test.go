package aisearch

import (
	"context"
	"errors"
	"net/http"
	"os"
	"testing"
	"time"
)

// TestLiveLLMRerankerFallback proves the degradation chain's second leg works
// against a real Ollama: with the cross-encoder pointed at a dead address, the
// chain must select the qwen3:4b LLM reranker and produce a usable ordering.
//
// Gated on KO_AISEARCH_LIVE (needs the ollama resource with qwen3:4b pulled):
//
//	KO_AISEARCH_LIVE=1 go test . -run TestLiveLLMRerankerFallback -v
func TestLiveLLMRerankerFallback(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 to run the live LLM-reranker fallback proof")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Cross-encoder at a dead address -> unavailable; chain must fall to LLM.
	dead := NewCrossEncoderRerankerWithClient("http://127.0.0.1:1", &deadClient{})
	llm := NewLLMReranker("")
	chain := NewRerankerChain(dead, llm)

	if !llm.Available(ctx) {
		t.Skip("qwen3:4b not available via resource-ollama; skipping live fallback proof")
	}
	if got := chain.ActiveName(ctx); got != llm.Name() {
		t.Fatalf("active leg = %q, want the LLM reranker %q", got, llm.Name())
	}

	cands := []RerankCandidate{
		{ID: "noise", Text: "A recipe for sourdough bread starter and proofing schedules."},
		{ID: "match", Text: "Restart a running scenario using the lifecycle: make stop then make start."},
	}
	scores, err := chain.Rerank(ctx, "how do I restart a scenario", cands)
	if err != nil {
		t.Fatalf("live LLM rerank: %v", err)
	}
	var matchScore, noiseScore float64
	var haveMatch, haveNoise bool
	for _, s := range scores {
		switch s.ID {
		case "match":
			matchScore, haveMatch = s.Score, true
		case "noise":
			noiseScore, haveNoise = s.Score, true
		}
	}
	t.Logf("LLM reranker scores: match=%.3f noise=%.3f", matchScore, noiseScore)
	if !haveMatch || !haveNoise {
		t.Fatalf("expected scores for both candidates, got %+v", scores)
	}
	if matchScore <= noiseScore {
		t.Errorf("LLM reranker ranked the relevant passage no higher than noise (match=%.3f noise=%.3f)", matchScore, noiseScore)
	}
}

// deadClient is an httpDoer that always fails, simulating a down resource.
type deadClient struct{}

var errDead = errors.New("dead reranker")

func (deadClient) Do(*http.Request) (*http.Response, error) {
	return nil, errDead
}
