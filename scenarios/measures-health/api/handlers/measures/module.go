package measures

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"measures-health/internal/module"
	"measures-health/internal/runhistory"

	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/measures/measures_v1connect"
)

// Module returns the measures domain's contribution to the API: the typed
// Connect-RPC MeasuresService plus the packages/measures-go serve registry
// mounted at /measures (GET /measures/declarations + POST /measures/execute).
// Both are backed by the same compute path over the validation_run history, so a
// measure and its RPC can never report different numbers.
//
// now anchors relative time-window resolution (nil → time.Now).
func Module(c Counter, now func() time.Time, logger *log.Logger) (module.Module, error) {
	if logger == nil {
		logger = log.Default()
	}
	serveHandler, err := MeasuresHandler(c, now)
	if err != nil {
		return module.Module{}, err
	}
	return module.Module{
		Name: "measures",
		Mount: func(r *mux.Router) {
			RegisterRoutes(r, c, now)
			// The framework-agnostic serve registry, mounted under /measures. This
			// is the contract the behavioral probe + search-hub central index call.
			r.PathPrefix("/measures/").Handler(http.StripPrefix("/measures", serveHandler))
		},
		Endpoints: Endpoints,
	}, nil
}

// Schema returns the validation_runs DDL — the measures domain owns the persisted
// validation_run entity (measures-health dogfooding the capability it enforces).
func Schema() string { return runhistory.Schema() }

// Endpoints is the machine-readable description of the measures module's public
// Connect surface. The two RPC paths reference the generated *Procedure constants
// so renaming an RPC breaks this at compile time; the global parity test asserts
// every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "measures_count_failed_validations",
		Path:        measuresconnect.MeasuresServiceCountFailedValidationsProcedure,
		Method:      "POST",
		Summary:     "Count scenarios that failed measures validation in a window",
		Description: "Answers \"how many scenarios failed measures validation this week\": a read-only, run-eligible, full-tier measure over the persisted validation_run history, parameterized by the canonical time_window. Backed by the same compute path as the /measures serve registry.",
		Category:    "measures",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"window": "vrooli.measures.v1.TimeWindow (optional; defaults to this_week)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"count": "int64 (scalar value_field)"},
		},
		Examples: []module.Example{
			{Name: "Failed this week", Curl: "curl http://localhost:${API_PORT}/vrooli.measures_health.v1.measures.MeasuresService/CountFailedValidations -H 'Content-Type: application/json' -d '{}'"},
		},
	},
	{
		ID:          "measures_count_validation_coverage",
		Path:        measuresconnect.MeasuresServiceCountValidationCoverageProcedure,
		Method:      "POST",
		Summary:     "Count scenarios that passed measures validation in a window",
		Description: "Answers \"how many scenarios passed measures validation this week\" — the fleet measure-coverage signal over time. Read-only, run-eligible, full-tier (canonical time_window). Backed by the same compute path as the /measures serve registry.",
		Category:    "measures",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"window": "vrooli.measures.v1.TimeWindow (optional; defaults to this_week)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"count": "int64 (scalar value_field)"},
		},
		Examples: []module.Example{
			{Name: "Coverage this week", Curl: "curl http://localhost:${API_PORT}/vrooli.measures_health.v1.measures.MeasuresService/CountValidationCoverage -H 'Content-Type: application/json' -d '{}'"},
		},
	},
}
