// Package artifacts wires the Connect-RPC ArtifactsService for the
// codegen lifecycle: inspect, generate, and clear generated/ files for
// one flow. Scenario-level generate/clear live on ScenariosService
// (handlers/scenarios) since GenerateScenarioArtifacts is server-
// streaming and the streaming surface is owned by ScenariosService.
//
// Generation delegates to pipeline.Verify via internal/artifacts; the
// runs recorder is shared with the verifications handler so generate
// calls land in the same history table the UI reads from.
package artifacts

import (
	"context"
	"database/sql"
	"log"

	"flow-verifier/internal/artifacts"
	"flow-verifier/internal/module"
	"flow-verifier/internal/pipeline"
	"flow-verifier/internal/runs"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	artifactsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/flow-verifier/v1/artifacts/artifacts_v1connect"
)

// Module returns the artifacts domain's Connect-RPC contribution.
func Module(db *sql.DB, clk schedule.Clock, scenariosSvc ScenariosService) module.Module {
	runsSvc := runs.NewService(runs.NewSQLiteRepository(db, clk))
	svc := NewService(runsSvc)
	return ModuleWithDeps(svc, scenariosSvc, log.Default())
}

// NewService constructs the production artifacts service used by Module.
// Exposed so main.go can hand the same service into handlers/scenarios
// (which serves the streaming GenerateScenarioArtifacts RPC).
func NewService(runsSvc *runs.Service) *artifacts.Service {
	return artifacts.NewService(pipelineGenerator{recorder: &runsRecorder{svc: runsSvc}})
}

// ModuleWithDeps is the test-friendly variant. Tests substitute a stub
// generator on the artifacts service before calling this.
func ModuleWithDeps(svc *artifacts.Service, scenariosSvc ScenariosService, logger *log.Logger) module.Module {
	path, handler := artifactsconnect.NewArtifactsServiceHandler(NewConnectHandler(Deps{
		Service:   svc,
		Scenarios: scenariosSvc,
		Logger:    logger,
	}))
	return module.Module{
		Name: "artifacts",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
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
// runs land in the same history table.
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
