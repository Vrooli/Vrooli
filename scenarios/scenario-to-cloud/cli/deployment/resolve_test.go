package deployment

import (
	"testing"
	"time"
)

func TestResolveByDNSFallbackReturnsAmbiguousError(t *testing.T) {
	oldLookup := lookupIPAddresses
	t.Cleanup(func() {
		lookupIPAddresses = oldLookup
	})
	lookupIPAddresses = func(host string) ([]string, error) {
		return []string{"203.0.113.10"}, nil
	}

	candidates := []DeploymentSummary{
		{ID: "dep-1", Host: "203.0.113.10", CreatedAt: time.Date(2026, 2, 6, 0, 0, 0, 0, time.UTC)},
		{ID: "dep-2", Host: "203.0.113.10", CreatedAt: time.Date(2026, 2, 7, 0, 0, 0, 0, time.UTC)},
	}

	_, err := resolveByDNSFallback(candidates, "vrooli.com")
	if err == nil {
		t.Fatal("expected ambiguity error")
	}
}

func TestNormalizeTokenExtractsHostnameFromURL(t *testing.T) {
	got := normalizeToken("https://VROOLI.com/health")
	if got != "vrooli.com" {
		t.Fatalf("expected hostname normalization, got %q", got)
	}
}
