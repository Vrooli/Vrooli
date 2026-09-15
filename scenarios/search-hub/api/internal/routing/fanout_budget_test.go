package routing

import (
	"testing"
	"time"
)

func TestDefaultFanoutBudgetFitsActiveFleet(t *testing.T) {
	// The honest baseline counted 34 active leaves. This is intentionally a
	// fleet-sized invariant rather than a two-provider unit fixture.
	const activeProviders = 34
	worst := fanoutWorstCase(activeProviders, defaultConcurrency, defaultPerProviderTimeout)
	if worst >= defaultQueryTimeout {
		t.Fatalf("fan-out budget exceeds query budget: %s >= %s (providers=%d concurrency=%d per-provider=%s)", worst, defaultQueryTimeout, activeProviders, defaultConcurrency, defaultPerProviderTimeout)
	}
	if got, want := worst, 20*time.Second; got != want {
		t.Fatalf("default fleet budget = %s, want %s", got, want)
	}
}

func TestFanoutWorstCaseUsesConcurrencyWaves(t *testing.T) {
	if got, want := fanoutWorstCase(9, 4, 2*time.Second), 6*time.Second; got != want {
		t.Fatalf("worst case = %s, want %s", got, want)
	}
}

// TestDefaultCrossEncoderBudgetCoversMeasuredLoadedLatency guards the
// re-measured cross-encoder bound. On 2026-09-12 the TEI reranker served
// 32x700-char docs to 32 concurrent callers at p95 ~1.73s (32x120-char p95
// ~0.38s), so the previous 500ms bound produced false timeouts and opened the
// 60s rerank breaker under concurrent host load (e.g. the eval scheduler).
func TestDefaultCrossEncoderBudgetCoversMeasuredLoadedLatency(t *testing.T) {
	if defaultCrossEncoderRerankTimeout < 2*time.Second {
		t.Fatalf("cross-encoder budget %s is below the measured loaded latency (2s)", defaultCrossEncoderRerankTimeout)
	}
	if defaultCrossEncoderRerankTimeout >= defaultLLMRerankTimeout {
		t.Fatalf("cross-encoder budget %s must stay below the LLM fallback budget %s", defaultCrossEncoderRerankTimeout, defaultLLMRerankTimeout)
	}
	if defaultCrossEncoderRerankTimeout+defaultResponseCushion >= defaultQueryTimeout {
		t.Fatalf("cross-encoder budget %s + response cushion %s must fit the query budget %s", defaultCrossEncoderRerankTimeout, defaultResponseCushion, defaultQueryTimeout)
	}
}
