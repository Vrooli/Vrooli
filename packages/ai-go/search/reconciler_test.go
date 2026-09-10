package aisearch

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- fakes ------------------------------------------------------------------

// memStore is an in-memory VectorStore that honors the same payload contract as
// qdrant (ScrollIDs projects payload_hash/source_id/source_hash/chunk_total),
// so the reconciler's drift logic is exercised end-to-end.
type memStore struct {
	mu      sync.Mutex
	points  map[string]Point
	upserts int
	refresh int
	deletes int
}

func newMemStore() *memStore { return &memStore{points: map[string]Point{}} }

// upsertCount reads the upsert counter under the lock — for tests that observe
// the store from a goroutine concurrent with a running SyncLoop.
func (m *memStore) upsertCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.upserts
}

func (m *memStore) EnsureCollection(context.Context, CollectionSpec) error { return nil }

func (m *memStore) Upsert(_ context.Context, p Point) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.points[p.ID] = p
	m.upserts++
	return nil
}

func (m *memStore) SetPayload(_ context.Context, id string, payload map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.points[id]; ok {
		p.Payload = payload
		m.points[id] = p
	}
	m.refresh++
	return nil
}

func (m *memStore) BatchDelete(_ context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		delete(m.points, id)
	}
	m.deletes += len(ids)
	return nil
}

func (m *memStore) Query(context.Context, HybridQuery) ([]SearchResult, error) { return nil, nil }
func (m *memStore) CountPoints(context.Context) (int, error)                   { return len(m.points), nil }
func (m *memStore) Available(context.Context) bool                             { return true }

func (m *memStore) ScrollIDs(context.Context) (map[string]ScrollItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]ScrollItem, len(m.points))
	for id, p := range m.points {
		hash, _ := p.Payload[payloadHashKey].(string)
		srcID, _ := p.Payload[sourceIDKey].(string)
		srcHash, _ := p.Payload[sourceHashKey].(string)
		total, _ := p.Payload[chunkTotalKey].(int)
		out[id] = ScrollItem{PayloadHash: hash, SourceID: srcID, SourceHash: srcHash, ChunkTotal: total}
	}
	return out, nil
}

// countingEmbedder returns a fixed-length vector and counts calls.
type countingEmbedder struct {
	mu    sync.Mutex
	calls int
}

func (e *countingEmbedder) Embed(context.Context, string) ([]float64, error) {
	e.mu.Lock()
	e.calls++
	e.mu.Unlock()
	return []float64{0.1, 0.2, 0.3}, nil
}
func (e *countingEmbedder) Available(context.Context) bool { return true }

type concurrencyProbeEmbedder struct {
	mu      sync.Mutex
	active  int
	maximum int
	want    int
	release chan struct{}
	once    sync.Once
}

