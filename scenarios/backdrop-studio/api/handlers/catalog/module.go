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
		ParentId: v.ParentID, QualityTier: qualityTierToProto(v.EffectiveQualityTier()), PlateSpec: plateSpecToProto(v.PlateSpec),
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

// The quality tier crosses the wire as an enum and lives in the store as the
// string a seed file and an operator write. The two mappings are here, beside
// each other, because a tier that reached the store and not the wire would be
// the same defect as treatment_params and inks: a field that decides how a
// style is served, held by the catalog, invisible to every consumer.
var qualityTierProto = map[string]sharedv1.QualityTier{
	catalog.TierProcedural:    sharedv1.QualityTier_QUALITY_TIER_PROCEDURAL,
	catalog.TierLocalModel:    sharedv1.QualityTier_QUALITY_TIER_LOCAL_MODEL,
	catalog.TierFrontierModel: sharedv1.QualityTier_QUALITY_TIER_FRONTIER_MODEL,
}

func qualityTierToProto(tier string) sharedv1.QualityTier {
	if mapped, ok := qualityTierProto[tier]; ok {
		return mapped
	}
	return sharedv1.QualityTier_QUALITY_TIER_PROCEDURAL
}

// An unspecified tier reads back as procedural rather than as empty, because
// that is what a style carrying no tier actually is: drawn in-process, free and
// offline. Returning "" would make the store's own default the second place
// that decision is made.
func qualityTierFromProto(tier sharedv1.QualityTier) string {
	for name, mapped := range qualityTierProto {
		if mapped == tier {
			return name
		}
	}
	return catalog.TierProcedural
}

func fromProto(v *sharedv1.Style) catalog.Style {
	out := catalog.Style{
		ID: v.GetId(), Name: v.GetName(), Version: int(v.GetVersion()), Role: v.GetRole(),
		Subject: v.GetSubject(), Lineage: v.GetLineage(), Treatments: v.GetTreatments(),
		Placements: v.GetPlacements(), Strategy: v.GetStrategy(),
		ContrastThreshold: v.GetContrastThreshold(), TreatmentParams: v.GetTreatmentParams(),
		Inks: v.GetInks(), ParentID: v.GetParentId(),
		QualityTier: qualityTierFromProto(v.GetQualityTier()), PlateSpec: plateSpecFromProto(v.GetPlateSpec()),
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

// plateSpecToProto and plateSpecFromProto carry a style's declared depth stack
// across the wire. An absent spec stays absent rather than becoming a
// one-element list: the default is materialised at render time by
// EffectivePlateSpec, and inflating it here would make every style that never
// declared a stack look like one that did.
func plateSpecToProto(spec []catalog.PlateSpec) []*sharedv1.PlateSpec {
	if len(spec) == 0 {
		return nil
	}
	out := make([]*sharedv1.PlateSpec, 0, len(spec))
	for _, plate := range spec {
		out = append(out, &sharedv1.PlateSpec{
			Name:            plate.Name,
			Depth:           int32(plate.Depth),
			Blend:           plate.Blend,
			Planes:          append([]string(nil), plate.Planes...),
			Opacity:         plate.Opacity,
			Treatments:      append([]string(nil), plate.Treatments...),
			TreatmentParams: plate.TreatmentParams,
		})
	}
	return out
}

func plateSpecFromProto(spec []*sharedv1.PlateSpec) []catalog.PlateSpec {
	if len(spec) == 0 {
		return nil
	}
	out := make([]catalog.PlateSpec, 0, len(spec))
	for _, plate := range spec {
		out = append(out, catalog.PlateSpec{
			Name:            plate.GetName(),
			Depth:           int(plate.GetDepth()),
			Blend:           plate.GetBlend(),
			Planes:          append([]string(nil), plate.GetPlanes()...),
			Opacity:         plate.GetOpacity(),
			Treatments:      append([]string(nil), plate.GetTreatments()...),
			TreatmentParams: plate.GetTreatmentParams(),
		})
	}
	return out
}
