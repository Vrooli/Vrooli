package scaffold

import (
	"context"
	"database/sql"
	"net/http"

	"backdrop-studio/internal/module"
	"backdrop-studio/internal/scaffold"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	scaffoldv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/scaffold"
	scaffoldconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/scaffold/scaffold_v1connect"
)

func Module(_ *sql.DB) module.Module {
	h := &handler{}
	return module.Module{Name: "scaffold", Mount: func(r *mux.Router) {
		path, svc := scaffoldconnect.NewScaffoldServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

type handler struct{}

func (*handler) ListPresets(context.Context, *connect.Request[scaffoldv1.ListPresetsRequest]) (*connect.Response[scaffoldv1.ListPresetsResponse], error) {
	resp := &scaffoldv1.ListPresetsResponse{}
	for _, p := range scaffold.ListPresets() {
		resp.Presets = append(resp.Presets, &scaffoldv1.Preset{Id: p.ID, Name: p.Name, Subject: p.Subject, Parameters: p.Parameters})
	}
	return connect.NewResponse(resp), nil
}

func (*handler) Render(ctx context.Context, req *connect.Request[scaffoldv1.RenderRequest]) (*connect.Response[scaffoldv1.RenderResponse], error) {
	var regions []scaffold.Region
	for _, r := range req.Msg.GetReservedRegions() {
		regions = append(regions, scaffold.Region{X: r.GetX(), Y: r.GetY(), Width: r.GetWidth(), Height: r.GetHeight()})
	}
	result, err := scaffold.Render(scaffold.Request{Preset: req.Msg.GetPreset(), Width: int(req.Msg.GetWidth()), Height: int(req.Msg.GetHeight()), Seed: req.Msg.GetSeed(), Conditioner: req.Msg.GetConditioner(), ParamsJSON: req.Msg.GetParamsJson(), Regions: regions})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&scaffoldv1.RenderResponse{ImagePng: result.PNG, Sha256: string(result.SHA256), Width: int32(result.Width), Height: int32(result.Height), Conditioner: result.Conditioner}), nil
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "scaffold_presets_list", Path: scaffoldconnect.ScaffoldServiceListPresetsProcedure, Method: http.MethodPost, Summary: "List procedural scaffold presets", Category: "scaffold"},
	{ID: "scaffold_render", Path: scaffoldconnect.ScaffoldServiceRenderProcedure, Method: http.MethodPost, Summary: "Render a seeded scaffold image", Category: "scaffold"},
}