func (e *concurrencyProbeEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	e.mu.Lock()
	e.active++
	if e.active > e.maximum {
		e.maximum = e.active
	}
	if e.active == e.want {
		e.once.Do(func() { close(e.release) })
	}
	e.mu.Unlock()
	select {
	case <-e.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	e.mu.Lock()
	e.active--
	e.mu.Unlock()
	return []float64{float64(len(text))}, nil
}

func (e *concurrencyProbeEmbedder) Available(context.Context) bool { return true }

// lineChunker splits a doc body on "\n" — one chunk per line — to exercise the
// 1-source→N-chunk fan-out.
type lineChunker struct{}

func (lineChunker) Chunk(doc SourceDoc) ([]Chunk, error) {
	lines := strings.Split(doc.Body, "\n")
	out := make([]Chunk, 0, len(lines))
	for i, ln := range lines {
		out = append(out, Chunk{SourceID: doc.ID, Index: i, Body: ln, Meta: doc.Meta})
	}
	return out, nil
}

// sliceSource serves whatever docs it currently holds.
type sliceSource struct{ docs []SourceDoc }

func (s *sliceSource) LoadAll(context.Context) ([]SourceDoc, error) { return s.docs, nil }

// doc builds a SourceDoc whose ContentHash is derived from its body, so editing
// the body changes the content hash (as a real source would).
func doc(id, body string) SourceDoc {
	return SourceDoc{ID: id, Kind: "doc", Body: body, ContentHash: "h:" + body}
}

func newDocReconciler(src *sliceSource, store VectorStore, emb Embedder) *Reconciler {
	return NewReconciler(emb, []SourceBinding{{
		Kind:     "doc",
		Store:    store,
		Source:   src,
		Chunker:  lineChunker{},
		Composer: NewIdentityComposer(),
		IDPrefix: "knowledge-observatory:",
	}}, 4)
}

// --- tests ------------------------------------------------------------------

func TestReconcileFanOut(t *testing.T) {
	src := &sliceSource{docs: []SourceDoc{doc("README.md", "alpha\nbeta\ngamma")}}
	store, emb := newMemStore(), &countingEmbedder{}
	rec := newDocReconciler(src, store, emb)

	plan, apply, err := rec.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(plan.Collections[0].ToUpsert); got != 3 {
		t.Fatalf("expected 3 chunks planned, got %d", got)
	}
	if emb.calls != 3 || apply.Collections[0].Upserted != 3 || len(store.points) != 3 {
		t.Fatalf("expected 3 embeds/upserts/points, got embeds=%d upserts=%d points=%d", emb.calls, apply.Collections[0].Upserted, len(store.points))
	}
}

func TestReconcileWarmNoOpEmbedsZero(t *testing.T) {
	src := &sliceSource{docs: []SourceDoc{doc("README.md", "alpha\nbeta\ngamma")}}
	store, emb := newMemStore(), &countingEmbedder{}
	rec := newDocReconciler(src, store, emb)

	if _, _, err := rec.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	embedsAfterCold := emb.calls

	plan, apply, err := rec.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if plan.Collections[0].UnchangedSources != 1 {
		t.Fatalf("expected source-level skip, got %d unchanged sources", plan.Collections[0].UnchangedSources)
	}
	if emb.calls != embedsAfterCold {
		t.Fatalf("warm tick must embed zero: cold=%d warm-total=%d", embedsAfterCold, emb.calls)
	}
	if apply.Collections[0].Upserted != 0 {
		t.Fatalf("warm tick must upsert zero, got %d", apply.Collections[0].Upserted)
	}
}

func TestReconcileSingleEditReEmbedsOnlyChangedChunk(t *testing.T) {
	src := &sliceSource{docs: []SourceDoc{doc("README.md", "alpha\nbeta\ngamma")}}
	store, emb := newMemStore(), &countingEmbedder{}
	rec := newDocReconciler(src, store, emb)
	if _, _, err := rec.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	coldEmbeds := emb.calls // 3

	// Edit only the middle chunk.
	src.docs = []SourceDoc{doc("README.md", "alpha\nBETA\ngamma")}
	plan, apply, err := rec.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Collections[0].ToUpsert) != 1 {
		t.Fatalf("expected exactly 1 changed chunk to upsert, got %d", len(plan.Collections[0].ToUpsert))
	}
	if len(plan.Collections[0].ToRefresh) != 2 {
		t.Fatalf("expected 2 unchanged chunks to refresh source_hash, got %d", len(plan.Collections[0].ToRefresh))
	}
	if emb.calls-coldEmbeds != 1 {
		t.Fatalf("a single-chunk edit must re-embed exactly one chunk, got %d", emb.calls-coldEmbeds)
	}
	if apply.Collections[0].Refreshed != 2 {
		t.Fatalf("expected 2 payload refreshes, got %d", apply.Collections[0].Refreshed)
	}

	// The source must now converge: a third tick does no work at all.
	plan3, _, err := rec.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if plan3.HasWork() {
		t.Fatalf("source should have converged to no-work, got %+v", plan3.Collections[0])
	}
}

func TestReconcileGhostDeletion(t *testing.T) {
	src := &sliceSource{docs: []SourceDoc{doc("README.md", "alpha\nbeta\ngamma")}}
	store, emb := newMemStore(), &countingEmbedder{}
	rec := newDocReconciler(src, store, emb)
	if _, _, err := rec.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.points) != 3 {
		t.Fatalf("setup expected 3 points, got %d", len(store.points))
	}

	// Source no longer yields the doc → all its chunks are ghosts.
	src.docs = nil
	plan, apply, err := rec.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Collections[0].ToDelete) != 3 {
		t.Fatalf("expected 3 ghost deletions, got %d", len(plan.Collections[0].ToDelete))
	}
	if apply.Collections[0].Deleted != 3 || len(store.points) != 0 {
		t.Fatalf("expected store emptied, got deleted=%d points=%d", apply.Collections[0].Deleted, len(store.points))
	}
}

