// Package reindex hosts the Connect-RPC handler for ui-health's
// ReindexService. Wires to the aisearch service.
package reindex

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	"ui-health/internal/aisearch"

	reindexv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/reindex"
)

type Reindexer interface {
	Reindex(ctx context.Context, scenario string, dryRun bool) (jobID string, plannedUpserts, plannedDeletes int, err error)
	ReindexStatus(jobID string) (state string, processed, total int, errMsg string, warnings []string, ok bool)
	ReindexCancel(jobID string) bool
}

type Deps struct {
	Logger    *log.Logger
	Reindexer Reindexer
}

type connectHandler struct {
	deps Deps
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Reindex(ctx context.Context, req *connect.Request[reindexv1.ReindexRequest]) (*connect.Response[reindexv1.ReindexResponse], error) {
	if h.deps.Reindexer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("reindex service not configured"))
	}
	r := req.Msg
	jobID, up, del, err := h.deps.Reindexer.Reindex(ctx, r.GetScenario(), r.GetDryRun())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&reindexv1.ReindexResponse{
		JobId:          jobID,
		PlannedUpserts: int32(up),
		PlannedDeletes: int32(del),
		DryRun:         r.GetDryRun(),
	}), nil
}

func (h *connectHandler) ReindexStatus(_ context.Context, req *connect.Request[reindexv1.ReindexStatusRequest]) (*connect.Response[reindexv1.ReindexStatusResponse], error) {
	if h.deps.Reindexer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("reindex service not configured"))
	}
	jobID := req.Msg.GetJobId()
	state, processed, total, errMsg, warnings, ok := h.deps.Reindexer.ReindexStatus(jobID)
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("reindex job not found"))
	}
	return connect.NewResponse(&reindexv1.ReindexStatusResponse{
		JobId:     jobID,
		State:     state,
		Processed: int32(processed),
		Total:     int32(total),
		Error:     errMsg,
		Warnings:  warnings,
	}), nil
}

func (h *connectHandler) ReindexCancel(_ context.Context, req *connect.Request[reindexv1.ReindexCancelRequest]) (*connect.Response[reindexv1.ReindexCancelResponse], error) {
	if h.deps.Reindexer == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("reindex service not configured"))
	}
	cancelled := h.deps.Reindexer.ReindexCancel(req.Msg.GetJobId())
	return connect.NewResponse(&reindexv1.ReindexCancelResponse{
		JobId:     req.Msg.GetJobId(),
		Cancelled: cancelled,
	}), nil
}

// ServiceAdapter wraps *aisearch.Service to satisfy the Reindexer interface.
type ServiceAdapter struct{ Service *aisearch.Service }

func (a ServiceAdapter) Reindex(ctx context.Context, scenario string, dryRun bool) (string, int, int, error) {
	job, err := a.Service.Reindex(ctx, scenario, dryRun)
	if err != nil {
		return "", 0, 0, err
	}
	exp := a.Service.JobExport(job)
	up, _ := exp["planned_upserts"].(int)
	del, _ := exp["planned_deletes"].(int)
	return safeString(exp["job_id"]), up, del, nil
}

func (a ServiceAdapter) ReindexStatus(jobID string) (string, int, int, string, []string, bool) {
	job, ok := a.Service.ReindexStatus(jobID)
	if !ok {
		return "", 0, 0, "", nil, false
	}
	exp := a.Service.JobExport(job)
	state, _ := exp["state"].(string)
	processed, _ := exp["processed"].(int)
	total, _ := exp["total"].(int)
	errMsg, _ := exp["error"].(string)
	warnings, _ := exp["warnings"].([]string)
	return state, processed, total, errMsg, warnings, true
}

func (a ServiceAdapter) ReindexCancel(jobID string) bool { return a.Service.ReindexCancel(jobID) }

func safeString(v interface{}) string {
	s, _ := v.(string)
	return s
}
