package aisearch

import (
	"path/filepath"
	"testing"

	pkg "github.com/vrooli/ai-go/search"
)

// recall_test.go is SDA's per-build search-SSOT guard. It stays thin:
// .vrooli/search.json owns the corpora + scoring policies, and the shared
// ai-go/search.GradeSuite owns the recall denominator. The LIVE recall gate
// (actual retrieval against a populated index) lives in internal/searchgate,
// which can wire the real fleet data providers without an import cycle.

func loadProvider(t *testing.T, id string) pkg.ProviderConfig {
	t.Helper()
	path := filepath.Join("..", "..", "..", ".vrooli", "search.json")
	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		t.Fatalf("load search.json: %v", err)
	}
	p, ok := file.Provider(id)
	if !ok {
		t.Fatalf("provider %q not in search.json", id)
	}
	return p
}

// TestSearchSSOTWellFormed guards the two new federated leaves: each provider
// carries a valid scoring block at the engine adequacy target (recall@5 ≥ 0.8,
// deep_k ≥ gate_k), a valid tests corpus, and ≥12 reviewed positives + ≥1
// gibberish negative (the junk-rejection guard). A drift here is caught at build
// time, before the live gate ever runs.
func TestSearchSSOTWellFormed(t *testing.T) {
	for _, id := range []string{
		"scenario-dependency-analyzer.scenarios",
		"scenario-dependency-analyzer.resources",
	} {
		t.Run(id, func(t *testing.T) {
			p := loadProvider(t, id)
			if err := p.Scoring.Validate(); err != nil {
				t.Fatalf("invalid scoring block: %v", err)
			}
			if err := p.Tests.Validate(); err != nil {
				t.Fatalf("invalid test corpus: %v", err)
			}
			policy := p.ResolvedScoring()
			if policy.GateK != 5 || policy.RecallTarget != 0.8 || policy.DeepK < policy.GateK {
				t.Fatalf("unexpected scoring policy: %+v", policy)
			}

			positives, negatives := 0, 0
			for _, tc := range p.Tests.Cases {
				switch {
				case tc.ExpectNoStrongHit:
					negatives++
				case len(tc.ExpectIDs) > 0:
					positives++
				}
			}
			if positives < 12 {
				t.Fatalf("positives = %d, want ≥12 (engine adequacy floor)", positives)
			}
			if negatives == 0 {
				t.Fatal("no gibberish negative (junk-rejection guard missing)")
			}

			// Embed recipe must be uniform across SDA corpora (the corpusSpec
			// INVARIANT: one shared embedder backs every binding).
			tuning := p.ResolvedTuning()
			if tuning.EmbedModel != pkg.DefaultEmbedModel || !tuning.EmbedTaskPrefix {
				t.Fatalf("embed recipe %q/task_prefix=%v diverges from the shared SDA recipe (%q/true)",
					tuning.EmbedModel, tuning.EmbedTaskPrefix, pkg.DefaultEmbedModel)
			}
		})
	}
}