func TestReconcileEmbedBudgetResumes(t *testing.T) {
	src := &sliceSource{docs: []SourceDoc{doc("README.md", "alpha\nbeta\ngamma")}}
	store, emb := newMemStore(), &countingEmbedder{}
	rec := newDocReconciler(src, store, emb)
	rec.MaxEmbedsPerTick = 1

	_, apply1, err := rec.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if apply1.Collections[0].Upserted != 1 || apply1.Deferred != 2 {
		t.Fatalf("expected 1 upsert + 2 deferred, got upserted=%d deferred=%d", apply1.Collections[0].Upserted, apply1.Deferred)
	}
	if len(store.points) != 1 {
		t.Fatalf("budget tick should index exactly 1 point, got %d", len(store.points))
	}

	// Next tick must NOT skip the partially-indexed source — it resumes.
	plan2, apply2, err := rec.RunOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if plan2.Collections[0].UnchangedSources != 0 {
		t.Fatal("a partially-indexed source must not be skipped by the source-level gate")
	}
	if apply2.Collections[0].Upserted != 1 {
		t.Fatalf("expected the next budgeted upsert, got %d", apply2.Collections[0].Upserted)
	}

	// Drain the remainder; the corpus fully indexes.
	if _, _, err := rec.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(store.points) != 3 {
		t.Fatalf("expected full index of 3 points after draining budget, got %d", len(store.points))
	}
}

func TestIdentityChunkerKeepsUnsuffixedPointID(t *testing.T) {
	// A 1:1 consumer (identity chunker) must produce the same point ID a
	// dense-only collection already uses — no #index suffix — so cli-health's
	// existing points are recognized on migration (Phase 2), not re-embedded.
	src := &sliceSource{docs: []SourceDoc{{ID: "vrooli scenario start", Kind: "command", Body: "start a scenario", ContentHash: "h1"}}}
	store, emb := newMemStore(), &countingEmbedder{}
	rec := NewReconciler(emb, []SourceBinding{{
		Kind: "command", Store: store, Source: src,
		Chunker: NewIdentityChunker(), Composer: NewIdentityComposer(), IDPrefix: "cli-health:",
	}}, 4)
	if _, _, err := rec.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := PointIDFor("cli-health:", "vrooli scenario start", 0, 1)
	if _, ok := store.points[want]; !ok {
		var got []string
		for id := range store.points {
			got = append(got, id)
		}
		sort.Strings(got)
		t.Fatalf("expected un-suffixed point ID %s, store has %v", want, got)
	}
}

type generatedPagedSource struct {
	total  int
	failAt int
}

func (s generatedPagedSource) LoadPage(_ context.Context, request PageRequest) (SourcePage, error) {
	start := 0
	if request.Cursor != "" {
		var err error
		start, err = strconv.Atoi(request.Cursor)
		if err != nil {
			return SourcePage{}, err
		}
	}
	if s.failAt > 0 && start >= s.failAt {
		return SourcePage{}, fmt.Errorf("fixture page failure")
	}
	end := start + request.Limit
	if end > s.total {
		end = s.total
	}
	docs := make([]SourceDoc, 0, end-start)
	for i := start; i < end; i++ {
		id := fmt.Sprintf("source-%08d", i)
		docs = append(docs, SourceDoc{ID: id, Kind: "fixture", ContentHash: "hash:" + id, Body: "body " + id})
	}
	page := SourcePage{Documents: docs, Done: end == s.total}
	if !page.Done {
		page.NextCursor = strconv.Itoa(end)
	}
	return page, nil
}

type fakeGenerationStore struct {
	begins     int
	maxLookup  int
	active     map[string]StoredSourceState
	staged     int
	begin      bool
	promoted   bool
	rolledBack bool
	cleaned    bool
}

type resumableGenerationStore struct {
	*fakeGenerationStore
	candidate map[string]StoredSourceState
}

