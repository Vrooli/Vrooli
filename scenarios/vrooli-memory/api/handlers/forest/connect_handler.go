package forest

import (
	"context"
	"log"

	"connectrpc.com/connect"

	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest/forest_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest"
	memoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest/forest_v1connect"

	"vrooli-memory/internal/ledgerclient"
)

type connectHandler struct {
	client sourceconnect.ForestServiceClient
	logger *log.Logger
}

func NewConnectHandler(client sourceconnect.ForestServiceClient, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{client: client, logger: logger}
}

func (h *connectHandler) RunCompactionPass(ctx context.Context, in *connect.Request[memoryv1.RunCompactionPassRequest]) (*connect.Response[memoryv1.RunCompactionPassResponse], error) {
	req := connect.NewRequest(&sourcev1.RunCompactionPassRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.RunCompactionPass(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("run compaction pass", err)
	}
	out := &memoryv1.RunCompactionPassResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetFrontier(ctx context.Context, in *connect.Request[memoryv1.GetFrontierRequest]) (*connect.Response[memoryv1.GetFrontierResponse], error) {
	req := connect.NewRequest(&sourcev1.GetFrontierRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.GetFrontier(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("get frontier", err)
	}
	out := &memoryv1.GetFrontierResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetNode(ctx context.Context, in *connect.Request[memoryv1.GetNodeRequest]) (*connect.Response[memoryv1.GetNodeResponse], error) {
	req := connect.NewRequest(&sourcev1.GetNodeRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.GetNode(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("get node", err)
	}
	out := &memoryv1.GetNodeResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) RebuildForest(ctx context.Context, in *connect.Request[memoryv1.RebuildForestRequest]) (*connect.Response[memoryv1.RebuildForestResponse], error) {
	req := connect.NewRequest(&sourcev1.RebuildForestRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.RebuildForest(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("rebuild forest", err)
	}
	out := &memoryv1.RebuildForestResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

var _ memoryconnect.ForestServiceHandler = (*connectHandler)(nil)
