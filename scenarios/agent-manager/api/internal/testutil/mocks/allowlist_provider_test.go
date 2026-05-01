package mocks

import (
	"context"
	"testing"

	"agent-manager/internal/domain"
)

func TestFakeAllowlistProviderReturnsDefaultsWhenUnset(t *testing.T) {
	provider := NewDefaultAllowlistProvider()

	got := provider.GetAllowlist(context.Background())
	want := domain.DefaultInvestigationTagAllowlist()

	if len(got) != len(want) {
		t.Fatalf("expected %d default rules, got %d", len(want), len(got))
	}
}

func TestFakeAllowlistProviderCopiesRules(t *testing.T) {
	provider := NewFakeAllowlistProvider(domain.InvestigationTagRule{
		Pattern:       "custom-*",
		CaseSensitive: true,
	})

	got := provider.GetAllowlist(context.Background())
	got[0].Pattern = "mutated"

	again := provider.GetAllowlist(context.Background())
	if again[0].Pattern != "custom-*" {
		t.Fatalf("expected provider rules to be copied out, got %q", again[0].Pattern)
	}
}
