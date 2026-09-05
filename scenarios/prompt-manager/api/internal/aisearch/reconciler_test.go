package aisearch

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Fakes -------------------------------------------------------------------

type fakeStore struct {
	mu          sync.Mutex
	points      map[string]map[string]interface{}
	upserts     int
	batchCalls  int
	scrollCalls int
	scrollErr   error
	upsertErr   error
	batchErr    error
}

func newFakeStore() *fakeStore { return &fakeStore{points: map[string]map[string]interface{}{}} }

func (f *fakeStore) EnsureCollection(context.Context) error { return nil }
func (f *fakeStore) Available(context.Context) bool         { return true }

func (f *fakeStore) Upsert(_ context.Context, id string, _ []float64, payload map[string]interface{}) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upserts++
	cp := make(map[string]interface{}, len(payload))
	for k, v := range payload {
		cp[k] = v
	}
	f.points[id] = cp
	return nil
}

func (f *fakeStore) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.points, id)
	return nil
}

func (f *fakeStore) BatchDelete(_ context.Context, ids []string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.batchErr != nil {
		return f.batchErr
	}
	f.batchCalls++
	for _, id := range ids {
		delete(f.points, id)
	}
	return nil
}

func (f *fakeStore) Search(context.Context, []float64, int, float64) ([]SearchResult, error) {
	return nil, nil
}

func (f *fakeStore) CountPoints(context.Context) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.points), nil
}

func (f *fakeStore) ScrollIDs(context.Context) (map[string]ScrollItem, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.scrollCalls++
	if f.scrollErr != nil {
		return nil, f.scrollErr
	}
	out := make(map[string]ScrollItem, len(f.points))
	for id, p := range f.points {
		h, _ := p[payloadHashKey].(string)
		out[id] = ScrollItem{PayloadHash: h}
	}
	return out, nil
}

// preload puts a point with a synthetic hash into the store, simulating prior
// reconcile state.
func (f *fakeStore) preload(id, hash string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.points[id] = map[string]interface{}{payloadHashKey: hash}
}

type fakeEmbedder struct {
	mu             sync.Mutex
	calls          int
	failOn         map[string]bool
	delay          time.Duration
	concurrent     int32
	peakConcurrent int32
}

func (f *fakeEmbedder) Embed(ctx context.Context, text string) ([]float64, error) {
	n := atomic.AddInt32(&f.concurrent, 1)
	defer atomic.AddInt32(&f.concurrent, -1)
	for {
		peak := atomic.LoadInt32(&f.peakConcurrent)
		if n <= peak || atomic.CompareAndSwapInt32(&f.peakConcurrent, peak, n) {
			break
		}
	}
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	if f.failOn[text] {
		return nil, errors.New("embed-fail: " + text)
	}
	return []float64{0.1, 0.2, 0.3}, nil
}
func (f *fakeEmbedder) Available(context.Context) bool { return true }

// --- Test descriptor ---------------------------------------------------------

type testItem struct {
	ID   string
	Text string
	Body map[string]interface{}
}

// newTestDescriptor returns a CollectionDescriptor that walks an in-memory
// slice. The returned function lets tests mutate the slice between Plan calls.
func newTestDescriptor(kind EntityKind, store VectorStore) (CollectionDescriptor, *[]testItem) {
	items := &[]testItem{}
	d := CollectionDescriptor{
		Kind:  kind,
		Store: store,
		LoadAll: func(ctx context.Context) ([]ItemSnapshot, error) {
			out := make([]ItemSnapshot, 0, len(*items))
			for i := range *items {
				v := (*items)[i]
				out = append(out, &v)
			}
			return out, nil
		},
		ComposeText: func(snap ItemSnapshot) string { return snap.(*testItem).Text },
		BuildPayload: func(snap ItemSnapshot, text string) map[string]interface{} {
			it := snap.(*testItem)
			p := map[string]interface{}{"id": it.ID}
			for k, v := range it.Body {
				p[k] = v
			}
			p[payloadHashKey] = composePayloadHash(text, p)
			return p
		},
		PointID:     func(snap ItemSnapshot) string { return "pt-" + snap.(*testItem).ID },
		DisplayName: func(snap ItemSnapshot) string { return snap.(*testItem).ID },
	}
	return d, items
}

