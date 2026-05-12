// Package runs wires the HTTP surface for verification run history.
// Reads from the SQLite-backed runs domain (internal/runs); writes
// happen through the verifications handler, which records via the same
// runs.Service.
package runs

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"flow-verifier/internal/clock"
	"flow-verifier/internal/httpx"
	"flow-verifier/internal/module"
	"flow-verifier/internal/runs"

	"github.com/gorilla/mux"
)

// Module returns the runs domain's HTTP contribution. The caller is
// expected to have applied modules.AllSchemas to the same *sql.DB; this
// handler does not run migrations.
func Module(db *sql.DB, clk clock.Clock) module.Module {
	svc := runs.NewService(runs.NewSQLiteRepository(db, clk))
	return ModuleWithService(svc)
}

// ModuleWithService is the test-friendly variant that accepts an
// already-constructed *runs.Service.
func ModuleWithService(svc *runs.Service) module.Module {
	return module.Module{
		Name: "runs",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/runs", listHandler(svc)).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/runs/{id}", getHandler(svc)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports runs.Schema so the modules registry can collect
// both endpoint descriptors and schema from one symbol per handler.
func Schema() string { return runs.Schema() }

func listHandler(svc *runs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := runs.ListQuery{FlowID: r.URL.Query().Get("flowId")}
		if raw := r.URL.Query().Get("limit"); raw != "" {
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 {
				httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "limit must be a non-negative integer")
				return
			}
			q.Limit = n
		}
		rows, err := svc.List(r.Context(), q)
		if err != nil {
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
			return
		}
		if rows == nil {
			rows = []runs.Run{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"runs": rows})
	}
}

func getHandler(svc *runs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		run, err := svc.Get(r.Context(), id)
		if err != nil {
			var nf runs.ErrNotFound
			if errors.As(err, &nf) {
				httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, run)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
