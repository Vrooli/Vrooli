package policy

import (
	"context"
	"errors"
	"log"
	"time"

	"connectrpc.com/connect"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
	internalpolicy "lighting-control/internal/policy"
)

type connectHandler struct {
	service *Service
	logger  *log.Logger
}

func NewConnectHandler(service *Service, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{service: service, logger: logger}
}

func (h *connectHandler) GetPolicy(ctx context.Context, _ *connect.Request[policyv1.GetPolicyRequest]) (*connect.Response[policyv1.GetPolicyResponse], error) {
	config, effective, err := h.service.Get(ctx)
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.GetPolicyResponse{Config: config, Effective: effective}), nil
}

func (h *connectHandler) SetPolicy(ctx context.Context, req *connect.Request[policyv1.SetPolicyRequest]) (*connect.Response[policyv1.SetPolicyResponse], error) {
	if err := (internalpolicy.Engine{Config: req.Msg.Config}).Validate(); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	config, err := h.service.SetConfig(ctx, req.Msg.Config)
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.SetPolicyResponse{Config: config}), nil
}

func (h *connectHandler) SetManualPreset(ctx context.Context, req *connect.Request[policyv1.SetManualPresetRequest]) (*connect.Response[policyv1.SetManualPresetResponse], error) {
	expires := time.Time{}
	var err error
	if req.Msg.ExpiresAt != nil {
		expires, err = req.Msg.ExpiresAt.AsTime(), req.Msg.ExpiresAt.CheckValid()
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}
	effective, results, err := h.service.SetManual(ctx, req.Msg.Preset, expires)
	if err != nil {
		if errors.Is(err, internalpolicy.ErrInvalidManualRequest) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.SetManualPresetResponse{Effective: effective, Results: results}), nil
}

func (h *connectHandler) ClearManualPreset(ctx context.Context, _ *connect.Request[policyv1.ClearManualPresetRequest]) (*connect.Response[policyv1.ClearManualPresetResponse], error) {
	cleared, _, _, err := h.service.ClearManual(ctx)
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.ClearManualPresetResponse{Cleared: cleared}), nil
}

func (h *connectHandler) EvaluatePolicy(ctx context.Context, _ *connect.Request[policyv1.EvaluatePolicyRequest]) (*connect.Response[policyv1.EvaluatePolicyResponse], error) {
	effective, results, err := h.service.Evaluate(ctx)
	if err != nil {
		return nil, policyError(err)
	}
	return connect.NewResponse(&policyv1.EvaluatePolicyResponse{Effective: effective, Results: results}), nil
}

func policyError(err error) error {
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return connect.NewError(connect.CodeCanceled, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
