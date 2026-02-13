package settings

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"
)

func TestHandler_UpdateRejectsDeprecatedRecommendationFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	handler := &Handler{store: NewStore(path)}

	payload := []byte(`{"recommendationMode":"off"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBuffer(payload))
	rec := httptest.NewRecorder()
	handler.Update(rec, req)

	testutil.AssertStatusBadRequest(t, rec)
}
