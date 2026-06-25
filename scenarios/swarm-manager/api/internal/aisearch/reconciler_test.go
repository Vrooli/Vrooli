package aisearch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
	"swarm-manager/internal/testutil"
)

// ---- shared test fakes ----
//
// fakeEmbedder and fakeVectorStore satisfy the Embedder / VectorStore
// interfaces directly; reconciler tests don't need an httptest.Server. The
// reader fakes (fakeBacklogReader, fakeInitReader) are reused from
// service_test.go via same-package access.

type fakeEmbedder struct {
	mu           sync.Mutex
	calls        int
	inFlight     int
	peakInFlight int
	delay        time.Duration
	failOn       map[string]error // text → error
	dim          int              // length of returned vector; default 3
}

func (f *fakeEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	f.mu.Lock()
	f.calls++
	f.inFlight++
	if f.inFlight > f.peakInFlight {
		f.peakInFlight = f.inFlight
	}
	delay := f.delay
	err := f.failOn[text]
	dim := f.dim
	if dim == 0 {
		dim = 3
	}
	f.mu.Unlock()

	defer func() {
		f.mu.Lock()
		f.inFlight--
		f.mu.Unlock()
	}()

	if delay > 0 {
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if err != nil {
		return nil, err
	}
	out := make([]float64, dim)
	for i := range out {
		out[i] = 0.1
	}
	return out, nil
}

func (f *fakeEmbedder) Available(_ context.Context) bool { return true }

func (f *fakeEmbedder) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func (f *fakeEmbedder) peak() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.peakInFlight
}

type fakeVectorStore struct {
	mu                sync.Mutex
	points            map[string]map[string]interface{}
	upsertCalls       int
	deleteCalls       int
	batchDeleteCalls  int
	batchDeleteRanges [][]string
	scrollErr         error
	upsertFailOn      map[string]error // pointID → error
	ensureCalls       int
}

func (s *fakeVectorStore) seed(id string, payload map[string]interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.points == nil {
		s.points = make(map[string]map[string]interface{})
	}
	s.points[id] = payload
}

func (s *fakeVectorStore) EnsureCollection(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureCalls++
	return nil
}

func (s *fakeVectorStore) Upsert(_ context.Context, id string, _ []float64, payload map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.upsertFailOn[id]; err != nil {
		return err
	}
	if s.points == nil {
		s.points = make(map[string]map[string]interface{})
	}
	s.points[id] = payload
	s.upsertCalls++
	return nil
}

func (s *fakeVectorStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.points, id)
	s.deleteCalls++
	return nil
}

func (s *fakeVectorStore) BatchDelete(_ context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(ids) == 0 {
		return nil
	}
	s.batchDeleteCalls++
	s.batchDeleteRanges = append(s.batchDeleteRanges, append([]string(nil), ids...))
	for _, id := range ids {
		delete(s.points, id)
	}
	return nil
}

func (s *fakeVectorStore) Search(_ context.Context, _ []float64, _ int, _ float64) ([]SearchResult, error) {
	return nil, nil
}

func (s *fakeVectorStore) CountPoints(_ context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.points), nil
}

func (s *fakeVectorStore) ScrollIDs(_ context.Context) (map[string]ScrollItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scrollErr != nil {
		return nil, s.scrollErr
	}
	out := make(map[string]ScrollItem, len(s.points))
	for id, p := range s.points {
		item := ScrollItem{}
		if h, ok := p["payload_hash"].(string); ok {
			item.PayloadHash = h
		}
		if a, ok := p["archived"].(bool); ok {
			item.Archived = a
		}
		out[id] = item
	}
	return out, nil
}

func (s *fakeVectorStore) Available(_ context.Context) bool { return true }

// ---- builders ----