// computeExpectedHash mirrors what the descriptor's BuildPayload + composePayloadHash
// would produce for a given text and body.
func computeExpectedHash(id, text string, body map[string]interface{}) string {
	p := map[string]interface{}{"id": id}
	for k, v := range body {
		p[k] = v
	}
	return composePayloadHash(text, p)
}

// --- Plan tests --------------------------------------------------------------

func TestReconciler_Plan_AllCollectionsEmpty(t *testing.T) {
	store := newFakeStore()
	desc, _ := newTestDescriptor(KindSkill, store)
	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)

	plan, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.HasWork() {
		t.Errorf("expected no work")
	}
	if len(plan.Collections) != 1 {
		t.Fatalf("expected 1 collection in plan, got %d", len(plan.Collections))
	}
	c := plan.Collections[0]
	if c.UnchangedCount != 0 || c.LegacyCount != 0 {
		t.Errorf("expected zero counters, got %+v", c)
	}
}

func TestReconciler_Plan_NewItemsOnly(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{
		{ID: "a", Text: "foo"},
		{ID: "b", Text: "bar"},
		{ID: "c", Text: "baz"},
	}
	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)

	plan, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	c := plan.Collections[0]
	if len(c.ToUpsert) != 3 {
		t.Errorf("expected 3 ToUpsert, got %d", len(c.ToUpsert))
	}
	if len(c.ToDelete) != 0 {
		t.Errorf("expected no deletes, got %d", len(c.ToDelete))
	}
}

func TestReconciler_Plan_OrphansOnly(t *testing.T) {
	store := newFakeStore()
	store.preload("pt-x", "sha256:01")
	store.preload("pt-y", "sha256:02")
	desc, _ := newTestDescriptor(KindSkill, store)
	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)

	plan, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	c := plan.Collections[0]
	if len(c.ToDelete) != 2 {
		t.Errorf("expected 2 ToDelete, got %v", c.ToDelete)
	}
	if len(c.ToUpsert) != 0 {
		t.Errorf("expected 0 upserts, got %d", len(c.ToUpsert))
	}
}

func TestReconciler_Plan_HashMatch_NoWork(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "foo"}}
	hash := computeExpectedHash("a", "foo", nil)
	store.preload("pt-a", hash)

	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)
	plan, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	c := plan.Collections[0]
	if c.UnchangedCount != 1 {
		t.Errorf("expected 1 unchanged, got %d", c.UnchangedCount)
	}
	if len(c.ToUpsert) != 0 || len(c.ToDelete) != 0 {
		t.Errorf("expected zero work, got %+v", c)
	}
}

func TestReconciler_Plan_HashChanged(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "v2"}}
	store.preload("pt-a", "sha256:stalehash")

	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)
	plan, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Collections[0].ToUpsert) != 1 {
		t.Errorf("expected 1 upsert, got %d", len(plan.Collections[0].ToUpsert))
	}
}

func TestReconciler_Plan_LegacyMissingHash(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "v1"}}
	store.preload("pt-a", "") // legacy: empty hash

	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)
	plan, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	c := plan.Collections[0]
	if c.LegacyCount != 1 {
		t.Errorf("expected LegacyCount=1, got %d", c.LegacyCount)
	}
	if len(c.ToUpsert) != 1 {
		t.Errorf("expected 1 ToUpsert, got %d", len(c.ToUpsert))
	}
}

func TestReconciler_Plan_PerCollectionFailureDoesNotAbort(t *testing.T) {
	good := newFakeStore()
	bad := newFakeStore()
	bad.scrollErr = errors.New("scroll boom")

	dGood, gItems := newTestDescriptor(KindSkill, good)
	dBad, bItems := newTestDescriptor(KindAgent, bad)
	*gItems = []testItem{{ID: "g1", Text: "ok"}}
	*bItems = []testItem{{ID: "b1", Text: "ok"}}

	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{dBad, dGood}, 1)
	plan, err := r.Plan(context.Background())
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Collections) != 2 {
		t.Fatalf("expected 2 collections, got %d", len(plan.Collections))
	}
	// Find each kind by name.
	got := map[EntityKind]CollectionDriftReport{}
	for _, c := range plan.Collections {
		got[c.Kind] = c
	}
	if len(got[KindSkill].ToUpsert) != 1 {
		t.Errorf("skill should still plan: got %+v", got[KindSkill])
	}
	if len(got[KindAgent].ToUpsert) != 0 {
		t.Errorf("agent (failed scroll) should be empty: got %+v", got[KindAgent])
	}
}

