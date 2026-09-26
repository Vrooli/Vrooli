// Package board exposes the single browser-reachable marketing board read.
// The read executes the governed content-desk.board-read declared program and
// returns its envelope unchanged, so the board the UI shows and the board a
// CLI agent reads come from one implementation.
package board

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"content-desk/internal/httpx"
	"content-desk/internal/module"

	"content-desk/integrations/programruntime"

	"github.com/gorilla/mux"
)

// ProgramName is the scenario-declared program this browser edge executes.
const ProgramName = "content-desk.board-read"

// readTimeout sits above the declared program's 60s wall budget so the
// runtime, not this handler, decides when a run is over.
const readTimeout = 90 * time.Second

// Module wires the board read into the API server.
func Module(runner programruntime.Runner) module.Module {
	return module.Module{
		Name:      "board",
		Mount:     func(r *mux.Router) { r.HandleFunc("/api/v1/board", readHandler(runner)).Methods(http.MethodGet) },
		Endpoints: Endpoints,
	}
}

// Schema returns "" — the board is a projection over other owners' records and
// owns no tables.
func Schema() string { return "" }

func readHandler(runner programruntime.Runner) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if runner == nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeUnavailable, "board read runner is not configured")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), readTimeout)
		defer cancel()
		result, err := runner.RunDeclared(ctx, ProgramName, nil)
		if err != nil {
			httpx.WriteError(w, http.StatusServiceUnavailable, httpx.CodeUnavailable, "board read unavailable: "+err.Error())
			return
		}
		if !json.Valid(result.Envelope) {
			httpx.WriteError(w, http.StatusBadGateway, httpx.CodeInternal, "board read returned a non-JSON envelope")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(result.Envelope)
	}
}
