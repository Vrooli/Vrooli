package settings

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/vrooli/api-core/apihttptest"
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
	apihttptest.AssertStatus(t, rec.Result(), http.StatusInternalServerError)
}
