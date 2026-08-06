package inference

import (
	"context"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"

	internalinference "ai-gateway/internal/inference"
)

type connectHandler struct {
	service *internalinference.Service
}

func NewConnectHandler(service *internalinference.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) Run(ctx context.Context, req *connect.Request[inferencev1.RunRequest]) (*connect.Response[inferencev1.RunResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalinference.ErrUnavailable)
	}
	response := h.service.Run(ctx, internalinference.ProviderRequest{
		Source: req.Msg.GetSource(), SchemaJSON: req.Msg.GetSchemaJson(), Instruction: req.Msg.GetInstruction(), Role: req.Msg.GetRole(),
	})
	return connect.NewResponse(response), nil
}

func (h *connectHandler) RunBatch(ctx context.Context, req *connect.Request[inferencev1.RunBatchRequest]) (*connect.Response[inferencev1.RunBatchResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalinference.ErrUnavailable)
	}
	requests := make([]internalinference.ProviderRequest, 0, len(req.Msg.GetItems()))
	for _, item := range req.Msg.GetItems() {
		requests = append(requests, internalinference.ProviderRequest{
			Source: item.GetSource(), SchemaJSON: req.Msg.GetSchemaJson(), Instruction: req.Msg.GetInstruction(), Role: req.Msg.GetRole(),
		})
	}
	return connect.NewResponse(h.service.RunBatch(ctx, requests)), nil
}
