package aisearch

import (
	"context"
	"sort"
	"testing"
)

// fakeEmbedder returns a tiny deterministic vector and reports availability.
type fakeEmbedder struct {
	available bool
	calls     int
	failOn    string // if text == failOn, return an error
}

func (f *fakeEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	f.calls++
	if text == f.failOn {
		return nil, context.DeadlineExceeded
	}
	// length-3 vector; content is irrelevant to the fake store's ranking.
	return []float64{float64(len(text)), 1, 0}, nil
}
func (f *fakeEmbedder) Available(context.Context) bool { return f.available }

// fakeVectorStore is an in-memory VectorStore for tests.
type fakeVectorStore struct {
	available  bool
	points     map[string]fakePoint
	ensureN    int
	scoreOrder []string // ids in the order Search should return them (highest first)
}

type fakePoint struct {
	vector  []float64
	payload map[string]interface{}
}

func newFakeStore() *fakeVectorStore {
	return &fakeVectorStore{available: true, points: map[string]fakePoint{}}
}

func (s *fakeVectorStore) EnsureCollection(context.Context) error { s.ensureN++; return nil }
func (s *fakeVectorStore) Upsert(_ context.Context, id string, vector []float64, payload map[string]interface{}) error {
	s.points[id] = fakePoint{vector: vector, payload: payload}
	return nil
}

func (s *fakeVectorStore) Delete(_ context.Context, id string) error {
	delete(s.points, id)
	return nil
}

func (s *fakeVectorStore) BatchDelete(_ context.Context, ids []string) error {
	for _, id := range ids {
		delete(s.points, id)
	}
	return nil
}

func (s *fakeVectorStore) Search(_ context.Context, _ []float64, limit int, _ float64) ([]VectorSearchResult, error) {
	ids := s.scoreOrder
	if len(ids) == 0 {
		ids = make([]string, 0, len(s.points))
		for id := range s.points {
			ids = append(ids, id)
		}
		sort.Strings(ids)
	}
	out := make([]VectorSearchResult, 0, len(ids))
	score := 1.0
	for _, id := range ids {
		p, ok := s.points[id]
		if !ok {
			continue
		}
		out = append(out, VectorSearchResult{ID: id, Score: score, Payload: p.payload})
		score -= 0.1
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
func (s *fakeVectorStore) CountPoints(context.Context) (int, error) { return len(s.points), nil }
func (s *fakeVectorStore) ScrollIDs(context.Context) (map[string]ScrollItem, error) {
	out := make(map[string]ScrollItem, len(s.points))
	for id, p := range s.points {
		hash, _ := p.payload[payloadHashKey].(string)
		out[id] = ScrollItem{PayloadHash: hash}
	}
	return out, nil
}
func (s *fakeVectorStore) Available(context.Context) bool { return s.available }

func TestIndexer_SyncUpsertsSkipsAndDeletes(t *testing.T) {
	ctx := context.Background()
	emb := &fakeEmbedder{available: true}
	vs := newFakeStore()
	ix := NewIndexer(emb, vs)

	items := []Item{
		{Key: "go|a|x|1", Text: "x version 1"},
		{Key: "npm|a|y|2", Text: "y version 2"},
	}
	up, del, err := ix.Sync(ctx, items)
	if err != nil || up != 2 || del != 0 {
		t.Fatalf("first sync = (%d,%d,%v), want (2,0,nil)", up, del, err)
	}
	if len(vs.points) != 2 {
		t.Fatalf("store has %d points, want 2", len(vs.points))
	}
	embCalls := emb.calls

	// Re-sync identical items → all skipped (no new embeds, no changes).
	up, del, err = ix.Sync(ctx, items)
	if err != nil || up != 0 || del != 0 {
		t.Fatalf("idempotent sync = (%d,%d,%v), want (0,0,nil)", up, del, err)
	}
	if emb.calls != embCalls {
		t.Errorf("unchanged items must not re-embed: calls %d→%d", embCalls, emb.calls)
	}

	// Change one item's text → exactly one re-embed/upsert.
	items[0].Text = "x version 1.1 with a CVE"
	up, _, _ = ix.Sync(ctx, items)
	if up != 1 {
		t.Errorf("changed item should upsert once, got %d", up)
	}

	// Drop one item → its point is deleted.
	_, del, _ = ix.Sync(ctx, items[:1])
	if del != 1 {
		t.Errorf("dropped item should delete once, got %d", del)
	}
	if len(vs.points) != 1 {
		t.Errorf("store should have 1 point after delete, got %d", len(vs.points))
	}
}

func TestIndexer_QueryMapsPayloadToKey(t *testing.T) {
	ctx := context.Background()
	vs := newFakeStore()
	ix := NewIndexer(&fakeEmbedder{available: true}, vs)
	_, _, err := ix.Sync(ctx, []Item{
		{Key: "go|a|x|1", Text: "alpha"},
		{Key: "npm|b|y|2", Text: "beta"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The upsert payload must carry the corpus key under corpusKeyField so a
	// vector hit maps back to the record(s) it covers.
	if got, _ := vs.points[pointID("go|a|x|1")].payload[corpusKeyField].(string); got != "go|a|x|1" {
		t.Errorf("upsert payload %s = %q, want %q", corpusKeyField, got, "go|a|x|1")
	}

	vs.scoreOrder = []string{pointID("npm|b|y|2"), pointID("go|a|x|1")}

	hits, err := ix.Query(ctx, "anything", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("got %d hits, want 2", len(hits))
	}
	if hits[0].Key != "npm|b|y|2" {
		t.Errorf("ranking not preserved: got %q first", hits[0].Key)
	}
	if hits[0].Score <= hits[1].Score {
		t.Errorf("scores not descending: %v", hits)
	}
}

func TestIndexer_QueryEmbedErrorPropagates(t *testing.T) {
	ix := NewIndexer(&fakeEmbedder{available: true, failOn: "boom"}, newFakeStore())
	if _, err := ix.Query(context.Background(), "boom", 5); err == nil {
		t.Fatal("expected embed error to propagate so the caller can fall back to TEXT")
	}
}

func TestIndexer_NilSafe(t *testing.T) {
	var ix *Indexer
	if o, q := ix.Available(context.Background()); o || q {
		t.Error("nil indexer must report both backends unavailable")
	}
	if up, del, err := ix.Sync(context.Background(), []Item{{Key: "k", Text: "t"}}); err != nil || up != 0 || del != 0 {
		t.Errorf("nil indexer Sync should be a no-op, got (%d,%d,%v)", up, del, err)
	}
	if hits, err := ix.Query(context.Background(), "q", 5); err != nil || hits != nil {
		t.Errorf("nil indexer Query should be a no-op, got (%v,%v)", hits, err)
	}
}

func TestPointIDDeterministicAndValidUUID(t *testing.T) {
	a := pointID("go|a|x|1")
	b := pointID("go|a|x|1")
	if a != b {
		t.Errorf("pointID not deterministic: %q vs %q", a, b)
	}
	if pointID("go|a|x|1") == pointID("go|a|x|2") {
		t.Error("distinct keys must yield distinct point IDs")
	}
	// UUID shape: 8-4-4-4-12 hex.
	if len(a) != 36 || a[8] != '-' || a[13] != '-' || a[18] != '-' || a[23] != '-' {
		t.Errorf("pointID is not a UUID: %q", a)
	}
}
