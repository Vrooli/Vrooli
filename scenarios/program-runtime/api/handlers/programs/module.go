package programs

import (
	"context"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	programsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs"
	programsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/programs/programs_v1connect"
	"program-runtime/internal/module"
	internalprograms "program-runtime/internal/programs"
)

type handler struct {
	programsconnect.UnimplementedProgramServiceHandler
	service *internalprograms.Service
}

func Module(service *internalprograms.Service) module.Module {
	return module.Module{Name: "programs", Mount: func(r *mux.Router) {
		path, h := programsconnect.NewProgramServiceHandler(&handler{service: service})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func (h *handler) SubmitProgram(ctx context.Context, req *connect.Request[programsv1.SubmitProgramRequest]) (*connect.Response[programsv1.SubmitProgramResponse], error) {
	p, err := h.service.Submit(ctx, req.Msg.SessionId, req.Msg.Source, req.Msg.Provenance, req.Msg.IncludeMaterialized)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&programsv1.SubmitProgramResponse{Program: p}), nil
}

func (h *handler) GetProgram(ctx context.Context, req *connect.Request[programsv1.GetProgramRequest]) (*connect.Response[programsv1.GetProgramResponse], error) {
	p, err := h.service.Get(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&programsv1.GetProgramResponse{Program: p}), nil
}

func (h *handler) ListPrograms(ctx context.Context, req *connect.Request[programsv1.ListProgramsRequest]) (*connect.Response[programsv1.ListProgramsResponse], error) {
	return connect.NewResponse(&programsv1.ListProgramsResponse{Programs: h.service.List(ctx, req.Msg.SessionId, req.Msg.IncludeOperator)}), nil
}

func (h *handler) MineFailures(ctx context.Context, req *connect.Request[programsv1.MineFailuresRequest]) (*connect.Response[programsv1.MineFailuresResponse], error) {
	shapes := h.service.MineFailures(ctx, req.Msg.IncludeOperator)
	return connect.NewResponse(&programsv1.MineFailuresResponse{Shapes: shapes, Count: int64(len(shapes))}), nil
}
