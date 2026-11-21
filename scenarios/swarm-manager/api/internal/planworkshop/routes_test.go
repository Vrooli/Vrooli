package planworkshop

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestRegisterRoutesDoesNotMountRetiredTransitionApplyAliases(t *testing.T) {
	router := mux.NewRouter()
	NewHandler(NewService(NewStore(t.TempDir()), nil)).RegisterRoutes(router)

	for _, path := range []string{"/api/v1/plan-workshops/pw-1/review/apply", "/api/v1/plan-workshops/pw-1/responses/response-1/reconciliation/apply", "/api/v1/plan-workshops/pw-1/responses/response-1/candidate/apply", "/api/v1/plan-workshops/pw-1/responses/response-1/candidate/discard"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("POST %s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}
