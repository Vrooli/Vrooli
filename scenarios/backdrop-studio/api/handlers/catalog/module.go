package catalog

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/catalog"
	catalogconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/catalog/catalog_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/shared"
)

type (
	Deps    struct{ DB *sql.DB }
	handler struct{ store *catalog.Store }
)

func Module(db *sql.DB) module.Module {
	h := &handler{store: catalog.NewStore(db)}
	return module.Module{Name: "catalog", Mount: func(r *mux.Router) {
		path, svc := catalogconnect.NewCatalogServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) ListStyles(ctx context.Context, req *connect.Request[catalogv1.ListStylesRequest]) (*connect.Response[catalogv1.ListStylesResponse], error) {
	q := req.Msg
	items, err := h.store.ListStyles(ctx, q.GetRole(), q.GetSubject(), q.GetTreatment(), q.GetLineage(), q.GetPlacement())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &catalogv1.ListStylesResponse{}
	for _, v := range items {
		resp.Styles = append(resp.Styles, toProto(v))
	}
	return connect.NewResponse(resp), nil
}

func (h *handler) CreateStyle(ctx context.Context, req *connect.Request[catalogv1.CreateStyleRequest]) (*connect.Response[sharedv1.Style], error) {
	if req.Msg.GetStyle() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("catalog: style is required"))
	}
	v := fromProto(req.Msg.GetStyle())
	if err := h.store.CreateStyle(ctx, v); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(toProto(v)), nil
}

func toProto(v catalog.Style) *sharedv1.Style {
	// TreatmentParams, Inks and ParentID are the three fields that decide what a
	// style looks like and where it came from, and all three used to stop at
	// this boundary: the store held them, the wire did not carry them, so the
	// studio could show a chain without its parameters and offer a fork it
	// could not record.
	out := &sharedv1.Style{
		Id: v.ID, Name: v.Name, Version: int32(v.Version), Role: v.Role, Subject: v.Subject,
		Lineage: v.Lineage, Treatments: v.Treatments, Placements: v.Placements, Strategy: v.Strategy,
		ContrastThreshold: v.ContrastThreshold, TreatmentParams: v.TreatmentParams, Inks: v.Inks,
		ParentId: v.ParentID,
	}
	if v.Scaffold != nil {
		out.Scaffold = &sharedv1.ScaffoldBinding{Preset: v.Scaffold.Preset, Conditioner: v.Scaffold.Conditioner, ParamsJson: v.Scaffold.ParamsJSON}
	}
	if v.Generation != nil {
		out.Generation = &sharedv1.GenerationBlock{PromptTemplate: v.Generation.PromptTemplate, Negative: v.Generation.Negative, Model: v.Generation.Model, ProviderUrl: v.Generation.ProviderURL, Credential: v.Generation.Credential}
	}
	for _, r := range v.Regions {
		out.Regions = append(out.Regions, &sharedv1.ReservedRegion{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height, Kind: r.Kind, TextColor: r.TextColor})
	}
	return out
}

func fromProto(v *sharedv1.Style) catalog.Style {
	out := catalog.Style{
		ID: v.GetId(), Name: v.GetName(), Version: int(v.GetVersion()), Role: v.GetRole(),
		Subject: v.GetSubject(), Lineage: v.GetLineage(), Treatments: v.GetTreatments(),
		Placements: v.GetPlacements(), Strategy: v.GetStrategy(),
		ContrastThreshold: v.GetContrastThreshold(), TreatmentParams: v.GetTreatmentParams(),
		Inks: v.GetInks(), ParentID: v.GetParentId(),
	}
	if v.GetScaffold() != nil {
		out.Scaffold = &catalog.ScaffoldBinding{Preset: v.GetScaffold().GetPreset(), Conditioner: v.GetScaffold().GetConditioner(), ParamsJSON: v.GetScaffold().GetParamsJson()}
	}
	if v.GetGeneration() != nil {
		out.Generation = &catalog.GenerationBlock{PromptTemplate: v.GetGeneration().GetPromptTemplate(), Negative: v.GetGeneration().GetNegative(), Model: v.GetGeneration().GetModel(), ProviderURL: v.GetGeneration().GetProviderUrl(), Credential: v.GetGeneration().GetCredential()}
	}
	for _, r := range v.GetRegions() {
		out.Regions = append(out.Regions, catalog.Region{X: r.GetX(), Y: r.GetY(), Width: r.GetWidth(), Height: r.GetHeight(), Kind: r.GetKind(), TextColor: r.GetTextColor()})
	}
	return out
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "catalog_styles_list", Path: catalogconnect.CatalogServiceListStylesProcedure, Method: http.MethodPost, Summary: "List styles with optional axis filters", Category: "catalog"},
	{ID: "catalog_styles_create", Path: catalogconnect.CatalogServiceCreateStyleProcedure, Method: http.MethodPost, Summary: "Create a validated style version", Category: "catalog"},
}
