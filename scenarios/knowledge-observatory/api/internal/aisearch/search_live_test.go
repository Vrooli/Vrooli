package aisearch

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	pkg "github.com/vrooli/ai-go/search"
)

// TestLiveSearchServiceHybridRerank is the Phase-5 end-to-end live proof: it
// indexes a bounded slice of the real repo, then drives the full SearchService
// (hybrid dense+sparse RRF -> authority boost -> reranker) and asserts the
// degradation chain reports an active reranker and results carry real bodies.
//
// Gated on KO_AISEARCH_LIVE so it never runs in CI without infra:
//
//	KO_AISEARCH_LIVE=1 \
//	KO_LIVE_SCENARIOS_ROOT=/abs/path/to/Vrooli/scenarios \
//	go test ./internal/aisearch/ -run TestLiveSearchServiceHybridRerank -v
//
// Optional: QDRANT_URL, KO_LIVE_COLLECTION, KO_LIVE_BUDGET (default 400 embeds),
// RERANKER_URL (defaults to the resource's 127.0.0.1:11453).
func TestLiveSearchServiceHybridRerank(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 (and KO_LIVE_SCENARIOS_ROOT) to run the live search proof")
	}
	scenariosRoot := os.Getenv("KO_LIVE_SCENARIOS_ROOT")
	if scenariosRoot == "" {
		t.Fatal("KO_LIVE_SCENARIOS_ROOT must point at the repo's scenarios/ directory")
	}
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = pkg.DefaultQdrantURL
	}
	collection := os.Getenv("KO_LIVE_COLLECTION")
	if collection == "" {
		collection = DefaultCollection
	}
	budget := 400
	if raw := os.Getenv("KO_LIVE_BUDGET"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			budget = v
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	embedder := pkg.NewEmbedder(pkg.DefaultEmbedModel)
	if !embedder.Available(ctx) {
		t.Fatal("ollama embedder unavailable; start the ollama resource first")
	}
	store := pkg.NewVectorStore(qdrantURL, os.Getenv("QDRANT_API_KEY"), collection)
	if !store.Available(ctx) {
		t.Fatal("qdrant unavailable; start the qdrant resource first")
	}

	idx, err := NewIndexer(Options{
		Embedder:         embedder,
		VectorStore:      store,
		Embedding:        liveEmbeddingPolicy(t, ctx),
		ScenariosRoot:    scenariosRoot,
		MaxEmbedsPerTick: budget,
	})
	if err != nil {
		t.Fatalf("NewIndexer: %v", err)
	}
	if err := idx.EnsureCollection(ctx); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}
	res, err := idx.Reindex(ctx, false)
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	t.Logf("reindex: planned=%d upserted=%d deferred=%d errors=%d", res.Planned, res.Upserted, res.Deferred, len(res.Errors))

	reranker := NewDefaultReranker()
	svc := NewSearchService(ServiceOptions{
		Embedder:      embedder,
		VectorStore:   store,
		RerankEnabled: true,
		Reranker:      reranker,
		Reconciler:    idx.Reconciler(),
	})

	status := svc.Status(ctx)
	t.Logf("status: available=%v ollama=%v qdrant=%v reranker=%q indexed=%d",
		status.Available, status.Ollama, status.Qdrant, status.Reranker, status.IndexedCount)
	if !status.Available || !status.Qdrant {
		t.Fatalf("search reports unavailable: %+v", status)
	}
	if status.Reranker == "none" {
		t.Errorf("expected an active reranker (cross-encoder resource is healthy); got none")
	}

	resp, err := svc.Search(ctx, pkg.SearchQuery{
		Query: "how does the documentation semantic search engine chunk and index files",
		Mode:  pkg.ModeHybrid,
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("Search(hybrid): %v", err)
	}
	if resp.Method != "hybrid" {
		t.Fatalf("method = %q, want hybrid", resp.Method)
	}
	t.Logf("hybrid query answered by method=%q reranker=%q total=%d", resp.Method, resp.Reranker, resp.Total)
	if len(resp.Results) == 0 {
		t.Fatal("hybrid search returned no results")
	}
	for i, h := range resp.Results {
		t.Logf("  #%d score=%.4f %s — %.80s", i+1, h.Score, h.RelativePath, h.Snippet)
		if h.Snippet == "" {
			t.Errorf("result %d has empty snippet (the legacy content:\"\" defect)", i+1)
		}
		if h.RelativePath == "" {
			t.Errorf("result %d has empty relative_path (federation contract field)", i+1)
		}
	}

	// Same-case architecture retrieval is part of the documentation truth
	// contract: the canonical source must be surfaced by the production auto
	// route after a live index reconciliation, not only by the text fallback.
	// These cases mirror the credential-free Connect fixture and the operator
	// tasks exercised by DR4.
	semanticSvc := NewSearchService(ServiceOptions{
		Embedder:        embedder,
		VectorStore:     store,
		Reconciler:      idx.Reconciler(),
		RerankShortlist: pkg.DocCorpusTuning().RerankShortlist,
	})
	for _, tc := range []struct {
		name         string
		query        string
		expectedPath string
	}{
		{name: "data ownership", query: "scenario-scoped SQLite database", expectedPath: "scenarios/knowledge-observatory/docs/concepts/DATA.md"},
		{name: "legacy viewer route", query: "legacy", expectedPath: "scenarios/knowledge-observatory/docs/concepts/ARCHITECTURE.md"},
		{name: "append API contract", query: "manifest-supported append log", expectedPath: "scenarios/knowledge-observatory/docs/reference/api-endpoints.md"},
	} {
		t.Run("same-case/"+tc.name, func(t *testing.T) {
			got, err := semanticSvc.Search(ctx, pkg.SearchQuery{
				Query: tc.query,
				Scope: pkg.Scope{Kind: pkg.ScopePath, Value: "scenarios/knowledge-observatory/docs"},
				Mode:  pkg.ModeAuto,
				Limit: 5,
			})
			if err != nil {
				t.Fatalf("Search(%q): %v", tc.query, err)
			}
			if got.Method == "text" {
				t.Fatalf("Search(%q) degraded to text; expected the live vector route", tc.query)
			}
			for _, hit := range got.Results {
				if hit.RelativePath == tc.expectedPath {
					return
				}
			}
			// On a top-five miss, inspect the same live query with a wider public
			// page to distinguish candidate recall from ranking/page crowding.
			// This is diagnostic only; the acceptance above remains the requested
			// top-five contract.
			wide, wideErr := semanticSvc.Search(ctx, pkg.SearchQuery{
				Query: tc.query,
				Scope: pkg.Scope{Kind: pkg.ScopePath, Value: "scenarios/knowledge-observatory/docs"},
				Mode:  pkg.ModeAuto,
				Limit: 100,
			})
			if wideErr != nil {
				t.Logf("wide diagnostic query failed: %v", wideErr)
			} else {
				paths := make([]string, 0, len(wide.Results))
				for _, hit := range wide.Results {
					paths = append(paths, hit.RelativePath)
				}
				t.Logf("wide diagnostic paths (%d): %v", len(paths), paths)
			}
			t.Fatalf("Search(%q) method=%q did not surface canonical source %q in top five: %+v", tc.query, got.Method, tc.expectedPath, got.Results)
		})
	}

	// Auto mode should also answer via a vector leg when infra is up.
	autoResp, err := svc.Search(ctx, pkg.SearchQuery{Query: "lifecycle make start stop", Mode: pkg.ModeAuto, Limit: 3})
	if err != nil {
		t.Fatalf("Search(auto): %v", err)
	}
	if autoResp.Method == "text" {
		t.Logf("note: auto degraded to text (vector legs returned nothing for this query)")
	}
	t.Logf("auto query answered by method=%q reranker=%q", autoResp.Method, autoResp.Reranker)
}
