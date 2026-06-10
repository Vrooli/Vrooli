package routing_test

import (
	"context"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"

	internalregistry "search-hub/internal/registry"
	"search-hub/internal/routing"
)

// TestClassifierWarmLatencyBudget pins web-search REQ-P2-002 (criterion
// restated 2026-06-10): classifier inference adds at most 2s p95 to the
// routing decision path, measured WARM — the model is already loaded (one
// throwaway warm-up call) and qwen3 thinking is suppressed (/no_think in
// buildClassifierPrompt). Cold-start (~10s model load) is exempt by design;
// the original auto-generated 50ms figure was never meetable for an LLM call.
//
// Live-Ollama-gated like TestClassifierRoutingRecall: skipped when the daemon
// or model is unavailable so hermetic CI stays deterministic.
func TestClassifierWarmLatencyBudget(t *testing.T) {
	if os.Getenv("SEARCH_HUB_SKIP_OLLAMA") != "" {
		t.Skip("SEARCH_HUB_SKIP_OLLAMA set")
	}
	clf := routing.NewOllamaClassifier()
	availCtx, availCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer availCancel()
	if !clf.Available(availCtx) {
		t.Skip("resource-ollama unavailable — classifier latency budget requires the local Ollama daemon + classifier model")
	}

	// A realistic multi-provider landscape: the ACTIVE corpus providers — the
	// router only lists active descriptors in the prompt, so including the
	// capability-gap stubs would double the prefill and measure a prompt
	// production never sends.
	seeds := loadProviderCorpus(t)
	profiles := make([]routing.ProviderProfile, 0, len(seeds))
	for _, d := range seeds {
		internalregistry.Normalize(d)
		if d.GetState() != registryv1.ProviderState_PROVIDER_STATE_ACTIVE {
			continue
		}
		profiles = append(profiles, routing.ProviderProfile{
			Type:        d.GetType(),
			Group:       d.GetProviderGroup(),
			Description: d.GetDescription(),
		})
	}
	require.NotEmpty(t, profiles)

	classify := func(q string) time.Duration {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		start := time.Now()
		_, err := clf.Classify(ctx, q, profiles)
		elapsed := time.Since(start)
		require.NoErrorf(t, err, "classify %q", q)
		return elapsed
	}

	// Warm-up: loads the model into memory; its latency is exempt.
	warmup := classify("warm-up query about scenario commands")
	t.Logf("warm-up (cold-start, exempt): %s", warmup)

	queries := []string{
		"how do I restart a scenario",
		"key features of Go 1.26",
		"which skill covers debugging",
		"latest news about postgres releases",
		"where are the findings stored",
	}
	latencies := make([]time.Duration, 0, len(queries))
	for _, q := range queries {
		d := classify(q)
		latencies = append(latencies, d)
		t.Logf("warm classify %-45q %s", q, d)
	}

	// With N=5, p95 is the max.
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p95 := latencies[len(latencies)-1]
	t.Logf("warm p95 = %s (budget 2s)", p95)
	require.LessOrEqualf(t, p95, 2*time.Second,
		"classifier warm p95 %s exceeds the 2s budget (REQ-P2-002 restated criterion)", p95)
}
