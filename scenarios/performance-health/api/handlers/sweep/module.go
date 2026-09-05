package sweep

import (
	"log"

	internalanalysis "performance-health/internal/analysis"
	"performance-health/internal/capture"
	"performance-health/internal/module"
	"performance-health/internal/readiness"

	"github.com/gorilla/mux"
	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
	sweepconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep/sweep_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted SweepService.
var ProtoFile = sweepv1.File_performance_health_v1_sweep_sweep_proto

// DefaultCommitBudgetMs mirrors the analysis hot-spot threshold so the sweep's
// own analyzer locates components identically to the analysis domain.
const DefaultCommitBudgetMs = 8.0

// Module mounts the SweepService over the same capture conductor, trace
// analyzer, and budgets gate the rest of the scenario uses. The trend store and
// budgets service are injected from the composition root (they own the
// flow-tagged samples + per-flow budget config). A nil sample writer disables
// persistence.
func Module(logger *log.Logger, repoRoot string, trendWriter SampleWriter, gate FlowGate) module.Module {
	captureSvc := capture.NewService(&capture.BASConnectClient{}, &capture.CLIBuildController{}).
		WithFlowResolver(&capture.FileFlowResolver{RepoRoot: repoRoot})
	analyzer := internalanalysis.NewService(internalanalysis.FileTraceLoader{
		Locator:  internalanalysis.SourceLocator{RepoRoot: repoRoot},
		BudgetMs: DefaultCommitBudgetMs,
	})
	tierer := readiness.NewService(readiness.NewCodeFactsClient(repoRoot))
	handler := NewHandler(captureSvc, analyzer, trendWriter, gate, tierer, logger)
	path, connectHandler := sweepconnect.NewSweepServiceHandler(handler)
	return module.Module{
		Name: "sweep",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the empty schema: sweep owns no database tables (it writes
// through the shared trend store).
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "sweep_run_sweep",
		Path:        sweepconnect.SweepServiceRunSweepProcedure,
		Method:      "POST",
		Summary:     "Run the per-flow capture-sweep for a scenario",
		Description: "For each interaction flow with a per-flow budget declared, drives a targeted audit, analyzes the trace, persists a flow-tagged trend sample, and reports the per-flow budget verdict. The browser CAPTURE runs out-of-band on this cadence (never inside a gated run); the per-flow budget CHECK then runs in the test-genie Performance phase, reading the latest persisted sample, so a breach fails the suite run.",
		Category:    "sweep",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "results": "FlowSweepResult[]"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}, {Status: 500, Code: "internal", Description: "Sweep orchestration failure"}},
	},
}
