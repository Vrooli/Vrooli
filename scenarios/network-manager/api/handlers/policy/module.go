package policy

import (
	"context"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"

	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy/policy_v1connect"
)

type handler struct{}

func Module() module.Module {
	path, h := policyconnect.NewPolicyServiceHandler(&handler{})
	return module.Module{Name: "policy", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return "" }

func (h *handler) PreviewPolicyChange(_ context.Context, req *connect.Request[policyv1.PreviewPolicyChangeRequest]) (*connect.Response[policyv1.PreviewPolicyChangeResponse], error) {
	return connect.NewResponse(&policyv1.PreviewPolicyChangeResponse{Preview: change("policy-preview", req.Msg.GetTarget(), req.Msg.GetAction(), "preview")}), nil
}

func (h *handler) ApplyPolicyChange(context.Context, *connect.Request[policyv1.ApplyPolicyChangeRequest]) (*connect.Response[policyv1.ApplyPolicyChangeResponse], error) {
	return connect.NewResponse(&policyv1.ApplyPolicyChangeResponse{Change: change("policy-preview", "resolver", "apply", "approval_required")}), nil
}

func (h *handler) RollbackPolicyChange(_ context.Context, req *connect.Request[policyv1.RollbackPolicyChangeRequest]) (*connect.Response[policyv1.RollbackPolicyChangeResponse], error) {
	return connect.NewResponse(&policyv1.RollbackPolicyChangeResponse{Change: change(req.Msg.GetId(), "resolver", "rollback", "preview")}), nil
}

func (h *handler) PauseFiltering(_ context.Context, req *connect.Request[policyv1.PauseFilteringRequest]) (*connect.Response[policyv1.PauseFilteringResponse], error) {
	return connect.NewResponse(&policyv1.PauseFilteringResponse{Change: change("pause-preview", req.Msg.GetTarget(), "pause_filtering", "preview")}), nil
}

func (h *handler) ResumeFiltering(_ context.Context, req *connect.Request[policyv1.ResumeFilteringRequest]) (*connect.Response[policyv1.ResumeFilteringResponse], error) {
	return connect.NewResponse(&policyv1.ResumeFilteringResponse{Change: change("resume-preview", req.Msg.GetTarget(), "resume_filtering", "preview")}), nil
}

func change(id, target, action, status string) *policyv1.PolicyChange {
	if id == "" {
		id = "policy-preview"
	}
	if target == "" {
		target = "network"
	}
	if action == "" {
		action = "inspect"
	}
	return &policyv1.PolicyChange{Id: id, Target: target, Action: action, Status: status, RollbackSupported: true, Effects: []string{"No live resolver policy changed in scaffold mode."}}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("policy_preview", policyconnect.PolicyServicePreviewPolicyChangeProcedure, "Preview filtering policy change"),
	connectEndpoint("policy_apply", policyconnect.PolicyServiceApplyPolicyChangeProcedure, "Apply approved filtering policy change"),
	connectEndpoint("policy_rollback", policyconnect.PolicyServiceRollbackPolicyChangeProcedure, "Rollback filtering policy change"),
	connectEndpoint("policy_pause", policyconnect.PolicyServicePauseFilteringProcedure, "Pause filtering"),
	connectEndpoint("policy_resume", policyconnect.PolicyServiceResumeFilteringProcedure, "Resume filtering"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "policy", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"change": "PolicyChange"}}}
}
