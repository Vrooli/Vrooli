package handlers

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
	"google.golang.org/protobuf/proto"
)

// AgentManagerConnectHandler mounts the typed agent-manager API without
// duplicating orchestration logic. The generated unimplemented embedding keeps
// the large compatibility service additive: methods are admitted deliberately
// as their typed adapters are implemented, while ListRuns is the first
// cross-scenario readiness-board contract.
type AgentManagerConnectHandler struct {
	apiconnect.UnimplementedAgentManagerServiceHandler
	h *Handler
}

func NewAgentManagerConnectHandler(h *Handler) *AgentManagerConnectHandler {
	return &AgentManagerConnectHandler{h: h}
}

func (h *AgentManagerConnectHandler) ListRuns(ctx context.Context, req *connect.Request[api.ListRunsRequest]) (*connect.Response[api.ListRunsResponse], error) {
	if req == nil || req.Msg == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("request is required"))
	}
	request := proto.Clone(req.Msg).(*api.ListRunsRequest)
	if request.Limit == nil {
		limit := int32(defaultRunListLimit)
		request.Limit = &limit
	}
	if h.h.validator != nil {
		if err := h.h.validator.Validate(request); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, protovalidateToDomainError(err))
		}
	}
	response, err := h.h.listRunsProto(ctx, request, nil, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(response), nil
}
