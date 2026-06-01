// Package reindex hosts the Connect-RPC handler for security-health's
// ReindexService — async reconciliation of the fleet dependency corpus.
package reindex

import (
	"context"
	"log"

	"connectrpc.com/connect"

	depdomain "security-health/internal/dependencies"

	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/security-health/v1/reindex"
)

// Reindexer is the slice of dependencies.Service the handler exercises.
type Reindexer interface {
	Reindex(ctx context.Context, scenario string, dryRun bool) (depdomain.ReindexResult, error)
	ReindexStatus(jobID string) (state string, processed, total int, errMsg string, ok bool)
	ReindexCancel(jobID string) (cancelled, ok bool)
}

// Deps wires the handler's seams.
type Deps struct {
	Logger  *log.Logger
	Service Reindexer
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler returns a handler satisfying ReindexServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Reindex(ctx context.Context, req *connect.Request[reindexv1.ReindexRequest]) (*connect.Response[reindexv1.ReindexResponse], error) {
	res, err := h.deps.Service.Reindex(ctx, req.Msg.GetScenario(), req.Msg.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&reindexv1.ReindexResponse{
		JobId:          res.JobID,
		PlannedUpserts: int32(res.PlannedUpserts),
		PlannedDeletes: int32(res.PlannedDeletes),
		DryRun:         res.DryRun,
	}), nil
}

func (h *connectHandler) ReindexStatus(_ context.Context, req *connect.Request[reindexv1.ReindexStatusRequest]) (*connect.Response[reindexv1.ReindexStatusResponse], error) {
	state, processed, total, errMsg, ok := h.deps.Service.ReindexStatus(req.Msg.GetJobId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errUnknownJob(req.Msg.GetJobId()))
	}
	return connect.NewResponse(&reindexv1.ReindexStatusResponse{
		JobId:     req.Msg.GetJobId(),
		State:     state,
		Processed: int32(processed),
		Total:     int32(total),
		Error:     errMsg,
	}), nil
}

func (h *connectHandler) ReindexCancel(_ context.Context, req *connect.Request[reindexv1.ReindexCancelRequest]) (*connect.Response[reindexv1.ReindexCancelResponse], error) {
	cancelled, ok := h.deps.Service.ReindexCancel(req.Msg.GetJobId())
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errUnknownJob(req.Msg.GetJobId()))
	}
	return connect.NewResponse(&reindexv1.ReindexCancelResponse{
		JobId:     req.Msg.GetJobId(),
		Cancelled: cancelled,
	}), nil
}
