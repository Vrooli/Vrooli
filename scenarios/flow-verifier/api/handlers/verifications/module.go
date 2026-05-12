// Package verifications wires the HTTP surface for kicking off
// verifications. POST runs synchronously and returns the per-flow runs
// it persisted via the runs domain. GET /api/v1/verifications/{runId}
// resolves to the same runs row a Phase E recorder wrote.
package verifications

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"flow-verifier/internal/clock"
	"flow-verifier/internal/httpx"
	"flow-verifier/internal/module"
	"flow-verifier/internal/pipeline"
	"flow-verifier/internal/runs"

	"github.com/gorilla/mux"
)

// Module returns the verifications domain's HTTP contribution. db and
// clock plumb through to the runs domain — verifications writes one row
// per flow per POST via runs.Service.
func Module(db *sql.DB, clk clock.Clock) module.Module {
	svc := runs.NewService(runs.NewSQLiteRepository(db, clk))
	return ModuleWithService(svc)
}

// ModuleWithService is the test-friendly variant.
func ModuleWithService(svc *runs.Service) module.Module {
	rec := &runsRecorder{svc: svc}
	return module.Module{
		Name: "verifications",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/verifications", postHandler(rec, svc)).Methods(http.MethodPost)
			r.HandleFunc("/api/v1/verifications/{runId}", getHandler(svc)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — verifications dispatches runs but persistence lives
// in the runs domain.
func Schema() string { return "" }

// runsRecorder adapts a *runs.Service to pipeline.Recorder. It also
// captures the inserted runs so the POST handler can return them.
type runsRecorder struct {
	svc      *runs.Service
	captured []runs.Run
}

func (r *runsRecorder) Record(ctx context.Context, entry pipeline.RunEntry) error {
	row := runs.Run{
		FlowID:       entry.FlowID,
		FlowPath:     entry.FlowPath,
		Root:         entry.Root,
		Mode:         pipelineModeToRunsMode(entry.Mode),
		Status:       runs.Status(entry.Status),
		Output:       entry.Output,
		ErrorMessage: entry.ErrorMessage,
		StartedAt:    entry.StartedAt,
		FinishedAt:   entry.FinishedAt,
	}
	inserted, err := r.svc.Record(ctx, row)
	if err != nil {
		return err
	}
	r.captured = append(r.captured, inserted)
	return nil
}

func pipelineModeToRunsMode(m pipeline.Mode) runs.Mode {
	if m == pipeline.ModeGenerate {
		return runs.ModeRun
	}
	return runs.ModeCheck
}

type postBody struct {
	Root   string `json:"root"`
	FlowID string `json:"flowId"`
	Mode   string `json:"mode"`
}

type postResponse struct {
	Status string     `json:"status"`
	Error  string     `json:"error,omitempty"`
	Runs   []runs.Run `json:"runs"`
}

func postHandler(parent *runsRecorder, _ *runs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body postBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, err.Error())
			return
		}
		if body.Root == "" {
			body.Root = "."
		}
		mode := pipeline.ModeCheck
		switch strings.ToLower(body.Mode) {
		case "", "check":
			mode = pipeline.ModeCheck
		case "run", "generate":
			mode = pipeline.ModeGenerate
		default:
			httpx.WriteError(w, http.StatusBadRequest, httpx.CodeInvalidRequest, "mode must be check or run")
			return
		}
		// Fresh capture buffer per request so concurrent POSTs don't
		// leak rows into each other's responses. parent is shared only
		// for the *runs.Service handle.
		rec := &runsRecorder{svc: parent.svc}
		_, runErr := pipeline.Verify(r.Context(), pipeline.VerifyOptions{
			Root:     body.Root,
			FlowID:   body.FlowID,
			Mode:     mode,
			Recorder: rec,
		})
		resp := postResponse{Runs: rec.captured}
		if resp.Runs == nil {
			resp.Runs = []runs.Run{}
		}
		if runErr != nil {
			resp.Status = "failed"
			resp.Error = runErr.Error()
		} else {
			resp.Status = "passed"
		}
		writeJSON(w, http.StatusOK, resp)
	}
}

func getHandler(svc *runs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["runId"]
		row, err := svc.Get(r.Context(), id)
		if err != nil {
			var nf runs.ErrNotFound
			if errors.As(err, &nf) {
				httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, row)
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
