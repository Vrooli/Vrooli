package measures

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/vrooli/vrooli/scenarios/template-manager/api/internal/module"

	tmmeasuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/template-manager/v1/measures/measures_v1connect"
)

func Module(store Store, now func() time.Time, logger *log.Logger) (module.Module, error) {
	if logger == nil {
		logger = log.Default()
	}
	serveHandler, err := MeasuresHandler(store, now)
	if err != nil {
		return module.Module{}, err
	}
	return module.Module{
		Name: "measures",
		Mount: func(r *mux.Router) {
			RegisterRoutes(r, store, now)
			r.PathPrefix("/measures/").Handler(http.StripPrefix("/measures", serveHandler))
		},
		Endpoints: Endpoints,
	}, nil
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "measures_open_debt_count",
		Path:        tmmeasuresconnect.MeasuresServiceOpenDebtCountProcedure,
		Method:      "POST",
		Summary:     "Count open template debt entries",
		Description: "Counts open inherited-template debt entries in a canonical time window. Backed by the same compute path as /measures.",
		Category:    "measures",
	},
	{
		ID:          "measures_deep_validate_green_streak",
		Path:        tmmeasuresconnect.MeasuresServiceDeepValidateGreenStreakProcedure,
		Method:      "POST",
		Summary:     "Count consecutive passing deep validations",
		Description: "Counts newest-first passing deep-validation runs until the first non-green run for a template.",
		Category:    "measures",
	},
	{
		ID:          "measures_fleet_standing_distribution",
		Path:        tmmeasuresconnect.MeasuresServiceFleetStandingDistributionProcedure,
		Method:      "POST",
		Summary:     "Return template standing distribution",
		Description: "Buckets governed templates by current, drift, version lag, and open-debt standing.",
		Category:    "measures",
	},
	{
		ID:          "measures_max_version_lag",
		Path:        tmmeasuresconnect.MeasuresServiceMaxVersionLagProcedure,
		Method:      "POST",
		Summary:     "Return maximum template version lag",
		Description: "Returns the maximum version lag count across template records. Backed by the same compute path as /measures.",
		Category:    "measures",
	},
	{
		ID:          "measures_declarations",
		Path:        "/measures/declarations",
		Method:      "GET",
		Summary:     "List declared measures",
		Description: "Measures-go registry declarations harvested by measures-health and search-hub.",
		Category:    "measures",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "measures-go serves a framework-neutral harvest endpoint consumed without a generated client.",
		},
	},
	{
		ID:          "measures_execute",
		Path:        "/measures/execute",
		Method:      "POST",
		Summary:     "Execute a declared measure",
		Description: "Measures-go registry execution endpoint used by measures-health behavioral probes and search-hub federation.",
		Category:    "measures",
		RESTException: &module.RESTException{
			Reason: module.RESTReasonOpsProbe,
			Note:   "measures-go serves a uniform JSON execution endpoint shared across scenarios.",
		},
	},
}