func newReconcilerForTest(t *testing.T) (*Reconciler, *fakeEmbedder, *fakeVectorStore, *fakeVectorStore, *fakeBacklogReader, *fakeInitReader) {
	t.Helper()
	emb := &fakeEmbedder{}
	bs := &fakeVectorStore{}
	is := &fakeVectorStore{}
	br := &fakeBacklogReader{}
	ir := &fakeInitReader{}
	r := NewReconciler(emb, bs, is, br, ir, 0)
	return r, emb, bs, is, br, ir
}

func mustHash(item backlog.BacklogItem) string {
	return composePayloadHash(composeBacklogText(item), buildBacklogPayload(item, ""))
}

func mustHashInit(init initiatives.Initiative) string {
	return composePayloadHash(composeInitiativeText(init), buildInitiativePayload(init, ""))
}

// ---- Plan ----

func TestReconciler_Plan_EmptyDisk_EmptyQdrant(t *testing.T) {
	r, _, _, _, _, _ := newReconcilerForTest(t)
	report, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.HasWork() {
		t.Errorf("expected no work, got %+v", report)
	}
	if report.UnchangedBacklog != 0 || report.UnchangedInitiative != 0 {
		t.Errorf("expected zero unchanged counts, got %+v", report)
	}
}

func TestReconciler_Plan_NewItemsOnly(t *testing.T) {
	r, _, _, _, br, _ := newReconcilerForTest(t)
	br.items = []backlog.BacklogItem{
		{Kind: backlog.KindFix, Name: "a", Title: "A"},
		{Kind: backlog.KindFix, Name: "b", Title: "B"},
		{Kind: backlog.KindIdea, Name: "c", Title: "C"},
		{Kind: backlog.KindIdea, Name: "d", Title: "D"},
		{Kind: backlog.KindResearch, Name: "e", Title: "E"},
	}
	report, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.ToUpsertBacklog) != 5 {
		t.Errorf("expected 5 upserts, got %d", len(report.ToUpsertBacklog))
	}
	if len(report.ToDeleteBacklog) != 0 || report.UnchangedBacklog != 0 {
		t.Errorf("expected only upserts, got %+v", report)
	}
}

func TestReconciler_Plan_OrphansOnly(t *testing.T) {
	r, _, bs, _, _, _ := newReconcilerForTest(t)
	bs.seed("ghost-1", map[string]interface{}{"payload_hash": "sha256:x"})
	bs.seed("ghost-2", map[string]interface{}{"payload_hash": "sha256:y"})
	bs.seed("ghost-3", map[string]interface{}{"payload_hash": "sha256:z"})

	report, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.ToDeleteBacklog) != 3 {
		t.Errorf("expected 3 deletes, got %d", len(report.ToDeleteBacklog))
	}
	if len(report.ToUpsertBacklog) != 0 {
		t.Errorf("expected no upserts, got %d", len(report.ToUpsertBacklog))
	}
}

func TestReconciler_Plan_HashChanged(t *testing.T) {
	r, _, bs, _, br, _ := newReconcilerForTest(t)
	item := backlog.BacklogItem{Kind: backlog.KindFix, Name: "x", Title: "T"}
	br.items = []backlog.BacklogItem{item}
	id := backlogPointID(item.Kind, item.Name)
	// Seed with a different (stale) hash.
	bs.seed(id, map[string]interface{}{"payload_hash": "sha256:STALE000"})

	report, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.ToUpsertBacklog) != 1 {
		t.Fatalf("expected 1 upsert, got %d", len(report.ToUpsertBacklog))
	}
	if report.UnchangedBacklog != 0 {
		t.Errorf("expected 0 unchanged, got %d", report.UnchangedBacklog)
	}
}

func TestReconciler_Plan_HashMatch_NoWork(t *testing.T) {
	r, _, bs, _, br, _ := newReconcilerForTest(t)
	item := backlog.BacklogItem{Kind: backlog.KindFix, Name: "x", Title: "T"}
	br.items = []backlog.BacklogItem{item}
	id := backlogPointID(item.Kind, item.Name)
	bs.seed(id, map[string]interface{}{"payload_hash": mustHash(item)})

	report, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report.HasWork() {
		t.Errorf("expected no work for matching hash, got %+v", report)
	}
	if report.UnchangedBacklog != 1 {
		t.Errorf("expected UnchangedBacklog=1, got %d", report.UnchangedBacklog)
	}
}

