package operatorinputs

import (
	"context"

	"connectrpc.com/connect"

	operatorinputsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorinputs/operatorinputsv1connect"

	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	internaloperatorinputs "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorinputs"
)

func Module(applier internaloperatorinputs.CapabilityApplier, currentRevision func(context.Context) (string, error), targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := operatorinputsconnect.NewOperatorInputsServiceHandler(
		NewConnectHandler(internaloperatorinputs.Service{Applier: applier, CurrentRevision: currentRevision}),
		connect.WithInterceptors(interceptors...),
	)
	return module.Connect("operatorinputs", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{
		ID: "operatorinputs_list", Path: operatorinputsconnect.OperatorInputsServiceListOperatorInputsProcedure, Method: "POST",
		Summary: "List pending operator inputs", Description: "Returns the typed metadata queue awaiting operator answers.", Category: "operatorinputs",
		Response: &module.Schema{Type: "object", Properties: map[string]string{"requests": "array<setup.OperatorInputRequest>"}},
	},
	{
		ID: "operatorinputs_resolve", Path: operatorinputsconnect.OperatorInputsServiceResolveOperatorInputsProcedure, Method: "POST",
		Summary: "Resolve operator inputs", Description: "Validates and applies the submitted operator answers.", Category: "operatorinputs",
		Request:  &module.Schema{Type: "object", Properties: map[string]string{"target": "string (required)", "expected_revision": "string", "answers": "array<Answer> (required)"}},
		Response: &module.Schema{Type: "object", Properties: map[string]string{"configuration_pending": "boolean", "outcomes": "array<AnswerOutcome>"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Malformed or incomplete answers"},
			{Status: 412, Code: "failed_precondition", Description: "A capability did not become ready; remediation is returned"},
			{Status: 409, Code: "aborted", Description: "The target configuration revision changed; reload before answering"},
			{Status: 500, Code: "internal", Description: "Queue or control-plane failure"},
		},
	},
}
