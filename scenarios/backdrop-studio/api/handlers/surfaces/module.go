package surfaces

import (
	"context"
	"database/sql"
	"net/http"

	"backdrop-studio/internal/catalog"
	"backdrop-studio/internal/module"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	surfacesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/surfaces"
	surfacesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/backdrop-studio/v1/surfaces/surfaces_v1connect"
)

type handler struct{ store *catalog.Store }

func Module(db *sql.DB) module.Module {
	h := &handler{store: catalog.NewStore(db)}
	return module.Module{Name: "surfaces", Mount: func(r *mux.Router) {
		path, svc := surfacesconnect.NewSurfacesServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) ListSurfaces(ctx context.Context, _ *connect.Request[surfacesv1.ListSurfacesRequest]) (*connect.Response[surfacesv1.ListSurfacesResponse], error) {
	items, err := h.store.ListSurfaces(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp := &surfacesv1.ListSurfacesResponse{}
	for _, v := range items {
		resp.Surfaces = append(resp.Surfaces, &surfacesv1.Surface{Id: v.ID, Name: v.Name, Kind: v.Kind, Width: int32(v.Width), Height: int32(v.Height), Placements: v.Placements, Authority: v.Authority, ConfirmedOn: v.ConfirmedOn})
	}
	return connect.NewResponse(resp), nil
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "surfaces_list", Path: surfacesconnect.SurfacesServiceListSurfacesProcedure, Method: http.MethodPost, Summary: "List declared output surfaces", Category: "surfaces", Description: "Returns declared pixel geometries and permitted placements for Backdrop Studio."},
}
