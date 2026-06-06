package aisearch

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"

	pkg "github.com/vrooli/aisearch-go"
)

// recall_test.go is cli-health's REQ-P0-004 acceptance gate: command search must
// achieve recall@5 >= 0.8 on the hand-labeled corpus at
// scenarios/cli-health/testdata/search_queries.json. It mirrors the KO docs
// accuracy harness — a thin per-build smoke gate, NOT a second eval system (the
// search-hub eval domain owns A/B tuning).
//
// Live-gated (needs ollama + qdrant + a populated index):
//
//	CLI_HEALTH_AISEARCH_LIVE=1 go test ./internal/aisearch/ -run TestCommandRecall -v -timeout 20m
//
// Optional env: QDRANT_URL, QDRANT_API_KEY, CLI_HEALTH_CORPUS_FILE (corpus
// override), CLI_HEALTH_REPO_ROOT (discovery repo root).
//
// On a miss the harness prints the actual top FullPaths so the FIRST live run is
// a calibration pass: fix any expected_paths that differ from what discovery
// emits, then the gate is meaningful.

type commandCase struct {
	ID            string   `json:"id"`
	Query         string   `json:"query"`
	ExpectedPaths []string `json:"expected_paths"`
}

type commandScoring struct {
	RecallAt     int     `json:"recall_at"`
	RecallTarget float64 `json:"recall_target"`
}

type commandCorpus struct {
	Scoring commandScoring `json:"scoring"`
	Cases   []commandCase  `json:"cases"`
}

func loadCommandCorpus(t *testing.T) commandCorpus {
	t.Helper()
	path := os.Getenv("CLI_HEALTH_CORPUS_FILE")
	if path == "" {
		// Resolved relative to this package dir: .../api/internal/aisearch.
		path = filepath.Join("..", "..", "..", "testdata", "search_queries.json")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus %s: %v", path, err)
	}
	var c commandCorpus
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("parse corpus %s: %v", path, err)
	}
	if len(c.Cases) == 0 {
		t.Fatal("corpus has no cases")
	}
	if c.Scoring.RecallAt == 0 {
		c.Scoring.RecallAt = 5
	}
	if c.Scoring.RecallTarget == 0 {
		c.Scoring.RecallTarget = 0.8
	}
	return c
}

// normPath lowercases + collapses whitespace so a FullPath comparison is robust
// to incidental spacing differences between the label and discovery's output.
func normPath(p string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(p))), " ")
}

func TestCommandRecall(t *testing.T) {
	if os.Getenv("CLI_HEALTH_AISEARCH_LIVE") == "" {
		t.Skip("set CLI_HEALTH_AISEARCH_LIVE=1 to run the live recall@5 gate (needs ollama + qdrant)")
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
	engine := pkg.NewDenseEngine(cfg, DefaultCollection)
	discovery := NewFilesystemDiscoverySource(repoRoot)
	discovery.ExternalCLIs = []ExternalCLI{{Name: "vrooli", Binary: "vrooli"}}
	svc := NewService(Options{
		Embedder:        engine.Embedder,
		VectorStore:     engine.VectorStore,
		Discovery:       discovery,
		Parallelism:     cfg.ReconcileParallelism,
		RerankEnabled:   cfg.RerankEnabled,
		Reranker:        engine.Reranker,
		RerankShortlist: cfg.RerankShortlist,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Minute)
	defer cancel()

	if err := svc.EnsureCollection(ctx); err != nil {
		t.Fatalf("ensure collection: %v", err)
	}
	// Populate the index (idempotent — a warm index is a no-op fast path).
	rec := svc.Reconciler()
	plan, err := rec.Plan(ctx)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := rec.Apply(ctx, plan); err != nil {
		t.Fatalf("apply (index): %v", err)
	}

	k := corpus.Scoring.RecallAt
	hits := 0
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
			t.Logf("MISS %s (%q): expected one of %v; got top-%d %v",
				c.ID, c.Query, c.ExpectedPaths, k, got)
		}
	}

	recall := float64(hits) / float64(len(corpus.Cases))
	t.Logf("recall@%d = %.3f (%d/%d), target %.2f", k, recall, hits, len(corpus.Cases), corpus.Scoring.RecallTarget)
	if recall < corpus.Scoring.RecallTarget {
		t.Fatalf("recall@%d = %.3f below target %.2f — calibrate testdata/search_queries.json expected_paths from the MISS logs above, or investigate the index",
			k, recall, corpus.Scoring.RecallTarget)
	}
}
