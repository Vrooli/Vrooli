package aisearch

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/testutil"

	"github.com/gorilla/mux"
	"github.com/vrooli/cli-core/cliutil"
)

// newTestHandler wires a Handler around an aisearch Service + Reconciler. The
// optional fakes let tests preload disk + qdrant state without httptest plumbing.
func newTestHandler(t *testing.T, opts handlerOpts) (*Handler, *Reconciler, *mux.Router) {
	t.Helper()
	emb := opts.embedder
	if emb == nil {
		emb = &fakeEmbedder{}
	}
	bs := opts.backlogStore
	if bs == nil {
		bs = &fakeVectorStore{}
	}
	gs := opts.goalStore
	if gs == nil {
		gs = &fakeVectorStore{}
	}
	br := opts.backlogReader
	if br == nil {
		br = &fakeBacklogReader{}
	}
	gr := opts.goalReader
	if gr == nil {
		gr = &fakeGoalReader{}
	}

	// Service uses the same fakes for search + status; Reconciler owns the
	// reconcile lifecycle. Threshold of 0 lets the default kick in.
	svc := NewService(emb, bs, gs, br, gr, 0)
	reconciler := NewReconciler(emb, bs, gs, br, gr, 1)
	h := NewHandler(svc, reconciler)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return h, reconciler, r
}

type handlerOpts struct {
	embedder      Embedder
	backlogStore  VectorStore
	goalStore     VectorStore
	backlogReader BacklogReader
	goalReader    GoalReader
}

func TestHandler_Search_InvalidBody(t *testing.T) {
	_, _, r := newTestHandler(t, handlerOpts{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandler_Search_EmptyQuery(t *testing.T) {
	_, _, r := newTestHandler(t, handlerOpts{})
	body, _ := json.Marshal(AISearchRequest{Query: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty query, got %d", w.Code)
	}
}

func TestHandler_Status_Reads(t *testing.T) {
	_, _, r := newTestHandler(t, handlerOpts{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/ai/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var st AvailabilityStatus
	if err := json.Unmarshal(w.Body.Bytes(), &st); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// fakeEmbedder.Available returns true; fakeVectorStore.Available returns true.
	if !st.Ollama || !st.Qdrant {
		t.Errorf("expected Ollama+Qdrant=true with fake stores, got %+v", st)
	}
}

func TestHandler_Reconcile_DryRun_ReturnsDriftReport(t *testing.T) {
	br := &fakeBacklogReader{items: []backlog.BacklogItem{
		{Kind: backlog.KindFix, Name: "a"},
		{Kind: backlog.KindFix, Name: "b"},
	}}
	emb := &fakeEmbedder{}
	bs := &fakeVectorStore{}
	_, _, r := newTestHandler(t, handlerOpts{embedder: emb, backlogStore: bs, backlogReader: br})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reconcile", nil)
	req.Header.Set(cliutil.DryRunHeader, "true")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for dry-run, got %d body=%s", w.Code, w.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload["dry_run"] != true {
		t.Errorf("expected dry_run=true, got %v", payload["dry_run"])
	}
	// Crucial: dry-run must NOT touch the embedder or upsert anything.
	if emb.callCount() != 0 {
		t.Errorf("dry-run should not embed, got %d calls", emb.callCount())
	}
	if bs.upsertCalls != 0 {
		t.Errorf("dry-run should not upsert, got %d calls", bs.upsertCalls)
	}
}

func TestHandler_Reconcile_Live_AcceptedAndAppliesPlan(t *testing.T) {
	br := &fakeBacklogReader{items: []backlog.BacklogItem{
		{Kind: backlog.KindFix, Name: "live"},
	}}
	emb := &fakeEmbedder{}
	bs := &fakeVectorStore{}
	_, reconciler, r := newTestHandler(t, handlerOpts{embedder: emb, backlogStore: bs, backlogReader: br})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reconcile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 on start, got %d body=%s", w.Code, w.Body.String())
	}

	// The goroutine completes asynchronously; wait for the upsert.
	testutil.Eventually(t, time.Second, "live reconcile applied", func() bool {
		bs.mu.Lock()
		defer bs.mu.Unlock()
		return bs.upsertCalls >= 1
	})
	// And the in-flight singleton clears once done.
	testutil.Eventually(t, time.Second, "reconciler idle after run", func() bool {
		return !reconciler.Status().Running
	})
}

func TestHandler_Reconcile_Conflict_WhenAlreadyRunning(t *testing.T) {
	emb := &fakeEmbedder{delay: 200 * time.Millisecond}
	br := &fakeBacklogReader{items: []backlog.BacklogItem{{Kind: backlog.KindFix, Name: "slow"}}}
	_, reconciler, r := newTestHandler(t, handlerOpts{embedder: emb, backlogReader: br})

	// Kick off a slow run.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reconcile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}
	// Wait for the singleton to actually be acquired before firing the second call.
	testutil.Eventually(t, time.Second, "first run acquires singleton", func() bool {
		return reconciler.Status().Running
	})

	// Second request must 409.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reconcile", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409 while running, got %d body=%s", w2.Code, w2.Body.String())
	}

	// Drain.
	testutil.Eventually(t, 2*time.Second, "first run drains", func() bool {
		return !reconciler.Status().Running
	})
}

func TestHandler_ReconcileStatus_ReturnsLastReport(t *testing.T) {
	br := &fakeBacklogReader{items: []backlog.BacklogItem{{Kind: backlog.KindFix, Name: "one"}}}
	_, reconciler, r := newTestHandler(t, handlerOpts{backlogReader: br})

	// Drive one run synchronously so LastPlan + LastResult populate.
	if _, _, err := reconciler.RunOnce(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/ai/reconcile/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "lastResult") {
		t.Errorf("expected lastResult in body, got %s", w.Body.String())
	}
}

func TestHandler_ReconcileCancel_StopsRunning(t *testing.T) {
	emb := &fakeEmbedder{delay: 500 * time.Millisecond}
	br := &fakeBacklogReader{items: []backlog.BacklogItem{
		{Kind: backlog.KindFix, Name: "a"}, {Kind: backlog.KindFix, Name: "b"}, {Kind: backlog.KindFix, Name: "c"},
	}}
	_, reconciler, r := newTestHandler(t, handlerOpts{embedder: emb, backlogReader: br})

	// Start run.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reconcile", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	testutil.Eventually(t, time.Second, "run starts", func() bool {
		return reconciler.Status().Running
	})

	// Cancel.
	cancelReq := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reconcile/cancel", nil)
	cancelW := httptest.NewRecorder()
	r.ServeHTTP(cancelW, cancelReq)
	if cancelW.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", cancelW.Code)
	}

	testutil.Eventually(t, 2*time.Second, "run stops after cancel", func() bool {
		st := reconciler.Status()
		return !st.Running && st.Canceled
	})
}