type batchGenerationStore struct {
	*fakeGenerationStore
	batches int
}

func (s *batchGenerationStore) StageSources(_ context.Context, _ string, writes []GenerationSourceWrite) error {
	s.batches++
	s.staged += len(writes)
	return nil
}

func (s *resumableGenerationStore) LookupGenerationSources(_ context.Context, _ string, ids []string) (map[string]StoredSourceState, error) {
	out := make(map[string]StoredSourceState)
	for _, id := range ids {
		if state, ok := s.candidate[id]; ok {
			out[id] = state
		}
	}
	return out, nil
}

func (s *fakeGenerationStore) BeginGeneration(context.Context, GenerationMetadata) error {
	s.begin = true
	s.begins++
	return nil
}

func (s *fakeGenerationStore) LookupActiveSources(_ context.Context, ids []string) (map[string]StoredSourceState, error) {
	if len(ids) > s.maxLookup {
		s.maxLookup = len(ids)
	}
	out := make(map[string]StoredSourceState)
	for _, id := range ids {
		if state, ok := s.active[id]; ok {
			out[id] = state
		}
	}
	return out, nil
}

func (s *fakeGenerationStore) StageSource(_ context.Context, _ string, _ GenerationSourceWrite) error {
	s.staged++
	return nil
}

func (s *fakeGenerationStore) StageDelete(context.Context, string, string) error { return nil }

func (s *fakeGenerationStore) ValidateGeneration(context.Context, string) (GenerationValidation, error) {
	return GenerationValidation{SourceCount: s.staged, PointCount: s.staged, Valid: true}, nil
}

func (s *fakeGenerationStore) PromoteGeneration(context.Context, string) error {
	s.promoted = true
	return nil
}

func (s *fakeGenerationStore) RollbackGeneration(context.Context, string) error {
	s.rolledBack = true
	return nil
}

func (s *fakeGenerationStore) CleanupGenerations(context.Context, int) error {
	s.cleaned = true
	return nil
}

func TestStreamingReconcilerBoundsLargeCorpusByPage(t *testing.T) {
	const (
		// Exceeds Agent Manager's measured 28,749-message operator corpus so
		// this remains a representative scale gate rather than a toy fixture.
		corpusSize = 50_000
		pageSize   = 64
	)
	store := &fakeGenerationStore{}
	embedder := &countingEmbedder{}
	reconciler := NewStreamingReconciler(embedder)
	result, err := reconciler.RunFull(context.Background(), StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: corpusSize},
		IDPrefix: "fixture:", PageSize: pageSize,
	}, GenerationMetadata{ID: "generation-1"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Promoted || !store.promoted || !store.cleaned || store.rolledBack {
		t.Fatalf("generation lifecycle mismatch: result=%+v store=%+v", result, store)
	}
	if result.Sources != corpusSize || result.Embedded != corpusSize {
		t.Fatalf("expected all generated sources, got sources=%d embedded=%d", result.Sources, result.Embedded)
	}
	if result.MaxPageDocuments > pageSize || store.maxLookup > pageSize {
		t.Fatalf("bounded planner exceeded page: result=%d lookup=%d limit=%d", result.MaxPageDocuments, store.maxLookup, pageSize)
	}
}

