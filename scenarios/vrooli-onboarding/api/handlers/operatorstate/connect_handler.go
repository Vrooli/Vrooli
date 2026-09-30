package operatorstate

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/operatorstate"
	operatorstatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate"
	operatorstatev1connect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/operatorstate/operatorstatev1connect"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/operatorstateapi"
	"google.golang.org/protobuf/encoding/protojson"
)

type connectHandler struct{ service operatorstateapi.Service }

func NewConnectHandler(service operatorstateapi.Service) operatorstatev1connect.OperatorStateServiceHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) GetOperatorState(ctx context.Context, _ *connect.Request[operatorstatev1.GetOperatorStateRequest]) (*connect.Response[operatorstatev1.GetOperatorStateResponse], error) {
	document, err := h.service.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("get operator state: %w", err))
	}
	state, err := documentToProto(document)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode operator state: %w", err))
	}
	return connect.NewResponse(&operatorstatev1.GetOperatorStateResponse{State: state}), nil
}

func (h *connectHandler) PatchOperatorState(ctx context.Context, req *connect.Request[operatorstatev1.PatchOperatorStateRequest]) (*connect.Response[operatorstatev1.PatchOperatorStateResponse], error) {
	if req.Msg.GetState() == nil || req.Msg.GetUpdateMask() == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("state and update_mask are required"))
	}
	stateJSON, err := (&protojson.MarshalOptions{UseProtoNames: true}).Marshal(req.Msg.GetState())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("encode operator state patch: %w", err))
	}
	document, err := h.service.Patch(ctx, stateJSON, req.Msg.GetUpdateMask().GetPaths(), req.Msg.GetExpectedRevision())
	if err != nil {
		if errors.Is(err, operatorstate.ErrRevisionConflict) {
			return nil, connect.NewError(connect.CodeAborted, err)
		}
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	state, err := documentToProto(document)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("encode patched operator state: %w", err))
	}
	return connect.NewResponse(&operatorstatev1.PatchOperatorStateResponse{State: state}), nil
}

func documentToProto(document operatorstate.Document) (*operatorstatev1.OperatorState, error) {
	data, err := operatorstateapi.MarshalDocument(document)
	if err != nil {
		return nil, err
	}
	state := new(operatorstatev1.OperatorState)
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, state); err != nil {
		return nil, err
	}
	return state, nil
}
