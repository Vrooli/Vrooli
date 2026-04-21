package aisearch

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/cli-core/cliutil"
)

// newTestHandler wires a Handler around an aisearch Service configured against
// supplied mock servers. If ollamaURL or qdrantURL is empty, that subsystem is
// disabled and the Handler exercises its graceful-degradation paths.
func newTestHandler(t *testing.T, ollamaURL, qdrantURL string) (*Handler, *mux.Router) {
	t.Helper()
	embedder := NewEmbedder(ollamaURL, "nomic-embed-text")
	backlogVS := NewVectorStore(qdrantURL, "", "sm-b", 3)
	initVS := NewVectorStore(qdrantURL, "", "sm-i", 3)
	svc := NewService(embedder, backlogVS, initVS, nil, nil, 0.5)
	h := NewHandler(svc)
	r := mux.NewRouter()
	h.RegisterRoutes(r)
	return h, r
}

func TestHandler_Search_InvalidBody(t *testing.T) {
	_, r := newTestHandler(t, "", "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandler_Search_EmptyQuery(t *testing.T) {
	_, r := newTestHandler(t, "", "")
	body, _ := json.Marshal(AISearchRequest{Query: ""})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty query, got %d", w.Code)
	}
}

func TestHandler_Search_FallbackUnavailable(t *testing.T) {
	// No ollama, no qdrant, no text searcher → response returns 200 with
	// fallback=unavailable and empty results.
	_, r := newTestHandler(t, "", "")
	body, _ := json.Marshal(AISearchRequest{Query: "x"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai", bytes.NewReader(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp AISearchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Fallback != FallbackUnavailable {
		t.Errorf("expected fallback=unavailable, got %s", resp.Fallback)
	}
}

func TestHandler_Status_Unavailable(t *testing.T) {
	_, r := newTestHandler(t, "", "")
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
	if st.Available {
		t.Error("expected Available=false for empty URLs")
	}
	if !strings.Contains(st.Message, "Ollama") || !strings.Contains(st.Message, "Qdrant") {
		t.Errorf("expected message to mention both subsystems, got %q", st.Message)
	}
}

func TestHandler_Reindex_DryRun(t *testing.T) {
	_, r := newTestHandler(t, "", "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reindex", nil)
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
		t.Errorf("expected dry_run=true in response, got %v", payload["dry_run"])
	}
}

func TestHandler_Reindex_Live(t *testing.T) {
	ollama := fakeOllamaServer(t)
	defer ollama.Close()
	qStub := &qdrantStub{}
	qServer := httptest.NewServer(qStub.handler(t))
	defer qServer.Close()

	h, r := newTestHandler(t, ollama.URL, qServer.URL)
	_ = h

	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reindex", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Errorf("expected 202 on start, got %d body=%s", w.Code, w.Body.String())
	}

	// Second concurrent start → 409.
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reindex", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	// It might already have finished and return 202 again; accept either.
	if w2.Code != http.StatusConflict && w2.Code != http.StatusAccepted {
		t.Errorf("expected 409 or 202 on second start, got %d", w2.Code)
	}
}

func TestHandler_ReindexStatus_ReadsService(t *testing.T) {
	_, r := newTestHandler(t, "", "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search/ai/reindex/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var st ReindexStatus
	if err := json.Unmarshal(w.Body.Bytes(), &st); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if st.Running {
		t.Error("expected Running=false on fresh service")
	}
}

func TestHandler_CancelReindex(t *testing.T) {
	_, r := newTestHandler(t, "", "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/search/ai/reindex/cancel", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
