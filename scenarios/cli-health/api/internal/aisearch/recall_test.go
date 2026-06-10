package aisearch

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"

	pkg "github.com/vrooli/aisearch-go"
)

// recall_test.go is cli-health's REQ-P0-004 acceptance gate: command search must
// achieve recall@5 >= 0.8 on the labelled corpus in the scenario-owned SSOT
// scenarios/cli-health/.vrooli/search.json (provider "cli-health.commands",
// tests.cases). It mirrors the KO docs accuracy harness — a thin per-build smoke
// gate, NOT a second eval system (the search-hub eval domain owns A/B tuning).
// The corpus is RANK-CENTRIC: a positive case asserts expect_ids (LEAF command
// name) landing within the gate's top-K; negatives (expect_no_strong_hit) and any
// unlabelled case are skipped from the recall denominator. K and the pass bar are
// gate POLICY — package constants here, not corpus fields.
//
// Live-gated (needs ollama + qdrant + a populated index):
//
//	CLI_HEALTH_AISEARCH_LIVE=1 go test ./internal/aisearch/ -run TestCommandRecall -v -timeout 20m
//
// Optional env: QDRANT_URL, QDRANT_API_KEY, CLI_HEALTH_SEARCH_FILE (search.json
// override), CLI_HEALTH_REPO_ROOT (discovery repo root).
//
// On a miss the harness prints the actual top leaf names so the FIRST live run is
// a calibration pass: fix any expect_ids that differ from what discovery emits,
// then the gate is meaningful.

// recallGateCollection is a dedicated, self-contained collection the gate
// rebuilds each run so it never depends on a warm/stale prod index or disturbs
// the live cli-health-commands collection.
const recallGateCollection = "cli-health-commands-recall-gate"

// Gate policy (NOT corpus fields): recall@recallGateK over the positive cases must
// reach recallGateTarget. The corpus stays rank-centric and regime-agnostic; the
// per-build bar lives here.
const (
	recallGateK      = 5
	recallGateTarget = 0.8
)

type commandCase struct {
	ID        string
	Query     string
	ExpectIDs []string
}

type commandScoring struct {
	RecallAt     int
	RecallTarget float64
}

type commandCorpus struct {
	Scoring commandScoring
	Cases   []commandCase
}

// searchProvider loads the cli-health.commands provider from the search.json
// SSOT (CLI_HEALTH_SEARCH_FILE overrides the package-relative default path).
func searchProvider(t *testing.T) pkg.ProviderConfig {
	t.Helper()
	path := os.Getenv("CLI_HEALTH_SEARCH_FILE")
	if path == "" {
		// Resolved relative to this package dir: .../api/internal/aisearch.
		path = filepath.Join("..", "..", "..", ".vrooli", "search.json")
	}
	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		t.Fatalf("load search.json %s: %v", path, err)
	}
	provider, ok := file.Provider("cli-health.commands")
	if !ok {
		t.Fatalf("provider cli-health.commands not found in %s", path)
	}
	return provider
}

// commandsTuning returns the resolved tuning for the command corpus from the SSOT.
func commandsTuning(t *testing.T, _ string) pkg.TuningConfig {
	t.Helper()
	return searchProvider(t).ResolvedTuning()
}

// loadCommandCorpus reads the gate corpus from the scenario-owned search.json
// SSOT (provider "cli-health.commands", tests block) and keeps only the positive,
// labelled cases — those carrying expect_ids and not marked expect_no_strong_hit.
// Negatives and any unlabelled case are excluded from the recall denominator. The
// scoring (K + target) is fixed gate policy, not read from the file.
func loadCommandCorpus(t *testing.T) commandCorpus {
	t.Helper()
	suite := searchProvider(t).Tests
	c := commandCorpus{Scoring: commandScoring{RecallAt: recallGateK, RecallTarget: recallGateTarget}}
	for _, tc := range suite.Cases {
		if len(tc.ExpectIDs) == 0 || tc.ExpectNoStrongHit {
			continue // negative or unlabelled — not a positive recall case
		}
		c.Cases = append(c.Cases, commandCase{ID: tc.ID, Query: tc.Query, ExpectIDs: tc.ExpectIDs})
	}
	if len(c.Cases) == 0 {
		t.Fatal("corpus has no gradeable positive cases")
	}
	return c
}