func TestStreamingReconcilerUsesBoundedPerSourceEmbeddingConcurrency(t *testing.T) {
	embedder := &concurrencyProbeEmbedder{want: 4, release: make(chan struct{})}
	reconciler := NewStreamingReconciler(embedder)
	tasks := []pendingEmbedding{
		{chunkIndex: 0, text: "zero"},
		{chunkIndex: 1, text: "one"},
		{chunkIndex: 2, text: "two"},
		{chunkIndex: 3, text: "three"},
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := reconciler.embedPending(ctx, StreamingBinding{EmbedConcurrency: 4}, "source", tasks); err != nil {
		t.Fatal(err)
	}
	if embedder.maximum != 4 {
		t.Fatalf("maximum concurrent embeddings = %d, want 4", embedder.maximum)
	}
	for index, task := range tasks {
		if len(task.point.Dense) != 1 || task.point.Dense[0] != float64(len(task.text)) {
			t.Fatalf("task %d result was not written in source order: %+v", index, task.point.Dense)
		}
	}
}

func TestStreamingReconcilerUsesBoundedCrossSourceEmbeddingConcurrency(t *testing.T) {
	embedder := &concurrencyProbeEmbedder{want: 4, release: make(chan struct{})}
	reconciler := NewStreamingReconciler(embedder)
	store := &fakeGenerationStore{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	result, err := reconciler.RunFull(ctx, StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: 4},
		IDPrefix: "fixture:", PageSize: 4, EmbedConcurrency: 4,
	}, GenerationMetadata{ID: "cross-source-concurrency"})
	if err != nil {
		t.Fatal(err)
	}
	if embedder.maximum != 4 {
		t.Fatalf("maximum concurrent source embeddings = %d, want 4", embedder.maximum)
	}
	if result.Embedded != 4 || result.Sources != 4 || !result.Promoted {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestStreamingReconcilerBatchesOneBoundedPageWrite(t *testing.T) {
	store := &batchGenerationStore{fakeGenerationStore: &fakeGenerationStore{}}
	result, err := NewStreamingReconciler(&countingEmbedder{}).RunFull(context.Background(), StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: 8}, PageSize: 4,
	}, GenerationMetadata{ID: "batch-write"})
	if err != nil {
		t.Fatal(err)
	}
	if store.batches != 2 || store.staged != 8 || result.Embedded != 8 {
		t.Fatalf("expected one batch per bounded page: batches=%d staged=%d result=%+v", store.batches, store.staged, result)
	}
}

func TestStreamingReconcilerRollsBackWithoutPromotionOnPageFailure(t *testing.T) {
	store := &fakeGenerationStore{}
	reconciler := NewStreamingReconciler(&countingEmbedder{})
	_, err := reconciler.RunFull(context.Background(), StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: 8, failAt: 2},
		IDPrefix: "fixture:", PageSize: 2,
	}, GenerationMetadata{ID: "generation-fails"})
	if err == nil {
		t.Fatal("expected page failure")
	}
	if !store.rolledBack || store.promoted {
		t.Fatalf("failed shadow generation must roll back without promotion: %+v", store)
	}
}

func TestStreamingReconcilerChecksAdopterInvariantBeforePromotion(t *testing.T) {
	store := &fakeGenerationStore{}
	_, err := NewStreamingReconciler(&countingEmbedder{}).RunFull(context.Background(), StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: 1}, PageSize: 1,
		BeforePromote: func(context.Context) error { return errors.New("privacy deletion pending") },
	}, GenerationMetadata{ID: "unsafe-generation"})
	if err == nil || !strings.Contains(err.Error(), "privacy deletion pending") {
		t.Fatalf("expected pre-promotion failure, got %v", err)
	}
	if !store.rolledBack || store.promoted {
		t.Fatalf("unsafe generation must roll back before promotion: %+v", store)
	}
}

func TestStreamingReconcilerSkipsCompleteUnchangedSourceBeforeChunking(t *testing.T) {
	id := "source-00000000"
	hash := "hash:" + id
	pointID := PointIDFor("fixture:", id, 0, 1)
	store := &fakeGenerationStore{active: map[string]StoredSourceState{
		id: {
			SourceHash:  hash,
			Model:       "fixture-model",
			ChunkPolicy: "identity-v1",
			Points: map[string]ScrollItem{
				pointID: {SourceHash: hash, SourceID: id, ChunkTotal: 1},
			},
		},
	}}
	embedder := &countingEmbedder{}
	result, err := NewStreamingReconciler(embedder).RunFull(context.Background(), StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: 1}, IDPrefix: "fixture:", PageSize: 1,
	}, GenerationMetadata{ID: "generation-reuse", Model: "fixture-model", ChunkPolicy: "identity-v1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Embedded != 0 || result.Reused != 1 || embedder.calls != 0 {
		t.Fatalf("unchanged source must skip chunk/embed wholesale: result=%+v calls=%d", result, embedder.calls)
	}
}

