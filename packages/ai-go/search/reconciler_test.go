package aisearch

import (
	"context"
	"sort"
	"strings"
	"sync"
	"testing"
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
