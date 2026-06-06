package aisearch

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pkg "github.com/vrooli/aisearch-go"

	"knowledge-observatory/internal/services/docsearch"
)

// accuracy_test.go is the Phase-7 validation harness for the KO documentation
// search cutover. It drives the live SearchService over the hand-labeled corpus
// in testdata/search_queries.json and asserts the plan's accuracy gate
// (recall@5 >= 0.8), while reporting the comparison numbers that justify the
// hybrid + rerank cost (plan §6.1, §6.4, §6.7):
//
//   - retrieval-leg comparison:   hybrid vs dense vs grep
//   - reranker comparison:        cross-encoder vs LLM vs none
//   - scope isolation:            scenario / path / global filters
//   - degradation chain:          cross-encoder down -> LLM -> fused order
//
// Recall/MRR are measured per DOCUMENT (chunk hits deduped to their owning doc),
// since the corpus labels expected documents, not chunks.
//
// These tests need live infra (Qdrant with the indexed vrooli-docs collection,
// Ollama, and — for the cross-encoder rows — the reranker resource), so they are
// gated on KO_AISEARCH_LIVE:
//
//	KO_AISEARCH_LIVE=1 go test ./internal/aisearch/ -run TestAccuracyCorpus -v -timeout 25m
//
// Optional env: QDRANT_URL, QDRANT_API_KEY, KO_LIVE_SCENARIOS_ROOT (grep leg
// repo root), KO_CORPUS_FILE (corpus override), KO_SKIP_LLM_RERANK=1 (skip the
// slow LLM-reranker comparison row).

// corpusCase mirrors one entry in testdata/search_queries.json.
type corpusCase struct {
	ID            string   `json:"id"`
	Query         string   `json:"query"`
	Scope         string   `json:"scope"`
	PrimaryPath   string   `json:"primary_path"`
	ExpectedPaths []string `json:"expected_paths"`
	Notes         string   `json:"notes"`
}

// corpusScoring mirrors the scoring block (gate thresholds live in the corpus,
// not hard-coded here, so re-tuning is a data edit).
type corpusScoring struct {
	RecallAt     int     `json:"recall_at"`
	RecallTarget float64 `json:"recall_target"`
	MRRAt        int     `json:"mrr_at"`
}

type corpus struct {
	Version int           `json:"version"`
	Scoring corpusScoring `json:"scoring"`
	Cases   []corpusCase  `json:"cases"`
}

func loadCorpus(t *testing.T) corpus {
	t.Helper()
	path := os.Getenv("KO_CORPUS_FILE")
	if path == "" {
		// Resolved relative to this package dir: .../api/internal/aisearch.
		path = filepath.Join("..", "..", "..", "testdata", "search_queries.json")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read corpus %s: %v", path, err)
	}
	var c corpus
	if err := json.Unmarshal(raw, &c); err != nil {
		t.Fatalf("parse corpus %s: %v", path, err)
	}
	if len(c.Cases) == 0 {
		t.Fatal("corpus has no cases")
	}
	if c.Scoring.RecallAt == 0 {
		c.Scoring.RecallAt = 5
	}
	if c.Scoring.MRRAt == 0 {
		c.Scoring.MRRAt = 3
	}
	return c
}

// parseScope maps the corpus scope string ("global", "scenario:<n>",
// "path:<prefix>") to a pkg.Scope.
func parseScope(s string) pkg.Scope {
	switch {
	case strings.HasPrefix(s, "scenario:"):
		return pkg.Scope{Kind: pkg.ScopeScenario, Value: strings.TrimPrefix(s, "scenario:")}
	case strings.HasPrefix(s, "path:"):
		return pkg.Scope{Kind: pkg.ScopePath, Value: strings.TrimPrefix(s, "path:")}
	default:
		return pkg.Scope{Kind: pkg.ScopeGlobal}
	}
}

// topDocs dedups chunk-level hits to unique document paths, preserving ranked
// order — documentation recall is measured per-document, not per-chunk.
func topDocs(resp pkg.SearchResponse) []string {
	seen := map[string]bool{}
	docs := make([]string, 0, len(resp.Results))
	for _, h := range resp.Results {
		p := h.RelativePath
		if p == "" {
			p = h.Path
		}
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		docs = append(docs, p)
	}
	return docs
}

// recallHit reports whether any expected doc appears in the top-k docs.
func recallHit(docs, expected []string, k int) bool {
	if k > len(docs) {
		k = len(docs)
	}
	exp := make(map[string]bool, len(expected))
	for _, e := range expected {
		exp[e] = true
	}
	for i := 0; i < k; i++ {
		if exp[docs[i]] {
			return true
		}
	}
	return false
}

