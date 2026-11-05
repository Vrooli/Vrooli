package benchmark

import (
	"log"

	internalbench "performance-health/internal/benchmark"
	"performance-health/internal/module"

	"github.com/gorilla/mux"
	benchmarkv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/benchmark"
	benchmarkconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/benchmark/benchmark_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted BenchmarkService.
var ProtoFile = benchmarkv1.File_performance_health_v1_benchmark_benchmark_proto

// Module mounts the BenchmarkService backed by the real build-timing runner:
// it times `go build ./...` (api/) and the UI package-manager build (ui/)
// against thresholds from .vrooli/testing.json, preserving the early-exit
// semantics of the migrated test-genie native perf phase. Measured runs persist
// a build-time sample through the injected trend writer (additive); the concrete
// store is wired from the composition root so this domain never imports the
// trend domain directly. A nil writer disables persistence.
func Module(logger *log.Logger, repoRoot string, trendWriter SampleWriter) module.Module {
	svc := internalbench.NewService(&internalbench.CLIRunner{RepoRoot: repoRoot})
	handler := NewHandler(svc, trendWriter, logger)
	path, connectHandler := benchmarkconnect.NewBenchmarkServiceHandler(handler)
	return module.Module{
		Name: "benchmark",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the empty schema: benchmark owns no database tables.
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "benchmark_run_benchmark",
		Path:        benchmarkconnect.BenchmarkServiceRunBenchmarkProcedure,
		Method:      "POST",
		Summary:     "Time a scenario's build surfaces",
		Description: "Times go build and the UI build against thresholds from .vrooli/testing.json, preserving the early-exit semantics of the migrated test-genie native perf phase.",
		Category:    "benchmark",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "outcome": "BenchmarkOutcome", "timings": "array<BuildTiming>", "reason": "string"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}, {Status: 500, Code: "internal", Description: "Benchmark failure"}},
	},
}
