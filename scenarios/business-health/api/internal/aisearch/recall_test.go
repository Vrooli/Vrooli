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

// recall_test.go is the BH-SRCH-002 acceptance gate. It stays thin:
// search.json owns the corpus and scoring policy; ai-go/search.GradeSuite
// owns the denominator, exclusion, and outcome math.
//
// Live-gated (needs ollama + qdrant):
//
//	BUSINESS_HEALTH_AISEARCH_LIVE=1 go test ./internal/aisearch/ -run TestIntentRecall -v -timeout 20m

// recallGateCollection is a dedicated throwaway collection so the gate
// never depends on (or disturbs) the live business-health-intent index.
const recallGateCollection = "business-health-intent-recall-gate"

func TestCollectionForTuningVersionsHybridSchema(t *testing.T) {
	if got := collectionForTuning(DefaultCollection, pkg.TuningConfig{Engine: pkg.EngineDense}); got != DefaultCollection {
		t.Fatalf("dense collection = %q, want %q", got, DefaultCollection)
	}
	want := DefaultCollection + HybridCollectionSuffix
	if got := collectionForTuning(DefaultCollection, pkg.TuningConfig{Engine: pkg.EngineHybrid}); got != want {
		t.Fatalf("hybrid collection = %q, want %q", got, want)
	}
	if got := collectionForTuning(want, pkg.TuningConfig{Engine: pkg.EngineHybrid}); got != want {
		t.Fatalf("already-versioned hybrid collection = %q, want %q", got, want)
	}
}

func searchProvider(t *testing.T) pkg.ProviderConfig {
	t.Helper()
	path := os.Getenv("BUSINESS_HEALTH_SEARCH_FILE")
	if path == "" {
		path = filepath.Join("..", "..", "..", ".vrooli", "search.json")
	}
	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		t.Fatalf("load search.json %s: %v", path, err)
	}
	provider, ok := file.Provider("business-health.intent")
	if !ok {
		t.Fatalf("provider business-health.intent not found in %s", path)
	}
	return provider
}

// [REQ:BH-SRCH-001] [REQ:BH-SRCH-002] Non-live per-build guard: the SSOT
// parses, carries the intended tuning (dense + task prefix + rerank ON per
// plan decision D5), valid scoring at the SDA bar, and a corpus with
// positives, gibberish negatives, and candidate-held generated cases.
func TestSearchSSOTWellFormed(t *testing.T) {
	provider := searchProvider(t)
	tuning := provider.ResolvedTuning()
	if tuning.Engine != pkg.EngineDense || !tuning.EmbedTaskPrefix || !tuning.RerankEnabled || !tuning.RerankBlend {
		t.Errorf("intent tuning is not the intended config (dense + task prefix + rerank + blend): %+v", tuning)
	}
	if err := provider.Scoring.Validate(); err != nil {
		t.Fatalf("invalid scoring block: %v", err)
	}
	if err := provider.Tests.Validate(); err != nil {
		t.Fatalf("invalid test corpus: %v", err)
	}
	policy := provider.ResolvedScoring()
	if policy.GateK != 5 || policy.RecallTarget != 0.8 || policy.DeepK < policy.GateK {
		t.Fatalf("unexpected scoring policy: %+v", policy)
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
	if positives < 13 {
		t.Fatalf("SSOT has %d positive cases; the plan requires ≥13-query gold coverage (positives + negatives ≥ 13 with both families present)", positives)
	}
	if negatives == 0 {
		t.Fatal("SSOT has no gibberish negatives (junk-rejection guard missing)")
	}
	if generatedCandidates == 0 {
		t.Fatal("SSOT has no generated candidate cases to exercise review-as-state")
	}
}

// [REQ:BH-SRCH-002] Live recall gate at the SDA bar.
func TestIntentRecall(t *testing.T) {
	if os.Getenv("BUSINESS_HEALTH_AISEARCH_LIVE") == "" {
		t.Skip("set BUSINESS_HEALTH_AISEARCH_LIVE=1 to run the live recall gate (needs ollama + qdrant)")
	}
	provider := searchProvider(t)
	policy := provider.ResolvedScoring()

	repoRoot := os.Getenv("BUSINESS_HEALTH_REPO_ROOT")
	if repoRoot == "" {
		root, err := repocontract.ResolveRepoRoot()
		if err != nil {
			t.Fatalf("resolve repo root: %v", err)
		}
		repoRoot = root
	}

	cfg := pkg.LoadConfig("BUSINESS_HEALTH")
	engine := pkg.NewServiceForTuning(provider.ResolvedTuning(), pkg.EngineDeps{
		QdrantURL:     cfg.QdrantURL,
		QdrantAPIKey:  cfg.QdrantAPIKey,
		Collection:    recallGateCollection,
		RerankerURL:   cfg.RerankerURL,
		RerankerModel: cfg.RerankerModel,
		RerankRole:    cfg.RerankRole,
	})
	svc := NewService(Options{
		Embedder:        engine.Embedder,
		VectorStore:     engine.VectorStore,
		Sparse:          engine.SparseEncoder,
		Source:          NewFleetIntentSource(repoRoot),
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
