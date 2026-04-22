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

func TestDelete_LoadError(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "queue.json")
	if err := os.MkdirAll(badPath, 0o755); err != nil {
		t.Fatalf("create dir: %v", err)
	}

	handler := NewHandler(badPath)
	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/queue/item-1", bytes.NewBufferString(""))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}
