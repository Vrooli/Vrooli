package legibility

import (
	"context"
	"net/http"

	internal "backdrop-studio/internal/legibility"
	"backdrop-studio/internal/module"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/legibility"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/legibility/legibility_v1connect"
)

type handler struct{}

func Module() module.Module {
	h := &handler{}
	return module.Module{Name: "legibility", Mount: func(r *mux.Router) {
		path, svc := connectv1.NewLegibilityServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}
func (*handler) Measure(_ context.Context, req *connect.Request[v1.MeasureRequest]) (*connect.Response[v1.Verdict], error) {
	regions := make([]internal.Region, 0, len(req.Msg.GetRegions()))
	for _, r := range req.Msg.GetRegions() {
		regions = append(regions, internal.Region{X: r.GetX(), Y: r.GetY(), Width: r.GetWidth(), Height: r.GetHeight(), Kind: r.GetKind(), TextColor: r.GetTextColor()})
	}
	verdict, err := internal.Measure(req.Msg.GetImagePng(), regions, req.Msg.GetThreshold(), req.Msg.GetPlacement())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	out := &v1.Verdict{Passes: verdict.Passes, MinimumRatio: verdict.MinimumRatio, Threshold: verdict.Threshold, Placement: verdict.Placement}
	for _, r := range verdict.Regions {
		out.Regions = append(out.Regions, &v1.RegionVerdict{RegionIndex: int32(r.Index), MinimumRatio: r.MinimumRatio, Passes: r.Passes})
	}
	for _, a := range verdict.Amendments {
		out.Amendments = append(out.Amendments, &v1.Amendment{Kind: a.Kind, Description: a.Description, Value: a.Value})
	}
	return connect.NewResponse(out), nil
}

var Endpoints = []module.EndpointDescriptor{{ID: "legibility_measure", Path: connectv1.LegibilityServiceMeasureProcedure, Method: http.MethodPost, Summary: "Measure worst-pixel contrast", Category: "legibility"}}
