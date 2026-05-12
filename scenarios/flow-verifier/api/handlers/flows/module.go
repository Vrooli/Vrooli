// Package flows wires the HTTP surface for flow discovery and inspection.
package flows

import (
	"encoding/json"
	"net/http"

	"flow-verifier/internal/flows"
	"flow-verifier/internal/httpx"
	"flow-verifier/internal/module"

	"github.com/gorilla/mux"
)

// Module returns the flows domain's HTTP contribution. List discovers
// every flow under ?root=<path>; detail returns the explain report for
// one flow id.
func Module() module.Module {
	return module.Module{
		Name: "flows",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/flows", listHandler).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/flows/{id}", getHandler).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — flow inventory is filesystem-truth, no tables.
func Schema() string { return "" }

func listHandler(w http.ResponseWriter, r *http.Request) {
	root := r.URL.Query().Get("root")
	if root == "" {
		root = "."
	}
	summaries, err := flows.List(root, "")
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"flows": summaries})
}

func getHandler(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	root := r.URL.Query().Get("root")
	if root == "" {
		root = "."
	}
	report, err := flows.Explain(root, id)
	if err != nil {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"flowId": id, "report": report})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
