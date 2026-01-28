package ideas

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"swarm-manager/internal/testutil"
)

func TestHandler_RegisterRoutes(t *testing.T) {
	ideasDir := t.TempDir()
	handler := NewHandler(ideasDir)

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ideas", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	testutil.AssertStatusOK(t, rec)
	resp := testutil.DecodeJSON[struct {
		Ideas []Idea `json:"ideas"`
	}](t, rec)
	if len(resp.Ideas) != 0 {
		t.Fatalf("expected empty ideas list, got %d", len(resp.Ideas))
	}
}