func TestReconciler_Plan_ContextCanceled(t *testing.T) {
	store := newFakeStore()
	desc, _ := newTestDescriptor(KindSkill, store)
	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := r.Plan(ctx)
	if err == nil {
		t.Fatal("expected ctx error")
	}
}

// --- Apply tests -------------------------------------------------------------

func TestReconciler_Apply_UpsertAndDeleteCounts(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "foo"}, {ID: "b", Text: "bar"}}
	store.preload("pt-ghost", "sha256:dead")

	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 4)

	plan, _ := r.Plan(context.Background())
	apply, err := r.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if len(apply.Collections) != 1 {
		t.Fatalf("expected 1 collection, got %d", len(apply.Collections))
	}
	got := apply.Collections[0]
	if got.Upserted != 2 {
		t.Errorf("Upserted = %d, want 2", got.Upserted)
	}
	if got.Deleted != 1 {
		t.Errorf("Deleted = %d, want 1", got.Deleted)
	}
	if store.batchCalls != 1 {
		t.Errorf("expected 1 batch delete call, got %d", store.batchCalls)
	}
}

func TestReconciler_Apply_ParallelismRespected(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	for i := 0; i < 10; i++ {
		*items = append(*items, testItem{ID: fmt.Sprintf("i%d", i), Text: fmt.Sprintf("t%d", i)})
	}
	emb := &fakeEmbedder{delay: 30 * time.Millisecond}
	r := NewReconciler(emb, []CollectionDescriptor{desc}, 2)

	plan, _ := r.Plan(context.Background())
	if _, err := r.Apply(context.Background(), plan); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if peak := atomic.LoadInt32(&emb.peakConcurrent); peak > 2 {
		t.Errorf("expected peak concurrency <= 2, got %d", peak)
	}
}

func TestReconciler_Apply_PartialEmbedFailure(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{
		{ID: "a", Text: "ok-a"},
		{ID: "b", Text: "boom"},
		{ID: "c", Text: "ok-c"},
	}
	emb := &fakeEmbedder{failOn: map[string]bool{"boom": true}}
	r := NewReconciler(emb, []CollectionDescriptor{desc}, 1)

	plan, _ := r.Plan(context.Background())
	apply, err := r.Apply(context.Background(), plan)
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if apply.Collections[0].Upserted != 2 {
		t.Errorf("expected 2 upserts, got %d", apply.Collections[0].Upserted)
	}
	if len(apply.Errors) != 1 || apply.Errors[0].Op != "embed" {
		t.Errorf("expected one embed error, got %+v", apply.Errors)
	}
}

// --- RunOnce tests -----------------------------------------------------------

func TestReconciler_RunOnce_ConvergesInOnePass(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "foo"}, {ID: "b", Text: "bar"}}
	emb := &fakeEmbedder{}
	r := NewReconciler(emb, []CollectionDescriptor{desc}, 1)

	// First run: 2 embeds, 2 upserts.
	if _, _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	firstCalls := emb.calls
	if firstCalls != 2 {
		t.Errorf("expected 2 embed calls on first run, got %d", firstCalls)
	}

	// Second run with no on-disk changes: zero embeds, zero upserts.
	plan, apply, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce 2: %v", err)
	}
	if emb.calls != firstCalls {
		t.Errorf("expected 0 new embed calls, got %d (total %d)", emb.calls-firstCalls, emb.calls)
	}
	if plan.HasWork() {
		t.Errorf("expected no work on second run, got %+v", plan.Collections)
	}
	if apply.Collections[0].Upserted != 0 || apply.Collections[0].Deleted != 0 {
		t.Errorf("expected no apply work, got %+v", apply.Collections[0])
	}
}

