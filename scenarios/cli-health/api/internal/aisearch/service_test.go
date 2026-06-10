package aisearch

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"
	"testing"

	pkg "github.com/vrooli/aisearch-go"
)

// staticDiscovery is a deterministic DiscoverySource for tests.
type staticDiscovery struct {
	scenarios    map[string][]CommandRecord
	externalCLIs []ExternalCLI
	external     map[string][]CommandRecord
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

func (s *staticDiscovery) ListExternalCLIs() []ExternalCLI {
	if len(s.externalCLIs) == 0 {
		return nil
	}
	out := make([]ExternalCLI, len(s.externalCLIs))
	copy(out, s.externalCLIs)
	return out
}

func (s *staticDiscovery) DiscoverExternal(_ context.Context, cli ExternalCLI) ([]CommandRecord, error) {
	return s.external[cli.Name], nil
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

// fakeVectorStore is an in-memory pkg.VectorStore that honors the same on-disk
// payload contract as Qdrant (ScrollIDs projects payload_hash / source_id /
// source_hash / chunk_total), so the shared reconciler's two-level drift logic
// is exercised end-to-end through the Service.
type fakeVectorStore struct {
	available bool
	count     int
	scrollErr error
	upsertErr error
	searchOut []pkg.SearchResult
	searchErr error

	mu     sync.Mutex
	points map[string]pkg.Point
}

func newFakeStore() *fakeVectorStore {
	return &fakeVectorStore{
		available: true,
		points:    map[string]pkg.Point{},
	}
}

func (s *fakeVectorStore) EnsureCollection(_ context.Context, _ pkg.CollectionSpec) error { return nil }

func (s *fakeVectorStore) Upsert(_ context.Context, p pkg.Point) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.points[p.ID] = p
	return nil
}

func (s *fakeVectorStore) SetPayload(_ context.Context, id string, payload map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if p, ok := s.points[id]; ok {
		p.Payload = payload
		s.points[id] = p
	}
	return nil
}

func (s *fakeVectorStore) BatchDelete(_ context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		delete(s.points, id)
	}
	return nil
}

func (s *fakeVectorStore) Query(_ context.Context, _ pkg.HybridQuery) ([]pkg.SearchResult, error) {
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

// ScrollIDs projects the on-disk drift fields by their payload-key string
// literals — the same keys the shared engine's buildChunkPayload writes.
func (s *fakeVectorStore) ScrollIDs(_ context.Context) (map[string]pkg.ScrollItem, error) {
	if s.scrollErr != nil {
		return nil, s.scrollErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]pkg.ScrollItem, len(s.points))
	for id, p := range s.points {
		hash, _ := p.Payload["payload_hash"].(string)
		srcID, _ := p.Payload["source_id"].(string)
		srcHash, _ := p.Payload["source_hash"].(string)
		total, _ := p.Payload["chunk_total"].(int)
		out[id] = pkg.ScrollItem{PayloadHash: hash, SourceID: srcID, SourceHash: srcHash, ChunkTotal: total}
	}
	return out, nil
}
func (s *fakeVectorStore) Available(_ context.Context) bool { return s.available }

func newTestService(disc DiscoverySource, embedder pkg.Embedder, store pkg.VectorStore) *Service {
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
				{
					Origin: "demo", Group: "things", Name: "list", FullPath: "demo things list",
					Description: "List things from the database", Source: SourceManifest,
				},
				{
					Origin: "demo", Group: "things", Name: "show", FullPath: "demo things show",
					Description: "Show one thing", Source: SourceManifest,
				},
			},
			"other": {
				{
					Origin: "other", Group: "validate", Name: "manifest", FullPath: "other validate manifest",
					Description: "Validate the CLI manifest", Source: SourceManifest,
					Tags: []string{"validate"},
				},
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
	rec := CommandRecord{Origin: "demo", Name: "x", FullPath: "demo x", Source: SourceManifest}
	store.searchOut = []pkg.SearchResult{{
		ID:      pointIDForCommand(rec.FullPath),
		Score:   0.91,
		Payload: commandMeta(rec),
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

// TestService_AIMode_AppliesRelevanceFloor asserts WS2: the query-adaptive
// relative cutoff drops a hit that trails the top by more than the default gap,
// while keeping the top and near hits.
func TestService_AIMode_AppliesRelevanceFloor(t *testing.T) {
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	mk := func(name string, score float64) pkg.SearchResult {
		rec := CommandRecord{Origin: "demo", Name: name, FullPath: "demo " + name, Source: SourceManifest}
		return pkg.SearchResult{ID: pointIDForCommand(rec.FullPath), Score: score, Payload: commandMeta(rec)}
	}
	// Default gap 0.15 off the 0.80 top -> cutoff 0.65; the 0.50 hit is dropped.
	store.searchOut = []pkg.SearchResult{mk("top", 0.80), mk("near", 0.70), mk("far", 0.50)}

	svc := newTestService(sampleCorpus(), emb, store)
	resp, err := svc.Search(context.Background(), "anything", 10, ModeAI)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected floor to drop the weak hit, got %d: %+v", len(resp.Results), resp.Results)
	}
	for _, r := range resp.Results {
		if r.FullPath == "demo far" {
			t.Fatalf("weak hit (0.50) should have been filtered, got %+v", resp.Results)
		}
	}
}

// fakeReranker is a programmable pkg.Reranker for WS4 tests.
type fakeReranker struct {
	name      string
	available bool
	scores    map[string]float64
	err       error
}

func (f *fakeReranker) Name() string                     { return f.name }
func (f *fakeReranker) Available(_ context.Context) bool { return f.available }
func (f *fakeReranker) Rerank(_ context.Context, _ string, cands []pkg.RerankCandidate) ([]pkg.RerankScore, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]pkg.RerankScore, 0, len(cands))
	for _, c := range cands {
		if s, ok := f.scores[c.ID]; ok {
			out = append(out, pkg.RerankScore{ID: c.ID, Score: s})
		}
	}
	return out, nil
}

