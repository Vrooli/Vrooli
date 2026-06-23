package validation

import (
	"context"
	"log"
	"path/filepath"

	"ui-health/internal/autofix"
	"ui-health/internal/codefacts"
	"ui-health/internal/module"
	"ui-health/internal/services/manifestvalidation"
	"ui-health/internal/uiruntime"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// ProtoFile exposes the shared validation proto FileDescriptor so the
// global parity test (api/internal/modules/registry_test.go) can walk it.
var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

// Module returns the validation domain's contribution to the API.
func Module(logger *log.Logger, repoRoot string) module.Module {
	validator := manifestvalidation.New(repoRoot, logger)
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "ui-health"))
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
	connectPath, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(NewConnectHandler(Deps{
		Logger:       logger,
		Validator:    validator,
		MaturitySpec: spec,
		Fixer:        autofix.New(validator),
		RepoRoot:     repoRoot,
		Environment:  environment,
		CodeFacts:    codefacts.New(),
		// Freshness backs the static UI-bundle freshness group via the canonical
		// content-hash engine (typed vrooli CLI). Degrades gracefully when the
		// engine can't be resolved.
		Freshness: vroolicli.New(),
		// Runtime drives the BAS iframe-bridge handshake + render group when a
		// caller requests execution (include_execution=true) against a UI scenario.
		// It degrades gracefully — unreachable BAS/UI yields skipped findings.
		Runtime: uiruntime.New(logger),
	}))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's UI manifest, slot directories, and overlay rules",
		Description: "Runs ui-health validators against a scenario and returns the shared scenario-validation response with findings in the maturity assessment.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":      "string",
				"status":        "scenario_validation.v1.ValidationStatus",
				"assessment":    "common.v1.MaturityAssessment",
				"native_detail": "google.protobuf.Any",
			},
		},
		Examples: []module.Example{
			{Name: "Validate scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"ui-health\"}'"},
		},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview ui-health's deterministic UI auto-fixes for a scenario",
		Description: "Previews (dry-run) the safe mechanical remediations ui-health can apply to a scenario's auto-fixable UI findings, returning candidate edits without writing.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)", "path": "string (optional, explicit scenario root)", "rule_ids": "string[] (optional, restrict to finding codes)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":   "string",
				"applied":    "bool (false for preview)",
				"candidates": "scenario_validation.v1.FixCandidate[]",
				"messages":   "string[]",
			},
		},
		Examples: []module.Example{
			{Name: "Preview fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/PreviewFix -H 'Content-Type: application/json' -d '{\"scenario\":\"ui-health\"}'"},
		},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply ui-health's deterministic UI auto-fixes for a scenario",
		Description: "Writes the safe mechanical remediations ui-health can apply to a scenario's auto-fixable UI findings. Idempotent: re-applying a fixed scenario yields no candidates.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)", "path": "string (optional, explicit scenario root)", "rule_ids": "string[] (optional, restrict to finding codes)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":   "string",
				"applied":    "bool (true)",
				"candidates": "scenario_validation.v1.FixCandidate[]",
				"messages":   "string[]",
			},
		},
		Examples: []module.Example{
			{Name: "Apply fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ApplyFix -H 'Content-Type: application/json' -d '{\"scenario\":\"ui-health\"}'"},
		},
	},
}
