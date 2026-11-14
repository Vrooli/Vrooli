package validation

import (
	"context"
	"log"
	"path/filepath"

	"measures-health/internal/module"
	internal "measures-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"

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
	// DescribeProvider answers readiness from this provider's own descriptor,
	// so a readiness probe no longer costs a full target analysis. A load
	// failure yields the zero Describer, which reports Unimplemented and makes
	// consumers fall back to the legacy probe.
	describer, _ := assessment.LoadDescriber(filepath.Join(repoRoot, "scenarios", "measures-health"))
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "measures-health"))
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment disabled: %v", err)
	}
	// Capture host facts once; they do not change during the process lifetime.
	// A failure (CLI unavailable) is non-fatal — the metrics collector backfills
	// os/arch/num_cpu from the stdlib, leaving richer facts unset.
	environment, envErr := vroolicli.New().HostCaptureEnvironment(context.Background())
	if envErr != nil {
		if logger != nil {
			logger.Printf("validation: host inventory unavailable, metrics environment limited to stdlib baseline: %v", envErr)
		}
		environment = nil
	}
	handler := NewConnectHandler(Deps{
		Validator:    v,
		Recorder:     recorder,
		Logger:       logger,
		MaturitySpec: spec,
		Environment:  environment,
	})
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(assessment.Serve(handler, describer))
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

// Schema returns "" — validation computes coverage on demand from the
// filesystem; it owns no tables.
func Schema() string { return "" }

// Endpoints is the machine-readable description of the validation module's
// public surface. Connect-RPC method paths reference the generated *Procedure
// constants so renaming an RPC in validation.proto breaks this at compile time;
// the global parity test asserts every rpc has exactly one entry here.
var Endpoints = []module.EndpointDescriptor{
	{ID: "validation_validate_target", Path: scenariovalidationconnect.ScenarioValidationServiceValidateTargetProcedure, Method: "POST", Summary: "Validate a first-class repository target", Category: "validation"},
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
		ID:          "validation_describe_provider",
		Path:        scenariovalidationconnect.ScenarioValidationServiceDescribeProviderProcedure,
		Method:      "POST",
		Summary:     "Describe this provider's identity and contract",
		Description: "Reports provider identity, backed phase, maturity spec version, contract, build provenance, and capabilities. Inspects no target, so readiness consumers can confirm this provider is live and current without paying for a full validation run.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{
			"provider":     "string",
			"phase":        "string",
			"spec_version": "string",
			"contract":     "string",
			"build":        "scenario_validation.v1.ProviderBuild",
			"capabilities": "scenario_validation.v1.ProviderCapabilities",
		}},
		Examples: []module.Example{{Name: "Describe provider", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/DescribeProvider -H 'Content-Type: application/json' -d '{}'"}},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic measures fixes",
		Description: "Measures Health does not currently ship deterministic fixes; the shared RPC returns Unimplemented and is listed so module descriptors match the proto surface.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string (required)",
				"rule_ids": "array<string> (optional rule filter)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"candidates": "array<scenario_validation.v1.FixCandidate>",
				"applied":    "bool (always false for preview)",
				"messages":   "array<string>",
			},
		},
		Examples: []module.Example{
			{Name: "Preview fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/PreviewFix -H 'Content-Type: application/json' -d '{\"scenario\":\"measures-health\"}'"},
		},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic measures fixes",
		Description: "Measures Health does not currently ship deterministic fixes; the shared RPC returns Unimplemented and is listed so module descriptors match the proto surface.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string (required)",
				"rule_ids": "array<string> (optional rule filter)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"candidates": "array<scenario_validation.v1.FixCandidate>",
				"applied":    "bool",
				"messages":   "array<string>",
			},
		},
		Examples: []module.Example{
			{Name: "Apply fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ApplyFix -H 'Content-Type: application/json' -d '{\"scenario\":\"measures-health\"}'"},
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
