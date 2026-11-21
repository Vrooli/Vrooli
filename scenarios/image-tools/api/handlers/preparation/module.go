package preparation

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation"
	pc "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/preparation/preparation_v1connect"
	"image-tools/internal/module"
	"image-tools/internal/sceneprep"
)

type Handler struct{ Service *sceneprep.Service }

func Module(db *database.RoutedDB, blobs blobstore.BlobStore) module.Module {
	h := &Handler{Service: &sceneprep.Service{Repository: sceneprep.NewSQLRepository(db), Blobs: blobs}}
	path, handler := pc.NewPreparationServiceHandler(h)
	return module.Module{Name: "preparation", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
func Schema() string { return sceneprep.Schema() }
func (h *Handler) AssessRepair(ctx context.Context, req *connect.Request[v1.AssessRepairRequest]) (*connect.Response[v1.AssessRepairResponse], error) {
	a, err := h.Service.AssessRepair(ctx, req.Msg)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&v1.AssessRepairResponse{Assessment: a}), nil
}
func (h *Handler) CreatePhotograph(ctx context.Context, req *connect.Request[v1.CreatePhotographRequest]) (*connect.Response[v1.PhotographResponse], error) {
	p, err := h.Service.Create(ctx, req.Msg)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&v1.PhotographResponse{Photograph: p}), nil
}
func (h *Handler) GetPhotograph(ctx context.Context, req *connect.Request[v1.GetPhotographRequest]) (*connect.Response[v1.PhotographResponse], error) {
	p, err := h.Service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&v1.PhotographResponse{Photograph: p}), nil
}
func (h *Handler) ListPhotographs(ctx context.Context, req *connect.Request[v1.ListPhotographsRequest]) (*connect.Response[v1.ListPhotographsResponse], error) {
	p, err := h.Service.List(ctx, req.Msg.Limit)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&v1.ListPhotographsResponse{Photographs: p}), nil
}
func (h *Handler) ReviewPhotograph(ctx context.Context, req *connect.Request[v1.ReviewPhotographRequest]) (*connect.Response[v1.PhotographResponse], error) {
	p, err := h.Service.Review(ctx, req.Msg)
	if err != nil {
		return nil, mapError(err)
	}
	return connect.NewResponse(&v1.PhotographResponse{Photograph: p}), nil
}
func mapError(err error) error {
	code := connect.CodeInternal
	if errors.Is(err, sceneprep.ErrInvalid) {
		code = connect.CodeInvalidArgument
	} else if errors.Is(err, sceneprep.ErrNotFound) {
		code = connect.CodeNotFound
	}
	return connect.NewError(code, err)
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "preparation_assess_repair", Path: pc.PreparationServiceAssessRepairProcedure, Method: "POST", Summary: "Measure removal-mask context before repair inference", Category: "preparation"},
	{ID: "preparation_create", Path: pc.PreparationServiceCreatePhotographProcedure, Method: "POST", Summary: "Assemble an immutable photograph preparation", Category: "preparation"},
	{ID: "preparation_get", Path: pc.PreparationServiceGetPhotographProcedure, Method: "POST", Summary: "Inspect a retained photograph preparation", Category: "preparation"},
	{ID: "preparation_list", Path: pc.PreparationServiceListPhotographsProcedure, Method: "POST", Summary: "List retained photograph preparations", Category: "preparation"},
	{ID: "preparation_review", Path: pc.PreparationServiceReviewPhotographProcedure, Method: "POST", Summary: "Retain a visual review as a new immutable revision", Category: "preparation"},
}
