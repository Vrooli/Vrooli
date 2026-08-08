package facets

import (
	"context"
	"log"

	"connectrpc.com/connect"

	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets/facets_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets"
	memoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets/facets_v1connect"

	"vrooli-memory/internal/ledgerclient"
)

type connectHandler struct {
	client sourceconnect.FacetsServiceClient
	logger *log.Logger
}

func NewConnectHandler(client sourceconnect.FacetsServiceClient, logger *log.Logger) *connectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &connectHandler{client: client, logger: logger}
}

func (h *connectHandler) ListFacets(ctx context.Context, in *connect.Request[memoryv1.ListFacetsRequest]) (*connect.Response[memoryv1.ListFacetsResponse], error) {
	return h.callListFacets(ctx, in)
}

func (h *connectHandler) callListFacets(ctx context.Context, in *connect.Request[memoryv1.ListFacetsRequest]) (*connect.Response[memoryv1.ListFacetsResponse], error) {
	req := connect.NewRequest(&sourcev1.ListFacetsRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.ListFacets(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("list facets", err)
	}
	out := &memoryv1.ListFacetsResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) AssignFacet(ctx context.Context, in *connect.Request[memoryv1.AssignFacetRequest]) (*connect.Response[memoryv1.AssignFacetResponse], error) {
	return h.assignFacet(ctx, in)
}

func (h *connectHandler) assignFacet(ctx context.Context, in *connect.Request[memoryv1.AssignFacetRequest]) (*connect.Response[memoryv1.AssignFacetResponse], error) {
	req := connect.NewRequest(&sourcev1.AssignFacetRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.AssignFacet(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("assign facet", err)
	}
	out := &memoryv1.AssignFacetResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) SetPin(ctx context.Context, in *connect.Request[memoryv1.SetPinRequest]) (*connect.Response[memoryv1.SetPinResponse], error) {
	req := connect.NewRequest(&sourcev1.SetPinRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	ledgerclient.ForwardHeaders(in.Header(), req.Header())
	resp, err := h.client.SetPin(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("set pin", err)
	}
	out := &memoryv1.SetPinResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListPinProposals(ctx context.Context, in *connect.Request[memoryv1.ListPinProposalsRequest]) (*connect.Response[memoryv1.ListPinProposalsResponse], error) {
	req := connect.NewRequest(&sourcev1.ListPinProposalsRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.ListPinProposals(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("list pin proposals", err)
	}
	out := &memoryv1.ListPinProposalsResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ListPinCandidates(ctx context.Context, in *connect.Request[memoryv1.ListPinCandidatesRequest]) (*connect.Response[memoryv1.ListPinCandidatesResponse], error) {
	req := connect.NewRequest(&sourcev1.ListPinCandidatesRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.ListPinCandidates(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("list pin candidates", err)
	}
	out := &memoryv1.ListPinCandidatesResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ResolvePinProposal(ctx context.Context, in *connect.Request[memoryv1.ResolvePinProposalRequest]) (*connect.Response[memoryv1.ResolvePinProposalResponse], error) {
	req := connect.NewRequest(&sourcev1.ResolvePinProposalRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.ResolvePinProposal(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("resolve pin proposal", err)
	}
	out := &memoryv1.ResolvePinProposalResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) MarkSuperseded(ctx context.Context, in *connect.Request[memoryv1.MarkSupersededRequest]) (*connect.Response[memoryv1.MarkSupersededResponse], error) {
	req := connect.NewRequest(&sourcev1.MarkSupersededRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.MarkSuperseded(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("mark superseded", err)
	}
	out := &memoryv1.MarkSupersededResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) ResolveThread(ctx context.Context, in *connect.Request[memoryv1.ResolveThreadRequest]) (*connect.Response[memoryv1.ResolveThreadResponse], error) {
	req := connect.NewRequest(&sourcev1.ResolveThreadRequest{})
	if err := ledgerclient.TranslateWithScope(in.Msg, req.Msg, "agent-memory"); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	resp, err := h.client.ResolveThread(ctx, req)
	if err != nil {
		return nil, ledgerclient.RPCError("resolve thread", err)
	}
	out := &memoryv1.ResolveThreadResponse{}
	if err := ledgerclient.Translate(resp.Msg, out); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

var _ memoryconnect.FacetsServiceHandler = (*connectHandler)(nil)
