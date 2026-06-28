package validation

import (
	"brand-manager/internal/module"

	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"github.com/gorilla/mux"
)

// Module returns the validation domain's contribution to the API: the served
// ScenarioValidationService Connect handler that test-genie's `branding`
// delegated phase calls. It owns no tables (the scan reads target scenarios'
// files), so it ships no endpoint descriptors for the REST codegen.
func Module() module.Module {
	handler := NewHandler()
	path, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(handler)
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
	}
}

// Schema returns "" — validation owns no tables; it scans target scenarios.
func Schema() string { return "" }
