package aisearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
	"swarm-manager/internal/testutil"
)

// saveBacklogItem mkdirs the item directory before calling SaveItem, matching
// what production write-handlers do (see backlog/handler_create.go,
// batch_handler.go, import.go — all MkdirAll before SaveItem). Defaults
// status to backlog when callers don't care — matches real handler intake.
func saveBacklogItem(t *testing.T, store *backlog.FileStore, item backlog.BacklogItem) {
	t.Helper()
	if item.Status == "" {
		item.Status = backlog.StatusBacklog
	}
	if err := os.MkdirAll(store.ItemDir(item.Kind, item.Name), 0o755); err != nil {
		t.Fatalf("mkdir item dir: %v", err)
	}
	if err := store.SaveItem(item); err != nil {
		t.Fatalf("SaveItem: %v", err)
	}
}

// buildTestService wires real stores behind a shared *Service backed by the
// supplied mock ollama/qdrant servers. Returns the store handles for test
// assertions.
func buildTestService(t *testing.T, embedder Embedder, qdrantURL string) (*Service, *backlog.FileStore, *goals.Service, string) {
	t.Helper()
	root := t.TempDir()
	// Pre-create the kind directories so LoadAll doesn't error.
	for _, d := range []string{"ideas", "research", "fix", "execute", "chore", "goals"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	bStore := backlog.NewFileStore(root)
	goalService := goals.NewService(goals.NewStore(root), bStore)
	backlogVS := NewVectorStore(qdrantURL, "", "sm-b", 3)
	goalVS := NewVectorStore(qdrantURL, "", "sm-g", 3)
	svc := NewService(
		embedder, backlogVS, goalVS,
		NewBacklogStoreAdapter(bStore),
		NewGoalServiceAdapter(goalService),
		0.5,
	)
	// Wire the write-through seam.
	bStore.SetAIIndexer(svc)
	goalService.SetAIIndexer(svc)

	return svc, bStore, goalService, root
}

func TestIntegration_BacklogSave_FiresIndexUpsert(t *testing.T) {
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	_, bStore, _, _ := buildTestService(t, fakeEmbedderOK(), qServer.URL)

	saveBacklogItem(t, bStore, backlog.BacklogItem{
		Name:  "alpha",
		Title: "Alpha",
		Kind:  backlog.KindExecute,
	})
	testutil.Eventually(t, 2*time.Second, "backlog save index upsert", func() bool {
		return atomic.LoadInt32(&qStub.upsertCalls) >= 1
	})
}

func TestIntegration_BacklogDelete_FiresIndexDelete(t *testing.T) {
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	_, bStore, _, _ := buildTestService(t, fakeEmbedderOK(), qServer.URL)

	saveBacklogItem(t, bStore, backlog.BacklogItem{
		Name:  "alpha",
		Title: "Alpha",
		Kind:  backlog.KindExecute,
	})
	// Wait for the initial upsert to land so we don't race it.
	testutil.Eventually(t, 2*time.Second, "initial backlog save index upsert", func() bool {
		return atomic.LoadInt32(&qStub.upsertCalls) >= 1
	})

	if err := bStore.DeleteItem(backlog.KindExecute, "alpha"); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	testutil.Eventually(t, 2*time.Second, "backlog delete index cleanup", func() bool {
		return atomic.LoadInt32(&qStub.deleteCalls) >= 1
	})
}

func TestIntegration_GoalCreate_FiresIndexUpsert(t *testing.T) {
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	_, _, goalService, _ := buildTestService(t, fakeEmbedderOK(), qServer.URL)

	if _, err := goalService.Create(goals.CreateRequest{Name: "obs-core", Title: "Observability Core"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	testutil.Eventually(t, 2*time.Second, "goal create index upsert", func() bool {
		return atomic.LoadInt32(&qStub.upsertCalls) >= 1
	})
}

func TestIntegration_QdrantFailure_DoesNotBreakCRUD(t *testing.T) {
	// Core fire-and-forget invariant: CRUD must succeed even when Qdrant
	// returns 500 on every call.
	qServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer qServer.Close()

	_, bStore, goalService, _ := buildTestService(t, fakeEmbedderOK(), qServer.URL)

	// CRUD calls must not propagate the upstream 500.
	for i := 0; i < 5; i++ {
		saveBacklogItem(t, bStore, backlog.BacklogItem{Name: "bad", Title: "x", Kind: backlog.KindIdea})
	}
	if err := bStore.DeleteItem(backlog.KindIdea, "bad"); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if _, err := goalService.Create(goals.CreateRequest{Name: "x", Title: "X"}); err != nil {
		t.Fatalf("goal Create: %v", err)
	}
	// Fixed sleep intentionally gives fire-and-forget goroutines a short
	// window to hit the failing Qdrant seam; the assertion is that CRUD did
	// not observe those asynchronous failures.
	time.Sleep(100 * time.Millisecond)
}

func TestIntegration_OllamaEmpty_CRUDStillSucceeds(t *testing.T) {
	// If OLLAMA_URL is empty, the indexer reports errors internally but CRUD
	// must still succeed — operators can bring up the stack incrementally.
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	_, bStore, _, _ := buildTestService(t, fakeEmbedderErr(), qServer.URL)

	saveBacklogItem(t, bStore, backlog.BacklogItem{Name: "alpha", Title: "A", Kind: backlog.KindIdea})
	// Fixed sleep intentionally validates no background upsert succeeds when
	// Ollama is disabled; the positive contract above is that CRUD returned nil.
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&qStub.upsertCalls) != 0 {
		t.Errorf("expected 0 upserts (ollama disabled), got %d", qStub.upsertCalls)
	}
}

func TestIntegration_Status_ReflectsOnDiskCounts(t *testing.T) {
	qStub := &qdrantStub{count: 0}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc, bStore, goalService, _ := buildTestService(t, fakeEmbedderOK(), qServer.URL)

	for _, name := range []string{"a", "b", "c"} {
		saveBacklogItem(t, bStore, backlog.BacklogItem{Name: name, Kind: backlog.KindIdea, Title: name})
	}
	if _, err := goalService.Create(goals.CreateRequest{Name: "i1", Title: "I1"}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	st := svc.GetStatus(context.Background())
	if st.OnDiskBacklog != 3 {
		t.Errorf("expected 3 backlog items on disk, got %d", st.OnDiskBacklog)
	}
	if st.OnDiskGoals != 1 {
		t.Errorf("expected 1 goal on disk, got %d", st.OnDiskGoals)
	}
}

// buildTestReconciler wires real on-disk stores (via adapters) and in-memory
// VectorStore + Embedder fakes into a Reconciler. This is the integration-test
// pairing that exercises the disk → reconciler → index pipeline end-to-end
// without HTTP overhead. Returns the reconciler plus the disk handles so the
// test can mutate disk between RunOnce calls and observe convergence.
func buildTestReconciler(t *testing.T) (*Reconciler, *backlog.FileStore, *goals.Store, *fakeVectorStore, *fakeVectorStore, *fakeEmbedder) {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"ideas", "research", "fix", "execute", "chore", "goals"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	bStore := backlog.NewFileStore(root)
	goalStore := goals.NewStore(root)
	goalService := goals.NewService(goalStore, bStore)

	emb := &fakeEmbedder{}
	bs := &fakeVectorStore{}
	gs := &fakeVectorStore{}
	r := NewReconciler(emb, bs, gs,
		NewBacklogStoreAdapter(bStore),
		NewGoalServiceAdapter(goalService),
		1,
	)
	return r, bStore, goalStore, bs, gs, emb
}

func TestIntegration_Reconciler_PopulatesEmptyIndex(t *testing.T) {
	r, bStore, goalStore, bs, _, emb := buildTestReconciler(t)

	// Seed disk directly (no SetAIIndexer hook) so the only path to qdrant
	// is the reconciler.
	saveBacklogItem(t, bStore, backlog.BacklogItem{Name: "a", Title: "A", Kind: backlog.KindIdea})
	saveBacklogItem(t, bStore, backlog.BacklogItem{Name: "b", Title: "B", Kind: backlog.KindIdea})
	if err := goalStore.Save(&goals.Goal{Name: "i1", Title: "I1", Status: goals.StatusActive}); err != nil {
		t.Fatalf("seed goal: %v", err)
	}

	plan, res, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("runonce: %v", err)
	}
	if plan == nil || res == nil {
		t.Fatal("expected non-nil plan and result")
	}
	if res.UpsertedBacklog != 2 {
		t.Errorf("expected 2 backlog upserts, got %d", res.UpsertedBacklog)
	}
	if res.UpsertedGoal != 1 {
		t.Errorf("expected 1 goal upsert, got %d", res.UpsertedGoal)
	}
	if emb.callCount() != 3 {
		t.Errorf("expected 3 embed calls (2 backlog + 1 goal), got %d", emb.callCount())
	}
	if bs.upsertCalls != 2 {
		t.Errorf("expected 2 backlog upsert calls, got %d", bs.upsertCalls)
	}
}

func TestIntegration_Reconciler_CleansGhostsLeftByOutOfBandFileDelete(t *testing.T) {
	r, bStore, _, bs, _, emb := buildTestReconciler(t)

	// First pass: seed disk + populate index.
	saveBacklogItem(t, bStore, backlog.BacklogItem{Name: "alpha", Kind: backlog.KindFix})
	saveBacklogItem(t, bStore, backlog.BacklogItem{Name: "beta", Kind: backlog.KindFix})
	if _, _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatalf("runonce 1: %v", err)
	}

	// Simulate out-of-band file deletion (operator rm -rf): index now has a
	// point whose backing item is gone. This is the exact scenario that
	// caused the production CPU spike loop.
	if err := os.RemoveAll(bStore.ItemDir(backlog.KindFix, "alpha")); err != nil {
		t.Fatalf("RemoveAll: %v", err)
	}

	embedsBefore := emb.callCount()
	_, res, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("runonce 2: %v", err)
	}
	// Ghost cleanup: 1 BatchDelete call carrying alpha's point ID; no embeds
	// (beta's hash matches), no upserts.
	if res.DeletedBacklog != 1 {
		t.Errorf("expected 1 ghost deleted, got %d", res.DeletedBacklog)
	}
	if bs.batchDeleteCalls != 1 {
		t.Errorf("expected 1 BatchDelete call, got %d", bs.batchDeleteCalls)
	}
	if emb.callCount() != embedsBefore {
		t.Errorf("expected zero new embeds (beta unchanged), got %d", emb.callCount()-embedsBefore)
	}

	// Third pass: now fully converged.
	plan3, _, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("runonce 3: %v", err)
	}
	if plan3.HasWork() {
		t.Errorf("expected zero work after ghost cleanup, got %+v", plan3)
	}
}

