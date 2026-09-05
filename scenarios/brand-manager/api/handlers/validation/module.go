package validation

import (
	"path/filepath"

	"brand-manager/internal/module"

	"github.com/vrooli/maturity-go/assessment"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"

	"github.com/gorilla/mux"
)

// Module returns the validation domain's contribution to the API: the served
// ScenarioValidationService Connect handler that test-genie's `branding`
// delegated phase calls. It owns no tables (the scan reads target scenarios'
// files), so it ships no endpoint descriptors for the REST codegen.
func Module() module.Module {
	handler := NewHandler()
	// DescribeProvider answers readiness from this provider's own descriptor, so a
	// readiness probe no longer costs a full target analysis. A load failure yields
	// the zero Describer, which reports Unimplemented and makes consumers fall back.
	describer, _ := assessment.LoadDescriber(filepath.Join(handler.repoRoot, "scenarios", "brand-manager"))
	path, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(handler, describer))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			r.PathPrefix(path).Handler(connectHandler)
		},
	}
}

// Schema returns "" — validation owns no tables; it scans target scenarios.
func Schema() string { return "" }
