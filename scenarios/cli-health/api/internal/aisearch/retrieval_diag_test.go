package aisearch

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"

	pkg "github.com/vrooli/aisearch-go"
)

// retrieval_diag_test.go isolates RETRIEVAL quality from RERANK. For each corpus
// query it reports the rank of the canonical command in the raw vector results
// (no rerank, no floor, limit 100) under each index strategy. This is the
// measurement the recall-experiment lacked: recall@5 there was decided by the
// cross-encoder re-sort, which masked whether the retrieval-side levers (task
// prefixes, enriched text, hybrid sparse) move the canonical command UP the
// shortlist at all.
//
//	CLI_HEALTH_AISEARCH_LIVE=1 CLI_HEALTH_RECALL_EXPERIMENT=1 \
//	  go test ./internal/aisearch/ -run TestRetrievalRankDiagnostic -v -timeout 20m

func TestRetrievalRankDiagnostic(t *testing.T) {
	if os.Getenv("CLI_HEALTH_AISEARCH_LIVE") == "" || os.Getenv("CLI_HEALTH_RECALL_EXPERIMENT") == "" {
		t.Skip("set CLI_HEALTH_AISEARCH_LIVE=1 CLI_HEALTH_RECALL_EXPERIMENT=1 to run the retrieval diagnostic")
	}
	corpus := loadCommandCorpus(t)
	repoRoot := os.Getenv("CLI_HEALTH_REPO_ROOT")
	if repoRoot == "" {
		root, err := repocontract.ResolveRepoRoot()
		if err != nil {
			t.Fatalf("resolve repo root: %v", err)
		}
		repoRoot = root
	}
	cfg := pkg.LoadConfig("CLI_HEALTH")
	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Minute)
	defer cancel()

	newDiscovery := func() DiscoverySource {
		d := NewFilesystemDiscoverySource(repoRoot)
		d.ExternalCLIs = []ExternalCLI{{Name: "vrooli", Binary: "vrooli"}}
		return d
	}
	noPrefix := pkg.NewEmbedderWithPrefixes(cfg.EmbedModel, "", "")
	prefixed := pkg.NewEmbedderForConfig(pkg.Config{EmbedModel: cfg.EmbedModel, EmbedTaskPrefix: true})
	bm25 := pkg.NewBM25SparseEncoder()

	type idx struct {
		key        string
		collection string
		embedder   pkg.Embedder
		compose    func(CommandRecord) string
		sparse     pkg.SparseEncoder
		queryDense bool // also probe dense-only retrieval on a hybrid index
	}
	indices := []idx{
		{"A noprefix+terse", "cli-health-diag-a", noPrefix, composeCommandEmbeddingText, nil, false},
		{"B prefix+terse", "cli-health-diag-b", prefixed, composeCommandEmbeddingText, nil, false},
		{"C prefix+enriched", "cli-health-diag-c", prefixed, composeCommandEmbeddingTextEnriched, nil, false},
		{"D prefix+enriched+hybrid", "cli-health-diag-d", prefixed, composeCommandEmbeddingTextEnriched, bm25, true},
	}

	for _, ix := range indices {
		dropCollection(t, cfg.QdrantURL, cfg.QdrantAPIKey, ix.collection)
		svc := NewService(Options{
			Embedder:    ix.embedder,
			VectorStore: pkg.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, ix.collection),
			Discovery:   newDiscovery(),
			Parallelism: cfg.ReconcileParallelism,
			Collection:  ix.collection,
			Compose:     ix.compose,
			Sparse:      ix.sparse,
		})
		if err := svc.EnsureCollection(ctx); err != nil {
			t.Fatalf("[%s] ensure: %v", ix.key, err)
		}
		rec := svc.Reconciler()
		plan, err := rec.Plan(ctx)
		if err != nil {
			t.Fatalf("[%s] plan: %v", ix.key, err)
		}
		if _, err := rec.Apply(ctx, plan); err != nil {
			t.Fatalf("[%s] apply: %v", ix.key, err)
		}
	}
	defer func() {
		for _, ix := range indices {
			dropCollection(t, cfg.QdrantURL, cfg.QdrantAPIKey, ix.collection)
		}
	}()

	const probe = 100
	// rawRank returns the 0-based rank of the first wanted full_path in the raw
	// vector results, or -1 if not in the top `probe`.
	rawRank := func(ix idx, query string, want map[string]bool, hybrid bool) int {
		var dense []float64
		var err error
		if te, ok := ix.embedder.(pkg.TaskEmbedder); ok {
			dense, err = te.EmbedQuery(ctx, query)
		} else {
			dense, err = ix.embedder.Embed(ctx, query)
		}
		if err != nil {
			t.Fatalf("embed: %v", err)
		}
		store := pkg.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, ix.collection)
		hq := pkg.HybridQuery{Dense: dense, Limit: probe, PrefetchLimit: probe}
		if hybrid && ix.sparse != nil {
			sv := ix.sparse.Encode(query)
			hq.Sparse = &sv
			hq.Fusion = "rrf"
		}
		raw, err := store.Query(ctx, hq)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		for i, r := range raw {
			fp, _ := r.Payload["full_path"].(string)
			if want[normPath(fp)] {
				return i
			}
		}
		return -1
	}

	rankStr := func(r int) string {
		if r < 0 {
			return ">100"
		}
		return fmt.Sprintf("%d", r)
	}

	t.Log("RAW RETRIEVAL RANK of canonical command (0-based; >100 = not retrieved)")
	t.Logf("%-20s | %5s | %5s | %5s | %7s | %7s", "query", "A", "B", "C", "D-hyb", "D-dense")
	for _, c := range corpus.Cases {
		want := make(map[string]bool, len(c.ExpectedPaths))
		for _, p := range c.ExpectedPaths {
			want[normPath(p)] = true
		}
		ra := rawRank(indices[0], c.Query, want, false)
		rb := rawRank(indices[1], c.Query, want, false)
		rc := rawRank(indices[2], c.Query, want, false)
		rdH := rawRank(indices[3], c.Query, want, true)
		rdD := rawRank(indices[3], c.Query, want, false)
		t.Logf("%-20s | %5s | %5s | %5s | %7s | %7s", c.ID, rankStr(ra), rankStr(rb), rankStr(rc), rankStr(rdH), rankStr(rdD))
	}
}
