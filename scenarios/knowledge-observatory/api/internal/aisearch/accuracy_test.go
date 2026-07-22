package aisearch

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pkg "github.com/vrooli/ai-go/search"

	"knowledge-observatory/internal/services/docsearch"
)

// accuracy_test.go is KO's live documentation-search gate. The corpus and
// scoring policy live in .vrooli/search.json; ai-go/search.GradeSuite owns the
// recall denominator, case status, stale handling, and outcome math.
//
// Live-gated (needs Qdrant with the indexed vrooli-docs collection and Ollama):
//
//	KO_AISEARCH_LIVE=1 go test ./internal/aisearch/ -run TestAccuracyCorpus -v -timeout 25m
//
// Optional env: QDRANT_URL, QDRANT_API_KEY, KO_LIVE_SCENARIOS_ROOT (grep leg
// repo root), KO_SEARCH_FILE (search.json override).

func searchProvider(t *testing.T) pkg.ProviderConfig {
	t.Helper()
	path := os.Getenv("KO_SEARCH_FILE")
	if path == "" {
		// Resolved relative to this package dir: .../api/internal/aisearch.
		path = filepath.Join("..", "..", "..", ".vrooli", "search.json")
	}
	file, err := pkg.LoadSearchFile(path)
	if err != nil {
		t.Fatalf("load search.json %s: %v", path, err)
	}
	provider, ok := file.Provider("knowledge-observatory.docs")
	if !ok {
		t.Fatalf("provider knowledge-observatory.docs not found in %s", path)
	}
	return provider
}

func TestSearchSSOTWellFormed(t *testing.T) {
	provider := searchProvider(t)
	tuning := provider.ResolvedTuning()
	if tuning.Engine != pkg.EngineHybrid || tuning.RerankEnabled || tuning.RerankBlend {
		t.Errorf("docs tuning is not the expected default hybrid/rerank-off config: %+v", tuning)
	}
	if err := provider.Scoring.Validate(); err != nil {
		t.Fatalf("invalid scoring block: %v", err)
	}
	if err := provider.Tests.Validate(); err != nil {
		t.Fatalf("invalid test corpus: %v", err)
	}
	policy := provider.ResolvedScoring()
	if policy.GateK != 5 || policy.RecallTarget != 0.8 || policy.DeepK < policy.GateK {
		t.Fatalf("unexpected docs scoring policy: %+v", policy)
	}
	if len(provider.Tests.Cases) != 23 {
		t.Fatalf("docs corpus cases = %d, want 23", len(provider.Tests.Cases))
	}
}

// liveStore builds the live embedder + vector store, failing the test (not
// skipping) when infra the harness depends on is down.
func liveStore(ctx context.Context, t *testing.T) (pkg.Embedder, pkg.VectorStore) {
	t.Helper()
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = pkg.DefaultQdrantURL
	}
	embedder := pkg.NewEmbedder(pkg.DefaultEmbedModel)
	if !embedder.Available(ctx) {
		t.Fatal("ollama embedder unavailable; start the ollama resource first")
	}
	store := pkg.NewVectorStore(qdrantURL, os.Getenv("QDRANT_API_KEY"), DefaultCollection)
	if !store.Available(ctx) {
		t.Fatal("qdrant unavailable; start the qdrant resource first")
	}
	count, err := store.CountPoints(ctx)
	if err != nil || count == 0 {
		t.Fatalf("collection %q is empty (count=%d, err=%v); run `reindex run` first", DefaultCollection, count, err)
	}
	t.Logf("evaluating against %d indexed chunks in %q", count, DefaultCollection)
	return embedder, store
}

func newGrepFallback(t *testing.T) TextFallback {
	t.Helper()
	root := scenariosRootForTest(t)
	svc, err := docsearch.NewService(root)
	if err != nil {
		t.Fatalf("docsearch.NewService(%q): %v", root, err)
	}
	return NewDocsearchFallback(svc)
}

func scenariosRootForTest(t *testing.T) string {
	t.Helper()
	if r := os.Getenv("KO_LIVE_SCENARIOS_ROOT"); r != "" {
		return r
	}
	// Package dir is .../scenarios/knowledge-observatory/api/internal/aisearch;
	// four parents up is the scenarios/ root.
	abs, err := filepath.Abs(filepath.Join("..", "..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve scenarios root: %v", err)
	}
	return abs
}

