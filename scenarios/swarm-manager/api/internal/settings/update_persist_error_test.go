package settings

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"swarm-manager/internal/testutil"
	"testing"
)

func TestHandler_UpdatePersistError(t *testing.T) {
	root := t.TempDir()
	readonlyDir := filepath.Join(root, "readonly")
	if err := os.MkdirAll(readonlyDir, 0o755); err != nil {
		t.Fatalf("create readonly dir: %v", err)
	}
	if err := os.Chmod(readonlyDir, 0o555); err != nil {
		t.Fatalf("chmod readonly dir: %v", err)
	}
	defer func() {
		_ = os.Chmod(readonlyDir, 0o755)
	}()

	path := filepath.Join(readonlyDir, "settings.json")
	handler := &Handler{store: NewStore(path)}

	payload := []byte(`{"theme":"light"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	testutil.AssertStatus(t, rec, http.StatusInternalServerError)
}
