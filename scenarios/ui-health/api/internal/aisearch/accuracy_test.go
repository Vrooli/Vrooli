package aisearch

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	pkg "github.com/vrooli/ai-go/search"
)

// accuracy_test.go is ui-health's search gate, mirroring the cli-health / KO
// adopters. It has two layers:
//
//   - TestSearchSSOTWellFormed (always runs, no infra): the scenario-owned
//     search.json must parse, expose the ui-health.surfaces provider with the
//     dense surfaces tuning, and carry a non-empty corpus including a junk
//     negative. Catches an SSOT regression in `make test` without ollama/qdrant.
//   - TestSurfaceJunkRejection (live-gated): against a live index, the gibberish
//     negative must not surface a strong hit. The rank-centric A/B over real
//     surface ids lives in the search-hub eval suite (mirrored from the tests
//     block at registration), not here — this is the thin per-build smoke.
//
// The surfaces corpus is discovered via InventoryService on the per-framework
// component-library scenarios, so a full recall gate needs those up; the live
// layer degrades gracefully (skips the recall assertions) when the corpus is
// empty.

const surfacesGateCollection = "ui-health-surface-recall-gate"

// surfacesProvider loads the ui-health.surfaces provider from the search.json
// SSOT (UI_HEALTH_SEARCH_FILE overrides the package-relative default path).
func surfacesProvider(t *testing.T) pkg.ProviderConfig {
	t.Helper()
	path := os.Getenv("UI_HEALTH_SEARCH_FILE")
	if path == "" {
		path = filepath.Join("..", "..", "..", ".vrooli", "search.json")
	}
	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		t.Fatalf("load search.json %s: %v", path, err)
	}
	provider, ok := file.Provider("ui-health.surfaces")
	if !ok {
		t.Fatalf("provider ui-health.surfaces not found in %s", path)
	}
	return provider
}

func TestSearchSSOTWellFormed(t *testing.T) {
	provider := surfacesProvider(t)
	tuning := provider.ResolvedTuning()
	// Surfaces are a dense single-chunk corpus; rerank is off for it.
	if tuning.Engine != pkg.EngineDense {
		t.Errorf("surfaces tuning engine = %q, want dense", tuning.Engine)
	}
	if tuning.RerankEnabled {
		t.Errorf("surfaces tuning should run rerank-off, got rerank_enabled=true")
	}
	cases := provider.Tests.Cases
	if len(cases) == 0 {
		t.Fatal("SSOT surfaces corpus has no cases")
	}
	positives, negatives := 0, 0
	for _, tc := range cases {
		if tc.ExpectNoStrongHit {
			negatives++
		} else {
			positives++
		}
		if tc.Query == "" {
			t.Errorf("case %q has an empty query", tc.ID)
		}
	}
	if positives == 0 {
		t.Error("SSOT has no positive cases")
	}
	if negatives == 0 {
		t.Error("SSOT has no negative cases (junk-rejection guard missing)")
	}
}

// TestSurfaceJunkRejection asserts that against a live, populated index the
// gibberish negative does not surface a strong (non-noise) hit. Live-gated; it
// skips when the corpus is empty (the component-library discovery dependency was
// not reindexed) rather than failing on infra absence.
func TestSurfaceJunkRejection(t *testing.T) {
	if os.Getenv("UI_HEALTH_AISEARCH_LIVE") == "" {
		t.Skip("set UI_HEALTH_AISEARCH_LIVE=1 to run the live junk-rejection smoke (needs ollama + qdrant + a populated index)")
	}
	provider := surfacesProvider(t)

	repoRoot := os.Getenv("UI_HEALTH_REPO_ROOT")
	if repoRoot == "" {
		t.Skip("set UI_HEALTH_REPO_ROOT to the repo root for the live smoke")
	}

	cfg := pkg.LoadConfig("UI_HEALTH")
	svc := NewSearchService(provider.ResolvedTuning(), Options{
		Discovery: NewFilesystemDiscoverySource(repoRoot),
		Threshold: DefaultSearchThreshold,
		EngineDeps: pkg.EngineDeps{
			QdrantURL:    cfg.QdrantURL,
			QdrantAPIKey: cfg.QdrantAPIKey,
			Collection:   surfacesGateCollection,
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := svc.EnsureCollection(ctx); err != nil {
		t.Fatalf("ensure collection: %v", err)
	}
	if st := svc.Status(ctx); !st.Qdrant || !st.Ollama {
		t.Fatalf("backends not reachable: %+v", st)
	}

	for _, tc := range provider.Tests.Cases {
		if !tc.ExpectNoStrongHit {
			continue
		}
		resp, err := svc.Search(ctx, tc.Query, 10, ModeAI)
		if err != nil {
			t.Fatalf("search %q: %v", tc.Query, err)
		}
		for _, h := range resp.Results {
			if tc.ExpectMaxScore > 0 && h.Score > tc.ExpectMaxScore {
				t.Errorf("junk case %q surfaced a strong hit %q at score %.3f (> ceiling %.2f)", tc.ID, h.DisplayName, h.Score, tc.ExpectMaxScore)
			}
		}
	}
}
