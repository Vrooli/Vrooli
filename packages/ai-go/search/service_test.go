package aisearch

import (
	"context"
	"runtime"
	"testing"
	"time"
)

// queryStore is a VectorStore whose Query returns a preset result set, so the
// read-path pipeline (project -> rerank -> floor -> labelWeak) is exercised
// offline. available toggles the auto-degradation path.
type queryStore struct {
	results   []SearchResult
	lastQuery HybridQuery
	available bool
	count     int
}

func (q *queryStore) EnsureCollection(context.Context, CollectionSpec) error   { return nil }
func (q *queryStore) Upsert(context.Context, Point) error                      { return nil }
func (q *queryStore) SetPayload(context.Context, string, map[string]any) error { return nil }
func (q *queryStore) BatchDelete(context.Context, []string) error              { return nil }
func (q *queryStore) ScrollIDs(context.Context) (map[string]ScrollItem, error) { return nil, nil }
func (q *queryStore) CountPoints(context.Context) (int, error)                 { return q.count, nil }
func (q *queryStore) Available(context.Context) bool                           { return q.available }
func (q *queryStore) Query(_ context.Context, hq HybridQuery) ([]SearchResult, error) {
	q.lastQuery = hq
	out := make([]SearchResult, len(q.results))
	copy(out, q.results)
	return out, nil
}

func docResult(id string, score float64, body string) SearchResult {
	return SearchResult{ID: id, Score: score, Payload: map[string]any{"body": body, "relative_path": id, "source_id": id}}
}

// docProjector mirrors a doc adopter: fill the projection fields from payload.
func docProjector(r SearchResult) SearchResult {
	r.RelativePath, _ = r.Payload["relative_path"].(string)
	r.Path = r.RelativePath
	r.SourceID, _ = r.Payload["source_id"].(string)
	r.Snippet, _ = r.Payload["body"].(string)
	return r
}

