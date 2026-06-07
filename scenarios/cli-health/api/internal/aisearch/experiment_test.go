package aisearch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"

	pkg "github.com/vrooli/aisearch-go"
)

// experiment_test.go is the recall-gap exploration harness (NOT the gate — that
// is TestCommandRecall). It builds several index strategies into scratch
// collections and measures recall@5 for each configuration, isolating the
// contribution of every lever:
//
//	H1  query/document task prefixes (nomic-embed-text)
//	H5  enriched embedding text (humanized identity + cleaned gloss)
//	H2  hybrid (dense + sparse BM25 RRF) leg
//	H4  canonical-origin authority prior
//	    + the rerank on/off contrast
//
// It prints a comparison table so the productionized config is chosen from data.
// Expensive (several full re-indexes + many reranked searches), so it is gated
// separately from the per-build live gate:
//
//	CLI_HEALTH_AISEARCH_LIVE=1 CLI_HEALTH_RECALL_EXPERIMENT=1 \
//	  go test ./internal/aisearch/ -run TestRecallExperiment -v -timeout 40m

func TestRecallExperiment(t *testing.T) {
	if os.Getenv("CLI_HEALTH_AISEARCH_LIVE") == "" || os.Getenv("CLI_HEALTH_RECALL_EXPERIMENT") == "" {
		t.Skip("set CLI_HEALTH_AISEARCH_LIVE=1 CLI_HEALTH_RECALL_EXPERIMENT=1 to run the recall experiment")
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
	ctx, cancel := context.WithTimeout(context.Background(), 38*time.Minute)
	defer cancel()

	newDiscovery := func() DiscoverySource {
		d := NewFilesystemDiscoverySource(repoRoot)
		d.ExternalCLIs = []ExternalCLI{{Name: "vrooli", Binary: "vrooli"}}
		return d
	}
	reranker := func() *pkg.RerankerChain {
		return pkg.NewRerankerChain(
			pkg.NewCrossEncoderReranker(cfg.RerankerURL, cfg.RerankerModel),
			pkg.NewLLMReranker(cfg.RerankModel),
		)
	}

	noPrefix := pkg.NewEmbedderWithPrefixes(cfg.EmbedModel, "", "")
	prefixed := pkg.NewEmbedderForConfig(pkg.Config{EmbedModel: cfg.EmbedModel, EmbedTaskPrefix: true}) // auto nomic search_query:/search_document:
	bm25 := pkg.NewBM25SparseEncoder()

	// Index strategies (the embedding-time variables). Each is built once into its
	// own scratch collection; query-time configs below reuse them.
	type idx struct {
		key        string
		collection string
		embedder   pkg.Embedder
		compose    func(CommandRecord) string
		sparse     pkg.SparseEncoder
	}
	idxA := idx{"A:noprefix+terse+dense", "cli-health-exp-a", noPrefix, composeCommandEmbeddingText, nil}
	idxB := idx{"B:prefix+terse+dense", "cli-health-exp-b", prefixed, composeCommandEmbeddingText, nil}
	idxC := idx{"C:prefix+enriched+dense", "cli-health-exp-c", prefixed, composeCommandEmbeddingTextEnriched, nil}
	idxD := idx{"D:prefix+enriched+hybrid", "cli-health-exp-d", prefixed, composeCommandEmbeddingTextEnriched, bm25}
	indices := []idx{idxA, idxB, idxC, idxD}

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
		start := time.Now()
		if _, err := rec.Apply(ctx, plan); err != nil {
			t.Fatalf("[%s] apply: %v", ix.key, err)
		}
		t.Logf("indexed %s in %s", ix.key, time.Since(start).Round(time.Second))
	}
	defer func() {
		for _, ix := range indices {
			dropCollection(t, cfg.QdrantURL, cfg.QdrantAPIKey, ix.collection)
		}
	}()

	// Query-time configs (reuse an index; vary rerank + authority prior).
	type cfgRow struct {
		name     string
		ix       idx
		rerank   bool
		decorate pkg.ScoreDecorator
	}
	rows := []cfgRow{
		{"C0 baseline (noprefix,terse,dense,rerank)", idxA, true, nil},
		{"C1 +prefix (H1)", idxB, true, nil},
		{"C2 +enriched (H5)", idxC, true, nil},
		{"C3 +hybrid (H2)", idxD, true, nil},
		{"C4 +authority (H4)", idxD, true, newAuthorityDecorator()},
		{"C5 hybrid+authority, NO rerank", idxD, false, newAuthorityDecorator()},
	}

	type result struct {
		name   string
		recall float64
		hits   int
	}
	var results []result
	for _, r := range rows {
		svc := NewService(Options{
			Embedder:        r.ix.embedder,
			VectorStore:     pkg.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, r.ix.collection),
			Discovery:       newDiscovery(),
			Collection:      r.ix.collection,
			Sparse:          r.ix.sparse,
			RerankEnabled:   r.rerank,
			Reranker:        reranker(),
			RerankShortlist: cfg.RerankShortlist,
			Decorate:        r.decorate,
		})
		recall, hits, misses := measureRecall(ctx, t, svc, corpus)
		results = append(results, result{r.name, recall, hits})
		t.Logf("=== %s: recall@5 = %.3f (%d/%d)", r.name, recall, hits, len(corpus.Cases))
		for _, m := range misses {
			t.Logf("    MISS %s", m)
		}
	}

	t.Log("================ RECALL EXPERIMENT SUMMARY ================")
	for _, r := range results {
		t.Logf("  %-44s recall@5 = %.3f (%d/%d)", r.name, r.recall, r.hits, len(corpus.Cases))
	}
}

// measureRecall runs every corpus case through svc and returns recall@5 plus the
// human-readable miss lines.
func measureRecall(ctx context.Context, t *testing.T, svc *Service, corpus commandCorpus) (float64, int, []string) {
	t.Helper()
	k := corpus.Scoring.RecallAt
	hits := 0
	var misses []string
	for _, c := range corpus.Cases {
		resp, err := svc.Search(ctx, c.Query, k, ModeAI)
		if err != nil {
			t.Fatalf("search %q: %v", c.ID, err)
		}
		want := make(map[string]bool, len(c.ExpectedPaths))
		for _, p := range c.ExpectedPaths {
			want[normPath(p)] = true
		}
		got := make([]string, 0, len(resp.Results))
		found := false
		for i, h := range resp.Results {
			if i >= k {
				break
			}
			got = append(got, h.FullPath)
			if want[normPath(h.FullPath)] {
				found = true
			}
		}
		if found {
			hits++
		} else {
			misses = append(misses, fmt.Sprintf("%s (%q): want %v; got %v", c.ID, c.Query, c.ExpectedPaths, got))
		}
	}
	sort.Strings(misses)
	return float64(hits) / float64(len(corpus.Cases)), hits, misses
}

// dropCollection deletes a scratch qdrant collection (idempotent; ignores 404).
func dropCollection(t *testing.T, baseURL, apiKey, name string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, baseURL+"/collections/"+name, nil)
	if err != nil {
		t.Fatalf("drop %s: %v", name, err)
	}
	if apiKey != "" {
		req.Header.Set("api-key", apiKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("drop %s: %v (continuing)", name, err)
		return
	}
	_ = resp.Body.Close()
}
