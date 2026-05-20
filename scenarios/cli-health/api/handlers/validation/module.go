package validation

import (
	"log"

	"cli-health/internal/module"
	"cli-health/internal/services/manifestvalidation"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/validation/validation_v1connect"
)

// ProtoFile exposes the validation domain's proto FileDescriptor so the
// global parity test (api/internal/modules/registry_test.go) can walk it
// without importing the gen/go package directly.
var ProtoFile = validationv1.File_cli_health_v1_validation_validation_proto

// Module returns the validation domain's contribution to the API: a single
// Connect-RPC service handler mounted at the generated procedure path. No
// REST exception — every RPC is proto-typed. The validator is constructed
// with the default filesystem/buf/JSONSchema seams rooted at repoRoot.
func Module(logger *log.Logger, repoRoot string) module.Module {
	validator := manifestvalidation.New(manifestvalidation.Deps{
		Manifests: manifestvalidation.NewFilesystemManifestLoader(repoRoot),
		Schema:    manifestvalidation.NewJSONSchemaValidator(repoRoot),
		Protos:    manifestvalidation.NewBufProtoLoader(repoRoot),
		Logger:    logger,
	})
	connectPath, connectHandler := validationconnect.NewValidationServiceHandler(NewConnectHandler(Deps{
		Logger:    logger,
		Validator: validator,
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
		Path:        validationconnect.ValidationServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's CLI manifest, proto, and endpoints.json",
		Description: "Runs the cli-health validators against a scenario and returns a structured Finding list. Phase 1 stub returns Unimplemented.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)"},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"scenario": "string",
				"passed":   "boolean",
				"findings": "array<Finding>",
				"summary":  "Summary",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Phase 1 stub; Phase 2 wires the real validators"},
		},
		Examples: []module.Example{
			{Name: "Validate scenario", Curl: "curl http://localhost:${API_PORT}/vrooli.cli_health.v1.validation.ValidationService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"cli-health\"}'"},
		},
		CLIMapping: &module.CLIMapping{
			Command: "cli-health validate scenario",
			Args:    []string{"<name>"},
		},
	},
}
