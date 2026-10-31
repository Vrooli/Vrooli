package selection

import (
	"connectrpc.com/connect"
	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection/selectionv1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/selection"
)

func Module(service selection.Service, targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := selectionconnect.NewSelectionServiceHandler(NewConnectHandler(service), connect.WithCodec(&strictJSONCodec{name: "json"}), connect.WithCodec(&strictJSONCodec{name: "json; charset=utf-8"}), connect.WithInterceptors(interceptors...))
	return module.Connect("selection", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "selection_scenarios", Path: selectionconnect.SelectionServiceListScenariosProcedure, Method: "POST", Summary: "List onboarding scenarios", Category: "selection"},
	{ID: "selection_core_set", Path: selectionconnect.SelectionServiceGetCoreSetProcedure, Method: "POST", Summary: "Read onboarding core set", Category: "selection", Request: &module.Schema{Type: "object", Properties: map[string]string{"seed": "array of strings"}}},
	{ID: "selection_recommendation", Path: selectionconnect.SelectionServiceGetRecommendationProcedure, Method: "POST", Summary: "Read onboarding recommendation", Category: "selection"},
	{ID: "selection_accept_recommendation", Path: selectionconnect.SelectionServiceAcceptRecommendationProcedure, Method: "POST", Summary: "Accept onboarding recommendation", Category: "selection"},
	{ID: "selection_closure", Path: selectionconnect.SelectionServiceGetClosureProcedure, Method: "POST", Summary: "Read selection closure", Category: "selection"},
	{ID: "selection_union", Path: selectionconnect.SelectionServiceGetUnionProcedure, Method: "POST", Summary: "Read deployment union", Category: "selection"},
	{ID: "selection_handoff", Path: selectionconnect.SelectionServiceCreateHandoffProcedure, Method: "POST", Summary: "Create onboarding handoff", Category: "selection"},
	{ID: "selection_handoff_get", Path: selectionconnect.SelectionServiceGetHandoffProcedure, Method: "POST", Summary: "Resolve onboarding handoff", Category: "selection"},
	{ID: "selection_handoff_rest", Path: "/api/v2/handoff", Method: "POST", Summary: "Create bridge onboarding handoff", Category: "selection"},
}
