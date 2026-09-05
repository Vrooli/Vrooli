// Package experiments owns the Connect transport for controlled experiments.
package experiments

import (
	"context"
	"net/http"
	"net/url"

	"connectrpc.com/connect"
	experimentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/experiments"
	experimentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/prompt-manager/v1/experiments/experiments_v1connect"

	"prompt-manager/handlers/transportbridge"
	"prompt-manager/internal/skills"
)

type connectHandler struct {
	experimentsconnect.UnimplementedExperimentsServiceHandler
	legacy *skills.ExperimentHandlers
}

func NewConnectMount(legacy *skills.ExperimentHandlers) (string, http.Handler) {
	return experimentsconnect.NewExperimentsServiceHandler(&connectHandler{legacy: legacy})
}

func (h *connectHandler) ListExperiments(ctx context.Context, req *connect.Request[experimentsv1.ListExperimentsRequest]) (*connect.Response[experimentsv1.ListExperimentsResponse], error) {
	if req.Msg.SkillId != nil {
		return transportbridge.InvokeWrappedJSON(ctx, req.Header(), h.legacy.ListExperimentsBySkill, http.MethodGet, "/skills/"+url.PathEscape(req.Msg.GetSkillId())+"/experiments", nil, map[string]string{"id": req.Msg.GetSkillId()}, "experiments", &experimentsv1.ListExperimentsResponse{})
	}
	return transportbridge.InvokeWrappedJSON(ctx, req.Header(), h.legacy.ListExperiments, http.MethodGet, "/experiments", nil, nil, "experiments", &experimentsv1.ListExperimentsResponse{})
}

func (h *connectHandler) GetExperiment(ctx context.Context, req *connect.Request[experimentsv1.GetExperimentRequest]) (*connect.Response[experimentsv1.Experiment], error) {
	return h.experimentCall(ctx, req.Header(), h.legacy.GetExperiment, http.MethodGet, req.Msg.GetExperimentId(), nil)
}

func (h *connectHandler) CreateExperiment(ctx context.Context, req *connect.Request[experimentsv1.CreateExperimentRequest]) (*connect.Response[experimentsv1.Experiment], error) {
	return transportbridge.InvokeProto(ctx, req.Header(), h.legacy.CreateExperiment, http.MethodPost, "/experiments", req.Msg, &experimentsv1.Experiment{})
}

func (h *connectHandler) UpdateExperiment(ctx context.Context, req *connect.Request[experimentsv1.UpdateExperimentRequest]) (*connect.Response[experimentsv1.Experiment], error) {
	return h.experimentCall(ctx, req.Header(), h.legacy.UpdateExperiment, http.MethodPut, req.Msg.GetExperimentId(), req.Msg)
}

func (h *connectHandler) DeleteExperiment(ctx context.Context, req *connect.Request[experimentsv1.DeleteExperimentRequest]) (*connect.Response[experimentsv1.DeleteExperimentResponse], error) {
	_, err := transportbridge.Invoke(ctx, req.Header(), h.legacy.DeleteExperiment, http.MethodDelete, experimentPath(req.Msg.GetExperimentId(), ""), nil, map[string]string{"eid": req.Msg.GetExperimentId()})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&experimentsv1.DeleteExperimentResponse{}), nil
}

func (h *connectHandler) StartExperiment(ctx context.Context, req *connect.Request[experimentsv1.StartExperimentRequest]) (*connect.Response[experimentsv1.Experiment], error) {
	return h.experimentCall(ctx, req.Header(), h.legacy.StartExperiment, http.MethodPost, req.Msg.GetExperimentId(), nil, "start")
}

func (h *connectHandler) ConcludeExperiment(ctx context.Context, req *connect.Request[experimentsv1.ConcludeExperimentRequest]) (*connect.Response[experimentsv1.Experiment], error) {
	return h.experimentCall(ctx, req.Header(), h.legacy.ConcludeExperiment, http.MethodPost, req.Msg.GetExperimentId(), req.Msg, "conclude")
}

