package validation

import (
	"log"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"proto-health/internal/module"
	"proto-health/internal/protosurface"
	internal "proto-health/internal/validation"

	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/validation/validation_v1connect"
)

var ProtoFile = validationv1.File_proto_health_v1_validation_validation_proto

func Module(logger *log.Logger, repoRoot string) module.Module {
	loader, err := protosurface.NewDescriptorLoaderFromFile(
		repoRoot,
		filepath.Join(repoRoot, "packages", "proto", "gen", "descriptor", "image.binpb"),
	)
	if err != nil {
		logger.Fatalf("proto descriptor loader: %v", err)
	}
	validator := internal.New(internal.Deps{Loader: loader, RepoRoot: repoRoot})
	connectPath, connectHandler := validationconnect.NewProtoHealthServiceHandler(NewConnectHandler(Deps{
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

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "validation_validate_scenario",
		Path:        validationconnect.ProtoHealthServiceValidateScenarioProcedure,
		Method:      "POST",
		Summary:     "Validate a scenario's proto contracts",
		Description: "Validates one scenario's Protocol Buffer structure, annotations, adoption signals, transport world, and conservative unused-message hints without computing fleet dependency graphs.",
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
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id or no proto files found for the scenario"},
		},
		Examples: []module.Example{
			{Name: "Validate proto-health", Curl: "curl http://localhost:${API_PORT}/vrooli.proto_health.v1.validation.ProtoHealthService/ValidateScenario -H 'Content-Type: application/json' -d '{\"scenario\":\"proto-health\"}'"},
		},
	},
	{
		ID:          "validation_describe_scenario_protos",
		Path:        validationconnect.ProtoHealthServiceDescribeScenarioProtosProcedure,
		Method:      "POST",
		Summary:     "Describe a scenario's proto surface",
		Description: "Returns the scenario-scoped proto surface fact: files, services, RPCs, messages, imports, annotations, adoption signals, and transport-world summary.",
		Category:    "validation",
		Request: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"scenario": "string (required, scenario id)"},
		},
		Response: &module.Schema{
			Type:       "object",
			Properties: map[string]string{"surface": "ProtoSurface"},
		},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing scenario id or no proto files found for the scenario"},
		},
		Examples: []module.Example{
			{Name: "Describe proto-health", Curl: "curl http://localhost:${API_PORT}/vrooli.proto_health.v1.validation.ProtoHealthService/DescribeScenarioProtos -H 'Content-Type: application/json' -d '{\"scenario\":\"proto-health\"}'"},
		},
	},
}