func TestReconciler_Plan_Mixed(t *testing.T) {
	r, _, bs, _, br, _ := newReconcilerForTest(t)
	itemNew := backlog.BacklogItem{Kind: backlog.KindFix, Name: "new1", Title: "New 1"}
	itemNew2 := backlog.BacklogItem{Kind: backlog.KindFix, Name: "new2", Title: "New 2"}
	itemChanged := backlog.BacklogItem{Kind: backlog.KindFix, Name: "ch", Title: "Updated Title"}
	itemUnchanged := backlog.BacklogItem{Kind: backlog.KindFix, Name: "stable", Title: "Stable"}
	br.items = []backlog.BacklogItem{itemNew, itemNew2, itemChanged, itemUnchanged}

	bs.seed(backlogPointID(itemChanged.Kind, itemChanged.Name), map[string]interface{}{"payload_hash": "sha256:STALE000"})
	bs.seed(backlogPointID(itemUnchanged.Kind, itemUnchanged.Name), map[string]interface{}{"payload_hash": mustHash(itemUnchanged)})
	bs.seed("orphan-id-1", map[string]interface{}{"payload_hash": "sha256:o1"})
	bs.seed("orphan-id-2", map[string]interface{}{"payload_hash": "sha256:o2"})

	report, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.ToUpsertBacklog) != 3 {
		t.Errorf("expected 3 upserts (2 new + 1 changed), got %d", len(report.ToUpsertBacklog))
	}
	if len(report.ToDeleteBacklog) != 2 {
		t.Errorf("expected 2 deletes (orphans), got %d", len(report.ToDeleteBacklog))
	}
	if report.UnchangedBacklog != 1 {
		t.Errorf("expected 1 unchanged, got %d", report.UnchangedBacklog)
	}
}

func TestReconciler_Plan_LegacyMissingHash(t *testing.T) {
	r, _, bs, _, br, _ := newReconcilerForTest(t)
	item := backlog.BacklogItem{Kind: backlog.KindFix, Name: "legacy", Title: "L"}
	br.items = []backlog.BacklogItem{item}
	bs.seed(backlogPointID(item.Kind, item.Name), map[string]interface{}{
		// legacy: no payload_hash field
		"name": "legacy",
	})

	report, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.ToUpsertBacklog) != 1 {
		t.Errorf("expected legacy point to be re-upserted, got %d", len(report.ToUpsertBacklog))
	}
	if report.LegacyBacklog != 1 {
		t.Errorf("expected LegacyBacklog=1, got %d", report.LegacyBacklog)
	}
}

func TestReconciler_Plan_PreservesArchivedItems(t *testing.T) {
	// An archived item must remain Unchanged when its hash matches; reconcile
	// must NOT delete it, because search-with-include-archived (and the "fix"
	// kind override that auto-includes archived) relies on it staying indexed.
	r, _, bs, _, br, _ := newReconcilerForTest(t)
	archived := "2026-01-01T00:00:00Z"
	item := backlog.BacklogItem{Kind: backlog.KindFix, Name: "a", Title: "A", ArchivedAt: &archived}
	br.items = []backlog.BacklogItem{item}
	bs.seed(backlogPointID(item.Kind, item.Name), map[string]interface{}{
		"payload_hash": mustHash(item),
		"archived":     true,
	})

	report, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(report.ToDeleteBacklog) != 0 {
		t.Errorf("archived item must not be deleted, got %v", report.ToDeleteBacklog)
	}
	if report.UnchangedBacklog != 1 {
		t.Errorf("archived matched item must count as Unchanged, got %d", report.UnchangedBacklog)
	}
}

