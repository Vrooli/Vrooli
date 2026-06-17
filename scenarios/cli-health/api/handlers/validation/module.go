package validation

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cli-health/internal/module"
	"cli-health/internal/services/manifestvalidation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/maturity-go/assessment"
	"github.com/vrooli/measures-go/manifestscan"

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
		Manifests: manifestvalidation.NewFilesystemManifestLoader(repoRoot),
		Schema:    manifestvalidation.NewJSONSchemaValidator(repoRoot),
		Protos:    manifestvalidation.NewBufProtoLoader(repoRoot),
		Measures:  manifestscan.NewDescriptorSchemaReader(repoRoot),
		Logger:    logger,
	})
	spec, err := loadMaturitySpec(repoRoot)
	if err != nil && logger != nil {
		logger.Printf("validation: maturity assessment disabled: %v", err)
	}
	connectPath, connectHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(NewConnectHandler(Deps{
		Logger:        logger,
		Validator:     validator,
		ReservedNames: reservedNames,
		MaturitySpec:  spec,
	}))
	return module.Module{
		Name: "validation",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func loadMaturitySpec(repoRoot string) (*assessment.Spec, error) {
	path := filepath.Join(repoRoot, "scenarios", "cli-health", ".vrooli", "maturity.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return assessment.ParseSpec(raw)
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
		CLIMapping: &module.CLIMapping{
			Command: "cli-health validate scenario",
			Args:    []string{"<name>"},
		},
	},
}
