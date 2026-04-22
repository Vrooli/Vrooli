package settings

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
)

func TestHandler_GetLoadError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid settings: %v", err)
	}

	handler := NewHandler(path)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := httptest.NewRecorder()
	handler.Get(rec, req)

	testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}
