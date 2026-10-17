package measures

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"ai-gateway/internal/module"

	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/measures/measures_v1connect"
)

// Module returns the measures domain's contribution to the API: the typed
// Connect-RPC MeasuresService plus the packages/measures-go serve registry
// mounted at /measures (GET /measures/declarations + POST /measures/execute).
// Both are backed by the same compute path over route_events, so a measure and
// its RPC can never report different numbers.
//
// The route_events table is owned and migrated by the routing domain, so this
// module contributes no schema of its own.
//
// now anchors relative time-window resolution (nil → time.Now).
func Module(metrics Metrics, now func() time.Time) (module.Module, error) {
	serveHandler, err := MeasuresHandler(metrics, now)
	if err != nil {
		return module.Module{}, err
	}
	return module.Module{
		Name: "measures",
		Mount: func(r *mux.Router) {
			RegisterRoutes(r, metrics, now)
			// The framework-agnostic serve registry, mounted under /measures. This
			// is the contract the behavioral probe + search-hub central index call.
			r.PathPrefix("/measures/").Handler(http.StripPrefix("/measures", serveHandler))
		},
		Endpoints: Endpoints,
	}, nil
}

// Endpoints is the machine-readable description of the measures module's public
// Connect surface. Each RPC path references the generated *Procedure constant so
// renaming an RPC breaks this at compile time; the global parity test asserts
// every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	measureEndpoint("measures_count_route_events", measuresconnect.MeasuresServiceCountRouteEventsProcedure,
		"Count AI routes executed in a window", "count", "int64 (scalar value_field)"),
	measureEndpoint("measures_route_success_rate", measuresconnect.MeasuresServiceRouteSuccessRateProcedure,
		"Fraction of AI routes that succeeded in a window", "rate", "double in [0,1] (scalar value_field)"),
	measureEndpoint("measures_route_fallback_rate", measuresconnect.MeasuresServiceRouteFallbackRateProcedure,
		"Fraction of successful routes that used a fallback in a window", "rate", "double in [0,1] (scalar value_field)"),
	measureEndpoint("measures_route_failure_rate", measuresconnect.MeasuresServiceRouteFailureRateProcedure,
		"Fraction of AI routes that failed in a window", "rate", "double in [0,1] (scalar value_field)"),
	measureEndpoint("measures_count_breaker_open_routes", measuresconnect.MeasuresServiceCountBreakerOpenRoutesProcedure,
		"Count routes blocked by an open provider breaker in a window", "count", "int64 (scalar value_field)"),
	measureEndpoint("measures_count_capacity_rejections", measuresconnect.MeasuresServiceCountCapacityRejectionsProcedure,
		"Count local routes rejected for insufficient capacity in a window", "count", "int64 (scalar value_field)"),
	measureEndpoint("measures_route_latency_p95", measuresconnect.MeasuresServiceRouteLatencyP95Procedure,
		"p95 AI route latency in milliseconds over a window", "latency_ms", "int64 (scalar value_field)"),
}

func measureEndpoint(id, path, summary, valueField, valueDesc string) module.EndpointDescriptor {
	return module.EndpointDescriptor{
		ID:          id,
		Path:        path,
		Method:      "POST",
		Summary:     summary,
		Description: summary + ": a read-only, run-eligible, full-tier measure over route_events, parameterized by the canonical time_window. Backed by the same compute path as the /measures serve registry.",
		Category:    "measures",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"window": "vrooli.measures.v1.TimeWindow (optional; defaults to this_week)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{valueField: valueDesc},
		},
	}
}
