package validation

import (
	"context"
	"log"
	"path/filepath"
	"time"

	"cli-health/internal/module"
	"cli-health/internal/services/manifestvalidation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/measures-go/manifestscan"
	vroolicli "github.com/vrooli/vrooli-cli-go"

	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// ProtoFile exposes the shared validation proto FileDescriptor so the
// global parity test (api/internal/modules/registry_test.go) can walk it
// without importing the gen/go package directly.
var ProtoFile = scenariovalidationv1.File_scenario_validation_v1_validation_proto

// Module returns the validation domain's contribution to the API: a single
// Connect-RPC service handler mounted at the generated procedure path. No
// REST exception — every RPC is proto-typed. The validator is constructed
// with the default filesystem/buf/JSONSchema seams rooted at repoRoot.
func Module(logger *log.Logger, repoRoot string, reservedNames []string) module.Module {
	validator := manifestvalidation.New(manifestvalidation.Deps{
		Manifests:    manifestvalidation.NewFilesystemManifestLoader(repoRoot),
		Schema:       manifestvalidation.NewJSONSchemaValidator(repoRoot),
		Protos:       manifestvalidation.NewBufProtoLoader(repoRoot),
		Measures:     manifestscan.NewDescriptorSchemaReader(repoRoot),
		RuntimeProbe: manifestvalidation.NewCLIRuntimeProbe(5 * time.Second),
		Logger:       logger,
	})
	spec, err := assessment.LoadSpecFromScenario(filepath.Join(repoRoot, "scenarios", "cli-health"))
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
		Logger:        logger,
		Validator:     validator,
		ReservedNames: reservedNames,
		MaturitySpec:  spec,
		Environment:   environment,
	}))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// Schema returns "" — validation is stateless in Phase 1 (no tables). The
// modules registry includes this re-export anyway so adding tables later is
// a uniform "edit Schema, EnsureSchemas picks it up" change.
func Schema() string { return "" }

// Endpoints is the machine-readable description of the validation module's
// public surface. References the generated *Procedure constants so renaming
// an RPC in validation.proto breaks this file at compile time. The global
// proto/Connect parity test enforces 1:1 coverage automatically.
var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        scenariovalidationconnect.ScenarioValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's CLI manifest, proto, and endpoints.json",
		Description: "Runs the cli-health validators against a scenario and returns the shared scenario-validation response with findings in the maturity assessment.",
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
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id, reserved CLI name, or validation input error"},
		},
		Examples: []module.Example{
			{Name: "Validate scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"cli-health\"}'"},
		},
	},
	{
		ID:          "validation_preview_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServicePreviewFixProcedure,
		Method:      "POST",
		Summary:     "Preview deterministic remediations for a scenario (unimplemented)",
		Description: "cli-health ships no deterministic autofixer; PreviewFix returns Unimplemented so the test-genie deterministic-fix aggregate records this provider as no_fixer and skips it.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string"}},
		Response:    &module.Schema{Type: "object"},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "cli-health has no deterministic fixer"},
		},
	},
	{
		ID:          "validation_apply_fix",
		Path:        scenariovalidationconnect.ScenarioValidationServiceApplyFixProcedure,
		Method:      "POST",
		Summary:     "Apply deterministic remediations for a scenario (unimplemented)",
		Description: "cli-health ships no deterministic autofixer; ApplyFix returns Unimplemented so the test-genie deterministic-fix aggregate records this provider as no_fixer and skips it.",
		Category:    "validation",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"scenario": "string"}},
		Response:    &module.Schema{Type: "object"},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "cli-health has no deterministic fixer"},
		},
	},
}
