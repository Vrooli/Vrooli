package settings

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"swarm-manager/internal/testutil"

	"github.com/gorilla/mux"

	"github.com/vrooli/api-core/apihttptest"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
)

func TestHandler_GetViaRouter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	handler := NewHandler(path)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	apihttptest.AssertStatus(t, rec.Result(), 200)
	resp := testutil.DecodeProtoJSON(t, rec, &apipb.SettingsResponse{})
	if resp.GetSettings().GetTheme() == "" {
		t.Fatalf("expected settings to be populated")
	}
}
