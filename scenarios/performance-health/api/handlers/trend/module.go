package trend

import (
	"log"

	"performance-health/internal/module"
	internaltrend "performance-health/internal/trend"

	"github.com/gorilla/mux"
	trendv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/trend"
	trendconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/trend/trend_v1connect"
)

// ProtoFile is the FileDescriptor backing the Connect-mounted TrendService.
var ProtoFile = trendv1.File_performance_health_v1_trend_trend_proto

// Module mounts the TrendService backed by the SQLite trend store.
func Module(logger *log.Logger, db internaltrend.Executor) module.Module {
	svc := internaltrend.NewService(internaltrend.NewStore(db))
	handler := NewHandler(svc, logger)
	path, connectHandler := trendconnect.NewTrendServiceHandler(handler)
	return module.Module{
		Name: "trend",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the trend store DDL.
func Schema() string { return internaltrend.Schema() }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "trend_get_trend",
		Path:        trendconnect.TrendServiceGetTrendProcedure,
		Method:      "POST",
		Summary:     "Read a scenario's persisted performance trend",
		Description: "Returns the persisted performance samples (build time, startup, LCP, bundle size) for a scenario, newest first.",
		Category:    "trend",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "limit": "int32"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "samples": "array<TrendSample>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario is missing"}, {Status: 500, Code: "internal", Description: "Trend read failure"}},
	},
}