func TestIntegration_Reconciler_NoOpAfterConvergence(t *testing.T) {
	r, bStore, _, bs, _, emb := buildTestReconciler(t)

	// Seed + initial reconcile.
	for _, name := range []string{"x", "y", "z"} {
		saveBacklogItem(t, bStore, backlog.BacklogItem{Name: name, Kind: backlog.KindIdea, Title: name})
	}
	if _, _, err := r.RunOnce(context.Background()); err != nil {
		t.Fatalf("runonce 1: %v", err)
	}
	embedsAfterFirst := emb.callCount()
	upsertsAfterFirst := bs.upsertCalls

	// Second reconcile must be a complete no-op: no embeds, no upserts, no
	// deletes. This is the test that pins the CPU-burn fix — convergent
	// hash compare means subsequent ticks do zero work when nothing changed.
	plan, _, err := r.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("runonce 2: %v", err)
	}
	if plan.HasWork() {
		t.Errorf("expected no work after convergence, got %+v", plan)
	}
	if emb.callCount() != embedsAfterFirst {
		t.Errorf("second pass embedded items it shouldn't have: was %d, now %d", embedsAfterFirst, emb.callCount())
	}
	if bs.upsertCalls != upsertsAfterFirst {
		t.Errorf("second pass upserted items it shouldn't have: was %d, now %d", upsertsAfterFirst, bs.upsertCalls)
	}
	if bs.batchDeleteCalls != 0 {
		t.Errorf("second pass should not delete anything, got %d BatchDelete calls", bs.batchDeleteCalls)
	}
}