func mkResult(name string, score float64) pkg.SearchResult {
	rec := CommandRecord{Origin: "demo", Name: name, FullPath: "demo " + name, Source: SourceManifest}
	return pkg.SearchResult{ID: pointIDForCommand(rec.FullPath), Score: score, Payload: commandMeta(rec)}
}

func newRerankService(disc DiscoverySource, emb pkg.Embedder, store pkg.VectorStore, enabled bool, r pkg.Reranker) *Service {
	return NewService(Options{
		Embedder:      emb,
		VectorStore:   store,
		Discovery:     disc,
		Parallelism:   2,
		RerankEnabled: enabled,
		Reranker:      pkg.NewRerankerChain(r),
	})
}

func TestService_AIMode_RerankReordersAndTruncates(t *testing.T) {
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	a, b, c := mkResult("alpha", 0.80), mkResult("bravo", 0.78), mkResult("charlie", 0.76)
	store.searchOut = []pkg.SearchResult{a, b, c}
	// Reranker prefers charlie over the dense order. Scores stay within the
	// default floor gap (0.15 off the 0.99 top) so all three survive the
	// post-rerank floor and the assertion isolates reorder + truncation.
	rr := &fakeReranker{name: "fake:rr", available: true, scores: map[string]float64{
		a.ID: 0.90, b.ID: 0.92, c.ID: 0.99,
	}}
	svc := newRerankService(sampleCorpus(), emb, store, true, rr)

	resp, err := svc.Search(context.Background(), "anything", 2, ModeAI)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Reranker != "fake:rr" {
		t.Errorf("Reranker = %q, want fake:rr", resp.Reranker)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("expected truncation to limit=2, got %d", len(resp.Results))
	}
	if resp.Results[0].FullPath != "demo charlie" {
		t.Errorf("rerank winner should sort first, got %+v", resp.Results)
	}
}

// TestService_AIMode_FloorRunsAfterRerank asserts the A2 reorder: a junk hit
// whose DENSE score is close to the strong hit (so the pre-rerank floor would
// keep it) but which the reranker drives to ~0 is dropped by the floor that now
// runs AFTER rerank (cap-fabecce56b518120). Under the old order (floor first)
// the junk would ride along on its dense score and inflate the page count.
func TestService_AIMode_FloorRunsAfterRerank(t *testing.T) {
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	strong, junk := mkResult("strong", 0.80), mkResult("junk", 0.79)
	// Dense scores are within the 0.15 gap → a pre-rerank floor keeps BOTH.
	store.searchOut = []pkg.SearchResult{strong, junk}
	// Reranker keeps the strong hit high and collapses the junk to ~0.
	rr := &fakeReranker{name: "fake:rr", available: true, scores: map[string]float64{
		strong.ID: 0.95, junk.ID: 0.001,
	}}
	svc := newRerankService(sampleCorpus(), emb, store, true, rr)

	resp, err := svc.Search(context.Background(), "anything", 10, ModeAI)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(resp.Results) != 1 || resp.Results[0].FullPath != "demo strong" {
		t.Fatalf("post-rerank floor must drop the junk hit reranked to ~0, got %+v", resp.Results)
	}
}

func TestService_AIMode_RerankUnavailableKeepsDenseOrder(t *testing.T) {
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	a, b := mkResult("alpha", 0.80), mkResult("bravo", 0.78)
	store.searchOut = []pkg.SearchResult{a, b}
	rr := &fakeReranker{name: "fake:rr", available: false} // no leg reachable
	svc := newRerankService(sampleCorpus(), emb, store, true, rr)

	resp, err := svc.Search(context.Background(), "anything", 10, ModeAI)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Reranker != "none" {
		t.Errorf("Reranker = %q, want none when no leg reachable", resp.Reranker)
	}
	if len(resp.Results) != 2 || resp.Results[0].FullPath != "demo alpha" {
		t.Fatalf("dense order must be preserved, got %+v", resp.Results)
	}
}

