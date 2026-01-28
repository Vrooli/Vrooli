package settings

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/testutil"
)

func TestHandler_GetViaRouter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	handler := NewHandler(path)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeJSON[SettingsResponse](t, rec)
	if resp.Settings.Theme == "" {
		t.Fatalf("expected settings to be populated")
	}
}
