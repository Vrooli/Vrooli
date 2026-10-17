package shapes

import (
	"context"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	shapesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shapes"
	shapesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shapes/shapes_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
	"program-runtime/internal/module"
	internal "program-runtime/internal/shapes"
)

type handler struct {
	shapesconnect.UnimplementedShapeServiceHandler
	repo *internal.Repository
}

func Module(repo *internal.Repository) module.Module {
	return module.Module{Name: "shapes", Mount: func(r *mux.Router) {
		path, h := shapesconnect.NewShapeServiceHandler(&handler{repo: repo})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func (h *handler) ListShapes(ctx context.Context, req *connect.Request[shapesv1.ListShapesRequest]) (*connect.Response[shapesv1.ListShapesResponse], error) {
	rows, err := h.repo.List(ctx, internal.Filter{MinOccurrences: req.Msg.GetMinOccurrences(), MinSessions: req.Msg.GetMinSessions(), UncoveredOnly: req.Msg.GetUncoveredOnly(), State: req.Msg.GetState()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	limit := int(req.Msg.GetLimit())
	if limit > 0 && len(rows) > limit {
		rows = rows[:limit]
	}
	response := &shapesv1.ListShapesResponse{}
	for _, row := range rows {
		response.Shapes = append(response.Shapes, toProto(row))
	}
	for _, row := range rows {
		switch row.State {
		case "observed":
			response.Observed++
		case "nominated":
			response.Nominated++
		case "covered":
			response.Covered++
		}
	}
	return connect.NewResponse(response), nil
}

func (h *handler) GetShape(ctx context.Context, req *connect.Request[shapesv1.GetShapeRequest]) (*connect.Response[shapesv1.GetShapeResponse], error) {
	row, err := h.repo.Get(ctx, strings.TrimSpace(req.Msg.GetShapeKey()))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&shapesv1.GetShapeResponse{Shape: toProto(row)}), nil
}

func (h *handler) ExpireShapes(ctx context.Context, req *connect.Request[shapesv1.ExpireShapesRequest]) (*connect.Response[shapesv1.ExpireShapesResponse], error) {
	window := internal.ShapeWindow
	if req.Msg.GetWindowSeconds() > 0 {
		window = time.Duration(req.Msg.GetWindowSeconds()) * time.Second
	}
	deleted, err := h.repo.Expire(ctx, time.Now().UTC(), window)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&shapesv1.ExpireShapesResponse{Deleted: deleted}), nil
}

func toProto(row internal.Shape) *sharedv1.ProgramShape {
	return &sharedv1.ProgramShape{ShapeKey: row.ShapeKey, BindingIds: row.BindingIDs, BindingCount: row.BindingCount, Occurrences: row.Occurrences, AgentRuns: row.AgentRuns, OperatorRuns: row.OperatorRuns, TestRuns: row.TestRuns, ReplayRuns: row.ReplayRuns, Sessions: row.Sessions, FirstSeen: row.FirstSeen, LastSeen: row.LastSeen, ExemplarProgramId: row.ExemplarProgramID, ExemplarBytes: row.ExemplarBytes, CoveredBy: row.CoveredBy, CoveredReason: row.CoveredReason, State: row.State, DominantScenario: row.DominantScenario}
}