// TestService_AIMode_LabelsWeakByRegime asserts the weak flag is computed once
// in the service against the ACTIVE leg's regime, not a fixed cosine 0.55. With
// a cross-encoder leg, a 0.40 hit is strong (xenc weak threshold 0.30) — under
// the old cosine 0.55 line it would have been mislabeled weak. The 0.25 hit
// survives the (permissive) cross-encoder floor and is correctly weak.
func TestService_AIMode_LabelsWeakByRegime(t *testing.T) {
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	strong, low := mkResult("strong", 0.70), mkResult("low", 0.60)
	store.searchOut = []pkg.SearchResult{strong, low}
	rr := &fakeReranker{name: "cross-encoder:bge", available: true, scores: map[string]float64{
		strong.ID: 0.40, low.ID: 0.25,
	}}
	svc := newRerankService(sampleCorpus(), emb, store, true, rr)

	resp, err := svc.Search(context.Background(), "anything", 10, ModeAI)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if resp.Reranker != "cross-encoder:bge" {
		t.Fatalf("Reranker = %q", resp.Reranker)
	}
	byPath := map[string]SearchHit{}
	for _, r := range resp.Results {
		byPath[r.FullPath] = r
	}
	if len(byPath) != 2 {
		t.Fatalf("expected both hits to survive the cross-encoder floor, got %+v", resp.Results)
	}
	if byPath["demo strong"].Weak {
		t.Errorf("0.40 under cross-encoder regime must NOT be weak (xenc threshold 0.30; cosine 0.55 would mislabel)")
	}
	if !byPath["demo low"].Weak {
		t.Errorf("0.25 under cross-encoder regime must be weak")
	}
}

func TestService_TextSearch_LabelsWeakCosine(t *testing.T) {
	// Text mode has no reranker → cosine regime (0.55). A normalized lexical
	// score below 0.55 is weak.
	svc := newTestService(sampleCorpus(), &fakeEmbedder{available: true}, newFakeStore())
	resp, err := svc.Search(context.Background(), "restart", 10, ModeText)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	for _, r := range resp.Results {
		if want := r.Score < 0.55; r.Weak != want {
			t.Errorf("%s score=%.3f Weak=%v, want %v (cosine 0.55)", r.FullPath, r.Score, r.Weak, want)
		}
	}
}

func TestService_Status_ReportsReranker(t *testing.T) {
	store := newFakeStore()
	rr := &fakeReranker{name: "fake:rr", available: true}
	svc := newRerankService(sampleCorpus(), &fakeEmbedder{available: true}, store, true, rr)
	if got := svc.Status(context.Background()).Reranker; got != "fake:rr" {
		t.Errorf("Status.Reranker = %q, want fake:rr", got)
	}

	off := newTestService(sampleCorpus(), &fakeEmbedder{available: true}, newFakeStore())
	if got := off.Status(context.Background()).Reranker; got != "none" {
		t.Errorf("disabled reranker status = %q, want none", got)
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
				{
					Origin: "demo", Group: "g", Name: "a", FullPath: "demo g a",
					Description: "alpha", Source: SourceManifest,
				},
				{
					Origin: "demo", Group: "g", Name: "b", FullPath: "demo g b",
					Description: "beta", Source: SourceManifest,
				},
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

func TestReindex_WalksScenariosAndExternalCLIs(t *testing.T) {
	disc := &staticDiscovery{
		scenarios: map[string][]CommandRecord{
			"demo": {
				{
					Origin: "demo", Group: "g", Name: "a", FullPath: "demo g a",
					Description: "alpha", Source: SourceManifest,
				},
			},
		},
		externalCLIs: []ExternalCLI{{Name: "vrooli", Binary: "vrooli"}},
		external: map[string][]CommandRecord{
			"vrooli": {
				{
					Origin: "vrooli", Group: "scenario", Name: "start",
					FullPath: "vrooli scenario start", Description: "Start a scenario",
					Source: SourceHelp,
				},
			},
		},
	}
	emb := &fakeEmbedder{available: true}
	store := newFakeStore()
	svc := newTestService(disc, emb, store)
	job, err := svc.Reindex(context.Background(), "", false)
	if err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	waitJob(t, svc, job.ID)

	// Both origins must be reflected in the store payload.
	origins := map[string]int{}
	for _, p := range store.points {
		if v, _ := p.Payload["origin"].(string); v != "" {
			origins[v]++
		}
	}
	if origins["demo"] == 0 {
		t.Errorf("scenario origin 'demo' missing; got %v", origins)
	}
	if origins["vrooli"] == 0 {
		t.Errorf("external origin 'vrooli' missing; got %v", origins)
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
	for i := 0; i < 100_000; i++ {
		job, ok := svc.ReindexStatus(jobID)
		if !ok {
			t.Fatalf("job %s missing", jobID)
		}
		if job.State == "succeeded" || job.State == "failed" || job.State == "cancelled" {
			return
		}
		// Yield without a real sleep (the reconcile goroutine is CPU-bound and
		// fast), mirroring packages/aisearch-go service_test.go waitJob.
		runtime.Gosched()
	}
	t.Fatalf("job %s never terminated", jobID)
}
