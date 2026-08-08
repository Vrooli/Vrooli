package recall

import (
	"connectrpc.com/connect"
	"context"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/recall/recall_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall"
	memoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/recall/recall_v1connect"
	"log"
	"vrooli-memory/internal/ledgerclient"
)

type connectHandler struct {
	client sourceconnect.RecallServiceClient
	logger *log.Logger
}

func NewConnectHandler(client sourceconnect.RecallServiceClient, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{client: client, logger: logger}
}
func (h *connectHandler) Recall(ctx context.Context, in *connect.Request[memoryv1.RecallRequest]) (*connect.Response[memoryv1.RecallResponse], error) {
	req := connect.NewRequest(&sourcev1.RecallRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.Recall(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("recall", err)
	}
	out := &memoryv1.RecallResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}
func (h *connectHandler) Wake(ctx context.Context, in *connect.Request[memoryv1.WakeRequest]) (*connect.Response[memoryv1.WakeResponse], error) {
	req := connect.NewRequest(&sourcev1.WakeRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.Wake(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("wake", err)
	}
	out := &memoryv1.WakeResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}
func (h *connectHandler) Zoom(ctx context.Context, in *connect.Request[memoryv1.ZoomRequest]) (*connect.Response[memoryv1.ZoomResponse], error) {
	req := connect.NewRequest(&sourcev1.ZoomRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.Zoom(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("zoom", err)
	}
	out := &memoryv1.ZoomResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}
func (h *connectHandler) ListSiblingEvents(ctx context.Context, in *connect.Request[memoryv1.ListSiblingEventsRequest]) (*connect.Response[memoryv1.ListSiblingEventsResponse], error) {
	req := connect.NewRequest(&sourcev1.ListSiblingEventsRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.ListSiblingEvents(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("list sibling events", err)
	}
	out := &memoryv1.ListSiblingEventsResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

var _ memoryconnect.RecallServiceHandler = (*connectHandler)(nil)
