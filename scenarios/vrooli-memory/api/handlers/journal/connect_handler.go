package journal

import (
	"context"
	"log"
	"strings"

	"connectrpc.com/connect"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/journal/journal_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal"
	memoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/journal/journal_v1connect"
	"vrooli-memory/internal/ledgerclient"
)

type connectHandler struct {
	client sourceconnect.JournalServiceClient
	logger *log.Logger
}

func NewConnectHandler(client sourceconnect.JournalServiceClient, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{client: client, logger: logger}
}

func (h *connectHandler) AppendEntry(ctx context.Context, in *connect.Request[memoryv1.AppendEntryRequest]) (*connect.Response[memoryv1.AppendEntryResponse], error) {
	if strings.TrimSpace(in.Msg.GetBody()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRequired("body"))
	}
	if in.Msg.GetKind() == "work-record" {
		for _, field := range []struct{ name, value string }{{"trigger", in.Msg.GetTrigger()}, {"approach", in.Msg.GetApproach()}, {"evidence", in.Msg.GetEvidence()}, {"outcome", in.Msg.GetOutcome()}} {
			if strings.TrimSpace(field.value) == "" {
				return nil, connect.NewError(connect.CodeInvalidArgument, errRequired(field.name))
			}
		}
	}
	req := connect.NewRequest(&sourcev1.AppendEntryRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.AppendEntry(ctx, req)
	if err != nil {
		h.logger.Printf("source-ledger AppendEntry: %v", err)
		return nil, ledgerclient.RPCError("append entry", err)
	}
	out := &memoryv1.AppendEntryResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) GetEntry(ctx context.Context, in *connect.Request[memoryv1.GetEntryRequest]) (*connect.Response[memoryv1.GetEntryResponse], error) {
	if strings.TrimSpace(in.Msg.GetId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errRequired("id"))
	}
	req := connect.NewRequest(&sourcev1.GetEntryRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.GetEntry(ctx, req)
	if err != nil {
		h.logger.Printf("source-ledger GetEntry: %v", err)
		return nil, ledgerclient.RPCError("get entry", err)
	}
	out := &memoryv1.GetEntryResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListEntries(ctx context.Context, in *connect.Request[memoryv1.ListEntriesRequest]) (*connect.Response[memoryv1.ListEntriesResponse], error) {
	req := connect.NewRequest(&sourcev1.ListEntriesRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.ListEntries(ctx, req)
	if err != nil {
		h.logger.Printf("source-ledger ListEntries: %v", err)
		return nil, ledgerclient.RPCError("list entries", err)
	}
	out := &memoryv1.ListEntriesResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ProcessClassificationRetries(ctx context.Context, in *connect.Request[memoryv1.ProcessClassificationRetriesRequest]) (*connect.Response[memoryv1.ProcessClassificationRetriesResponse], error) {
	req := connect.NewRequest(&sourcev1.ProcessClassificationRetriesRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.ProcessClassificationRetries(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("process classification retries", err)
	}
	out := &memoryv1.ProcessClassificationRetriesResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ProcessEmbeddingRetries(ctx context.Context, in *connect.Request[memoryv1.ProcessEmbeddingRetriesRequest]) (*connect.Response[memoryv1.ProcessEmbeddingRetriesResponse], error) {
	req := connect.NewRequest(&sourcev1.ProcessEmbeddingRetriesRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, in.Msg.GetScope()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.ProcessEmbeddingRetries(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("process embedding retries", err)
	}
	out := &memoryv1.ProcessEmbeddingRetriesResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

type errRequired string

func (e errRequired) Error() string { return string(e) + " is required" }

var _ memoryconnect.JournalServiceHandler = (*connectHandler)(nil)
