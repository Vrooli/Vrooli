package optimization

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	optimizationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/optimization"
	optimizationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/optimization/optimization_v1connect"
	"program-runtime/internal/module"
	internaloptimization "program-runtime/internal/optimization"
)

type handler struct {
	optimizationconnect.UnimplementedOptimizationServiceHandler
	service *internaloptimization.Service
}

func Module(service *internaloptimization.Service) module.Module {
	return module.Module{Name: "optimization", Mount: func(r *mux.Router) {
		path, h := optimizationconnect.NewOptimizationServiceHandler(&handler{service: service})
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func (h *handler) Submit(ctx context.Context, req *connect.Request[optimizationv1.SubmitOptimizationRequest]) (*connect.Response[optimizationv1.SubmitOptimizationResponse], error) {
	record, err := h.service.Submit(ctx, req.Msg.GetRequestJson())
	if err != nil {
		switch {
		case errors.Is(err, internaloptimization.ErrInvalidRequest):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, internaloptimization.ErrQueueFull):
			return nil, connect.NewError(connect.CodeResourceExhausted, err)
		case errors.Is(err, internaloptimization.ErrNoAdapter):
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		default:
			return nil, connect.NewError(connect.CodeUnavailable, err)
		}
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&optimizationv1.SubmitOptimizationResponse{RunId: record.Request.RunID, RecordJson: string(encoded)}), nil
}

func (h *handler) Wait(ctx context.Context, req *connect.Request[optimizationv1.WaitOptimizationRequest]) (*connect.Response[optimizationv1.WaitOptimizationResponse], error) {
	if req.Msg.GetWaitMs() < 1 || req.Msg.GetWaitMs() > 60000 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("wait_ms must be 1..60000"))
	}
	started := time.Now()
	record, terminal, err := h.service.Wait(ctx, req.Msg.GetRunId(), time.Duration(req.Msg.GetWaitMs())*time.Millisecond)
	if errors.Is(err, internaloptimization.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if errors.Is(err, internaloptimization.ErrInvalidRequest) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&optimizationv1.WaitOptimizationResponse{RecordJson: string(encoded), Terminal: terminal, WaitedMs: uint32(time.Since(started).Milliseconds())}), nil
}
