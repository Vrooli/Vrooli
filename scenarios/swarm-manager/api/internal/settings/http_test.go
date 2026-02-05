package settings

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/mux"

	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
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
	resp := testutil.DecodeProtoJSON(t, rec, &apipb.SettingsResponse{})
	if resp.GetSettings().GetTheme() == "" {
		t.Fatalf("expected settings to be populated")
	}
}
