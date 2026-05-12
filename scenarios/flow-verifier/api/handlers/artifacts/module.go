// Package artifacts wires the HTTP surface for the codegen lifecycle:
// inspect, generate, and clear generated/ files for one flow, one
// scenario, or every scenario. Generation delegates to pipeline.Verify
// via the internal/artifacts service — this handler only resolves the
// flow's root (using the existing scenarios.Service lookup) and
// translates errors to envelopes.
package artifacts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"flow-verifier/internal/artifacts"
	"flow-verifier/internal/clock"
	"flow-verifier/internal/httpx"
	"flow-verifier/internal/module"
	"flow-verifier/internal/pipeline"
	"flow-verifier/internal/runs"
	"flow-verifier/internal/scenarios"

	"github.com/gorilla/mux"
)

// ScenariosService is the subset of scenarios.Service this module uses
// to resolve a flow id → scenario root. Mirrors the inline interface
// used by handlers/flows so the dependency arrow stays inward.
type ScenariosService interface {
	List() ([]scenarios.Summary, error)
	Detail(id string) (scenarios.Detail, error)
}

// Module returns the artifacts domain's HTTP contribution. The runs
// recorder is shared with the verifications handler so generate calls
// land in the same history table the UI reads from.
func Module(db *sql.DB, clk clock.Clock, scenariosSvc ScenariosService) module.Module {
	runsSvc := runs.NewService(runs.NewSQLiteRepository(db, clk))
	svc := artifacts.NewService(pipelineGenerator{recorder: &runsRecorder{svc: runsSvc}})
	return ModuleWithDeps(svc, runsSvc, scenariosSvc)
}

// ModuleWithDeps is the test-friendly variant. Tests substitute a stub
// generator on the artifacts service before calling this.
func ModuleWithDeps(svc *artifacts.Service, runsSvc *runs.Service, scenariosSvc ScenariosService) module.Module {
	return module.Module{
		Name: "artifacts",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/flows/{id}/artifacts", statusHandler(svc, scenariosSvc)).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/flows/{id}/artifacts:generate", generateHandler(svc, scenariosSvc)).Methods(http.MethodPost)
			r.HandleFunc("/api/v1/flows/{id}/artifacts", clearHandler(svc, scenariosSvc)).Methods(http.MethodDelete)
			r.HandleFunc("/api/v1/scenarios/{id}/artifacts:generate", scenarioGenerateHandler(svc, scenariosSvc)).Methods(http.MethodPost)
			r.HandleFunc("/api/v1/scenarios/{id}/artifacts", scenarioClearHandler(svc, scenariosSvc)).Methods(http.MethodDelete)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — artifacts state is on-disk truth, no tables.
func Schema() string { return "" }

// pipelineGenerator is the production Generator that drives
// pipeline.Verify in generate mode and records one runs row per flow.
type pipelineGenerator struct {
	recorder *runsRecorder
}

func (g pipelineGenerator) Generate(ctx context.Context, root, flowID string) error {
	_, err := pipeline.Verify(ctx, pipeline.VerifyOptions{
		Root:     root,
		FlowID:   flowID,
		Mode:     pipeline.ModeGenerate,
		Recorder: g.recorder,
	})
	return err
}

// runsRecorder mirrors the verifications handler's recorder so generate
// runs land in the same history table. We don't capture; only persist.
type runsRecorder struct {
	svc *runs.Service
}

func (r *runsRecorder) Record(ctx context.Context, entry pipeline.RunEntry) error {
	row := runs.Run{
		FlowID:           entry.FlowID,
		FlowPath:         entry.FlowPath,
		Root:             entry.Root,
		Mode:             runs.ModeRun,
		Status:           runs.Status(entry.Status),
		Output:           entry.Output,
		ErrorMessage:     entry.ErrorMessage,
		FailureReason:    entry.FailureReason,
		MissingArtifacts: entry.MissingArtifacts,
		StartedAt:        entry.StartedAt,
		FinishedAt:       entry.FinishedAt,
	}
	_, err := r.svc.Record(ctx, row)
	return err
}

func statusHandler(svc *artifacts.Service, scenariosSvc ScenariosService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		root, err := resolveFlowRoot(scenariosSvc, r.URL.Query().Get("root"), r.URL.Query().Get("scenario"), id)
		if err != nil {
			writeResolveError(w, err)
			return
		}
		report, err := svc.Status(root, id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}

func generateHandler(svc *artifacts.Service, scenariosSvc ScenariosService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		root, err := resolveFlowRoot(scenariosSvc, r.URL.Query().Get("root"), r.URL.Query().Get("scenario"), id)
		if err != nil {
			writeResolveError(w, err)
			return
		}
		report, err := svc.Generate(r.Context(), root, id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, report)
	}
}

func clearHandler(svc *artifacts.Service, scenariosSvc ScenariosService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		root, err := resolveFlowRoot(scenariosSvc, r.URL.Query().Get("root"), r.URL.Query().Get("scenario"), id)
		if err != nil {
			writeResolveError(w, err)
			return
		}
		result, err := svc.Clear(root, id)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}

func scenarioGenerateHandler(svc *artifacts.Service, scenariosSvc ScenariosService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		root, err := resolveScenarioRoot(scenariosSvc, id)
		if err != nil {
			writeResolveError(w, err)
			return
		}
		reports, err := svc.GenerateForScenario(r.Context(), root)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scenarioId": id, "flows": reports})
	}
}

func scenarioClearHandler(svc *artifacts.Service, scenariosSvc ScenariosService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		root, err := resolveScenarioRoot(scenariosSvc, id)
		if err != nil {
			writeResolveError(w, err)
			return
		}
		results, err := svc.ClearForScenario(root)
		if err != nil {
			writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"scenarioId": id, "flows": results})
	}
}

func resolveFlowRoot(svc ScenariosService, root, scenarioID, flowID string) (string, error) {
	if root != "" {
		return root, nil
	}
	if svc == nil {
		return "", errors.New("scenarios service not configured")
	}
	if scenarioID != "" {
		detail, err := svc.Detail(scenarioID)
		if err != nil {
			return "", err
		}
		return detail.Path, nil
	}
	all, err := svc.List()
	if err != nil {
		return "", err
	}
	for _, scenario := range all {
		if scenario.DiscoveryErr != "" || scenario.FlowCount == 0 {
			continue
		}
		detail, err := svc.Detail(scenario.ID)
		if err != nil {
			continue
		}
		for _, row := range detail.Flows {
			if row.FlowID == flowID {
				return scenario.Path, nil
			}
		}
	}
	return "", artifacts.ErrFlowNotFound
}

func resolveScenarioRoot(svc ScenariosService, scenarioID string) (string, error) {
	if svc == nil {
		return "", errors.New("scenarios service not configured")
	}
	detail, err := svc.Detail(scenarioID)
	if err != nil {
		return "", err
	}
	return detail.Path, nil
}

func writeResolveError(w http.ResponseWriter, err error) {
	if errors.Is(err, scenarios.ErrScenarioNotFound) || errors.Is(err, artifacts.ErrFlowNotFound) {
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
		return
	}
	httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
}

func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, artifacts.ErrFlowNotFound):
		httpx.WriteError(w, http.StatusNotFound, httpx.CodeNotFound, err.Error())
	case errors.Is(err, artifacts.ErrPathTraversal):
		httpx.WriteError(w, http.StatusConflict, httpx.CodeInvalidRequest, err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, httpx.CodeInternal, err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
