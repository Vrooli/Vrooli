package validation

import (
	"context"
	"log"
	"path/filepath"

	"storage-health/internal/module"
	"storage-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// ProtoFile exposes the shared validation proto FileDescriptor so the global
// parity test (api/internal/modules/registry_test.go) can walk it.
var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

// Module returns the validation domain's contribution to the API: the shared
// ScenarioValidationService handler mounted at its generated procedure path.
// The validator is constructed with the real detector/analyzer seams rooted at
// repoRoot.
func Module(logger *log.Logger, repoRoot string) module.Module {
	validator := validation.New(validation.Deps{RepoRoot: repoRoot, Logger: logger})
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "storage-health"))
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment disabled: %v", err)
	}
	// Capture host facts once; they do not change during the process lifetime.
	// A failure (CLI unavailable) is non-fatal — the metrics collector backfills
	// os/arch/num_cpu from the stdlib.
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
		RepoRoot:     repoRoot,
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

// Schema returns "" — validation is stateless (no tables). The registry
// re-exports it anyway so the per-domain shape stays uniform.
func Schema() string { return "" }

// Endpoints is the machine-readable description of the validation module's
// public surface. References the generated *Procedure constant so renaming the
// RPC in validation.proto breaks this file at compile time.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's storage judgment",
		Description: "Detects the target scenario's storage surface (engines + API language) and runs the applicable static storage analyzers, returning the shared scenario-validation response with findings in the maturity assessment.",
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
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id or scenario not found under scenarios/"},
		},
		Examples: []module.Example{
			{Name: "Validate scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"storage-health\"}'"},
		},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview the deterministic storage autofixes for a scenario",
		Description: "Shared ScenarioValidationService Fix RPC. Returns the candidate edits the storage-health autofix registry would apply (DB_ROWS_NOT_CLOSED, ENSURE_SCHEMAS_NOT_WIRED) without writing anything. Optional rule_ids restricts the preview to specific finding codes.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "[]string"}},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "[]scenario_validation.v1.FixCandidate", "messages": "[]string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id/path or scenario directory not resolvable"},
		},
		Examples: []module.Example{
			{Name: "Preview storage fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/PreviewFix -H 'Content-Type: application/json' -d '{\"scenario\":\"storage-health\"}'"},
		},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply the deterministic storage autofixes for a scenario",
		Description: "Shared ScenarioValidationService Fix RPC. Applies the storage-health autofix registry's deterministic edits (DB_ROWS_NOT_CLOSED, ENSURE_SCHEMAS_NOT_WIRED) and reports what changed. Idempotent: a second apply over an already-fixed tree is a no-op.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "[]string"}},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "[]scenario_validation.v1.FixCandidate", "messages": "[]string"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id/path or scenario directory not resolvable"},
		},
		Examples: []module.Example{
			{Name: "Apply storage fixes", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ApplyFix -H 'Content-Type: application/json' -d '{\"scenario\":\"storage-health\"}'"},
		},
	},
}