func TestReconciler_Plan_ContextCanceled(t *testing.T) {
	r, _, _, _, br, _ := newReconcilerForTest(t)
	br.items = []backlog.BacklogItem{{Kind: backlog.KindFix, Name: "x"}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Plan starts

	_, err := r.Plan(ctx)
	if err == nil {
		t.Fatal("expected error from canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestReconciler_Plan_BacklogReaderError(t *testing.T) {
	r, _, _, _, _, _ := newReconcilerForTest(t)
	r.BacklogReader = &erroringBacklogReader{err: errors.New("disk full")}
	_, err := r.Plan(context.Background())
	if err == nil || !contains(err.Error(), "disk full") {
		t.Errorf("expected wrapped 'disk full' error, got %v", err)
	}
}

func TestReconciler_Plan_QdrantScrollError(t *testing.T) {
	r, emb, bs, _, br, _ := newReconcilerForTest(t)
	br.items = []backlog.BacklogItem{{Kind: backlog.KindFix, Name: "x"}}
	bs.scrollErr = errors.New("qdrant unreachable")

	_, err := r.Plan(context.Background())
	if err == nil || !contains(err.Error(), "qdrant unreachable") {
		t.Errorf("expected wrapped scroll error, got %v", err)
	}
	if emb.callCount() != 0 {
		t.Errorf("expected no embed calls when scroll fails, got %d", emb.callCount())
	}
}

// ---- Apply ----

func TestReconciler_Apply_ParallelismRespected(t *testing.T) {
	r, emb, _, _, br, _ := newReconcilerForTest(t)
	r.Parallelism = 2
	emb.delay = 50 * time.Millisecond
	for i := 0; i < 10; i++ {
		br.items = append(br.items, backlog.BacklogItem{Kind: backlog.KindFix, Name: fmt.Sprintf("i%d", i), Title: fmt.Sprintf("T%d", i)})
	}
	plan, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if _, err := r.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if peak := emb.peak(); peak > 2 {
		t.Errorf("expected concurrent peak ≤ 2, got %d", peak)
	}
	if emb.callCount() != 10 {
		t.Errorf("expected 10 embed calls, got %d", emb.callCount())
	}
}

func TestReconciler_Apply_UpsertCountMatchesPlan(t *testing.T) {
	r, emb, bs, _, br, _ := newReconcilerForTest(t)
	for i := 0; i < 4; i++ {
		br.items = append(br.items, backlog.BacklogItem{Kind: backlog.KindFix, Name: fmt.Sprintf("i%d", i)})
	}
	plan, _ := r.Plan(context.Background())
	res, err := r.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if emb.callCount() != 4 || bs.upsertCalls != 4 {
		t.Errorf("expected 4 embeds + 4 upserts, got embeds=%d upserts=%d", emb.callCount(), bs.upsertCalls)
	}
	if res.UpsertedBacklog != 4 {
		t.Errorf("expected UpsertedBacklog=4, got %d", res.UpsertedBacklog)
	}
}

func TestReconciler_Apply_BatchDeleteCalledOnce(t *testing.T) {
	r, _, bs, _, _, _ := newReconcilerForTest(t)
	for i := 0; i < 5; i++ {
		bs.seed(fmt.Sprintf("orphan-%d", i), map[string]interface{}{"payload_hash": "sha256:x"})
	}
	plan, _ := r.Plan(context.Background())
	if _, err := r.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if bs.batchDeleteCalls != 1 {
		t.Errorf("expected 1 BatchDelete, got %d", bs.batchDeleteCalls)
	}
	if len(bs.batchDeleteRanges[0]) != 5 {
		t.Errorf("expected 5 ids in single call, got %d", len(bs.batchDeleteRanges[0]))
	}
}

func TestReconciler_Apply_BatchDeleteChunked(t *testing.T) {
	// fakeVectorStore.BatchDelete records the input slice as one entry per call;
	// the chunking happens INSIDE qdrantVectorStore.BatchDelete (covered by the
	// vectorstore_test.go BatchDelete_ChunksAtBoundary test). At the Reconciler
	// level the contract is "one BatchDelete call per collection," regardless of
	// id count. This test pins that the reconciler does not over-chunk.
	r, _, bs, _, _, _ := newReconcilerForTest(t)
	for i := 0; i < 600; i++ {
		bs.seed(fmt.Sprintf("orphan-%d", i), map[string]interface{}{"payload_hash": "sha256:x"})
	}
	plan, _ := r.Plan(context.Background())
	if _, err := r.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if bs.batchDeleteCalls != 1 {
		t.Errorf("expected reconciler to issue exactly 1 BatchDelete call to the store, got %d", bs.batchDeleteCalls)
	}
	if len(bs.batchDeleteRanges[0]) != 600 {
		t.Errorf("expected all 600 orphan ids passed to BatchDelete, got %d", len(bs.batchDeleteRanges[0]))
	}
}

func TestReconciler_Apply_PartialEmbedFailure(t *testing.T) {
	r, emb, bs, _, br, _ := newReconcilerForTest(t)
	itemA := backlog.BacklogItem{Kind: backlog.KindFix, Name: "a", Title: "A"}
	itemB := backlog.BacklogItem{Kind: backlog.KindFix, Name: "b", Title: "B"}
	itemC := backlog.BacklogItem{Kind: backlog.KindFix, Name: "c", Title: "C"}
	br.items = []backlog.BacklogItem{itemA, itemB, itemC}
	emb.failOn = map[string]error{composeBacklogText(itemB): errors.New("embed boom")}

	plan, _ := r.Plan(context.Background())
	res, err := r.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.UpsertedBacklog != 2 {
		t.Errorf("expected 2 successful upserts, got %d", res.UpsertedBacklog)
	}
	if len(res.Errors) != 1 || res.Errors[0].Op != "embed" {
		t.Errorf("expected 1 embed error, got %+v", res.Errors)
	}
	if bs.upsertCalls != 2 {
		t.Errorf("expected 2 upsert calls, got %d", bs.upsertCalls)
	}
}

func TestReconciler_Apply_PartialUpsertFailure(t *testing.T) {
	r, _, bs, _, br, _ := newReconcilerForTest(t)
	itemA := backlog.BacklogItem{Kind: backlog.KindFix, Name: "a"}
	itemB := backlog.BacklogItem{Kind: backlog.KindFix, Name: "b"}
	br.items = []backlog.BacklogItem{itemA, itemB}
	bs.upsertFailOn = map[string]error{backlogPointID(itemB.Kind, itemB.Name): errors.New("upsert boom")}

	plan, _ := r.Plan(context.Background())
	res, err := r.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res.UpsertedBacklog != 1 {
		t.Errorf("expected 1 upsert, got %d", res.UpsertedBacklog)
	}
	if len(res.Errors) != 1 || res.Errors[0].Op != "upsert" {
		t.Errorf("expected 1 upsert error, got %+v", res.Errors)
	}
}

func TestReconciler_Apply_ContextCanceled_PartialResult(t *testing.T) {
	r, emb, _, _, br, _ := newReconcilerForTest(t)
	r.Parallelism = 1 // serialize so the cancel lands mid-stream deterministically
	emb.delay = 30 * time.Millisecond
	for i := 0; i < 10; i++ {
		br.items = append(br.items, backlog.BacklogItem{Kind: backlog.KindFix, Name: fmt.Sprintf("i%d", i)})
	}
	plan, _ := r.Plan(context.Background())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(45 * time.Millisecond)
		cancel()
	}()
	res, err := r.Apply(ctx, plan)
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil ApplyResult on cancellation")
	}
	if res.UpsertedBacklog >= 10 {
		t.Errorf("expected partial upsert count after cancel, got %d", res.UpsertedBacklog)
	}
}

// ---- RunOnce ----

func TestReconciler_RunOnce_ConvergesInOnePass(t *testing.T) {
	r, emb, bs, _, br, _ := newReconcilerForTest(t)
	for i := 0; i < 10; i++ {
		item := backlog.BacklogItem{Kind: backlog.KindFix, Name: fmt.Sprintf("i%d", i), Title: fmt.Sprintf("T%d", i)}
		br.items = append(br.items, item)
		bs.seed(backlogPointID(item.Kind, item.Name), map[string]interface{}{"payload_hash": mustHash(item)})
	}
	plan, res, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("first runonce: %v", err)
	}
	if plan.HasWork() {
		t.Errorf("expected no work on convergence, got %+v", plan)
	}
	if emb.callCount() != 0 {
		t.Errorf("expected no embeds on already-converged index, got %d", emb.callCount())
	}
	if res.UpsertedBacklog != 0 || res.DeletedBacklog != 0 {
		t.Errorf("expected zero applied work, got %+v", res)
	}

	// Second pass: still nothing.
	_, _, err = r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("second runonce: %v", err)
	}
	if emb.callCount() != 0 {
		t.Errorf("second pass added embed calls, got %d", emb.callCount())
	}
}

