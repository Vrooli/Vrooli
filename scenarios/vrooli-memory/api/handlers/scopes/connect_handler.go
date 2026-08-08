package scopes

import (
	"connectrpc.com/connect"
	"context"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/scopes/scopesv1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/scopes"
	memoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/scopes/scopesv1connect"
	"google.golang.org/protobuf/proto"
	"log"
	"vrooli-memory/internal/ledgerclient"
)

type connectHandler struct {
	client sourceconnect.ScopesServiceClient
	logger *log.Logger
}

func NewConnectHandler(client sourceconnect.ScopesServiceClient, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{client: client, logger: logger}
}
func (h *connectHandler) CreateScope(ctx context.Context, in *connect.Request[memoryv1.CreateScopeRequest]) (*connect.Response[memoryv1.CreateScopeResponse], error) {
	return proxy(ctx, in, &sourcev1.CreateScopeRequest{}, &memoryv1.CreateScopeResponse{}, h.client.CreateScope, "create scope")
}
func (h *connectHandler) ListScopes(ctx context.Context, in *connect.Request[memoryv1.ListScopesRequest]) (*connect.Response[memoryv1.ListScopesResponse], error) {
	return proxy(ctx, in, &sourcev1.ListScopesRequest{}, &memoryv1.ListScopesResponse{}, h.client.ListScopes, "list scopes")
}
func proxy[MI, SI, MO, SO any](ctx context.Context, in *connect.Request[MI], src *SI, out *MO, invoke func(context.Context, *connect.Request[SI]) (*connect.Response[SO], error), op string) (*connect.Response[MO], error) {
	req := connect.NewRequest(src)
	if err := ledgerclient.TranslateWithScope(any(in.Msg).(proto.Message), any(req.Msg).(proto.Message), "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := invoke(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError(op, err)
	}
	if err := ledgerclient.Translate(any(resp.Msg).(proto.Message), any(out).(proto.Message)); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

var _ memoryconnect.ScopesServiceHandler = (*connectHandler)(nil)
