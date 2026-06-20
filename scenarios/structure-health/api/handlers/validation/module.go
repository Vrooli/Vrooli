package validation

import (
	"context"
	"log"
	"path/filepath"

	"structure-health/internal/module"
	"structure-health/internal/profile"
	internalvalidation "structure-health/internal/validation"

	"github.com/gorilla/mux"
	"github.com/vrooli/maturity-go/assessment"
	vroolicli "github.com/vrooli/vrooli-cli-go"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/validation/validation_v1connect"
)

// ProtoFile and ScenarioValidationProtoFile are the FileDescriptors backing the
// two Connect-mounted services; the global parity test walks them against the
// Endpoints slice.
var (
	ProtoFile                   = validationv1.File_structure_health_v1_validation_validation_proto
	ScenarioValidationProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto
)

// Module mounts both the native ValidationService and the shared
// ScenarioValidationService (dual-mount), backed by one internal engine.
func Module(logger *log.Logger, repoRoot string) module.Module {
	svc := internalvalidation.New()
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "structure-health"))
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment unavailable: %v", err)
	}
	svc.Spec = spec
	svc.Facts = profile.CodeFactsClient{Locator: profile.DefaultLocator{RepoRoot: repoRoot}}
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
	handler := NewHandlerWithDeps(Deps{
		Service:      svc,
		Logger:       logger,
		MaturitySpec: spec,
		Environment:  environment,
	})
	connectPath, connectHandler := validationconnect.NewValidationServiceHandler(handler)
	sharedPath, sharedHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(NewSharedHandler(handler))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			r.PathPrefix(connectPath).Handler(connectHandler)
			r.PathPrefix(sharedPath).Handler(sharedHandler)
		},
		Endpoints: Endpoints,
	}
}

// Schema returns the empty schema: validation owns no database tables.
func Schema() string { return "" }

// Endpoints is the static endpoint metadata for codegen and the parity test.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        validationconnect.ValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's structure and lifecycle wiring",
		Description: "Reconciles code-facts ground truth against declared service.json intent and returns profile-aware skeleton and lifecycle-wiring findings plus a shared maturity assessment.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "include_execution": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"status": "string", "profile": "DetectedProfile", "surfaces": "array<SurfaceReconcile>", "findings": "array<StructureFinding>", "assessment": "common.v1.MaturityAssessment"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "validation_preview_fix_config",
		Path:        validationconnect.ValidationServicePreviewFixConfigProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic structure/service.json fixes",
		Description: "Returns the format-preserving service.json edits and skeleton files structure-health could apply for the requested rules, without writing anything.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>", "apply": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<AutofixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "validation_apply_fix_config",
		Path:        validationconnect.ValidationServiceApplyFixConfigProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic structure/service.json fixes",
		Description: "Applies the format-preserving service.json edits and skeleton files for the requested rules and reports what changed.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "rule_ids": "array<string>", "apply": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "applied": "bool", "candidates": "array<AutofixCandidate>", "messages": "array<string>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
	{
		ID:          "scenario_validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate scenario structure through the shared provider contract",
		Description: "Runs Structure Health's engine and returns the shared scenario-validation response; the native structure-health ValidateScenarioResponse is packed into native_detail.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "path": "string", "include_execution": "bool"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string", "status": "scenario_validation.v1.ValidationStatus", "assessment": "common.v1.MaturityAssessment", "native_detail": "google.protobuf.Any<structure_health.v1.validation.ValidateScenarioResponse>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Scenario/path is missing or cannot be resolved"}},
	},
}
