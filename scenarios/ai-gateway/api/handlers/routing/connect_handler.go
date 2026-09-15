package routing

import (
	"context"
	"database/sql"
	"strings"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"

	"ai-gateway/internal/providers"
	internalrouting "ai-gateway/internal/routing"
)

type Deps struct {
	DB            *sql.DB
	Adapters      []providers.Adapter
	Service       *internalrouting.Service
	MediaExecutor internalrouting.MediaExecutor
}

type connectHandler struct {
	service *internalrouting.Service
	media   *internalrouting.MediaService
}

func NewConnectHandler(d Deps) *connectHandler {
	if d.Service == nil {
		adapters := d.Adapters
		if len(adapters) == 0 {
			adapters = providers.NewDefaultAdapters(nil)
		}
		d.Service = internalrouting.NewSQLService(d.DB, adapters)
	}
	return &connectHandler{service: d.Service, media: internalrouting.NewMediaService(d.DB, d.MediaExecutor)}
}

// RecoverMedia resumes durable, non-terminal receipts after schema setup and
// before the handler accepts new requests. It is called once from the module
// lifecycle rather than racing every handler construction.
func (h *connectHandler) RecoverMedia(ctx context.Context) {
	if h != nil && h.media != nil {
		h.media.Recover(ctx)
	}
}

func (h *connectHandler) PreviewRoute(ctx context.Context, req *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error) {
	resp, err := h.service.Preview(ctx, req.Msg.GetRequest())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) ExecuteRoute(ctx context.Context, req *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error) {
	resp, err := h.service.Execute(ctx, req.Msg.GetRequest(), req.Msg.GetInputText())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(resp), nil
}

// Media execution is intentionally a separate asynchronous contract from text
// routing. The receipt store and executor wiring land with the media runtime;
// until then, fail explicitly rather than manufacturing a successful render.
func (h *connectHandler) SubmitMedia(ctx context.Context, req *connect.Request[routingv1.SubmitMediaRequest]) (*connect.Response[routingv1.SubmitMediaResponse], error) {
	execution, err := h.media.Submit(ctx, req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&routingv1.SubmitMediaResponse{Execution: execution}), nil
}

func (h *connectHandler) GetMediaExecution(ctx context.Context, req *connect.Request[routingv1.GetMediaExecutionRequest]) (*connect.Response[routingv1.GetMediaExecutionResponse], error) {
	execution, err := h.media.Get(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&routingv1.GetMediaExecutionResponse{Execution: execution}), nil
}

func (h *connectHandler) WaitMediaExecution(ctx context.Context, req *connect.Request[routingv1.WaitMediaExecutionRequest]) (*connect.Response[routingv1.WaitMediaExecutionResponse], error) {
	execution, err := h.media.Wait(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeDeadlineExceeded, err)
	}
	return connect.NewResponse(&routingv1.WaitMediaExecutionResponse{Execution: execution}), nil
}

func (h *connectHandler) CancelMediaExecution(ctx context.Context, req *connect.Request[routingv1.CancelMediaExecutionRequest]) (*connect.Response[routingv1.CancelMediaExecutionResponse], error) {
	execution, err := h.media.Cancel(ctx, req.Msg.GetExecutionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&routingv1.CancelMediaExecutionResponse{Execution: execution}), nil
}

func (h *connectHandler) RetryMediaExecution(ctx context.Context, req *connect.Request[routingv1.RetryMediaExecutionRequest]) (*connect.Response[routingv1.RetryMediaExecutionResponse], error) {
	execution, err := h.media.Retry(ctx, req.Msg.GetExecutionId(), req.Msg.GetIdempotencyKey())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&routingv1.RetryMediaExecutionResponse{Execution: execution}), nil
}

func (h *connectHandler) ListRouteEvidence(ctx context.Context, req *connect.Request[routingv1.ListRouteEvidenceRequest]) (*connect.Response[routingv1.ListRouteEvidenceResponse], error) {
	events, err := h.service.ListEvidence(ctx, internalrouting.EvidenceFilter{
		Scenario: strings.TrimSpace(req.Msg.GetScenario()),
		Limit:    int(req.Msg.GetLimit()),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&routingv1.ListRouteEvidenceResponse{Events: events}), nil
}

func (h *connectHandler) GetRouteEvidence(ctx context.Context, req *connect.Request[routingv1.GetRouteEvidenceRequest]) (*connect.Response[routingv1.GetRouteEvidenceResponse], error) {
	event, err := h.service.GetEvidence(ctx, req.Msg.GetEventId())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&routingv1.GetRouteEvidenceResponse{Event: event}), nil
}

func (h *connectHandler) ListProviderHealth(ctx context.Context, _ *connect.Request[routingv1.ListProviderHealthRequest]) (*connect.Response[routingv1.ListProviderHealthResponse], error) {
	items, err := h.service.ListProviderHealth(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&routingv1.ListProviderHealthResponse{Items: items}), nil
}
