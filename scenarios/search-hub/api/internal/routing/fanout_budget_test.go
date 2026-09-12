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
