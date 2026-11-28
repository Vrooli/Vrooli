package module

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestModuleMountsRoutesAndRetainsEndpointDescriptors(t *testing.T) {
	router := mux.NewRouter()
	mod := Module{
		Name: "health",
		Mount: func(router *mux.Router) {
			router.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
		},
		Endpoints: []EndpointDescriptor{{ID: "health", Path: "/health", Method: http.MethodGet}},
	}

	if mod.Name != "health" || len(mod.Endpoints) != 1 || mod.Endpoints[0].ID != "health" {
		t.Fatalf("module metadata = %+v", mod)
	}
	mod.Mount(router)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("mounted route status = %d, want %d", response.Code, http.StatusNoContent)
	}
}