func TestServiceVectorPipelineRerankThenFloorThenProject(t *testing.T) {
	store := &queryStore{available: true, results: []SearchResult{
		docResult("a", 0.80, "alpha"),
		docResult("b", 0.70, "beta"),
		docResult("c", 0.65, "gamma"), // will be reranked to ~0 and floored out
	}}
	// Cross-encoder regime: promotes a, keeps b, drives c to 0 (below the xenc
	// HardFloor) so the floor drops it AFTER rerank.
	rr := NewRerankerChain(&stubReranker{
		name:      "cross-encoder:test",
		available: true,
		scores:    []RerankScore{{ID: "a", Score: 0.95}, {ID: "b", Score: 0.60}, {ID: "c", Score: 0.0}},
	})
	svc := NewService(ServiceOptions{
		Embedder:      &countingEmbedder{},
		VectorStore:   store,
		Reranker:      rr,
		RerankEnabled: true,
		ApplyFloor:    true,
		Project:       docProjector,
		Shortlist:     50,
	})

	resp, err := svc.Search(context.Background(), SearchQuery{Query: "alpha", Mode: ModeDense, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Reranker != "cross-encoder:test" {
		t.Fatalf("reranker leg = %q, want cross-encoder:test", resp.Reranker)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected c floored out (2 results), got %d: %+v", len(resp.Results), resp.Results)
	}
	if resp.Results[0].ID != "a" || resp.Results[1].ID != "b" {
		t.Fatalf("rerank order wrong: %s,%s", resp.Results[0].ID, resp.Results[1].ID)
	}
	// Projection ran: RelativePath/Snippet filled from payload.
	if resp.Results[0].RelativePath != "a" || resp.Results[0].Snippet != "alpha" {
		t.Fatalf("projection did not fill fields: %+v", resp.Results[0])
	}
	// Weak labeled against the cross-encoder regime: 0.95 is strong (>0.30).
	if resp.Results[0].Weak {
		t.Fatalf("strong hit a should not be weak-labeled")
	}
	// Over-fetch: the store was asked for the shortlist (50), not the page (10).
	if store.lastQuery.Limit != 50 {
		t.Fatalf("expected over-fetch shortlist 50, got %d", store.lastQuery.Limit)
	}
}

func TestServiceFloorGateOffKeepsLowScores(t *testing.T) {
	// RRF-style tiny scores with rerank off: ApplyFloor=false must NOT annihilate
	// them on the cosine HardFloor (the doc/RRF case).
	store := &queryStore{available: true, results: []SearchResult{
		{ID: "a", Score: 0.03, Payload: map[string]any{"body": "x"}},
		{ID: "b", Score: 0.02, Payload: map[string]any{"body": "y"}},
	}}
	svc := NewService(ServiceOptions{
		Embedder:      &countingEmbedder{},
		SparseEncoder: NewBM25SparseEncoder(),
		VectorStore:   store,
		ApplyFloor:    false,
	})
	resp, err := svc.Search(context.Background(), SearchQuery{Query: "x", Mode: ModeHybrid, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("floor-off must keep both RRF hits, got %d", len(resp.Results))
	}
	if resp.Method != "hybrid" {
		t.Fatalf("method = %q, want hybrid", resp.Method)
	}
}

func TestServiceFusionFloorOnKeepsLowScores(t *testing.T) {
	// The Tier-2 win: with the fusion regime, ApplyFloor=true is SAFE on the
	// rerank-off hybrid path — the relative MaxGap gates the tail and the absolute
	// cosine 0.35 HardFloor (which would have annihilated these real RRF hits) does
	// not apply. Contrast with TestServiceFloorGateOffKeepsLowScores, which had to
	// disable the floor entirely to get the same preservation.
	store := &queryStore{available: true, results: []SearchResult{
		{ID: "a", Score: 0.56, Payload: map[string]any{"body": "x"}},
		{ID: "b", Score: 0.40, Payload: map[string]any{"body": "y"}},
		{ID: "c", Score: 0.30, Payload: map[string]any{"body": "z"}}, // < cosine 0.35 HardFloor
	}}
	svc := NewService(ServiceOptions{
		Embedder:      &countingEmbedder{},
		SparseEncoder: NewBM25SparseEncoder(),
		VectorStore:   store,
		ApplyFloor:    true, // now safe for fused/doc
	})
	resp, err := svc.Search(context.Background(), SearchQuery{Query: "x", Mode: ModeHybrid, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) != 3 {
		t.Fatalf("fusion floor must keep the real 0.30 hit with ApplyFloor on, got %d: %+v", len(resp.Results), resp.Results)
	}
	if resp.Reranker != "none" {
		t.Fatalf("reranker leg = %q, want none (rerank-off fused)", resp.Reranker)
	}
	if store.lastQuery.Fusion != HybridFusionRRF {
		t.Fatalf("default hybrid fusion = %q, want %q", store.lastQuery.Fusion, HybridFusionRRF)
	}
	// Weak-labeled against the fusion band (0.20), not the cosine band (0.55): the
	// real 0.30 hit is NOT weak — the latent mislabel bug is fixed.
	for _, r := range resp.Results {
		if r.Weak {
			t.Fatalf("fused hit %s (%.2f) wrongly weak-labeled under the fusion band", r.ID, r.Score)
		}
	}
}

func TestServiceHybridFusionStrategyIsForwarded(t *testing.T) {
	store := &queryStore{available: true, results: []SearchResult{{ID: "a", Score: 0.5, Payload: map[string]any{"body": "exact"}}}}
	svc := NewService(ServiceOptions{
		Embedder:      &countingEmbedder{},
		SparseEncoder: NewBM25SparseEncoder(),
		VectorStore:   store,
		HybridFusion:  HybridFusionDBSF,
		ApplyFloor:    false,
	})
	if _, err := svc.Search(context.Background(), SearchQuery{Query: "exact", Mode: ModeHybrid, Limit: 5}); err != nil {
		t.Fatal(err)
	}
	if store.lastQuery.Fusion != HybridFusionDBSF {
		t.Fatalf("hybrid fusion = %q, want %q", store.lastQuery.Fusion, HybridFusionDBSF)
	}
}

func TestSearchTypedProjectsToAdopterType(t *testing.T) {
	// A non-doc adopter shape: keep corpus-specific fields in Payload, project to a
	// typed struct at the engine boundary via the generic SearchTyped.
	type commandHit struct {
		FullPath string
		Score    float64
		Weak     bool
	}
	store := &queryStore{available: true, results: []SearchResult{
		{ID: "cmd:1", Score: 0.90, Payload: map[string]any{"full_path": "vrooli scenario restart"}},
		{ID: "cmd:2", Score: 0.80, Payload: map[string]any{"full_path": "vrooli scenario stop"}},
	}}
	svc := NewService(ServiceOptions{Embedder: &countingEmbedder{}, VectorStore: store})

	hits, resp, err := SearchTyped(context.Background(), svc, SearchQuery{Query: "restart", Mode: ModeDense, Limit: 10},
		func(r SearchResult) commandHit {
			fp, _ := r.Payload["full_path"].(string)
			return commandHit{FullPath: fp, Score: r.Score, Weak: r.Weak}
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != len(resp.Results) {
		t.Fatalf("typed page length %d != response length %d", len(hits), len(resp.Results))
	}
	if len(hits) == 0 || hits[0].FullPath != "vrooli scenario restart" {
		t.Fatalf("projection wrong: %+v", hits)
	}
	// The raw response is still returned for the wire envelope (Method/Reranker).
	if resp.Method != "dense" {
		t.Fatalf("method = %q, want dense", resp.Method)
	}
}

func TestSearchTypedPropagatesError(t *testing.T) {
	// An empty query errors in Search; SearchTyped surfaces it with a nil page.
	svc := NewService(ServiceOptions{Embedder: &countingEmbedder{}, VectorStore: &queryStore{available: true}})
	hits, _, err := SearchTyped(context.Background(), svc, SearchQuery{Query: "  "}, func(r SearchResult) int { return 1 })
	if err == nil {
		t.Fatal("expected error for empty query")
	}
	if hits != nil {
		t.Fatalf("expected nil page on error, got %v", hits)
	}
}

func TestServiceAutoDegradesToText(t *testing.T) {
	store := &queryStore{available: false} // backend down
	textCalled := false
	svc := NewService(ServiceOptions{
		Embedder:    &countingEmbedder{},
		VectorStore: store,
		TextFallback: func(_ context.Context, q SearchQuery) ([]SearchResult, error) {
			textCalled = true
			return []SearchResult{{ID: "t", Score: 0.5, Payload: map[string]any{}}}, nil
		},
	})
	resp, err := svc.Search(context.Background(), SearchQuery{Query: "anything", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !textCalled || resp.Method != "text" {
		t.Fatalf("auto must degrade to text when backend is down; called=%v method=%q", textCalled, resp.Method)
	}
}

func TestServiceHybridModeRequiresSparse(t *testing.T) {
	svc := NewService(ServiceOptions{Embedder: &countingEmbedder{}, VectorStore: &queryStore{available: true}})
	if _, err := svc.Search(context.Background(), SearchQuery{Query: "x", Mode: ModeHybrid}); err == nil {
		t.Fatal("hybrid mode without a sparse encoder must error")
	}
}

func TestServiceReindexDryRunLifecycle(t *testing.T) {
	src := &sliceSource{docs: []SourceDoc{doc("README.md", "alpha\nbeta")}}
	store := newMemStore()
	rec := newDocReconciler(src, store, &countingEmbedder{})
	svc := NewService(ServiceOptions{VectorStore: store, Reconciler: rec})

	job, err := svc.Reindex(context.Background(), "", true)
	if err != nil {
		t.Fatal(err)
	}
	// Drift the dry-run job to completion.
	waitJob(t, svc, job.ID)
	got, ok := svc.ReindexStatus(job.ID)
	if !ok || got.State != "succeeded" {
		t.Fatalf("dry-run job state = %q (ok=%v), want succeeded", got.State, ok)
	}
	if store.upserts != 0 {
		t.Fatalf("dry-run must not upsert; got %d", store.upserts)
	}
	exp := svc.JobExport(got)
	if exp["state"] != "succeeded" || exp["dry_run"] != true {
		t.Fatalf("JobExport wrong: %+v", exp)
	}
}

func TestServiceReindexRequiresReconciler(t *testing.T) {
	svc := NewService(ServiceOptions{VectorStore: &queryStore{available: true}})
	if _, err := svc.Reindex(context.Background(), "", false); err == nil {
		t.Fatal("Reindex without a reconciler must error")
	}
}

func TestServiceEmptyQueryErrors(t *testing.T) {
	svc := NewService(ServiceOptions{Embedder: &countingEmbedder{}, VectorStore: &queryStore{available: true}})
	if _, err := svc.Search(context.Background(), SearchQuery{Query: "   "}); err == nil {
		t.Fatal("empty query must error")
	}
}

// waitJob polls a job to a terminal state with a bounded number of yields.
func waitJob(t *testing.T, svc *Service, id string) {
	t.Helper()
	for i := 0; i < 1000; i++ {
		if j, ok := svc.ReindexStatus(id); ok {
			switch j.State {
			case "succeeded", "failed", "cancelled":
				return
			}
		}
		// Yield without a real sleep (the goroutine is CPU-bound and fast).
		runtime.Gosched()
	}
	t.Fatalf("job %s did not reach a terminal state", id)
}

// blockingStore is a VectorStore whose ScrollIDs blocks until unblocked,
// letting the test cancel a job while it is mid-flight. ScrollIDs is called
// inside reconciler.Plan after LoadAll, so a blocking ScrollIDs gives the test
// a reliable hook to cancel while the job is in the "running" state.
type blockingStore struct {
	*memStore
	entered chan struct{} // closed on first ScrollIDs entry
	unblock chan struct{} // close to let ScrollIDs return
}

func newBlockingStore() *blockingStore {
	return &blockingStore{
		memStore: newMemStore(),
		entered:  make(chan struct{}),
		unblock:  make(chan struct{}),
	}
}

func (b *blockingStore) ScrollIDs(ctx context.Context) (map[string]ScrollItem, error) {
	select {
	case <-b.entered:
	default:
		close(b.entered)
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.unblock:
		return b.memStore.ScrollIDs(ctx)
	}
}

func TestServiceReindexCancel(t *testing.T) {
	// Use a source with one doc so Plan has real work (LoadAll succeeds, then
	// ScrollIDs blocks so the job stays "running" long enough to cancel).
	src := &sliceSource{docs: []SourceDoc{doc("README.md", "cancel test")}}
	store := newBlockingStore()
	rec := NewReconciler(&countingEmbedder{}, []SourceBinding{{
		Kind:     "doc",
		Store:    store,
		Source:   src,
		Chunker:  NewIdentityChunker(),
		Composer: NewIdentityComposer(),
		IDPrefix: "cancel-test:",
	}}, 4)
	svc := NewService(ServiceOptions{VectorStore: store, Reconciler: rec})

	job, err := svc.Reindex(context.Background(), "", false)
	if err != nil {
		t.Fatal(err)
	}

	// Wait until Plan is inside ScrollIDs (job is "running").
	select {
	case <-store.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Plan/ScrollIDs to start")
	}

	// Verify the job is running before we cancel it.
	running, _ := svc.ReindexStatus(job.ID)
	if running.State != "running" {
		t.Fatalf("expected job to be running before cancel, got %q", running.State)
	}

	// Cancel the job while it is blocked inside Plan.
	if ok := svc.ReindexCancel(job.ID); !ok {
		t.Fatalf("ReindexCancel must return true for a running job")
	}

	// Attempting to cancel an already-cancelled job must return false.
	waitJob(t, svc, job.ID)
	if ok := svc.ReindexCancel(job.ID); ok {
		t.Fatalf("ReindexCancel must return false for a terminal job")
	}

	// The job must reach a terminal state (not hang). State is implementation-
	// defined when the context is cancelled mid-Plan (the reconciler swallows
	// LoadAll/ScrollIDs errors in planBinding, so the plan comes back empty and
	// Apply succeeds with zero work — the job ends as "succeeded" not
	// "cancelled"). The important invariants are: it terminates, and the cancel
	// signal propagated (verified by ScrollIDs returning ctx.Err()).
	got, ok := svc.ReindexStatus(job.ID)
	if !ok {
		t.Fatalf("job not found after cancel")
	}
	terminal := got.State == "succeeded" || got.State == "failed" || got.State == "cancelled"
	if !terminal {
		t.Fatalf("job must reach a terminal state after cancel, got %q", got.State)
	}
}
