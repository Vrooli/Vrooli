package validation

import (
	"context"
	"log"
	"path/filepath"

	internalaudit "quality-health/internal/audit"
	"quality-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

func Module(logger *log.Logger, repoRoot string) module.Module {
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "quality-health"))
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment unavailable: %v", err)
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
		Auditor:      internalaudit.New(nil),
		Logger:       logger,
		MaturitySpec: spec,
		Environment:  environment,
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
		Summary:     "Validate a scenario's static quality posture",
		Description: "Runs Quality Health's static audit and returns the shared scenario-validation response; the native AuditQualityResponse is packed into native_detail.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":          "string (required when path is empty)",
				"path":              "string (required when scenario is empty)",
				"include_execution": "bool (run command-backed checks)",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario":      "string",
				"status":        "scenario_validation.v1.ValidationStatus",
				"assessment":    "common.v1.MaturityAssessment",
				"native_detail": "google.protobuf.Any<quality_health.v1.audit.AuditQualityResponse>",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"},
		},
		Examples: []module.Example{
			{Name: "Validate scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"quality-health\"}'"},
		},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic static-quality fixes",
		Description: "Returns deterministic config edits quality-health can apply without writing files.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string (required when path is empty)",
				"path":     "string (required when scenario is empty)",
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
			{Name: "Preview fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/PreviewFix -H 'Content-Type: application/json' -d '{\"scenario\":\"quality-health\"}'"},
		},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic static-quality fixes",
		Description: "Applies deterministic config edits quality-health can safely perform.",
		Category:    "validation",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string (required when path is empty)",
				"path":     "string (required when scenario is empty)",
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
			{Name: "Apply fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ApplyFix -H 'Content-Type: application/json' -d '{\"scenario\":\"quality-health\"}'"},
		},
	},
}
