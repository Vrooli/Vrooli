package captures

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestRegisterRoutesDoesNotMountRetiredClassificationAliases(t *testing.T) {
	h, _ := setupTestHandler(t)
	router := mux.NewRouter()
	h.RegisterRoutes(router)

	for _, path := range []string{"/api/v1/captures/capture-1/classify", "/api/v1/captures/capture-1/classify/execution-1/apply"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusNotFound {
			t.Fatalf("POST %s status = %d, want %d", path, response.Code, http.StatusNotFound)
		}
	}
}
