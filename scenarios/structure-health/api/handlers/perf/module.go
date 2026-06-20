package perf

import (
	"context"
	"log"

	"structure-health/internal/module"
	internalperf "structure-health/internal/perf"

	"github.com/gorilla/mux"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	perfv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/perf"
	perfconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/perf/perf_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted PerfService; the
// global parity test walks it against the Endpoints slice.
var ProtoFile = perfv1.File_structure_health_v1_perf_perf_proto

// SelfScenario is this scenario's own slug; benchmarking it would restart the
// process answering the request, so the service rejects it.
const SelfScenario = "structure-health"

// Module mounts the PerfService, backed by a perf service that restarts the
// target scenario behind the CLI runner seam and persists measurements to the
// perf trend store.
func Module(logger *log.Logger, db internalperf.Executor) module.Module {
	store := internalperf.NewStore(db)
	cli := vroolicli.New()
	environment, envErr := cli.HostCaptureEnvironment(context.Background())
	if envErr != nil {
		if logger != nil {
			logger.Printf("perf: host inventory unavailable, metrics environment limited to stdlib baseline: %v", envErr)
		}
		environment = nil
	}
	runner := &internalperf.CLIRunner{Status: cli, Env: environment}
	svc := internalperf.NewService(runner, store, SelfScenario)
	handler := NewHandler(svc, logger)
	path, connectHandler := perfconnect.NewPerfServiceHandler(handler)
	return module.Module{
		Name: "perf",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the perf trend store DDL.
func Schema() string { return internalperf.Schema() }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "perf_benchmark_startup",
		Path:        perfconnect.PerfServiceBenchmarkStartupProcedure,
		Method:      "POST",
		Summary:     "Benchmark a scenario's startup performance",
		Description: "Restarts the target scenario, records per-surface time-to-healthy plus a resource envelope (CPU/mem/GPU/host facts), persists a trend row, and returns the measurement. Decoupled from validation — never invoked by a test-genie phase.",
		Category:    "perf",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "timeout_seconds": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"measurement": "PerfMeasurement"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}, {Status: 500, Code: "internal", Description: "Restart/measurement failure"}},
	},
	{
		ID:          "perf_get_perf_trend",
		Path:        perfconnect.PerfServiceGetPerfTrendProcedure,
		Method:      "POST",
		Summary:     "Read a scenario's persisted startup-performance trend",
		Description: "Returns the persisted startup measurements for a scenario, newest first.",
		Category:    "perf",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "measurements": "array<PerfMeasurement>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}, {Status: 500, Code: "internal", Description: "Trend read failure"}},
	},
}
