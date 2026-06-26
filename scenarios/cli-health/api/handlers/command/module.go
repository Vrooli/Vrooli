package command

import (
	"log"

	"cli-health/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	commandv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/command"
	commandconnect "github.com/vrooli/vrooli/packages/proto/gen/go/cli-health/v1/command/command_v1connect"
)

var ProtoFile = commandv1.File_cli_health_v1_command_command_proto

func Module(logger *log.Logger, validator Validator) module.Module {
	connectPath, connectHandler := commandconnect.NewCommandReferenceValidationServiceHandler(NewConnectHandler(Deps{
		Logger:    logger,
		Validator: validator,
	}))
	return module.Module{
		Name: "command",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

var Endpoints = []module.EndpointDescriptor{
	{
		ID:          "command_validate",
		Path:        commandconnect.CommandReferenceValidationServiceValidateCommandReferenceProcedure,
		Method:      "POST",
		Summary:     "Validate one command reference",
		Description: "Validates a Vrooli-owned command reference without executing it.",
		Category:    "command",
		Request: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"command_text": "string",
				"policy":       "CommandReferencePolicy",
				"qualifiers":   "array<string>",
			},
		},
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"result": "CommandReferenceValidationResult",
			},
		},
		Errors: []module.ErrorDesc{
			{Status: 501, Code: "unimplemented", Description: "Returned only when command validation is not configured"},
		},
		Examples: []module.Example{
			{Name: "Validate", Curl: "curl http://localhost:${API_PORT}/vrooli.cli_health.v1.command.CommandReferenceValidationService/ValidateCommandReference -H 'Content-Type: application/json' -d '{\"command_text\":\"vrooli scenario test cli-health\",\"policy\":\"COMMAND_REFERENCE_POLICY_PLAN\"}'"},
		},
	},
	{
		ID:          "command_validate_batch",
		Path:        commandconnect.CommandReferenceValidationServiceValidateCommandReferencesProcedure,
		Method:      "POST",
		Summary:     "Validate command references",
		Description: "Batch-validates Vrooli-owned command references without executing them.",
		Category:    "command",
		Response: &module.Schema{
			Type: "object",
			Properties: map[string]string{
				"results": "array<CommandReferenceValidationResult>",
			},
		},
	},
}
