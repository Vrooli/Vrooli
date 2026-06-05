package aisearch

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	pkg "github.com/vrooli/aisearch-go"
)

// fakeEmbedder returns a fixed-length deterministic dense vector.
type fakeEmbedder struct{ calls int }

func (f *fakeEmbedder) Embed(_ context.Context, _ string) ([]float64, error) {
	f.calls++
	return make([]float64, pkg.DefaultVectorSize), nil
}
func (f *fakeEmbedder) Available(context.Context) bool { return true }

// fakeStore is an in-memory VectorStore mirroring the payload contract the real
// Qdrant store persists, enough to exercise the two-level-drift reconciler.
type fakeStore struct {
	mu     sync.Mutex
	spec   pkg.CollectionSpec
	points map[string]pkg.Point
}

func newFakeStore() *fakeStore { return &fakeStore{points: map[string]pkg.Point{}} }

func (s *fakeStore) EnsureCollection(_ context.Context, spec pkg.CollectionSpec) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spec = spec
	return nil
}

func (s *fakeStore) Upsert(_ context.Context, p pkg.Point) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.points[p.ID] = p
	return nil
}

func (s *fakeStore) SetPayload(_ context.Context, id string, payload map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.points[id]; ok {
		p.Payload = payload
		s.points[id] = p
	}
	return nil
}

func (s *fakeStore) BatchDelete(_ context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		delete(s.points, id)
	}
	return nil
}

func (s *fakeStore) Query(context.Context, pkg.HybridQuery) ([]pkg.SearchResult, error) {
	return nil, nil
}

func (s *fakeStore) CountPoints(context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.points), nil
}

func (s *fakeStore) ScrollIDs(context.Context) (map[string]pkg.ScrollItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]pkg.ScrollItem, len(s.points))
	for id, p := range s.points {
		item := pkg.ScrollItem{}
		item.PayloadHash, _ = p.Payload["payload_hash"].(string)
		item.SourceID, _ = p.Payload["source_id"].(string)
		item.SourceHash, _ = p.Payload["source_hash"].(string)
		switch v := p.Payload["chunk_total"].(type) {
		case int:
			item.ChunkTotal = v
		case float64:
			item.ChunkTotal = int(v)
		}
		out[id] = item
	}
	return out, nil
}

func (s *fakeStore) Available(context.Context) bool { return true }

func newTestIndexer(t *testing.T) (*Indexer, *fakeStore, *fakeEmbedder, string) {
	t.Helper()
	_, scenariosRoot := buildFixtureRepo(t)
	store := newFakeStore()
	emb := &fakeEmbedder{}
	idx, err := NewIndexer(Options{Embedder: emb, VectorStore: store, ScenariosRoot: scenariosRoot})
	if err != nil {
		t.Fatalf("NewIndexer: %v", err)
	}
	return idx, store, emb, scenariosRoot
}

func TestIndexerCollectionSpecIsHybrid(t *testing.T) {
	idx, store, _, _ := newTestIndexer(t)
	if err := idx.EnsureCollection(context.Background()); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}
	spec := store.spec
	if spec.Name != DefaultCollection {
		t.Errorf("collection name = %q, want %q", spec.Name, DefaultCollection)
	}
	if !spec.Sparse || spec.SparseModifier != pkg.DefaultSparseModifier {
		t.Errorf("expected hybrid collection (sparse + idf), got sparse=%v modifier=%q", spec.Sparse, spec.SparseModifier)
	}
	if spec.DenseSize != pkg.DefaultVectorSize {
		t.Errorf("dense size = %d, want %d", spec.DenseSize, pkg.DefaultVectorSize)
	}
}

func TestIndexerIndexesDocsWithHybridPayload(t *testing.T) {
	idx, store, _, _ := newTestIndexer(t)
	ctx := context.Background()
	if err := idx.EnsureCollection(ctx); err != nil {
		t.Fatalf("EnsureCollection: %v", err)
	}

	res, err := idx.Reindex(ctx, false)
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if res.Upserted == 0 {
		t.Fatal("expected docs to be indexed")
	}
	if len(res.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", res.Errors)
	}

	// Every stored point must carry a sparse vector (hybrid), a non-empty body
	// (fixing the legacy content:"" defect), and the heading-path + scope meta.
	for id, p := range store.points {
		if p.Sparse == nil || len(p.Sparse.Indices) == 0 {
			t.Errorf("point %s has no sparse vector", id)
		}
		if body, _ := p.Payload["body"].(string); body == "" {
			t.Errorf("point %s has empty body payload", id)
		}
		if _, ok := p.Payload[MetaHeadingPath]; !ok {
			t.Errorf("point %s missing heading_path", id)
		}
		if _, ok := p.Payload[MetaScope]; !ok {
			t.Errorf("point %s missing scope", id)
		}
	}
}

func TestIndexerIsIdempotent(t *testing.T) {
	idx, _, emb, _ := newTestIndexer(t)
	ctx := context.Background()
	_ = idx.EnsureCollection(ctx)

	first, err := idx.Reindex(ctx, false)
	if err != nil {
		t.Fatalf("Reindex 1: %v", err)
	}
	embedsAfterFirst := emb.calls

	second, err := idx.Reindex(ctx, false)
	if err != nil {
		t.Fatalf("Reindex 2: %v", err)
	}
	if second.Upserted != 0 {
		t.Errorf("warm reindex upserted %d, want 0", second.Upserted)
	}
	if emb.calls != embedsAfterFirst {
		t.Errorf("warm reindex embedded %d new chunks, want 0", emb.calls-embedsAfterFirst)
	}
	if first.Upserted == 0 {
		t.Error("cold reindex upserted nothing")
	}
}

func TestIndexerReembedsOnlyChangedFile(t *testing.T) {
	idx, _, emb, scenariosRoot := newTestIndexer(t)
	ctx := context.Background()
	_ = idx.EnsureCollection(ctx)
	if _, err := idx.Reindex(ctx, false); err != nil {
		t.Fatalf("cold reindex: %v", err)
	}
	embedsBefore := emb.calls

	// Edit exactly one short (single-chunk) doc.
	if err := os.WriteFile(filepath.Join(scenariosRoot, "bar", "README.md"), []byte("# Bar\n\nUpdated bar."), 0o644); err != nil {
		t.Fatalf("edit: %v", err)
	}
	res, err := idx.Reindex(ctx, false)
	if err != nil {
		t.Fatalf("warm reindex: %v", err)
	}
	if res.Upserted != 1 {
		t.Errorf("changed-file reindex upserted %d, want 1", res.Upserted)
	}
	if delta := emb.calls - embedsBefore; delta != 1 {
		t.Errorf("changed-file reindex embedded %d, want 1", delta)
	}
}

func TestIndexerDryRunWritesNothing(t *testing.T) {
	idx, store, _, _ := newTestIndexer(t)
	ctx := context.Background()
	_ = idx.EnsureCollection(ctx)
	res, err := idx.Reindex(ctx, true)
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if res.Planned == 0 {
		t.Error("dry run should report planned work for an empty collection")
	}
	if n, _ := store.CountPoints(ctx); n != 0 {
		t.Errorf("dry run wrote %d points, want 0", n)
	}
}

func TestNewIndexerRequiresStore(t *testing.T) {
	_, scenariosRoot := buildFixtureRepo(t)
	if _, err := NewIndexer(Options{ScenariosRoot: scenariosRoot}); err == nil {
		t.Fatal("expected error when vector store is nil")
	}
}
