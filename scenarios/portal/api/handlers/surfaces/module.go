package surfaces

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/targetmodel"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	surfacesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces"
	surfacesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces/surfacesv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"portal/internal/module"
	"portal/internal/surfaces"
)

var Endpoints = []module.EndpointDescriptor{
	{ID: "surfaces_list", Path: surfacesconnect.SurfaceCatalogServiceListProcedure, Method: "POST", Summary: "List owner surfaces", Description: "Aggregate safe surface references and explicit partial-source status.", Category: "surfaces"},
	{ID: "surfaces_resolve", Path: surfacesconnect.SurfaceCatalogServiceResolveProcedure, Method: "POST", Summary: "Resolve a surface", Description: "Return exact or name-matched references without implicitly choosing ambiguous names or granting access.", Category: "surfaces"},
}

type Handler struct{ Catalog *surfaces.Catalog }

func Module(catalog *surfaces.Catalog) module.Module {
	path, handler := surfacesconnect.NewSurfaceCatalogServiceHandler(&Handler{Catalog: catalog}, connect.WithReadMaxBytes(16<<10))
	return module.Module{Name: "surfaces", Endpoints: Endpoints, Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }}
}

func project(snapshot surfaces.Snapshot) ([]*commonv1.SurfaceDescriptor, []*surfacesv1.SourceStatus, error) {
	descriptors := make([]*commonv1.SurfaceDescriptor, 0, len(snapshot.Surfaces))
	for _, surface := range snapshot.Surfaces {
		wire, err := surface.Proto()
		if err != nil {
			return nil, nil, err
		}
		descriptors = append(descriptors, wire)
	}
	sources := make([]*surfacesv1.SourceStatus, 0, len(snapshot.Sources))
	for _, source := range snapshot.Sources {
		sources = append(sources, &surfacesv1.SourceStatus{OwnerScenario: source.Owner, State: source.State, ReasonCode: source.ReasonCode})
	}
	return descriptors, sources, nil
}

func (h *Handler) List(ctx context.Context, req *connect.Request[surfacesv1.ListRequest]) (*connect.Response[surfacesv1.ListResponse], error) {
	if h.Catalog == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("surface catalog unavailable"))
	}
	// Optional Bridge providers need the same bearer token the caller used for
	// Portal. Keep it in request context only; descriptors never carry tokens.
	ctx = surfaces.WithBearerToken(ctx, bearerTokenFromHeader(req))
	snapshot := h.Catalog.List(ctx)
	descriptors, sources, err := project(snapshot)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid catalog projection"))
	}
	return connect.NewResponse(&surfacesv1.ListResponse{Surfaces: descriptors, Sources: sources, ObservedAt: timestamppb.New(snapshot.ObservedAt)}), nil
}

func (h *Handler) Resolve(ctx context.Context, req *connect.Request[surfacesv1.ResolveRequest]) (*connect.Response[surfacesv1.ResolveResponse], error) {
	if h.Catalog == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("surface catalog unavailable"))
	}
	var exact *targetmodel.SurfaceRef
	if req.Msg.ExactRef != nil {
		value, err := targetmodel.SurfaceRefFromProto(req.Msg.ExactRef)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid surface reference"))
		}
		exact = &value
	}
	// Validate input before reaching optional providers.
	if _, err := (surfaces.Snapshot{}).Resolve(req.Msg.DisplayLabel, exact); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	ctx = surfaces.WithBearerToken(ctx, bearerTokenFromHeader(req))
	snapshot := h.Catalog.List(ctx)
	snapshot.Surfaces, _ = snapshot.Resolve(req.Msg.DisplayLabel, exact)
	descriptors, sources, err := project(snapshot)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid catalog projection"))
	}
	return connect.NewResponse(&surfacesv1.ResolveResponse{Matches: descriptors, Sources: sources, ObservedAt: timestamppb.New(snapshot.ObservedAt)}), nil
}

type requestHeader interface{ Header() http.Header }

func bearerTokenFromHeader(req requestHeader) string {
	if req == nil {
		return ""
	}
	token, ok := strings.CutPrefix(req.Header().Get("Authorization"), "Bearer ")
	if !ok || token == "" || len(token) > 16384 {
		return ""
	}
	return token
}
