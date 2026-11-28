package apply

import (
	"connectrpc.com/connect"
	applyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/apply/applyv1connect"
	applydomain "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/apply"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/module"
)

func Module(service applydomain.Service, targetInterceptors ...connect.Interceptor) module.Module {
	interceptors := []connect.Interceptor{authz.MutationInterceptor()}
	interceptors = append(interceptors, targetInterceptors...)
	path, handler := applyconnect.NewApplyServiceHandler(NewConnectHandler(service), connect.WithInterceptors(interceptors...))
	return module.Connect("apply", path, handler, Endpoints)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "apply_start", Path: applyconnect.ApplyServiceStartApplyProcedure, Method: "POST", Summary: "Start onboarding apply", Description: "Starts a durable host configuration run.", Category: "apply"},
	{ID: "apply_run", Path: applyconnect.ApplyServiceGetApplyRunProcedure, Method: "POST", Summary: "Read onboarding apply run", Description: "Returns typed state and step events for one apply run.", Category: "apply", Request: &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string (required)"}}},
	{ID: "apply_plan", Path: applyconnect.ApplyServiceGetApplyPlanProcedure, Method: "POST", Summary: "Read onboarding apply plan", Description: "Returns the typed actions implied by the current selection.", Category: "apply"},
	{ID: "apply_review", Path: applyconnect.ApplyServiceReviewApplyProcedure, Method: "POST", Summary: "Review onboarding apply", Description: "Freezes the exact target, revision, and plan effects for explicit consent.", Category: "apply"},
	{ID: "apply_cancel", Path: applyconnect.ApplyServiceCancelApplyProcedure, Method: "POST", Summary: "Cancel onboarding apply", Description: "Requests cancellation at the next safe item boundary.", Category: "apply", Request: &module.Schema{Type: "object", Properties: map[string]string{"run_id": "string (required)"}}},
}
