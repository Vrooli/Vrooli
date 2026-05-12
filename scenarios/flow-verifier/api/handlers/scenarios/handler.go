// Package scenarios mounts the scenarios HTTP surface: list every
// scenario discovered under the Vrooli root, and a per-scenario
// detail endpoint with embedded flows.
package scenarios

import (
	"encoding/json"
	"errors"
	"net/http"

	"flow-verifier/internal/httpx"
	"flow-verifier/internal/module"
	"flow-verifier/internal/scenarios"

	"github.com/gorilla/mux"
)

// Service is the in-process surface this module depends on. main.go
// constructs scenarios.Service and passes it in; tests pass a stub.
type Service interface {
	List() ([]scenarios.Summary, error)
	Detail(id string) (scenarios.Detail, error)
	Root() string
}

// Module returns the scenarios domain's HTTP contribution.
func Module(svc Service) module.Module {
	return module.Module{
		Name: "scenarios",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/scenarios", listHandler(svc)).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/scenarios/{id}", getHandler(svc)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — scenarios are filesystem-truth, no tables.
func Schema() string { return "" }

func listHandler(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		rows, err := svc.List()
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"vrooliRoot": svc.Root(),
			"scenarios":  rows,
		})
	}
}

func getHandler(svc Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		detail, err := svc.Detail(id)
		if err != nil {
			if errors.Is(err, scenarios.ErrScenarioNotFound) {
				httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, detail)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