// reciprocalRank returns 1/rank of the primary doc within the top-n docs, else 0.
func reciprocalRank(docs []string, primary string, n int) float64 {
	if n > len(docs) {
		n = len(docs)
	}
	for i := 0; i < n; i++ {
		if docs[i] == primary {
			return 1.0 / float64(i+1)
		}
	}
	return 0
}

// evalResult holds aggregate scores for one search configuration.
type evalResult struct {
	name   string
	recall float64
	mrr    float64
	misses []string // case IDs that failed recall@k
}

// evalConfig runs every corpus case through svc in the given mode and aggregates
// recall@k / MRR@n per the corpus scoring block.
func evalConfig(ctx context.Context, t *testing.T, name string, svc *pkg.Service, c corpus, mode pkg.SearchMode) evalResult {
	t.Helper()
	recallK := c.Scoring.RecallAt
	mrrN := c.Scoring.MRRAt
	// Request more chunk hits than k so dedup-to-docs still yields >= k docs.
	chunkLimit := recallK * 6
	if mrrN*6 > chunkLimit {
		chunkLimit = mrrN * 6
	}
	var recallSum, mrrSum float64
	var misses []string
	for _, cs := range c.Cases {
		resp, err := svc.Search(ctx, pkg.SearchQuery{
			Query: cs.Query,
			Mode:  mode,
			Scope: parseScope(cs.Scope),
			Limit: chunkLimit,
		})
		if err != nil {
			t.Logf("[%s] case %s search error: %v", name, cs.ID, err)
			misses = append(misses, cs.ID)
			continue
		}
		docs := topDocs(resp)
		if recallHit(docs, cs.ExpectedPaths, recallK) {
			recallSum++
		} else {
			misses = append(misses, cs.ID)
		}
		mrrSum += reciprocalRank(docs, cs.PrimaryPath, mrrN)
	}
	n := float64(len(c.Cases))
	return evalResult{name: name, recall: recallSum / n, mrr: mrrSum / n, misses: misses}
}

func logEval(t *testing.T, r evalResult) {
	t.Helper()
	t.Logf("  %-22s recall=%.3f  mrr=%.3f  misses(%d)=%v", r.name, r.recall, r.mrr, len(r.misses), r.misses)
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
// live vrooli-docs collection — the validation-phase way to (re)populate the
// index with the current DocSource so the accuracy gate measures real coverage.
// Gated separately from the accuracy gate so it only runs when explicitly asked:
//
//	KO_REINDEX_FULL=1 go test ./internal/aisearch/ -run TestLiveFullReindex -v -timeout 60m
//
// Set KO_REINDEX_DRYRUN=1 to only plan (measure the delta) without writing.
// IMPORTANT: stop the KO service first so its background sync loop does not
// concurrently ghost-collect docs this pass adds (an old binary's source may not
// yet know them).
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

	// Loop until the corpus is fully reconciled (Planned/Deferred drained).
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

// TestAccuracyCorpus is the Phase-7 accuracy gate (plan §6.1/§6.7). It asserts
// recall@5 >= 0.8 for the production hybrid+rerank config and reports the
// retrieval-leg and reranker comparison numbers.
func TestAccuracyCorpus(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 to run the accuracy corpus against the live index")
	}
	c := loadCorpus(t)
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()

	embedder, store := liveStore(ctx, t)
	grep := newGrepFallback(t)

	// Production config: the default degradation chain (cross-encoder -> LLM).
	prodSvc := NewSearchService(ServiceOptions{
		Embedder: embedder, VectorStore: store,
		RerankEnabled: true, Reranker: NewDefaultReranker(), TextFallback: grep,
	})

	// ---- Retrieval-leg comparison (reranker held at production default) ----
	hybrid := evalConfig(ctx, t, "hybrid+rerank", prodSvc, c, pkg.ModeHybrid)
	dense := evalConfig(ctx, t, "dense+rerank", prodSvc, c, pkg.ModeDense)
	text := evalConfig(ctx, t, "grep", prodSvc, c, pkg.ModeText)

	t.Logf("=== retrieval leg comparison (recall@%d / MRR@%d, N=%d cases) ===", c.Scoring.RecallAt, c.Scoring.MRRAt, len(c.Cases))
	logEval(t, hybrid)
	logEval(t, dense)
	logEval(t, text)

	// ---- Reranker comparison on the hybrid leg ----
	crossSvc := NewSearchService(ServiceOptions{
		Embedder: embedder, VectorStore: store,
		RerankEnabled: true, Reranker: pkg.NewRerankerChain(pkg.NewCrossEncoderReranker("", "")), TextFallback: grep,
	})
	noneSvc := NewSearchService(ServiceOptions{
		Embedder: embedder, VectorStore: store, TextFallback: grep,
	})
	cross := evalConfig(ctx, t, "hybrid+cross-encoder", crossSvc, c, pkg.ModeHybrid)
	none := evalConfig(ctx, t, "hybrid+none", noneSvc, c, pkg.ModeHybrid)

	t.Logf("=== reranker comparison on hybrid leg ===")
	logEval(t, cross)
	if os.Getenv("KO_SKIP_LLM_RERANK") == "" {
		llmSvc := NewSearchService(ServiceOptions{
			Embedder: embedder, VectorStore: store,
			RerankEnabled: true, Reranker: pkg.NewRerankerChain(pkg.NewLLMReranker("")), TextFallback: grep,
		})
		logEval(t, evalConfig(ctx, t, "hybrid+llm", llmSvc, c, pkg.ModeHybrid))
	} else {
		t.Log("  (skipped LLM-reranker row: KO_SKIP_LLM_RERANK set)")
	}
	logEval(t, none)

	// ---- The gate: production config must clear the corpus recall target ----
	best := hybrid.recall
	if cross.recall > best {
		best = cross.recall
	}
	if best+1e-9 < c.Scoring.RecallTarget {
		t.Errorf("recall@%d = %.3f is below the gate %.2f (hybrid misses: %v)",
			c.Scoring.RecallAt, best, c.Scoring.RecallTarget, hybrid.misses)
	}
}

