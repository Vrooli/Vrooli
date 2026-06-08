package validation

import (
	"log"

	"measures-health/internal/module"
	internal "measures-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/validation/validation_v1connect"
)

// Module returns the validation domain's contribution to the API: the generated
// Connect-RPC ValidationService handler. The Validator is constructed with the
// production filesystem seams rooted at repoRoot (manifests + proto domain
// folders + the committed descriptor image + a live behavioral prober).
func Module(repoRoot string, logger *log.Logger) module.Module {
	v := internal.NewFilesystemValidator(repoRoot)
	connectPath, connectHandler := validationconnect.NewValidationServiceHandler(NewConnectHandler(Deps{
		Validator: v,
		Logger:    logger,
	}))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — validation computes coverage on demand from the
// filesystem; it owns no tables.
func Schema() string { return "" }

// Endpoints is the machine-readable description of the validation module's
// public surface. Connect-RPC method paths reference the generated *Procedure
// constants so renaming an RPC in validation.proto breaks this at compile time;
// the global parity test asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        validationconnect.ValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's measure coverage",
		Description: "Grades a scenario's measure adoption: derives its stateful domains, classifies each covered/waived/uncovered against the manifest measure blocks, grades per-measure extraction tier, and (when probe=true) behaviorally probes each declared measure end-to-end. Returns an expected/covered/waived report with normalized producer findings and a pass/fail verdict.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string (required, scenario id under scenarios/)",
				"probe":    "bool (run the behavioral adoption probe against live endpoints)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string",
				"passed":   "bool",
				"domains":  "array<DomainCoverage>",
				"findings": "array<Finding>",
				"summary":  "Summary",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Manifest/proto read failure"},
		},
		Examples: []module.Example{
			{Name: "Validate swarm-manager", Curl: "curl http://localhost:${API_PORT}/vrooli.measures_health.v1.validation.ValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"swarm-manager\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "measures-health validate scenario",
			Args:    []string{"<name>", "--probe"},
		},
	},
	{
		ID:          "validation_list_fleet_coverage",
		Path:        validationconnect.ValidationServiceListFleetCoverageProcedure,
		Method:      "POST",
		Summary:     "Roll up measure coverage across scenarios",
		Description: "Statically grades every discovered scenario (or the requested subset) and returns one coverage rollup each: expected/covered/waived/uncovered counts, worst tier, and the pass/fail verdict. Powers the fleet-view UI.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenarios": "array<string> (empty = every discovered scenario)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"entries": "array<FleetEntry>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Scenario enumeration failure"},
		},
		Examples: []module.Example{
			{Name: "Fleet coverage", Curl: "curl http://localhost:${API_PORT}/vrooli.measures_health.v1.validation.ValidationService/ListFleetCoverage -H 'Content-Type: application/json' -d '{}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "measures-health validate coverage",
		},
	},
}
