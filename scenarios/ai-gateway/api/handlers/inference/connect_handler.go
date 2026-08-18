package inference

import (
	"context"

	"connectrpc.com/connect"
	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
	"google.golang.org/protobuf/proto"

	internalinference "ai-gateway/internal/inference"
	"ai-gateway/internal/providers"
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
	ctx = providers.WithAccessToken(ctx, req.Header().Get("Authorization"))
	request := internalinference.ProviderRequest{
		Source: req.Msg.GetSource(), SchemaJSON: req.Msg.GetSchemaJson(), Instruction: req.Msg.GetInstruction(), Role: req.Msg.GetRole(), Profile: req.Msg.GetProfile(),
		Turns: req.Msg.GetTurns(), Attachments: req.Msg.GetAttachments(),
		MaxOutputTokens: req.Msg.GetMaxOutputTokens(),
	}
	// A present SamplingControls with no temperature is not the same as an
	// explicit 0, so read presence through the optional accessor rather than the
	// zero value.
	if sampling := req.Msg.GetSampling(); sampling != nil && sampling.Temperature != nil {
		request.Temperature = proto.Float64(sampling.GetTemperature())
	}
	response := h.service.Run(ctx, request)
	return connect.NewResponse(response), nil
}

func (h *connectHandler) RunBatch(ctx context.Context, req *connect.Request[inferencev1.RunBatchRequest]) (*connect.Response[inferencev1.RunBatchResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalinference.ErrUnavailable)
	}
	ctx = providers.WithAccessToken(ctx, req.Header().Get("Authorization"))
	requests := make([]internalinference.ProviderRequest, 0, len(req.Msg.GetItems()))
	for _, item := range req.Msg.GetItems() {
		requests = append(requests, internalinference.ProviderRequest{
			Source: item.GetSource(), SchemaJSON: req.Msg.GetSchemaJson(), Instruction: req.Msg.GetInstruction(), Role: req.Msg.GetRole(),
		})
	}
	return connect.NewResponse(h.service.RunBatch(ctx, requests)), nil
}

func (h *connectHandler) Embed(ctx context.Context, req *connect.Request[inferencev1.EmbedRequest]) (*connect.Response[inferencev1.EmbedResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, internalinference.ErrUnavailable)
	}
	ctx = providers.WithAccessToken(ctx, req.Header().Get("Authorization"))
	response := h.service.Embed(ctx, req.Msg.GetRole(), req.Msg.GetTexts(), req.Msg.GetSampling() != nil)
	return connect.NewResponse(response), nil
}