// TestLiveFullReindex drives an uncapped reconcile over the real corpus into the
// live vrooli-docs collection. Gated separately from the accuracy gate so it
// only runs when explicitly asked:
//
//	KO_REINDEX_FULL=1 go test ./internal/aisearch/ -run TestLiveFullReindex -v -timeout 60m
//
// Set KO_REINDEX_DRYRUN=1 to only plan without writing.
func TestLiveFullReindex(t *testing.T) {
	if os.Getenv("KO_REINDEX_FULL") == "" {
		t.Skip("set KO_REINDEX_FULL=1 to run a full reindex against the live collection")
	}
	dryRun := os.Getenv("KO_REINDEX_DRYRUN") != ""
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	embedder := pkg.NewEmbedder(pkg.DefaultEmbedModel)
	if !embedder.Available(ctx) {
		t.Fatal("ollama embedder unavailable")
	}
	qdrantURL := os.Getenv("QDRANT_URL")
	if qdrantURL == "" {
		qdrantURL = pkg.DefaultQdrantURL
	}
	store := pkg.NewVectorStore(qdrantURL, os.Getenv("QDRANT_API_KEY"), DefaultCollection)
	if !store.Available(ctx) {
		t.Fatal("qdrant unavailable")
	}
	idx, err := NewIndexer(Options{
		Embedder: embedder, VectorStore: store,
		ScenariosRoot:    scenariosRootForTest(t),
		MaxEmbedsPerTick: 0, // uncapped: index the whole delta in this pass
	})
	if err != nil {
		t.Fatalf("NewIndexer: %v", err)
	}
	if err := idx.EnsureCollection(ctx); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	before, _ := store.CountPoints(ctx)
	t.Logf("collection holds %d chunks before reindex (dryRun=%v)", before, dryRun)
	for pass := 1; ; pass++ {
		res, err := idx.Reindex(ctx, dryRun)
		if err != nil {
			t.Fatalf("reindex pass %d: %v", pass, err)
		}
		t.Logf("pass %d: planned=%d upserted=%d refreshed=%d deleted=%d deferred=%d errors=%d",
			pass, res.Planned, res.Upserted, res.Refreshed, res.Deleted, res.Deferred, len(res.Errors))
		for _, e := range res.Errors {
			t.Logf("  error: %s", e)
		}
		if dryRun || (res.Upserted == 0 && res.Deferred == 0) {
			break
		}
		if pass > 50 {
			t.Fatal("reindex did not converge in 50 passes")
		}
	}
	after, _ := store.CountPoints(ctx)
	t.Logf("collection holds %d chunks after reindex (delta %+d)", after, after-before)
}

