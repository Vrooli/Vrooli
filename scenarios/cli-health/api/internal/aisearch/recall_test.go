package aisearch

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"

	pkg "github.com/vrooli/ai-go/search"
)

// recall_test.go is cli-health's REQ-P0-004 acceptance gate. It stays thin:
// search.json owns the corpus and scoring policy, and ai-go/search.GradeSuite
// owns the denominator, candidate/stale exclusion, and outcome math.
//
// Live-gated (needs ollama + qdrant + a populated command index):
//
//	CLI_HEALTH_AISEARCH_LIVE=1 go test ./internal/aisearch/ -run TestCommandRecall -v -timeout 20m
//
// Optional env: QDRANT_URL, QDRANT_API_KEY, CLI_HEALTH_SEARCH_FILE (search.json
// override), CLI_HEALTH_REPO_ROOT (discovery repo root).

// recallGateCollection is a dedicated, self-contained collection the gate
// rebuilds each run so it never depends on a warm/stale prod index or disturbs
// the live cli-health-commands collection.
const recallGateCollection = "cli-health-commands-recall-gate"

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

// TestSearchSSOTWellFormed is a non-live per-build guard: the scenario-owned
// search.json SSOT must parse, expose the cli-health.commands provider with the
// measured-best command tuning, carry valid scoring, and keep generated cases in
// candidate status until reviewed.
func TestSearchSSOTWellFormed(t *testing.T) {
	provider := searchProvider(t)
	tuning := provider.ResolvedTuning()
	if tuning.Engine != pkg.EngineDense || !tuning.EmbedTaskPrefix || !tuning.RerankEnabled || !tuning.RerankBlend {
		t.Errorf("command tuning is not the measured-best config: %+v", tuning)
	}
	if err := provider.Scoring.Validate(); err != nil {
		t.Fatalf("invalid scoring block: %v", err)
	}
	if err := provider.Tests.Validate(); err != nil {
		t.Fatalf("invalid test corpus: %v", err)
	}
	policy := provider.ResolvedScoring()
	if policy.GateK != 5 || policy.RecallTarget != 0.8 || policy.DeepK < policy.GateK {
		t.Fatalf("unexpected command scoring policy: %+v", policy)
	}

	positives, negatives, generatedCandidates := 0, 0, 0
	for _, tc := range provider.Tests.Cases {
		switch {
		case tc.ExpectNoStrongHit:
			negatives++
		case len(tc.ExpectIDs) > 0:
			positives++
		}
		if tc.HasTag(pkg.TagGenerated) {
			if !tc.IsCandidate() {
				t.Errorf("generated case %q must stay candidate until promoted", tc.ID)
			}
			generatedCandidates++
		}
	}
	if positives == 0 {
		t.Fatal("SSOT has no positive cases")
	}
	if negatives == 0 {
		t.Fatal("SSOT has no negative cases (junk-rejection guard missing)")
	}
	if generatedCandidates == 0 {
		t.Fatal("SSOT has no generated candidate cases to exercise review-as-state")
	}
}

func TestCommandRecall(t *testing.T) {
	if os.Getenv("CLI_HEALTH_AISEARCH_LIVE") == "" {
		t.Skip("set CLI_HEALTH_AISEARCH_LIVE=1 to run the live recall gate (needs ollama + qdrant)")
	}
	provider := searchProvider(t)
	policy := provider.ResolvedScoring()

	repoRoot := os.Getenv("CLI_HEALTH_REPO_ROOT")
	if repoRoot == "" {
		root, err := repocontract.ResolveRepoRoot()
		if err != nil {
			t.Fatalf("resolve repo root: %v", err)
		}
		repoRoot = root
	}

	cfg := pkg.LoadConfig("CLI_HEALTH")
	engine := pkg.NewServiceForTuning(provider.ResolvedTuning(), pkg.EngineDeps{
		QdrantURL:     cfg.QdrantURL,
		QdrantAPIKey:  cfg.QdrantAPIKey,
		Collection:    recallGateCollection,
		RerankerURL:   cfg.RerankerURL,
		RerankerModel: cfg.RerankerModel,
		RerankRole:    cfg.RerankRole,
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
	rec := svc.Reconciler()
	plan, err := rec.Plan(ctx)
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := rec.Apply(ctx, plan); err != nil {
		t.Fatalf("apply (index): %v", err)
	}

	report, err := pkg.GradeSuite(ctx, svc.current().svc, provider.Tests, policy)
	if err != nil {
		t.Fatalf("grade suite: %v", err)
	}
	for _, miss := range report.Misses {
		t.Logf("MISS %s (%q): rank=%d top=%.3f referential=%s err=%v",
			miss.CaseID, miss.Query, miss.ExpectedRank, miss.ObservedTopScore, miss.Referential, miss.Error)
	}
	for _, stale := range report.Stale {
		t.Logf("STALE %s (%q): expected id absent within deep_k=%d", stale.CaseID, stale.Query, policy.DeepK)
	}
	t.Logf("recall@%d = %.3f (%d/%d), target %.2f; excluded_candidates=%d stale=%d",
		policy.GateK, report.Recall, report.Hits, report.GradeablePositives, policy.RecallTarget,
		len(report.ExcludedCandidate), len(report.Stale))
	if report.GradeablePositives == 0 {
		t.Fatal("no reviewed, non-stale positive cases were gradeable")
	}
	if !report.MeetsTarget() {
		t.Fatalf("recall@%d = %.3f below target %.2f", policy.GateK, report.Recall, policy.RecallTarget)
	}
}

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
