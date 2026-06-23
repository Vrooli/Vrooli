package policy

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"
	domainpolicy "network-manager/internal/policy"

	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy"
	policyconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/policy/policy_v1connect"
)

type handler struct {
	service *domainpolicy.Service
}

func Module(db domainpolicy.SQLExecutor) module.Module {
	service := domainpolicy.NewService(domainpolicy.Config{
		Repo:    domainpolicy.NewSQLiteRepository(db),
		Adapter: domainpolicy.ConservativeResolverPolicyAdapter{},
	})
	path, h := policyconnect.NewPolicyServiceHandler(&handler{service: service})
	return module.Module{Name: "policy", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return domainpolicy.Schema() }

func (h *handler) PreviewPolicyChange(ctx context.Context, req *connect.Request[policyv1.PreviewPolicyChangeRequest]) (*connect.Response[policyv1.PreviewPolicyChangeResponse], error) {
	change, err := h.service.Preview(ctx, req.Msg.GetTarget(), req.Msg.GetAction(), req.Msg.GetValues())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&policyv1.PreviewPolicyChangeResponse{Preview: toProtoChange(change)}), nil
}

func (h *handler) ApplyPolicyChange(ctx context.Context, req *connect.Request[policyv1.ApplyPolicyChangeRequest]) (*connect.Response[policyv1.ApplyPolicyChangeResponse], error) {
	change, err := h.service.Apply(ctx, req.Msg.GetPreviewId(), req.Msg.GetApproved())
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.ApplyPolicyChangeResponse{Change: toProtoChange(change)}), nil
}

func (h *handler) RollbackPolicyChange(ctx context.Context, req *connect.Request[policyv1.RollbackPolicyChangeRequest]) (*connect.Response[policyv1.RollbackPolicyChangeResponse], error) {
	change, err := h.service.Rollback(ctx, req.Msg.GetId())
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.RollbackPolicyChangeResponse{Change: toProtoChange(change)}), nil
}

func (h *handler) PauseFiltering(ctx context.Context, req *connect.Request[policyv1.PauseFilteringRequest]) (*connect.Response[policyv1.PauseFilteringResponse], error) {
	change, err := h.service.Pause(ctx, req.Msg.GetTarget(), req.Msg.GetDuration())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&policyv1.PauseFilteringResponse{Change: toProtoChange(change)}), nil
}

func (h *handler) ResumeFiltering(ctx context.Context, req *connect.Request[policyv1.ResumeFilteringRequest]) (*connect.Response[policyv1.ResumeFilteringResponse], error) {
	change, err := h.service.Resume(ctx, req.Msg.GetTarget())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&policyv1.ResumeFilteringResponse{Change: toProtoChange(change)}), nil
}

func policyError(err error) error {
	switch {
	case errors.Is(err, domainpolicy.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
}

func toProtoChange(change domainpolicy.Change) *policyv1.PolicyChange {
	return &policyv1.PolicyChange{
		Id:                change.ID,
		Target:            change.Target,
		Action:            change.Action,
		Status:            change.Status,
		Effects:           change.Effects,
		RollbackSupported: change.RollbackSupported,
	}
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
