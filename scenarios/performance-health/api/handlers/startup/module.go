package startup

import (
	"context"
	"log"

	"performance-health/internal/module"
	internalstartup "performance-health/internal/startup"
	"performance-health/internal/trend"

	"github.com/gorilla/mux"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	startupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/startup"
	startupconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/startup/startup_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted StartupService.
var ProtoFile = startupv1.File_performance_health_v1_startup_startup_proto

// SelfScenario is this scenario's own slug; benchmarking it would restart the
// process answering the request, so the service rejects it.
const SelfScenario = "performance-health"

// Module mounts the StartupService backed by the startup trend store and the
// real CLIRunner, which restarts the target scenario behind the vrooli-cli-go
// status seam and persists per-surface time-to-healthy plus a resource envelope.
// This is the migrated home of structure-health's former perf domain (axis ②).
func Module(logger *log.Logger, db internalstartup.Executor) module.Module {
	store := internalstartup.NewStore(db)
	cli := vroolicli.New()
	environment, envErr := cli.HostCaptureEnvironment(context.Background())
	if envErr != nil {
		if logger != nil {
			logger.Printf("startup: host inventory unavailable, metrics environment limited to stdlib baseline: %v", envErr)
		}
		environment = nil
	}
	runner := &internalstartup.CLIRunner{Status: cli, Env: environment}
	// Cross-write time-to-healthy into the shared perf_samples trend so the
	// startup budget axis has a producer (in addition to the rich
	// startup_measurements store).
	svc := internalstartup.NewService(runner, store, SelfScenario, internalstartup.WithPerfTrend(trend.NewStore(db)))
	handler := NewHandler(svc, logger)
	path, connectHandler := startupconnect.NewStartupServiceHandler(handler)
	return module.Module{
		Name: "startup",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the startup trend store DDL.
func Schema() string { return internalstartup.Schema() }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "startup_benchmark_startup",
		Path:        startupconnect.StartupServiceBenchmarkStartupProcedure,
		Method:      "POST",
		Summary:     "Benchmark a scenario's startup performance",
		Description: "Restarts the target scenario, records per-surface time-to-healthy plus a resource envelope, persists a trend row, and returns the measurement. Decoupled from validation — never invoked by a test-genie phase.",
		Category:    "startup",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "timeout_seconds": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"measurement": "StartupMeasurement"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}, {Status: 500, Code: "internal", Description: "Restart/measurement failure"}},
	},
	{
		ID:          "startup_get_startup_trend",
		Path:        startupconnect.StartupServiceGetStartupTrendProcedure,
		Method:      "POST",
		Summary:     "Read a scenario's persisted startup-performance trend",
		Description: "Returns the persisted startup measurements for a scenario, newest first.",
		Category:    "startup",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "measurements": "array<StartupMeasurement>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}, {Status: 500, Code: "internal", Description: "Trend read failure"}},
	},
}
