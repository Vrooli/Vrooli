package operatorstate

import (
	"connectrpc.com/connect"
	operatorstatev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate/operatorstatev1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
)

func Module(service operatorstateapi.Service, targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := operatorstatev1connect.NewOperatorStateServiceHandler(NewConnectHandler(service), connect.WithInterceptors(interceptors...))
	return module.Connect("operatorstate", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "operator_state_get", Path: operatorstatev1connect.OperatorStateServiceGetOperatorStateProcedure, Method: "POST", Summary: "Read operator state", Description: "Returns the typed persisted operator selection document.", Category: "operatorstate"},
	{ID: "operator_state_patch", Path: operatorstatev1connect.OperatorStateServicePatchOperatorStateProcedure, Method: "POST", Summary: "Patch operator state", Description: "Applies only the operator state fields named by the update mask, checks the optimistic-concurrency revision, and returns a durable secret-free mutation receipt.", Category: "operatorstate", Request: &module.Schema{Type: "object", Properties: map[string]string{"expected_revision": "string (recommended for concurrent clients)", "request_id": "string (stable across retries; server-derived if omitted)", "update_mask": "field mask (required)"}}},
}