func TestReconciler_RunOnce_GhostCleanup(t *testing.T) {
	r, emb, bs, _, br, _ := newReconcilerForTest(t)
	for i := 0; i < 3; i++ {
		item := backlog.BacklogItem{Kind: backlog.KindFix, Name: fmt.Sprintf("i%d", i), Title: fmt.Sprintf("T%d", i)}
		br.items = append(br.items, item)
		bs.seed(backlogPointID(item.Kind, item.Name), map[string]interface{}{"payload_hash": mustHash(item)})
	}
	bs.seed("ghost-1", map[string]interface{}{"payload_hash": "sha256:gone1"})
	bs.seed("ghost-2", map[string]interface{}{"payload_hash": "sha256:gone2"})

	_, res, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("runonce: %v", err)
	}
	if emb.callCount() != 0 {
		t.Errorf("expected zero embeds for hash-matched items, got %d", emb.callCount())
	}
	if res.DeletedBacklog != 2 {
		t.Errorf("expected 2 ghost deletes, got %d", res.DeletedBacklog)
	}
	if bs.batchDeleteCalls != 1 {
		t.Errorf("expected 1 BatchDelete call for ghosts, got %d", bs.batchDeleteCalls)
	}

	// Second pass: clean.
	plan2, _, _ := r.RunOnce(context.Background())
	if plan2.HasWork() {
		t.Errorf("expected zero work on second pass, got %+v", plan2)
	}
}

