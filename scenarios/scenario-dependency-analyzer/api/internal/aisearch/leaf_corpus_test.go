package aisearch

import "testing"

// TestComposeScenarioEmbeddingText locks the connection corpus's embedding-text
// rendering — the bytes the embedder and ContentHash see. A drift silently
// re-embeds the live collection and moves every retrieval score.
func TestComposeScenarioEmbeddingText(t *testing.T) {
	got := composeScenarioEmbeddingText(ScenarioConnection{
		Scenario:  "plan-manager",
		DependsOn: []string{"prompt-manager", "search-hub"},
		UsedBy:    []string{"swarm-manager"},
	})
	want := "Scenario: plan-manager. Depends on: prompt-manager, search-hub. Used by: swarm-manager."
	if got != want {
		t.Fatalf("body =\n%q\nwant\n%q", got, want)
	}

	empty := composeScenarioEmbeddingText(ScenarioConnection{Scenario: "isolated"})
	wantEmpty := "Scenario: isolated. Depends on: (no scenario dependencies). Used by: (no scenario dependents)."
	if empty != wantEmpty {
		t.Fatalf("empty body = %q, want %q", empty, wantEmpty)
	}
}

// TestComposeResourceEmbeddingText locks the resource-usage corpus's rendering.
func TestComposeResourceEmbeddingText(t *testing.T) {
	got := composeResourceEmbeddingText(ResourceUsage{
		Resource: "postgres",
		Type:     "postgres",
		UsedBy:   []string{"agent-manager", "plan-manager"},
	})
	want := "Resource: postgres. Used by scenarios: agent-manager, plan-manager."
	if got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}

	// A type distinct from the name is rendered; an empty consumer list is honest.
	typed := composeResourceEmbeddingText(ResourceUsage{Resource: "vault-kv", Type: "vault"})
	wantTyped := "Resource: vault-kv. Type: vault. Used by scenarios: (none)."
	if typed != wantTyped {
		t.Fatalf("typed body = %q, want %q", typed, wantTyped)
	}
}

// TestScenarioContentHashStable confirms the drift hash changes iff the rendered
// body changes (so a no-op reconcile skips re-embedding, and a real edit does not).
func TestScenarioContentHashStable(t *testing.T) {
	a := ScenarioConnection{Scenario: "x", DependsOn: []string{"a"}}
	aAgain := ScenarioConnection{Scenario: "x", DependsOn: []string{"a"}}
	if scenarioContentHash(a) != scenarioContentHash(aAgain) {
		t.Fatal("hash not deterministic for equal records")
	}
	b := a
	b.DependsOn = []string{"a", "b"}
	if scenarioContentHash(a) == scenarioContentHash(b) {
		t.Fatal("hash unchanged after dependency edit")
	}
}
