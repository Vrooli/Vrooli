package search

import "testing"

func TestCompositionSearchSummaryCarriesSelectedScopeAndQuery(t *testing.T) {
	got := compositionSearchSummary("ScenarioApp command groups", "cli-health", map[string]any{"scenario": "cli-health"}, 1)
	want := `Static CLI composition for query "ScenarioApp command groups" (scenario:cli-health): found 1 result(s); runtime dispatch remains unverified.`
	if got != want {
		t.Fatalf("summary=%q want %q", got, want)
	}
}

func TestCompositionSearchSummaryKeepsUnavailableScopeExplicitOnNoMatch(t *testing.T) {
	got := compositionSearchSummary("lunar chlorophyll", "../outside", map[string]any{}, 0)
	want := `Static CLI composition for query "lunar chlorophyll" (requested scenario "../outside"; selected source scope unavailable): found 0 result(s); runtime dispatch remains unverified.`
	if got != want {
		t.Fatalf("summary=%q want %q", got, want)
	}
}
