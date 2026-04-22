package queue

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"

	"github.com/gorilla/mux"
)

func TestList_LoadError(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "queue.json")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	handler := NewHandler(badPath)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/queue", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}

func TestCreate_InvalidJSON(t *testing.T) {
	handler, _ := setupQueueHandler(t)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusBadRequest(t, rec)
}

func TestCreate_LoadError(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "queue.json")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	handler := NewHandler(badPath)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue", bytes.NewBufferString(`{"kind":"idea"}`))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}

func TestDelete_MissingID(t *testing.T) {
	handler, _ := setupQueueHandler(t)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/queue/", nil)
	rec := httptest.NewRecorder()
	handler.Delete(rec, req)

	testutil.AssertStatusBadRequest(t, rec)
}
