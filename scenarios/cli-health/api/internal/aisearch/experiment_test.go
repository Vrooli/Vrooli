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

	pkg "github.com/vrooli/ai-go/search"
)

// experiment_test.go is the recall-gap exploration harness (NOT the gate — that
// is TestCommandRecall). It builds several index strategies into scratch
// collections and measures recall@5 for each configuration, isolating the
// contribution of every LIVE lever:
//
//	H1  query/document task prefixes (nomic-embed-text)
//	H2  hybrid (dense + sparse BM25 RRF) leg
//	    + the rerank on/off contrast
//
// The enriched-embedding-text (H5) and canonical-origin authority (H4) arms
// were measured to HURT recall (0.70→0.40 and 0.70→0.65) and have been removed;
// their verdicts live in packages/ai-go/search/docs/graduation-retrospective.md.
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
			pkg.NewLLMReranker(cfg.RerankRole),
		)
	}

	noPrefix := pkg.NewEmbedderWithPrefixes(cfg.EmbedModel, "", "")
	prefixed := pkg.NewEmbedderForConfig(pkg.Config{EmbedModel: cfg.EmbedModel, EmbedRole: cfg.EmbedRole, EmbedTaskPrefix: true})
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
	idxC := idx{"C:prefix+terse+hybrid", "cli-health-exp-c", prefixed, composeCommandEmbeddingText, bm25}
	indices := []idx{idxA, idxB, idxC}

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

	// Query-time configs (reuse an index; vary rerank).
	type cfgRow struct {
		name   string
		ix     idx
		rerank bool
	}
	rows := []cfgRow{
		{"C0 baseline (noprefix,terse,dense,rerank)", idxA, true},
		{"C1 +prefix (H1)", idxB, true},
		{"C2 +hybrid (H2)", idxC, true},
		{"C3 hybrid, NO rerank", idxC, false},
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
			RerankShortlist: pkg.DefaultRerankShortlist,
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
		want := make(map[string]bool, len(c.ExpectIDs))
		for _, id := range c.ExpectIDs {
			want[normPath(id)] = true
		}
		got := make([]string, 0, len(resp.Results))
		found := false
		for i, h := range resp.Results {
			if i >= k {
				break
			}
			got = append(got, h.Name)
			if want[normPath(h.Name)] {
				found = true
			}
		}
		if found {
			hits++
		} else {
			misses = append(misses, fmt.Sprintf("%s (%q): want %v; got %v", c.ID, c.Query, c.ExpectIDs, got))
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