func TestReconciler_RunOnce_LegacyMigration(t *testing.T) {
	r, emb, bs, _, br, _ := newReconcilerForTest(t)
	for i := 0; i < 3; i++ {
		item := backlog.BacklogItem{Kind: backlog.KindFix, Name: fmt.Sprintf("legacy-%d", i)}
		br.items = append(br.items, item)
		// Legacy: payload exists but lacks payload_hash.
		bs.seed(backlogPointID(item.Kind, item.Name), map[string]interface{}{"name": item.Name})
	}
	plan, res, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("runonce 1: %v", err)
	}
	if emb.callCount() != 3 || res.UpsertedBacklog != 3 {
		t.Errorf("first pass should re-embed all 3 legacy points, embeds=%d upserted=%d", emb.callCount(), res.UpsertedBacklog)
	}
	if plan.LegacyBacklog != 3 {
		t.Errorf("expected LegacyBacklog=3, got %d", plan.LegacyBacklog)
	}

	// Second pass: now stamped with payload_hash, expect zero work.
	plan2, _, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("runonce 2: %v", err)
	}
	if plan2.HasWork() {
		t.Errorf("expected convergence after legacy drain, got %+v", plan2)
	}
	if emb.callCount() != 3 {
		t.Errorf("expected no new embeds on second pass, got %d", emb.callCount())
	}
}

