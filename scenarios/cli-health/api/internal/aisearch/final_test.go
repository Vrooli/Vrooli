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

// final_test.go is the deciding run: on the winning index (prefix+terse+dense)
// it compares the three rerank policies on BOTH axes that matter —
//   - recall@5 on the labeled corpus (the gate metric), and
//   - gibberish rejection (the cross-encoder's real job: junk must score ~0 /
//     drop out / be weak-labeled),
//
// so the production policy is chosen from the precision/recall trade, not assumed.
//
//	CLI_HEALTH_AISEARCH_LIVE=1 CLI_HEALTH_RECALL_EXPERIMENT=1 \
//	  go test ./internal/aisearch/ -run TestRecallFinal -v -timeout 20m

func TestRecallFinal(t *testing.T) {
	if os.Getenv("CLI_HEALTH_AISEARCH_LIVE") == "" || os.Getenv("CLI_HEALTH_RECALL_EXPERIMENT") == "" {
		t.Skip("set CLI_HEALTH_AISEARCH_LIVE=1 CLI_HEALTH_RECALL_EXPERIMENT=1 to run the final decision matrix")
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
			pkg.NewLLMReranker(cfg.RerankModel),
		)
	}
	prefixed := pkg.NewEmbedderForConfig(pkg.Config{EmbedModel: cfg.EmbedModel, EmbedTaskPrefix: true})
	const collection = "cli-health-final"

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

	gibberish := []string{
		"asdfqwer zxcvbnm qwertyuiop florbnax",
		"wibble snorp gleurgh ffffzzzz",
	}

	type policy struct {
		name   string
		rerank bool
		blend  bool
	}
	policies := []policy{
		{"rerank-ON (pure reorder)", true, false},
		{"rerank-OFF", false, false},
		{"rerank-BLEND (RRF)", true, true},
	}

	type row struct {
		name      string
		recall    float64
		hits      int
		gibSummry string
	}
	var rows []row
	for _, p := range policies {
		svc := NewService(Options{
			Embedder:        prefixed,
			VectorStore:     pkg.NewVectorStore(cfg.QdrantURL, cfg.QdrantAPIKey, collection),
			Discovery:       newDiscovery(),
			Collection:      collection,
			RerankEnabled:   p.rerank,
			RerankBlend:     p.blend,
			Reranker:        reranker(),
			RerankShortlist: cfg.RerankShortlist,
		})
		recall, hits, _ := measureRecall(ctx, t, svc, corpus)

		// Gibberish probe: a strong-looking (non-weak) hit is a precision failure.
		gibStrong := 0
		gibDetail := ""
		for _, gq := range gibberish {
			resp, err := svc.Search(ctx, gq, 5, ModeAI)
			if err != nil {
				t.Fatalf("gibberish search: %v", err)
			}
			topScore := 0.0
			strongHere := 0
			for _, h := range resp.Results {
				if h.Score > topScore {
					topScore = h.Score
				}
				if !h.Weak {
					strongHere++
				}
			}
			gibStrong += strongHere
			gibDetail += fmt.Sprintf(" [n=%d top=%.3f strong=%d]", len(resp.Results), topScore, strongHere)
		}
		rows = append(rows, row{p.name, recall, hits, fmt.Sprintf("strongJunk=%d%s", gibStrong, gibDetail)})
		t.Logf("=== %-26s recall@5=%.3f (%d/%d) | gibberish: %s", p.name, recall, hits, len(corpus.Cases), rows[len(rows)-1].gibSummry)
	}

	t.Log("================ FINAL DECISION MATRIX ================")
	for _, r := range rows {
		t.Logf("  %-26s recall@5=%.3f (%2d/%d) | %s", r.name, r.recall, r.hits, len(corpus.Cases), r.gibSummry)
	}
}
