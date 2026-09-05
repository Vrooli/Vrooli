package relay

import (
	"context"
	"errors"
	"log"

	"vrooli-bridge/internal/audit"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/dispatch"
	"vrooli-bridge/internal/module"
	"vrooli-bridge/internal/registry"
	internalrelay "vrooli-bridge/internal/relay"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
)

type Deps struct {
	Service internalrelay.Service
	Logger  *log.Logger
}

type connectHandler struct{ deps Deps }

type nodeReaderAdapter struct{ svc registry.Service }

func (a nodeReaderAdapter) GetTarget(ctx context.Context, id string) (dispatch.TargetNode, error) {
	node, err := a.svc.Get(ctx, id)
	if err != nil {
		var notFound registry.ErrNodeNotFound
		if errors.As(err, &notFound) {
			return dispatch.TargetNode{}, dispatch.ErrNodeNotFound{ID: id}
		}
		return dispatch.TargetNode{}, err
	}
	return dispatch.TargetNode{
		ID: node.ID, Kind: node.Kind, OS: node.OS, Arch: node.Arch,
		Scopes: append([]string(nil), node.Scopes...), Revoked: node.Revoked(),
	}, nil
}

func NewService(registrySvc registry.Service, presence internalrelay.Presence, sink audit.Sink, pusher internalrelay.Pusher, broker *internalrelay.Broker) internalrelay.Service {
	return internalrelay.NewService(nodeReaderAdapter{svc: registrySvc}, presence, sink, pusher, broker)
}

func NewConnectHandler(deps Deps) *connectHandler {
	if deps.Logger == nil {
		deps.Logger = log.Default()
	}
	return &connectHandler{deps: deps}
}

func (h *connectHandler) Call(ctx context.Context, req *connect.Request[relayv1.RelayCallRequest]) (*connect.Response[relayv1.RelayCallResponse], error) {
	owner, err := auth.RequireOwner(ctx)
	if err != nil {
		return nil, auth.ToConnectError(err)
	}
	actor := owner.OwnerID
	if actor == "" {
		actor = owner.Email
	}
	if actor == "" {
		actor = "owner"
	}
	response, err := h.deps.Service.Call(ctx, internalrelay.Request{
		Actor: actor, NodeID: req.Msg.GetNodeId(), Scenario: req.Msg.GetScenario(),
		Command: req.Msg.GetCommand(), Args: req.Msg.GetArgs(),
		TimeoutSeconds: req.Msg.GetTimeoutSeconds(), MaxResponseBytes: req.Msg.GetMaxResponseBytes(),
	})
	if err != nil {
		connectErr := toConnectError(err)
		if connect.CodeOf(connectErr) == connect.CodeInternal {
			h.deps.Logger.Printf("relay.Call(node=%q command=%q): %v", req.Msg.GetNodeId(), req.Msg.GetCommand(), err)
		}
		return nil, connectErr
	}
	return connect.NewResponse(&relayv1.RelayCallResponse{
		CorrelationId: response.CorrelationID,
		Outcome:       outcome(response.Kind),
		Data:          append([]byte(nil), response.Data...),
		Reason:        response.Reason,
		ExitCode:      response.ExitCode,
		TotalBytes:    response.TotalBytes,
	}), nil
}

func toConnectError(err error) error {
	if err == nil {
		return nil
	}
	var limit internalrelay.ErrResponseLimit
	switch {
	case errors.As(err, &limit):
		return connect.NewError(connect.CodeResourceExhausted, limit)
	case errors.Is(err, context.Canceled):
		return connect.NewError(connect.CodeCanceled, err)
	case errors.Is(err, context.DeadlineExceeded):
		return connect.NewError(connect.CodeDeadlineExceeded, err)
	case errors.Is(err, internalrelay.ErrInvalidRequest):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, internalrelay.ErrCorrelationConflict):
		return connect.NewError(connect.CodeAlreadyExists, err)
	default:
		return dispatch.ToConnectError(err)
	}
}

func outcome(kind string) relayv1.RelayCallOutcome {
	switch kind {
	case internalrelay.KindCompleted:
		return relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_COMPLETED
	case internalrelay.KindTerminated:
		return relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_TERMINATED
	default:
		return relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_FAILED
	}
}

func Module(svc internalrelay.Service, logger *log.Logger) module.Module {
	path, handler := relayconnect.NewRelayServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "relay",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
		},
		Endpoints: Endpoints,
	}
}