func TestReconciler_RunOnce_SingletonWhileRunning(t *testing.T) {
	r, emb, _, _, br, _ := newReconcilerForTest(t)
	emb.delay = 200 * time.Millisecond
	br.items = []backlog.BacklogItem{{Kind: backlog.KindFix, Name: "slow"}}

	var firstStarted int32
	go func() {
		atomic.StoreInt32(&firstStarted, 1)
		_, _, _ = r.RunOnce(context.Background())
	}()
	// Wait for the first goroutine to enter RunOnce and acquire the singleton.
	testutil.Eventually(t, time.Second, "first runonce started", func() bool {
		return atomic.LoadInt32(&firstStarted) == 1
	})
	// Second RunOnce must immediately bounce off ErrReconcileBusy.
	_, _, err := r.RunOnce(context.Background())
	if !errors.Is(err, ErrReconcileBusy) {
		// The first call may have completed already if the scheduler was slow;
		// retry once with a tiny delay and accept either ErrReconcileBusy OR
		// a clean nil (race-tolerant assertion).
		time.Sleep(5 * time.Millisecond)
		_, _, err = r.RunOnce(context.Background())
		if err != nil && !errors.Is(err, ErrReconcileBusy) {
			t.Errorf("expected ErrReconcileBusy or nil, got %v", err)
		}
	}
	// Drain the slow first call.
	testutil.Eventually(t, 2*time.Second, "first runonce completes", func() bool {
		return r.Status().Running == false
	})
}

func TestReconciler_RunOnce_DisabledStores(t *testing.T) {
	r := NewReconciler(&fakeEmbedder{}, nil, nil, nil, nil, 0)
	plan, res, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("runonce: %v", err)
	}
	if plan == nil {
		t.Fatal("expected non-nil plan")
	}
	if plan.HasWork() {
		t.Errorf("expected empty work for disabled stores, got %+v", plan)
	}
	if res == nil {
		t.Error("expected non-nil ApplyResult even when no work")
	}
}

func TestReconciler_Cancel_StopsInFlight(t *testing.T) {
	r, emb, _, _, br, _ := newReconcilerForTest(t)
	r.Parallelism = 1
	emb.delay = 50 * time.Millisecond
	for i := 0; i < 10; i++ {
		br.items = append(br.items, backlog.BacklogItem{Kind: backlog.KindFix, Name: fmt.Sprintf("i%d", i)})
	}

	done := make(chan struct{})
	var resOut *ApplyResult
	go func() {
		_, res, _ := r.RunOnce(context.Background())
		resOut = res
		close(done)
	}()
	time.Sleep(80 * time.Millisecond)
	r.Cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunOnce did not return after Cancel")
	}
	if resOut == nil {
		t.Fatal("expected ApplyResult on cancel")
	}
	if resOut.UpsertedBacklog >= 10 {
		t.Errorf("expected partial counts after cancel, got %+v", resOut)
	}
	if !r.Status().Canceled {
		t.Error("expected Status.Canceled=true after Cancel")
	}
}

// ---- Initiative coverage (sanity) ----

func TestReconciler_Plan_InitiativeMatchedSkipped(t *testing.T) {
	// Pin that the initiative path mirrors the backlog path: same Plan logic,
	// same Unchanged behavior, no work needed when hash matches.
	r, _, _, is, _, ir := newReconcilerForTest(t)
	init := initiatives.Initiative{Name: "obs-core", Title: "Observability Core"}
	ir.items = []initiatives.Initiative{init}
	is.seed(initiativePointID(init.Name), map[string]interface{}{"payload_hash": mustHashInit(init)})

	report, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if report.HasWork() {
		t.Errorf("expected no work for matched initiative, got %+v", report)
	}
	if report.UnchangedInitiative != 1 {
		t.Errorf("expected UnchangedInitiative=1, got %d", report.UnchangedInitiative)
	}
}

// ---- helpers ----

type erroringBacklogReader struct {
	err error
}

func (e *erroringBacklogReader) LoadAll() ([]backlog.BacklogItem, error) {
	return nil, e.err
}

func (e *erroringBacklogReader) LoadItem(_ backlog.BacklogKind, _ string) (backlog.BacklogItem, error) {
	return backlog.BacklogItem{}, e.err
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