func (h *connectHandler) RecordOutcome(ctx context.Context, req *connect.Request[experimentsv1.RecordOutcomeRequest]) (*connect.Response[experimentsv1.ExperimentOutcome], error) {
	return h.outcomeCall(ctx, req.Header(), h.legacy.RecordOutcome, http.MethodPost, req.Msg.GetExperimentId(), req.Msg)
}

func (h *connectHandler) ListOutcomes(ctx context.Context, req *connect.Request[experimentsv1.ListOutcomesRequest]) (*connect.Response[experimentsv1.ListOutcomesResponse], error) {
	return transportbridge.InvokeWrappedJSON(ctx, req.Header(), h.legacy.ListOutcomes, http.MethodGet, experimentPath(req.Msg.GetExperimentId(), "outcomes"), nil, map[string]string{"eid": req.Msg.GetExperimentId()}, "outcomes", &experimentsv1.ListOutcomesResponse{})
}

func (h *connectHandler) AssignExperiment(ctx context.Context, req *connect.Request[experimentsv1.AssignExperimentRequest]) (*connect.Response[experimentsv1.ExperimentAssignment], error) {
	return transportbridge.InvokeProto(ctx, req.Header(), h.legacy.AssignExperiment, http.MethodPost, experimentPath(req.Msg.GetExperimentId(), "assignments"), req.Msg, &experimentsv1.ExperimentAssignment{})
}

func (h *connectHandler) RecordAuditReceipt(ctx context.Context, req *connect.Request[experimentsv1.RecordAuditReceiptRequest]) (*connect.Response[experimentsv1.Experiment], error) {
	return h.experimentCall(ctx, req.Header(), h.legacy.RecordAuditReceipt, http.MethodPost, req.Msg.GetExperimentId(), req.Msg, "audit-receipt")
}

func (h *connectHandler) RecordHoldoutReceipt(ctx context.Context, req *connect.Request[experimentsv1.RecordHoldoutReceiptRequest]) (*connect.Response[experimentsv1.Experiment], error) {
	return h.experimentCall(ctx, req.Header(), h.legacy.RecordHoldoutReceipt, http.MethodPost, req.Msg.GetExperimentId(), req.Msg, "holdout-receipt")
}

func (h *connectHandler) PromoteExperiment(ctx context.Context, req *connect.Request[experimentsv1.PromoteExperimentRequest]) (*connect.Response[experimentsv1.Experiment], error) {
	return h.experimentCall(ctx, req.Header(), h.legacy.PromoteExperiment, http.MethodPost, req.Msg.GetExperimentId(), req.Msg, "promote")
}

func (h *connectHandler) GetExperimentReport(ctx context.Context, req *connect.Request[experimentsv1.GetExperimentReportRequest]) (*connect.Response[experimentsv1.ExperimentReport], error) {
	return transportbridge.InvokeJSON(ctx, req.Header(), h.legacy.GetExperimentReport, http.MethodGet, experimentPath(req.Msg.GetExperimentId(), "report"), nil, map[string]string{"eid": req.Msg.GetExperimentId()}, &experimentsv1.ExperimentReport{})
}

func (h *connectHandler) experimentCall(ctx context.Context, headers http.Header, handler http.HandlerFunc, method, id string, body any, suffix ...string) (*connect.Response[experimentsv1.Experiment], error) {
	tail := ""
	if len(suffix) > 0 {
		tail = suffix[0]
	}
	return transportbridge.InvokeJSON(ctx, headers, handler, method, experimentPath(id, tail), body, map[string]string{"eid": id}, &experimentsv1.Experiment{})
}

func (h *connectHandler) outcomeCall(ctx context.Context, headers http.Header, handler http.HandlerFunc, method, id string, body any) (*connect.Response[experimentsv1.ExperimentOutcome], error) {
	return transportbridge.InvokeJSON(ctx, headers, handler, method, experimentPath(id, "outcomes"), body, map[string]string{"eid": id}, &experimentsv1.ExperimentOutcome{})
}

func experimentPath(id, suffix string) string {
	path := "/experiments/" + url.PathEscape(id)
	if suffix != "" {
		path += "/" + suffix
	}
	return path
}
