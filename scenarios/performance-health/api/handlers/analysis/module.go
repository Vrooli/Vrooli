package analysis

import (
	"log"

	internalanalysis "performance-health/internal/analysis"
	"performance-health/internal/module"

	"github.com/gorilla/mux"
	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis"
	analysisconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/analysis/analysis_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted AnalysisService.
var ProtoFile = analysisv1.File_performance_health_v1_analysis_analysis_proto

// DefaultCommitBudgetMs is the average-commit-time budget (ms) above which a
// component is flagged as a hot spot. Per-budget overrides land in P8.
const DefaultCommitBudgetMs = 8.0

// Module mounts the AnalysisService over the real CDP-trace parser: it pairs ⚛
// blink.user_timing begin/end marks by id2.local into a per-component table,
// derives long-task / paint / LCP from the sibling web-vitals, and locates hot
// components (component → file:line) within the scenario's ui/src.
func Module(logger *log.Logger, repoRoot string) module.Module {
	loader := internalanalysis.FileTraceLoader{
		Locator:  internalanalysis.SourceLocator{RepoRoot: repoRoot},
		BudgetMs: DefaultCommitBudgetMs,
	}
	svc := internalanalysis.NewService(loader)
	handler := NewHandler(svc, logger)
	path, connectHandler := analysisconnect.NewAnalysisServiceHandler(handler)
	return module.Module{
		Name: "analysis",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the empty schema: analysis owns no database tables.
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "analysis_analyze_trace",
		Path:        analysisconnect.AnalysisServiceAnalyzeTraceProcedure,
		Method:      "POST",
		Summary:     "Analyze a captured perf trace",
		Description: "Parses a captured performance.json into a per-component count/avg/max table plus deterministic, located findings (component → file:line with quantified evidence).",
		Category:    "analysis",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "trace_artifact": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "components": "array<ComponentTiming>", "long_task_ms": "int64", "lcp_ms": "int64", "fcp_ms": "int64", "findings": "array<PerfFinding>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario or trace artifact is missing"}},
	},
	{
		ID:          "analysis_compare_traces",
		Path:        analysisconnect.AnalysisServiceCompareTracesProcedure,
		Method:      "POST",
		Summary:     "Compare two perf traces of the same interaction",
		Description: "Diffs two traces by component (commit-time delta) plus long-task and LCP deltas.",
		Category:    "analysis",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "baseline_artifact": "string", "candidate_artifact": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "components": "array<ComponentDelta>", "long_task_delta_ms": "int64", "lcp_delta_ms": "int64"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario or artifacts are missing"}},
	},
}
