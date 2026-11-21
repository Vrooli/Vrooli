package sweep

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
)

func (h *Handler) RunWorkload(ctx context.Context, req *connect.Request[sweepv1.WorkloadRequest]) (*connect.Response[sweepv1.WorkloadReading], error) {
	if h.workloads == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("workload owner unavailable"))
	}
	r, err := h.workloads.Run(ctx, req.Msg.GetScenario(), req.Msg.GetWorkload())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(r), nil
}

func (h *Handler) GetWorkload(ctx context.Context, req *connect.Request[sweepv1.WorkloadRequest]) (*connect.Response[sweepv1.WorkloadReading], error) {
	if h.workloads == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("workload owner unavailable"))
	}
	r, err := h.workloads.Get(ctx, req.Msg.GetScenario(), req.Msg.GetWorkload())
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewResponse(r), nil
}
