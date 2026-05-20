package aisearch

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

// staticDiscovery is a deterministic DiscoverySource for tests.
type staticDiscovery struct {
	scenarios map[string][]CommandRecord
}

func (s *staticDiscovery) ListScenarios(_ context.Context) ([]string, error) {
	names := make([]string, 0, len(s.scenarios))
	for k := range s.scenarios {
		names = append(names, k)
	}
	return names, nil
}

func (s *staticDiscovery) Discover(_ context.Context, scenario string) ([]CommandRecord, error) {
	return s.scenarios[scenario], nil
}

// fakeEmbedder returns a deterministic vector with one dimension set per term
// position; "available" tracks whether we should report up.
type fakeEmbedder struct {
	available bool
	embedErr  error
	mu        sync.Mutex
	calls     int
}

func (f *fakeEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.embedErr != nil {
		return nil, f.embedErr
	}
	v := make([]float64, 8)
	for i, r := range text {
		v[i%8] += float64(r) / 1000.0
	}
	return v, nil
}
func (f *fakeEmbedder) Available(_ context.Context) bool { return f.available }

// fakeVectorStore tracks upserts/deletes and can simulate availability.
type fakeVectorStore struct {
	available bool
	count     int
	scrollErr error
	upsertErr error
	searchOut []SearchResult
	searchErr error

	mu     sync.Mutex
	points map[string]map[string]interface{}
	hashes map[string]string
}

func newFakeStore() *fakeVectorStore {
	return &fakeVectorStore{
		available: true,
		points:    map[string]map[string]interface{}{},
		hashes:    map[string]string{},
	}
}

func (s *fakeVectorStore) EnsureCollection(_ context.Context) error { return nil }
func (s *fakeVectorStore) Upsert(_ context.Context, id string, _ []float64, payload map[string]interface{}) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.points[id] = payload
	if h, ok := payload[payloadHashKey].(string); ok {
		s.hashes[id] = h
	}
	return nil
}
func (s *fakeVectorStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.points, id)
	delete(s.hashes, id)
	return nil
}
func (s *fakeVectorStore) BatchDelete(ctx context.Context, ids []string) error {
	for _, id := range ids {
		_ = s.Delete(ctx, id)
	}
	return nil
}
func (s *fakeVectorStore) Search(_ context.Context, _ []float64, _ int, _ float64) ([]SearchResult, error) {
	if s.searchErr != nil {
		return nil, s.searchErr
	}
	return s.searchOut, nil
}
func (s *fakeVectorStore) CountPoints(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count > 0 {
		return s.count, nil
	}
	return len(s.points), nil
}
func (s *fakeVectorStore) ScrollIDs(_ context.Context) (map[string]ScrollItem, error) {
	if s.scrollErr != nil {
		return nil, s.scrollErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]ScrollItem, len(s.hashes))
	for id, h := range s.hashes {
		out[id] = ScrollItem{PayloadHash: h}
	}
	return out, nil
}
func (s *fakeVectorStore) Available(_ context.Context) bool { return s.available }

func newTestService(disc DiscoverySource, embedder Embedder, store VectorStore) *Service {
	return NewService(Options{
		Embedder:    embedder,
		VectorStore: store,
		Discovery:   disc,
		Parallelism: 2,
	})
}

func sampleCorpus() *staticDiscovery {
	return &staticDiscovery{
		scenarios: map[string][]CommandRecord{
			"demo": {
				{Scenario: "demo", Group: "things", Name: "list", FullPath: "demo things list",
					Description: "List things from the database", Source: SourceManifest},
				{Scenario: "demo", Group: "things", Name: "show", FullPath: "demo things show",
					Description: "Show one thing", Source: SourceManifest},
			},
			"other": {
				{Scenario: "other", Group: "validate", Name: "manifest", FullPath: "other validate manifest",
					Description: "Validate the CLI manifest", Source: SourceManifest,
					Tags: []string{"validate"}},
			},
		},
	}
}

func TestService_TextSearch_RanksManifestHits(t *testing.T) {
	svc := newTestService(sampleCorpus(), &fakeEmbedder{}, newFakeStore())
	resp, err := svc.Search(context.Background(), "validate manifest", 5, ModeText)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Method != "text" {
		t.Errorf("Method = %q, want text", resp.Method)
	}
	if len(resp.Results) == 0 {
		t.Fatalf("no text-search hits")
	}
	if resp.Results[0].FullPath != "other validate manifest" {
		t.Errorf("top result = %q, want %q", resp.Results[0].FullPath, "other validate manifest")
	}
}

func TestService_AutoFallsBackToText_WhenEmbedderFails(t *testing.T) {
	emb := &fakeEmbedder{embedErr: errors.New("ollama down")}
	svc := newTestService(sampleCorpus(), emb, newFakeStore())
	resp, err := svc.Search(context.Background(), "list things", 5, ModeAuto)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Method != "text" {
		t.Errorf("Method = %q, want text fallback", resp.Method)
	}
}

