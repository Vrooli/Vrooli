package aisearch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/initiatives"
	"sync/atomic"
	"testing"
	"time"
)

// waitFor polls predicate every 10ms until it returns true or timeout expires.
// Used because index notifications are fire-and-forget goroutines.
func waitFor(t *testing.T, timeout time.Duration, predicate func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if predicate() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return predicate()
}

// saveBacklogItem mkdirs the item directory before calling SaveItem, matching
// what production write-handlers do (see backlog/handler_create.go,
// batch_handler.go, import.go — all MkdirAll before SaveItem).
func saveBacklogItem(t *testing.T, store *backlog.FileStore, item backlog.BacklogItem) {
	t.Helper()
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
func buildTestService(t *testing.T, ollamaURL, qdrantURL string) (*Service, *backlog.FileStore, *initiatives.Store, string) {
	t.Helper()
	root := t.TempDir()
	// Pre-create the kind directories so LoadAll doesn't error.
	for _, d := range []string{"ideas", "research", "fix", "execute", "chore", "initiatives"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}

	bStore := backlog.NewFileStore(root)
	iStore := initiatives.NewStore(root)

	embedder := NewEmbedder(ollamaURL, "nomic-embed-text")
	backlogVS := NewVectorStore(qdrantURL, "", "sm-b", 3)
	initVS := NewVectorStore(qdrantURL, "", "sm-i", 3)
	svc := NewService(
		embedder, backlogVS, initVS,
		NewBacklogStoreAdapter(bStore),
		NewInitiativeStoreAdapter(iStore),
		0.5,
	)
	// Wire the write-through seam.
	bStore.SetAIIndexer(svc)
	iStore.SetAIIndexer(svc)

	return svc, bStore, iStore, root
}

func TestIntegration_BacklogSave_FiresIndexUpsert(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	_, bStore, _, _ := buildTestService(t, ollama.URL, qServer.URL)

	saveBacklogItem(t, bStore, backlog.BacklogItem{
		Name:  "alpha",
		Title: "Alpha",
		Kind:  backlog.KindExecute,
	})
	if !waitFor(t, 2*time.Second, func() bool { return atomic.LoadInt32(&qStub.upsertCalls) >= 1 }) {
		t.Errorf("expected at least one upsert call, got %d", qStub.upsertCalls)
	}
}

func TestIntegration_BacklogDelete_FiresIndexDelete(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	_, bStore, _, _ := buildTestService(t, ollama.URL, qServer.URL)

	saveBacklogItem(t, bStore, backlog.BacklogItem{
		Name:  "alpha",
		Title: "Alpha",
		Kind:  backlog.KindExecute,
	})
	// Wait for the initial upsert to land so we don't race it.
	waitFor(t, 2*time.Second, func() bool { return atomic.LoadInt32(&qStub.upsertCalls) >= 1 })

	if err := bStore.DeleteItem(backlog.KindExecute, "alpha"); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if !waitFor(t, 2*time.Second, func() bool { return atomic.LoadInt32(&qStub.deleteCalls) >= 1 }) {
		t.Errorf("expected delete call, got %d", qStub.deleteCalls)
	}
}

func TestIntegration_InitiativeSave_FiresIndexUpsert(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	_, _, iStore, _ := buildTestService(t, ollama.URL, qServer.URL)

	init := &initiatives.Initiative{Name: "obs-core", Title: "Observability Core", Status: "active"}
	if err := iStore.Save(init); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !waitFor(t, 2*time.Second, func() bool { return atomic.LoadInt32(&qStub.upsertCalls) >= 1 }) {
		t.Errorf("expected initiative upsert, got %d", qStub.upsertCalls)
	}
}

func TestIntegration_QdrantFailure_DoesNotBreakCRUD(t *testing.T) {
	// Core fire-and-forget invariant: CRUD must succeed even when Qdrant
	// returns 500 on every call.
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer qServer.Close()

	_, bStore, iStore, _ := buildTestService(t, ollama.URL, qServer.URL)

	// CRUD calls must not propagate the upstream 500.
	for i := 0; i < 5; i++ {
		saveBacklogItem(t, bStore, backlog.BacklogItem{Name: "bad", Title: "x", Kind: backlog.KindIdea})
	}
	if err := bStore.DeleteItem(backlog.KindIdea, "bad"); err != nil {
		t.Fatalf("DeleteItem: %v", err)
	}
	if err := iStore.Save(&initiatives.Initiative{Name: "x", Title: "X", Status: "active"}); err != nil {
		t.Fatalf("initiative Save: %v", err)
	}
	// Give the goroutines time to fail in the background.
	time.Sleep(100 * time.Millisecond)
}

func TestIntegration_OllamaEmpty_CRUDStillSucceeds(t *testing.T) {
	// If OLLAMA_URL is empty, the indexer reports errors internally but CRUD
	// must still succeed — operators can bring up the stack incrementally.
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	_, bStore, _, _ := buildTestService(t, "", qServer.URL)

	saveBacklogItem(t, bStore, backlog.BacklogItem{Name: "alpha", Title: "A", Kind: backlog.KindIdea})
	// No upsert should succeed because the embedder can't embed; but CRUD returned nil.
	time.Sleep(100 * time.Millisecond)
	if atomic.LoadInt32(&qStub.upsertCalls) != 0 {
		t.Errorf("expected 0 upserts (ollama disabled), got %d", qStub.upsertCalls)
	}
}

func TestIntegration_Status_ReflectsOnDiskCounts(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{count: 0}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc, bStore, iStore, _ := buildTestService(t, ollama.URL, qServer.URL)

	for _, name := range []string{"a", "b", "c"} {
		saveBacklogItem(t, bStore, backlog.BacklogItem{Name: name, Kind: backlog.KindIdea, Title: name})
	}
	if err := iStore.Save(&initiatives.Initiative{Name: "i1", Title: "I1", Status: "active"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	st := svc.GetStatus(context.Background())
	if st.OnDiskBacklog != 3 {
		t.Errorf("expected 3 backlog items on disk, got %d", st.OnDiskBacklog)
	}
	if st.OnDiskInitiatives != 1 {
		t.Errorf("expected 1 initiative on disk, got %d", st.OnDiskInitiatives)
	}
}

func TestIntegration_ReindexAll_PopulatesBothCollections(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	svc, bStore, iStore, _ := buildTestService(t, ollama.URL, qServer.URL)

	// Seed disk *without* triggering the indexer — we want ReindexAll to be
	// the thing that populates Qdrant. Detach indexer, seed, reattach.
	bStore.SetAIIndexer(nil)
	iStore.SetAIIndexer(nil)
	for _, name := range []string{"a", "b"} {
		saveBacklogItem(t, bStore, backlog.BacklogItem{Name: name, Kind: backlog.KindIdea, Title: name})
	}
	if err := iStore.Save(&initiatives.Initiative{Name: "i1", Title: "I1", Status: "active"}); err != nil {
		t.Fatalf("seed initiative: %v", err)
	}
	bStore.SetAIIndexer(svc)
	iStore.SetAIIndexer(svc)

	resp, err := svc.ReindexAll(context.Background())
	if err != nil {
		t.Fatalf("ReindexAll: %v", err)
	}
	if resp.Indexed != 3 {
		t.Errorf("expected 3 indexed (2 backlog + 1 initiative), got %d", resp.Indexed)
	}
	if got := atomic.LoadInt32(&qStub.upsertCalls); got != 3 {
		t.Errorf("expected 3 upsert calls to qdrant, got %d", got)
	}
}