func TestStreamingReconcilerResumesCompleteCandidateSources(t *testing.T) {
	id := "source-00000000"
	hash := "hash:" + id
	pointID := PointIDFor("fixture:", id, 0, 1)
	store := &resumableGenerationStore{
		fakeGenerationStore: &fakeGenerationStore{},
		candidate: map[string]StoredSourceState{id: {
			SourceHash: hash, Model: "fixture-model", ChunkPolicy: "identity-v1",
			Points: map[string]ScrollItem{pointID: {SourceHash: hash, SourceID: id, ChunkTotal: 1}},
		}},
	}
	embedder := &countingEmbedder{}
	result, err := NewStreamingReconciler(embedder).RunFull(context.Background(), StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: 1}, IDPrefix: "fixture:", PageSize: 1,
	}, GenerationMetadata{ID: "interrupted", Model: "fixture-model", ChunkPolicy: "identity-v1"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Embedded != 0 || result.Reused != 1 || embedder.calls != 0 || store.staged != 0 {
		t.Fatalf("resume must preserve complete candidate source: result=%+v calls=%d staged=%d", result, embedder.calls, store.staged)
	}
}

// cleanupFailingGenerationStore promotes normally but cannot retire old
// generations, the shape of a Qdrant that is serving but slow to delete.
type cleanupFailingGenerationStore struct {
	*fakeGenerationStore
}

func (s *cleanupFailingGenerationStore) CleanupGenerations(context.Context, int) error {
	return errors.New("qdrant busy")
}

func TestStreamingRunChangesStagesEveryPageInOneGeneration(t *testing.T) {
	store := &fakeGenerationStore{}
	reconciler := NewStreamingReconciler(&countingEmbedder{})
	changes := ChangeSet{}
	for index := range 5 {
		id := "fixture:" + strconv.Itoa(index)
		changes.Changes = append(changes.Changes, SourceChange{Operation: ChangeUpsert, Document: SourceDoc{ID: id, Kind: "fixture", ContentHash: id, Body: "body " + id}})
	}
	changes.Changes = append(changes.Changes, SourceChange{Operation: ChangeDelete, SourceID: "fixture:gone"})
	result, err := reconciler.RunChanges(context.Background(), StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: 0}, PageSize: 2,
	}, GenerationMetadata{ID: "generation-changes"}, changes)
	if err != nil {
		t.Fatal(err)
	}
	if store.begins != 1 {
		t.Fatalf("a change set must cost exactly one generation, began %d", store.begins)
	}
	if result.Pages != 3 || result.MaxPageDocuments != 2 {
		t.Fatalf("6 changes at page size 2 must stage as 3 pages of at most 2, got pages=%d max=%d", result.Pages, result.MaxPageDocuments)
	}
	if result.Sources != 5 || result.Deleted != 1 || !result.Promoted || !store.promoted || store.rolledBack {
		t.Fatalf("unexpected result %+v store %+v", result, store)
	}
	if result.CleanupError != "" {
		t.Fatalf("unexpected cleanup error %q", result.CleanupError)
	}
}

func TestStreamingRunChangesAcceptsEmptyChangeSet(t *testing.T) {
	store := &fakeGenerationStore{}
	reconciler := NewStreamingReconciler(&countingEmbedder{})
	result, err := reconciler.RunChanges(context.Background(), StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: 0}, PageSize: 2,
	}, GenerationMetadata{ID: "generation-empty"}, ChangeSet{})
	if err != nil {
		t.Fatal(err)
	}
	if store.begins != 1 || !result.Promoted || result.Pages != 1 {
		t.Fatalf("empty change set must still run one generation, got %+v store %+v", result, store)
	}
}

func TestStreamingCleanupFailureDoesNotFailPromotedRun(t *testing.T) {
	store := &cleanupFailingGenerationStore{fakeGenerationStore: &fakeGenerationStore{}}
	reconciler := NewStreamingReconciler(&countingEmbedder{})
	result, err := reconciler.RunFull(context.Background(), StreamingBinding{
		Kind: "fixture", Store: store, Source: generatedPagedSource{total: 3}, IDPrefix: "fixture:", PageSize: 2,
	}, GenerationMetadata{ID: "generation-1"})
	if err != nil {
		t.Fatalf("a promoted generation must not fail on retirement cleanup: %v", err)
	}
	if !result.Promoted || result.CleanupError == "" || store.rolledBack {
		t.Fatalf("cleanup failure must be surfaced on the promoted result, got %+v", result)
	}
}