func TestReconciler_RunOnce_GhostCleanup(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "foo"}}
	store.preload("pt-ghost", "sha256:zz")

	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)

	if _, _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if _, ok := store.points["pt-ghost"]; ok {
		t.Errorf("ghost should have been deleted")
	}

	// Second pass: zero work.
	plan, _, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce 2: %v", err)
	}
	if plan.HasWork() {
		t.Errorf("expected no work after convergence, got %+v", plan.Collections)
	}
}

func TestReconciler_RunOnce_LegacyMigration(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "foo"}}
	store.preload("pt-a", "") // legacy: missing hash

	emb := &fakeEmbedder{}
	r := NewReconciler(emb, []CollectionDescriptor{desc}, 1)

	if _, _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if emb.calls != 1 {
		t.Errorf("expected 1 embed on legacy migration, got %d", emb.calls)
	}

	// Second pass: zero work.
	plan, _, _ := r.RunOnce(context.Background())
	if plan.HasWork() {
		t.Errorf("expected no work after legacy migration, got %+v", plan.Collections)
	}
}

func TestReconciler_RunOnce_SingletonWhileRunning(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "foo"}}
	emb := &fakeEmbedder{delay: 100 * time.Millisecond}

	r := NewReconciler(emb, []CollectionDescriptor{desc}, 1)

	done := make(chan error, 1)
	go func() {
		_, _, err := r.RunOnce(context.Background())
		done <- err
	}()

	// Give the first run a chance to enter.
	time.Sleep(20 * time.Millisecond)

	_, _, err := r.RunOnce(context.Background())
	if !errors.Is(err, ErrReconcileBusy) {
		t.Errorf("expected ErrReconcileBusy, got %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("first run failed: %v", err)
	}
}

func TestReconciler_RunOnce_AllStoresNil(t *testing.T) {
	r := NewReconciler(&fakeEmbedder{}, nil, 1)
	plan, apply, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if plan.HasWork() {
		t.Errorf("expected no work, got %+v", plan)
	}
	if apply == nil {
		t.Fatal("expected non-nil apply")
	}
}

func TestReconciler_NewReconciler_ClampsParallelism(t *testing.T) {
	r := NewReconciler(&fakeEmbedder{}, nil, 0)
	if r.Parallelism != DefaultReconcileParallelism {
		t.Errorf("expected default parallelism, got %d", r.Parallelism)
	}
	r2 := NewReconciler(&fakeEmbedder{}, nil, 999)
	if r2.Parallelism != MaxReconcileParallelism {
		t.Errorf("expected clamped parallelism %d, got %d", MaxReconcileParallelism, r2.Parallelism)
	}
}

func TestReconciler_Status_ReflectsLastRun(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	*items = []testItem{{ID: "a", Text: "foo"}}
	r := NewReconciler(&fakeEmbedder{}, []CollectionDescriptor{desc}, 1)

	if _, _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	st := r.Status()
	if st.Running {
		t.Errorf("Status.Running should be false after run completes")
	}
	if st.LastPlan == nil || st.LastResult == nil {
		t.Errorf("expected LastPlan + LastResult to be set, got %+v", st)
	}
	if st.StartedAt == "" || st.FinishedAt == "" {
		t.Errorf("expected timestamps set, got start=%q finish=%q", st.StartedAt, st.FinishedAt)
	}
}

func TestReconciler_Cancel_StopsInFlight(t *testing.T) {
	store := newFakeStore()
	desc, items := newTestDescriptor(KindSkill, store)
	for i := 0; i < 5; i++ {
		*items = append(*items, testItem{ID: fmt.Sprintf("i%d", i), Text: fmt.Sprintf("t%d", i)})
	}
	emb := &fakeEmbedder{delay: 200 * time.Millisecond}
	r := NewReconciler(emb, []CollectionDescriptor{desc}, 1)

	done := make(chan struct{})
	go func() {
		_, _, _ = r.RunOnce(context.Background())
		close(done)
	}()

	time.Sleep(40 * time.Millisecond)
	r.Cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunOnce did not exit after Cancel")
	}

	st := r.Status()
	if !st.Canceled {
		t.Errorf("expected Status.Canceled true, got %+v", st)
	}
}
