package release

import (
	"context"
	"fmt"
	"net/http"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/module"
	internal "backdrop-studio/internal/release"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/release/release_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/shared"
)

type handler struct{ store *internal.Store }

func Module(store *internal.Store) module.Module {
	h := &handler{store: store}
	return module.Module{Name: "release", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/backdrops/{id}/asset", h.asset).Methods(http.MethodGet)
		path, svc := connectv1.NewReleaseServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) Release(_ context.Context, req *connect.Request[v1.ReleaseRequest]) (*connect.Response[v1.ReleasedBackdrop], error) {
	r := req.Msg
	regions := make([]catalog.Region, 0, len(r.GetReservedRegions()))
	for _, x := range r.GetReservedRegions() {
		regions = append(regions, catalog.Region{X: x.GetX(), Y: x.GetY(), Width: x.GetWidth(), Height: x.GetHeight(), Kind: x.GetKind(), TextColor: x.GetTextColor()})
	}
	b, err := h.store.Release(internal.Request{CandidateID: r.GetCandidateId(), StyleID: r.GetStyleId(), Strategy: r.GetStrategy(), SurfaceID: r.GetSurfaceId(), Placement: r.GetPlacement(), AltText: r.GetAltText(), Width: int(r.GetWidth()), Height: int(r.GetHeight()), ExpectedWidth: int(r.GetExpectedWidth()), ExpectedHeight: int(r.GetExpectedHeight()), Decorative: r.GetDecorative(), AIGeneratedSet: r.GetAiGeneratedSet(), AIGenerated: r.GetAiGenerated(), LegibilityPasses: r.GetLegibilityPasses(), ContrastRatio: r.GetContrastRatio(), ContrastThreshold: r.GetContrastThreshold(), Regions: regions, ImagePNG: r.GetImagePng()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(toProto(b)), nil
}

func (h *handler) asset(w http.ResponseWriter, req *http.Request) {
	b, err := h.store.Get(mux.Vars(req)["id"])
	if err != nil || len(b.ImagePNG) == 0 {
		http.Error(w, "backdrop asset not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(b.ImagePNG)
}

func (h *handler) GetReference(_ context.Context, req *connect.Request[v1.GetReferenceRequest]) (*connect.Response[v1.ReleasedBackdrop], error) {
	b, err := h.store.Get(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(toProto(b)), nil
}

func toProto(b internal.Backdrop) *v1.ReleasedBackdrop {
	out := &v1.ReleasedBackdrop{Id: b.ID, CandidateId: b.CandidateID, StyleId: b.StyleID, SurfaceId: b.SurfaceID, Placement: b.Placement, Width: int32(b.Width), Height: int32(b.Height), AltText: b.AltText, Decorative: b.Decorative, AiGenerated: b.AIGenerated, ContrastRatio: b.ContrastRatio, ContrastThreshold: b.ContrastThreshold, Uri: fmt.Sprintf("/api/v1/backdrops/%s/asset", b.ID), AssetStudioRef: b.AssetStudioRef}
	for _, r := range b.Regions {
		out.ReservedRegions = append(out.ReservedRegions, &sharedv1.ReservedRegion{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height, Kind: r.Kind, TextColor: r.TextColor})
	}
	return out
}

var Endpoints = []module.EndpointDescriptor{{ID: "release_create", Path: connectv1.ReleaseServiceReleaseProcedure, Method: http.MethodPost, Summary: "Release a backdrop", Category: "release"}, {ID: "release_reference_get", Path: connectv1.ReleaseServiceGetReferenceProcedure, Method: http.MethodPost, Summary: "Get released backdrop metadata", Category: "release"}}