// TestSearchSSOTWellFormed is a non-live per-build guard: the scenario-owned
// search.json SSOT must parse, expose the cli-health.commands provider with the
// measured-best command tuning, and carry a non-empty positive corpus + at least
// one negative case. It catches an SSOT regression without needing ollama/qdrant.
func TestSearchSSOTWellFormed(t *testing.T) {
	provider := searchProvider(t)
	tuning := provider.ResolvedTuning()
	if tuning.Engine != pkg.EngineDense || !tuning.EmbedTaskPrefix || !tuning.RerankEnabled || !tuning.RerankBlend {
		t.Errorf("command tuning is not the measured-best config: %+v", tuning)
	}
	corpus := loadCommandCorpus(t)
	if len(corpus.Cases) == 0 {
		t.Fatal("no gradeable positive cases in the SSOT")
	}
	negatives := 0
	for _, tc := range provider.Tests.Cases {
		if tc.ExpectNoStrongHit {
			negatives++
		}
	}
	if negatives == 0 {
		t.Error("SSOT has no negative cases (junk-rejection guard missing)")
	}
	for _, c := range corpus.Cases {
		if c.Query == "" || len(c.ExpectIDs) == 0 {
			t.Errorf("gradeable case %q is malformed: %+v", c.ID, c)
		}
	}
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
	// The gate reads the SAME tuning the booted service uses — the SSOT in
	// search.json (provider cli-health.commands) — so the gate and production can
	// never drift. For the command corpus that tuning is the measured-best config
	// (dense + nomic task prefixes + RRF rerank blend) that lifts recall@5
	// 0.50 -> 0.70 without losing junk rejection.
	tuning := commandsTuning(t, repoRoot)
	engine := pkg.NewServiceForTuning(tuning, pkg.EngineDeps{
		QdrantURL:     cfg.QdrantURL,
		QdrantAPIKey:  cfg.QdrantAPIKey,
		Collection:    recallGateCollection,
		RerankerURL:   cfg.RerankerURL,
		RerankerModel: cfg.RerankerModel,
		RerankModel:   cfg.RerankModel,
	})
	discovery := NewFilesystemDiscoverySource(repoRoot)
	discovery.ExternalCLIs = []ExternalCLI{{Name: "vrooli", Binary: "vrooli"}}
	svc := NewService(Options{
		Embedder:        engine.Embedder,
		VectorStore:     engine.VectorStore,
		Sparse:          engine.SparseEncoder,
		Discovery:       discovery,
		Parallelism:     cfg.ReconcileParallelism,
		Collection:      recallGateCollection,
		Floor:           engine.Tuning.Floor.Config(),
		RerankEnabled:   engine.Tuning.RerankEnabled,
		RerankBlend:     engine.Tuning.RerankBlend,
		Reranker:        engine.Reranker,
		RerankShortlist: engine.Tuning.RerankShortlist,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 18*time.Minute)
	defer cancel()

	// Build a prefix-fresh collection: the prod index may predate the task-prefix
	// embedder, and the recipe-aware drift hash would re-embed it anyway, but a
	// dedicated collection keeps the gate self-contained and off the live index.
	dropCollection(t, cfg.QdrantURL, cfg.QdrantAPIKey, recallGateCollection)
	defer dropCollection(t, cfg.QdrantURL, cfg.QdrantAPIKey, recallGateCollection)
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
			t.Logf("MISS %s (%q): expected one of %v; got top-%d %v",
				c.ID, c.Query, c.ExpectIDs, k, got)
		}
	}

	recall := float64(hits) / float64(len(corpus.Cases))
	t.Logf("recall@%d = %.3f (%d/%d), target %.2f", k, recall, hits, len(corpus.Cases), corpus.Scoring.RecallTarget)
	if recall < corpus.Scoring.RecallTarget {
		t.Fatalf("recall@%d = %.3f below target %.2f — calibrate .vrooli/search.json expect_ids from the MISS logs above, or investigate the index",
			k, recall, corpus.Scoring.RecallTarget)
	}
}
