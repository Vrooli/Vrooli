package aisearch

import (
	"context"
	"os"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"

	pkg "github.com/vrooli/ai-go/search"
)

// bvariants_test.go pins down the production config on the winning index
// (prefix + terse + dense, "index B" from the retrieval diagnostic): the raw
// retrieval there puts the canonical command in the top-5 for ~14/20 queries
// (~0.70), so the question is which pipeline (rerank / floor) preserves that
// rather than eroding it. The authority-prior arm was measured to HURT recall
// and removed (see graduation-retrospective.md).
//
//	CLI_HEALTH_AISEARCH_LIVE=1 CLI_HEALTH_RECALL_EXPERIMENT=1 \
//	  go test ./internal/aisearch/ -run TestRecallBVariants -v -timeout 20m

func TestRecallBVariants(t *testing.T) {
	if os.Getenv("CLI_HEALTH_AISEARCH_LIVE") == "" || os.Getenv("CLI_HEALTH_RECALL_EXPERIMENT") == "" {
		t.Skip("set CLI_HEALTH_AISEARCH_LIVE=1 CLI_HEALTH_RECALL_EXPERIMENT=1 to run the B-variant recall matrix")
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
	reranker := func() *pkg.RerankerChain {
		return pkg.NewRerankerChain(
			pkg.NewCrossEncoderReranker(cfg.RerankerURL, cfg.RerankerModel),
			pkg.NewLLMReranker(cfg.RerankRole),
		)
	}
	prefixed := pkg.NewEmbedderForConfig(pkg.Config{EmbedModel: cfg.EmbedModel, EmbedTaskPrefix: true})
	const collection = "cli-health-bvar"

	// Build index B once.
	dropCollection(t, cfg.QdrantURL, cfg.QdrantAPIKey, collection)
	idxSvc := NewService(Options{
		Embedder:    prefixed,
		VectorStore: pkg.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, collection),
		Discovery:   newDiscovery(),
		Parallelism: cfg.ReconcileParallelism,
		Collection:  collection,
	})
	if err := idxSvc.EnsureCollection(ctx); err != nil {
		t.Fatalf("ensure: %v", err)
	}
	plan, err := idxSvc.Reconciler().Plan(ctx)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := idxSvc.Reconciler().Apply(ctx, plan); err != nil {
		t.Fatalf("apply: %v", err)
	}
	defer dropCollection(t, cfg.QdrantURL, cfg.QdrantAPIKey, collection)

	type row struct {
		name     string
		rerank   bool
		floorOff bool
	}
	rows := []row{
		{"B1 rerank+floor (prod-equiv+prefix)", true, false},
		{"B3 norerank+floor", false, false},
		{"B5 norerank+NOfloor", false, true},
	}

	type res struct {
		name   string
		recall float64
		hits   int
	}
	var out []res
	for _, r := range rows {
		svc := NewService(Options{
			Embedder:        prefixed,
			VectorStore:     pkg.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, collection),
			Discovery:       newDiscovery(),
			Collection:      collection,
			RerankEnabled:   r.rerank,
			Reranker:        reranker(),
			RerankShortlist: pkg.DefaultRerankShortlist,
			DisableFloor:    r.floorOff,
		})
		recall, hits, misses := measureRecall(ctx, t, svc, corpus)
		out = append(out, res{r.name, recall, hits})
		t.Logf("=== %s: recall@5 = %.3f (%d/%d)", r.name, recall, hits, len(corpus.Cases))
		for _, m := range misses {
			t.Logf("    MISS %s", m)
		}
	}
	t.Log("================ B-VARIANT SUMMARY ================")
	for _, r := range out {
		t.Logf("  %-40s recall@5 = %.3f (%d/%d)", r.name, r.recall, r.hits, len(corpus.Cases))
	}
}
