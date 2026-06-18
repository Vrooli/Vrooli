package validation

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"measures-health/internal/module"
	internal "measures-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"

	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/measures-health/v1/validation/validation_v1connect"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

// Module returns the validation domain's contribution to the API: the generated
// Connect-RPC ValidationService handler. The Validator is constructed with the
// production filesystem seams rooted at repoRoot (manifests + proto domain
// folders + the committed descriptor image + a live behavioral prober). recorder
// (optional, may be nil) persists each ValidateScenario run to the
// validation_run history.
func Module(repoRoot string, recorder RunRecorder, logger *log.Logger) module.Module {
	v := internal.NewFilesystemValidator(repoRoot)
	spec, err := loadMaturitySpec(repoRoot)
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment disabled: %v", err)
	}
	handler := NewConnectHandler(Deps{
		Validator:    v,
		Recorder:     recorder,
		Logger:       logger,
		MaturitySpec: spec,
	})
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(handler)
	nativePath, nativeHandler := validationconnect.NewValidationServiceHandler(handler)
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r,
				connectx.ServiceMount{Path: sharedPath, Handler: sharedHandler},
				connectx.ServiceMount{Path: nativePath, Handler: nativeHandler},
			)
		},
		Endpoints: Endpoints,
	}
}

func loadMaturitySpec(repoRoot string) (*assessment.Spec, error) {
	path := filepath.Join(repoRoot, "scenarios", "measures-health", ".vrooli", "maturity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return assessment.ParseSpec(raw)
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
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's measure coverage",
		Description: "Grades a scenario's measure adoption through the shared scenario-validation response; the native ScenarioCoverageReport is packed into native_detail.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":          "string (required, scenario id under scenarios/)",
				"include_execution": "bool (run the behavioral adoption probe against live endpoints)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":      "string",
				"status":        "scenario_validation.v1.ValidationStatus",
				"assessment":    "common.v1.MaturityAssessment",
				"native_detail": "google.protobuf.Any<measures_health.v1.validation.ScenarioCoverageReport>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 500, Code: "internal", Description: "Manifest/proto read failure"},
		},
		Examples: []module.Example{
			{Name: "Validate swarm-manager", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"swarm-manager\"}'"},
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
	},
}