func TestService_AIMode_PropagatesEmbedderError(t *testing.T) {
	emb := &fakeEmbedder{embedErr: errors.New("ollama down")}
	svc := newTestService(sampleCorpus(), emb, newFakeStore())
	_, err := svc.Search(context.Background(), "list", 5, ModeAI)
	if err == nil {
		t.Fatalf("want error in AI mode when embedder fails")
	}
	if !strings.Contains(err.Error(), "embed query") {
		t.Errorf("error = %v", err)
	}
}

func TestService_AIMode_ReturnsVectorHits(t *testing.T) {
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	rec := CommandRecord{Scenario: "demo", Name: "x", FullPath: "demo x", Source: SourceManifest}
	store.searchOut = []SearchResult{{
		ID:      PointIDForCommand(rec.FullPath),
		Score:   0.91,
		Payload: buildCommandPayload(rec, composeCommandEmbeddingText(rec)),
	}}
	svc := newTestService(sampleCorpus(), emb, store)
	resp, err := svc.Search(context.Background(), "anything", 5, ModeAI)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Method != "ai" {
		t.Errorf("Method = %q", resp.Method)
	}
	if len(resp.Results) != 1 || resp.Results[0].FullPath != "demo x" {
		t.Fatalf("results = %+v", resp.Results)
	}
	if resp.Results[0].ScorePercent != 91 {
		t.Errorf("ScorePercent = %d, want 91", resp.Results[0].ScorePercent)
	}
}

func TestService_Status_ReportsBackendAvailability(t *testing.T) {
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	store.count = 7
	svc := newTestService(sampleCorpus(), emb, store)
	st := svc.Status(context.Background())
	if !st.Available || !st.Ollama || !st.Qdrant {
		t.Errorf("expected fully available: %+v", st)
	}
	if st.IndexedCount != 7 {
		t.Errorf("IndexedCount = %d", st.IndexedCount)
	}
}

func TestService_Status_DegradedWhenOllamaDown(t *testing.T) {
	emb := &fakeEmbedder{available: false}
	store := newFakeStore()
	svc := newTestService(sampleCorpus(), emb, store)
	st := svc.Status(context.Background())
	if st.Available {
		t.Errorf("expected unavailable when ollama is down: %+v", st)
	}
	if !st.Qdrant {
		t.Errorf("Qdrant should still be up: %+v", st)
	}
}

func TestReconciler_ModifiedRecord_ProducesExactlyOneUpsert(t *testing.T) {
	disc := &staticDiscovery{
		scenarios: map[string][]CommandRecord{
			"demo": {
				{Scenario: "demo", Group: "g", Name: "a", FullPath: "demo g a",
					Description: "alpha", Source: SourceManifest},
				{Scenario: "demo", Group: "g", Name: "b", FullPath: "demo g b",
					Description: "beta", Source: SourceManifest},
			},
		},
	}
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	svc := newTestService(disc, emb, store)

	// Initial build.
	job1, err := svc.Reindex(context.Background(), "", false)
	if err != nil {
		t.Fatalf("initial reindex: %v", err)
	}
	waitJob(t, svc, job1.ID)
	if len(store.points) != 2 {
		t.Fatalf("after initial build: points=%d, want 2", len(store.points))
	}
	callsAfterFirst := emb.calls

	// Edit one record's description; the other should be unchanged.
	disc.scenarios["demo"][0].Description = "alpha-edited"

	job2, err := svc.Reindex(context.Background(), "", false)
	if err != nil {
		t.Fatalf("second reindex: %v", err)
	}
	waitJob(t, svc, job2.ID)

	upserts := 0
	for _, c := range job2.Apply.Collections {
		upserts += c.Upserted
	}
	if upserts != 1 {
		t.Errorf("upserts = %d, want exactly 1 (only the edited record)", upserts)
	}
	// And only one new embed call: we re-embedded the edited record only.
	if emb.calls-callsAfterFirst != 1 {
		t.Errorf("embed calls = %d, want 1", emb.calls-callsAfterFirst)
	}
}

func TestReindexDryRun_DoesNotUpsert(t *testing.T) {
	disc := sampleCorpus()
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	svc := newTestService(disc, emb, store)
	job, err := svc.Reindex(context.Background(), "", true)
	if err != nil {
		t.Fatalf("dry-run reindex: %v", err)
	}
	waitJob(t, svc, job.ID)
	if len(store.points) != 0 {
		t.Errorf("dry-run wrote %d points; want 0", len(store.points))
	}
	if job.State != "succeeded" {
		t.Errorf("State = %q, want succeeded", job.State)
	}
}

func waitJob(t *testing.T, svc *Service, jobID string) {
	t.Helper()
	for i := 0; i < 200; i++ {
		job, ok := svc.ReindexStatus(jobID)
		if !ok {
			t.Fatalf("job %s missing", jobID)
		}
		if job.State == "succeeded" || job.State == "failed" || job.State == "cancelled" {
			return
		}
		// 5ms * 200 = 1s budget — plenty for goroutines on test infra.
		_ = stallNanos(5_000_000)
	}
	t.Fatalf("job %s never terminated", jobID)
}

// stallNanos exists to avoid importing time at the top of the file just for a
// short sleep loop. It's a no-op divisor.
func stallNanos(n int64) int64 {
	x := int64(0)
	for i := int64(0); i < n; i++ {
		x++
	}
	return x
}
