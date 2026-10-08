package queue

import (
	"bytes"
	"github.com/vrooli/api-core/apihttptest"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/repo-contract-go/repocontracttest"
)

type listResponse struct {
	Items []Item `json:"items"`
}

type itemResponse struct {
	Item Item `json:"item"`
}

func setupQueueHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	dir := t.TempDir()
	queuePath := filepath.Join(dir, "queue.json")
	return NewHandler(queuePath), queuePath
}

func TestList_Empty(t *testing.T) {
	handler, _ := setupQueueHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queue", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	apihttptest.AssertStatus(t, rec.Result(), 200)
	resp := apihttptest.MustDecodeJSON[listResponse](t, rec.Body.Bytes())
	if len(resp.Items) != 0 {
		t.Errorf("expected empty queue, got %d items", len(resp.Items))
	}
}

func TestCreate_AddsItem(t *testing.T) {
	handler, queuePath := setupQueueHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	body := []byte(`{"kind":"idea","payload":{"name":"alpha"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	apihttptest.AssertStatus(t, rec.Result(), 201)
	resp := apihttptest.MustDecodeJSON[itemResponse](t, rec.Body.Bytes())

	if resp.Item.ID == "" {
		t.Errorf("expected id to be set")
	}
	if resp.Item.Kind != "idea" {
		t.Errorf("expected kind 'idea', got %q", resp.Item.Kind)
	}
	if resp.Item.Created == "" {
		t.Errorf("expected created timestamp to be set")
	}

	items := repocontracttest.ReadJSONFileInto[[]Item](t, queuePath)
	if len(items) != 1 {
		t.Fatalf("expected 1 item persisted, got %d", len(items))
	}
}

func TestDelete_Idempotent(t *testing.T) {
	handler, _ := setupQueueHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	// Create item
	body := []byte(`{"kind":"idea"}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/queue", bytes.NewBuffer(body))
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	apihttptest.AssertStatus(t, createRec.Result(), 201)
	created := apihttptest.MustDecodeJSON[itemResponse](t, createRec.Body.Bytes())

	// Delete once
	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/queue/"+created.Item.ID, nil)
	delRec := httptest.NewRecorder()
	router.ServeHTTP(delRec, delReq)
	apihttptest.AssertStatus(t, delRec.Result(), http.StatusNoContent)

	// Delete again (idempotent)
	delReq2 := httptest.NewRequest(http.MethodDelete, "/api/v1/queue/"+created.Item.ID, nil)
	delRec2 := httptest.NewRecorder()
	router.ServeHTTP(delRec2, delReq2)
	apihttptest.AssertStatus(t, delRec2.Result(), http.StatusNoContent)
}

func TestCreate_Invalid(t *testing.T) {
	handler, _ := setupQueueHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue", bytes.NewBuffer([]byte(`{}`)))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	apihttptest.AssertStatus(t, rec.Result(), 400)
}
