package providerpool

import (
	"testing"
	"time"
)

func TestObservationFromProviderFailureMapsGatewayVocabulary(t *testing.T) {
	cases := []struct {
		class string
		want  string
		ok    bool
	}{
		{GatewayFailureInsufficientCredits, ClassAPICreditExhausted, true},
		{GatewayFailureRateLimited, ClassRateLimit, true},
		{GatewayFailureProviderOverloaded, ClassProviderOverload, true},
		{"missing_binary", "", false},
		{"timeout", "", false},
		{"malformed_json", "", false},
		{"policy_error", "", false},
		{"execution_error", "", false},
		{"cancellation", "", false},
		{"unavailable", "", false},
		{"unsupported_sampling", "", false},
		{"", "", false},
		{" ", "", false},
	}
	for _, tc := range cases {
		obs, ok := ObservationFromProviderFailure(ProviderFailure{Class: tc.class})
		if ok != tc.ok {
			t.Fatalf("class %q: ok=%v want %v", tc.class, ok, tc.ok)
		}
		if tc.ok && obs.Class != tc.want {
			t.Fatalf("class %q mapped to %q want %q", tc.class, obs.Class, tc.want)
		}
	}
}

func TestObservationFromProviderFailureCarriesProvenanceVerbatim(t *testing.T) {
	reset := time.Date(2026, 9, 13, 13, 0, 0, 0, time.UTC).Format(time.RFC3339)
	obs, ok := ObservationFromProviderFailure(ProviderFailure{
		Pool:       "openrouter",
		Class:      GatewayFailureRateLimited,
		HTTPStatus: 429,
		RetryAfter: "37",
		ResetAt:    reset,
		Source:     "http:429",
		Runner:     "opencode",
		Provider:   "openrouter",
		Model:      "deepseek-v4.1-flash",
	})
	if !ok {
		t.Fatal("rate_limited must be a shared-pool recovery condition")
	}
	if obs.Class != ClassRateLimit || obs.RetryAfter != "37" || obs.ResetAt != reset || obs.Source != "http:429" {
		t.Fatalf("observed recovery provenance not carried verbatim: %+v", obs)
	}
	if obs.Pool != "openrouter" || obs.Provider != "openrouter" || obs.Model != "deepseek-v4.1-flash" || obs.Runner != "opencode" {
		t.Fatalf("pool identity not carried: %+v", obs)
	}
}

func TestProviderFailureCreditExhaustionPausesPool(t *testing.T) {
	p, _ := newTestPool(t, "openrouter")

	credit, ok := ObservationFromProviderFailure(ProviderFailure{
		Class: GatewayFailureInsufficientCredits, HTTPStatus: 402, Source: "http:402",
	})
	if !ok {
		t.Fatal("insufficient_credits must be a shared-pool recovery condition")
	}
	state, err := p.Observe(credit)
	if err != nil {
		t.Fatalf("observe credit exhaustion: %v", err)
	}
	if !state.Blocked || state.Eligible || !state.RequiresObservation || state.Class != ClassAPICreditExhausted {
		t.Fatalf("credit exhaustion with no observed reset must require a later observation: %+v", state)
	}
	if _, err := p.Reserve(Limit{}, Reservation{AttemptID: "a1", Known: true}); err == nil {
		t.Fatal("a paused pool must refuse a reservation")
	}
}

func TestProviderFailureRateLimitReleasesAfterObservedReset(t *testing.T) {
	p, now := newTestPool(t, "openrouter")
	reset := now.Add(15 * time.Minute)
	rate, ok := ObservationFromProviderFailure(ProviderFailure{
		Class: GatewayFailureRateLimited, HTTPStatus: 429,
		RetryAfter: "900", ResetAt: reset.Format(time.RFC3339), Source: "http:429",
	})
	if !ok {
		t.Fatal("rate_limited must be a shared-pool recovery condition")
	}
	state, err := p.Observe(rate)
	if err != nil {
		t.Fatalf("observe rate limit: %v", err)
	}
	if !state.Blocked || !state.ResetKnown || state.ResetAt != reset.Format(time.RFC3339) {
		t.Fatalf("rate limit must pause on its observed reset: %+v", state)
	}
	*now = reset.Add(time.Second)
	state, err = p.State()
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if !state.Eligible {
		t.Fatalf("a known reset that has passed must re-enable the pool: %+v", state)
	}
}

func TestProviderFailureObservationRecordsRawProvenance(t *testing.T) {
	p, _ := newTestPool(t, "openrouter")
	obs, ok := ObservationFromProviderFailure(ProviderFailure{
		Class: GatewayFailureRateLimited, HTTPStatus: 429, RetryAfter: "37", Source: "http:429",
	})
	if !ok {
		t.Fatal("rate_limited must be a shared-pool recovery condition")
	}
	if _, err := p.Observe(obs); err != nil {
		t.Fatalf("observe: %v", err)
	}
	current, err := p.load()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(current.Observations) != 1 {
		t.Fatalf("expected one recorded observation, got %d", len(current.Observations))
	}
	recorded := current.Observations[0]
	if recorded.RetryAfter != "37" || recorded.Source != "http:429" || recorded.Class != ClassRateLimit {
		t.Fatalf("raw retry-after/source provenance was not persisted: %+v", recorded)
	}
}
