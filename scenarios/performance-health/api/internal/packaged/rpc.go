package packaged

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	packagedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/packaged"
	packagedv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/packaged/packaged_v1connect"
)

type RPCHandler struct{ service *Service }

type reportQuery struct {
	Scenario   string `json:"scenario"`
	Ramp       string `json:"ramp"`
	Strategy   string `json:"strategy"`
	Condition  string `json:"condition"`
	Experiment string `json:"experiment_id"`
}

var _ packagedv1connect.PackagedServiceHandler = (*RPCHandler)(nil)

func NewRPCHandler(service *Service) (string, http.Handler) {
	return packagedv1connect.NewPackagedServiceHandler(&RPCHandler{service: service})
}

func (h *RPCHandler) ValidateExperiment(_ context.Context, req *connect.Request[packagedv1.ExperimentRequest]) (*connect.Response[packagedv1.ExperimentResponse], error) {
	var manifest ExperimentManifest
	if err := json.Unmarshal([]byte(req.Msg.GetManifestJson()), &manifest); err != nil {
		return connect.NewResponse(&packagedv1.ExperimentResponse{Error: "invalid packaged-experiment.v1 manifest: " + err.Error()}), nil
	}
	if err := manifest.Validate(); err != nil {
		return connect.NewResponse(&packagedv1.ExperimentResponse{Error: err.Error()}), nil
	}
	return connect.NewResponse(&packagedv1.ExperimentResponse{Valid: true}), nil
}

func (h *RPCHandler) Capture(ctx context.Context, req *connect.Request[packagedv1.CaptureRequest]) (*connect.Response[packagedv1.CaptureResponse], error) {
	var input CaptureRequest
	if err := json.Unmarshal([]byte(req.Msg.GetRequestJson()), &input); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	samples, err := h.service.Capture(ctx, input)
	if err != nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, err)
	}
	experimentID := input.Experiment
	if experimentID == "" && len(samples) > 0 {
		experimentID = samples[0].ExperimentID
	}
	result, err := json.Marshal(map[string]any{"experiment_id": experimentID, "samples": samples})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&packagedv1.CaptureResponse{ResultJson: string(result)}), nil
}

func (h *RPCHandler) Report(ctx context.Context, req *connect.Request[packagedv1.ReportRequest]) (*connect.Response[packagedv1.ReportResponse], error) {
	var query reportQuery
	if err := json.Unmarshal([]byte(req.Msg.GetQueryJson()), &query); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	report, err := h.service.Report(ctx, query.Scenario, query.Ramp, query.Strategy, query.Condition, query.Experiment)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	data, err := json.Marshal(report)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	markdown := ""
	if strings.TrimSpace(query.Experiment) != "" {
		markdown = RenderMarkdown(report)
	}
	return connect.NewResponse(&packagedv1.ReportResponse{ReportJson: string(data), Markdown: markdown}), nil
}