func TestAccuracyCorpus(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 to run the accuracy corpus against the live index")
	}
	provider := searchProvider(t)
	policy := provider.ResolvedScoring()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	embedder, store := liveStore(ctx, t)
	svc := NewSearchService(ServiceOptions{
		Embedder: embedder, VectorStore: store,
		RerankEnabled: provider.ResolvedTuning().RerankEnabled,
		RerankBlend:   provider.ResolvedTuning().RerankBlend,
		Reranker:      NewDefaultReranker(),
		TextFallback:  newGrepFallback(t),
		Floor:         provider.ResolvedTuning().Floor.Config(),
	})

	report, err := pkg.GradeSuite(ctx, docGradeSearcher{svc: svc}, provider.Tests, policy)
	if err != nil {
		t.Fatalf("grade suite: %v", err)
	}
	for _, miss := range report.Misses {
		t.Logf("MISS %s (%q): rank=%d top=%.3f referential=%s err=%v",
			miss.CaseID, miss.Query, miss.ExpectedRank, miss.ObservedTopScore, miss.Referential, miss.Error)
	}
	for _, stale := range report.Stale {
		t.Logf("STALE %s (%q): expected path absent within deep_k=%d", stale.CaseID, stale.Query, policy.DeepK)
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

// TestAccuracyDiagnostic dumps ranked docs each query returns. It is not a
// grading path; the gate above owns assertions through GradeSuite.
func TestAccuracyDiagnostic(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 to run the ranking diagnostic")
	}
	provider := searchProvider(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	embedder, store := liveStore(ctx, t)
	svc := NewSearchService(ServiceOptions{
		Embedder: embedder, VectorStore: store,
		RerankEnabled: true,
		Reranker:      pkg.NewRerankerChain(pkg.NewCrossEncoderReranker("", "")),
		TextFallback:  newGrepFallback(t),
	})

	only := map[string]bool{}
	if raw := os.Getenv("KO_DIAG_CASES"); raw != "" {
		for _, id := range strings.Split(raw, ",") {
			only[strings.TrimSpace(id)] = true
		}
	}
	for _, tc := range provider.Tests.Cases {
		if len(only) > 0 && !only[tc.ID] {
			continue
		}
		resp, err := docGradeSearcher{svc: svc}.Search(ctx, pkg.SearchQuery{
			Query: tc.Query, Scope: tc.ResolvedScope(), Mode: pkg.ModeHybrid, Limit: 48,
		})
		if err != nil {
			t.Logf("[%s] error: %v", tc.ID, err)
			continue
		}
		top := resp.Results
		if len(top) > 8 {
			top = top[:8]
		}
		t.Logf("[%s] scope=%s want=%v", tc.ID, tc.Scope, tc.ExpectIDs)
		for i, h := range top {
			t.Logf("    #%d %s", i+1, h.ID)
		}
	}
}

type docGradeSearcher struct {
	svc *pkg.Service
}

func (s docGradeSearcher) Search(ctx context.Context, q pkg.SearchQuery, opts ...pkg.SearchOption) (pkg.SearchResponse, error) {
	resp, err := s.svc.Search(ctx, q, opts...)
	if err != nil {
		return resp, err
	}
	return docLevelResponse(resp), nil
}

func docLevelResponse(resp pkg.SearchResponse) pkg.SearchResponse {
	seen := map[string]bool{}
	docs := make([]pkg.SearchResult, 0, len(resp.Results))
	for _, h := range resp.Results {
		p := h.RelativePath
		if p == "" {
			p = h.Path
		}
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		h.ID = p
		h.RelativePath = p
		h.Path = p
		docs = append(docs, h)
	}
	resp.Results = docs
	resp.Total = len(docs)
	return resp
}

// TestScopeFiltering asserts scope isolation: scenario scope never leaks other
// scenarios, path scope keeps only the prefix, and global reaches project-level
// /docs.
func TestScopeFiltering(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 to run scope-filter validation against the live index")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	embedder, store := liveStore(ctx, t)
	svc := NewSearchService(ServiceOptions{Embedder: embedder, VectorStore: store, RerankEnabled: true, Reranker: NewDefaultReranker()})

	scoped, err := svc.Search(ctx, pkg.SearchQuery{
		Query: "architecture and seams", Mode: pkg.ModeHybrid,
		Scope: pkg.Scope{Kind: pkg.ScopeScenario, Value: "cli-health"}, Limit: 25,
	})
	if err != nil {
		t.Fatalf("scenario-scoped search: %v", err)
	}
	if len(scoped.Results) == 0 {
		t.Error("scenario scope returned no results")
	}
	for _, h := range scoped.Results {
		scn, _ := h.Payload[MetaScenario].(string)
		if scn != "cli-health" {
			t.Errorf("scenario scope leaked a hit from scenario %q: %s", scn, h.RelativePath)
		}
	}

	pathScoped, err := svc.Search(ctx, pkg.SearchQuery{
		Query: "platform architecture", Mode: pkg.ModeHybrid,
		Scope: pkg.Scope{Kind: pkg.ScopePath, Value: "docs/concepts"}, Limit: 25,
	})
	if err != nil {
		t.Fatalf("path-scoped search: %v", err)
	}
	if len(pathScoped.Results) == 0 {
		t.Error("path scope returned no results")
	}
	for _, h := range pathScoped.Results {
		if !strings.HasPrefix(h.RelativePath, "docs/concepts") && !strings.HasPrefix(h.Path, "docs/concepts") {
			t.Errorf("path scope leaked a hit outside docs/concepts: %s", h.RelativePath)
		}
	}

	global, err := svc.Search(ctx, pkg.SearchQuery{
		Query: "overall platform architecture and operating model", Mode: pkg.ModeHybrid,
		Scope: pkg.Scope{Kind: pkg.ScopeGlobal}, Limit: 25,
	})
	if err != nil {
		t.Fatalf("global search: %v", err)
	}
	if len(global.Results) == 0 {
		t.Fatal("global scope returned no results")
	}
	var sawProjectScope bool
	for _, h := range global.Results {
		if sc, _ := h.Payload[MetaScope].(string); sc == ScopeProject {
			sawProjectScope = true
			break
		}
	}
	if !sawProjectScope {
		t.Error("global scope surfaced no project-scope (root /docs) document")
	}
}

// TestRerankerDegradation asserts the degradation chain: with the cross-encoder
// unreachable the chain falls through to the LLM reranker, and with both
// unreachable search still returns fused-order results with reranker "none".
func TestRerankerDegradation(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 to run reranker degradation validation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	embedder, store := liveStore(ctx, t)

	deadCross := pkg.NewCrossEncoderRerankerWithClient("http://127.0.0.1:0", &http.Client{Timeout: 2 * time.Second})
	llmChain := pkg.NewRerankerChain(deadCross, pkg.NewLLMReranker(""))
	if got := llmChain.ActiveName(ctx); !strings.HasPrefix(got, "llm:") {
		t.Errorf("cross-encoder down: expected an llm:* active reranker, got %q", got)
	}

	deadLLM := pkg.NewLLMRerankerWithRunner("", func(context.Context, []string, []byte) ([]byte, error) {
		return nil, fmt.Errorf("ollama down (simulated)")
	})
	noneChain := pkg.NewRerankerChain(deadCross, deadLLM)
	if got := noneChain.ActiveName(ctx); got != "none" {
		t.Errorf("both rerankers down: expected active=none, got %q", got)
	}

	svc := NewSearchService(ServiceOptions{Embedder: embedder, VectorStore: store, RerankEnabled: true, Reranker: noneChain})
	resp, err := svc.Search(ctx, pkg.SearchQuery{Query: "documentation chunking and indexing", Mode: pkg.ModeHybrid, Limit: 5})
	if err != nil {
		t.Fatalf("search with no reranker available: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Error("hybrid search returned nothing when both rerankers are down (should keep fused order)")
	}
	if resp.Reranker != "none" {
		t.Errorf("reranker reported %q; want none when the chain has no available leg", resp.Reranker)
	}
}
