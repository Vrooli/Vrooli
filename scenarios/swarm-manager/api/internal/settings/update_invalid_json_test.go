package settings

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"swarm-manager/internal/testutil"
	"testing"
)

func TestHandler_UpdateInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	handler := &Handler{store: NewStore(path)}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString("{"))
	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	testutil.AssertStatusBadRequest(t, rec)
}