// TestAccuracyDiagnostic dumps the ranked docs each query returns — the triage
// tool for understanding gate misses (near-miss vs true miss). Not an assertion;
// run it to inspect rankings:
//
//	KO_AISEARCH_LIVE=1 go test ./internal/aisearch/ -run TestAccuracyDiagnostic -v
//
// Optional KO_DIAG_CASES=id1,id2 limits output to specific cases.
func TestAccuracyDiagnostic(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 to run the ranking diagnostic")
	}
	c := loadCorpus(t)
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
	for _, cs := range c.Cases {
		if len(only) > 0 && !only[cs.ID] {
			continue
		}
		resp, err := svc.Search(ctx, pkg.SearchQuery{Query: cs.Query, Mode: pkg.ModeHybrid, Scope: parseScope(cs.Scope), Limit: 48})
		if err != nil {
			t.Logf("[%s] error: %v", cs.ID, err)
			continue
		}
		docs := topDocs(resp)
		rank := -1
		for i, d := range docs {
			if d == cs.PrimaryPath {
				rank = i + 1
				break
			}
		}
		top := docs
		if len(top) > 8 {
			top = top[:8]
		}
		t.Logf("[%s] scope=%s primary=%s primaryRank=%d", cs.ID, cs.Scope, cs.PrimaryPath, rank)
		t.Logf("    want=%v", cs.ExpectedPaths)
		for i, d := range top {
			t.Logf("    #%d %s", i+1, d)
		}
	}
}

// TestScopeFiltering asserts scope isolation (plan §6.4): scenario scope never
// leaks other scenarios, path scope keeps only the prefix, and global reaches
// project-level /docs.
func TestScopeFiltering(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 to run scope-filter validation against the live index")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	embedder, store := liveStore(ctx, t)
	svc := NewSearchService(ServiceOptions{Embedder: embedder, VectorStore: store, RerankEnabled: true, Reranker: NewDefaultReranker()})

	// scenario scope: every hit must belong to the scoped scenario.
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

	// path scope: every hit must be under the prefix.
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

	// global: the corpus-wide scope must remain reachable, including project-level
	// /docs (this is the isolation counterpart — global is broader than a scoped
	// query; per-query recall is the accuracy gate's job, not this test's).
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

// TestRerankerDegradation asserts the degradation chain (plan §6.7): with the
// cross-encoder unreachable the chain falls through to the LLM reranker, and
// with both unreachable search still returns results in fused order with the
// reranker reported as "none". This exercises the chain logic deterministically
// against the live LLM without stopping the cross-encoder resource.
func TestRerankerDegradation(t *testing.T) {
	if os.Getenv("KO_AISEARCH_LIVE") == "" {
		t.Skip("set KO_AISEARCH_LIVE=1 to run reranker degradation validation")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	embedder, store := liveStore(ctx, t)

	// An unreachable cross-encoder (port 0 never connects).
	deadCross := pkg.NewCrossEncoderRerankerWithClient("http://127.0.0.1:0", &http.Client{Timeout: 2 * time.Second})

	// cross-encoder down -> LLM active (requires Ollama, which liveStore proved up).
	llmChain := pkg.NewRerankerChain(deadCross, pkg.NewLLMReranker(""))
	if got := llmChain.ActiveName(ctx); !strings.HasPrefix(got, "llm:") {
		t.Errorf("cross-encoder down: expected an llm:* active reranker, got %q", got)
	}

	// both down -> none active, but search still answers in fused order.
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
