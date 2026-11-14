package sources

import (
	"bytes"
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	internal "signal-inbox/internal/sources"

	sourcesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/signal-inbox/v1/sources"
)

type connectHandler struct{ service *internal.Service }

func NewConnectHandler(service *internal.Service) *connectHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListAdapters(ctx context.Context, _ *connect.Request[sourcesv1.ListAdaptersRequest]) (*connect.Response[sourcesv1.ListAdaptersResponse], error) {
	states, err := h.service.List(ctx)
	if err != nil {
		return nil, toConnectError(err)
	}
	response := &sourcesv1.ListAdaptersResponse{Adapters: make([]*sourcesv1.AdapterState, 0, len(states))}
	for _, state := range states {
		response.Adapters = append(response.Adapters, stateToProto(state))
	}
	return connect.NewResponse(response), nil
}

func (h *connectHandler) SetAdapterEnabled(ctx context.Context, req *connect.Request[sourcesv1.SetAdapterEnabledRequest]) (*connect.Response[sourcesv1.SetAdapterEnabledResponse], error) {
	state, err := h.service.SetEnabled(ctx, req.Msg.AdapterId, req.Msg.Enabled)
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&sourcesv1.SetAdapterEnabledResponse{Adapter: stateToProto(state)}), nil
}

func (h *connectHandler) ImportArchive(ctx context.Context, req *connect.Request[sourcesv1.ImportArchiveRequest]) (*connect.Response[sourcesv1.ImportArchiveResponse], error) {
	if req.Msg.AdapterId == "" || len(req.Msg.Content) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("adapter_id and content are required"))
	}
	result, err := h.service.Import(ctx, req.Msg.AdapterId, bytes.NewReader(req.Msg.Content))
	if err != nil {
		return nil, toConnectError(err)
	}
	return connect.NewResponse(&sourcesv1.ImportArchiveResponse{Result: &sourcesv1.ImportResult{RunId: result.RunID, AdapterId: result.AdapterID, Created: result.Created, Duplicated: result.Duplicated, Failed: result.Failed}}), nil
}

func stateToProto(state internal.State) *sourcesv1.AdapterState {
	result := &sourcesv1.AdapterState{AdapterId: state.AdapterID, Kind: state.Kind, RiskTier: state.RiskTier, Enabled: state.Enabled, LastError: state.LastError, DisabledReason: state.DisabledReason}
	if !state.LastRunAt.IsZero() {
		result.LastRunAt = timestamppb.New(state.LastRunAt)
	}
	return result
}

func toConnectError(err error) error {
	var unknown internal.ErrUnknownAdapter
	var disabled internal.ErrAdapterDisabled
	var invalid internal.ErrInvalidDescriptor
	switch {
	case errors.As(err, &unknown):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.As(err, &disabled):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.As(err, &invalid):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}
