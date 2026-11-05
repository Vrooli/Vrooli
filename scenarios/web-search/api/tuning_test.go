package main

import (
	"testing"
	"time"
)

// The control surface must never stop boot: malformed or out-of-range values
// fall back to zero ("use the compiled default"), valid values pass through.
func TestTuningFromEnv(t *testing.T) {
	t.Run("unset means compiled defaults", func(t *testing.T) {
		got := tuningFromEnv()
		if got != (Tuning{}) {
			t.Fatalf("expected zero Tuning with no env set, got %+v", got)
		}
	})

	t.Run("valid overrides pass through", func(t *testing.T) {
		t.Setenv("WEB_SEARCH_HIGH_CONFIDENCE_THRESHOLD", "0.9")
		t.Setenv("WEB_SEARCH_DECAY_HALF_LIFE", "2160h")
		t.Setenv("WEB_SEARCH_MAX_GATHER_FINDINGS", "10")
		t.Setenv("WEB_SEARCH_L3_MAX_LOOPS", "6")
		t.Setenv("WEB_SEARCH_GOVERNOR_CAPACITY", "120")
		t.Setenv("WEB_SEARCH_CACHE_TTL", "30s")
		got := tuningFromEnv()
		want := Tuning{
			ConfidenceGate:   0.9,
			DecayHalfLife:    2160 * time.Hour,
			GatherCap:        10,
			MaxResearchLoops: 6,
			GovernorCapacity: 120,
			CacheTTL:         30 * time.Second,
		}
		if got != want {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("malformed and non-positive values are ignored", func(t *testing.T) {
		t.Setenv("WEB_SEARCH_HIGH_CONFIDENCE_THRESHOLD", "not-a-number")
		t.Setenv("WEB_SEARCH_DECAY_HALF_LIFE", "-24h")
		t.Setenv("WEB_SEARCH_MAX_GATHER_FINDINGS", "0")
		t.Setenv("WEB_SEARCH_GOVERNOR_CAPACITY", "-1")
		t.Setenv("WEB_SEARCH_CACHE_TTL", "five minutes")
		got := tuningFromEnv()
		if got != (Tuning{}) {
			t.Fatalf("expected zero Tuning for malformed env, got %+v", got)
		}
	})
}
