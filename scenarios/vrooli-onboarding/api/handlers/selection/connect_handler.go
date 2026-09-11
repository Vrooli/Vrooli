package selection

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	selectionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection/selectionv1connect"
	internalselection "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/selection"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type strictJSONCodec struct{ name string }

func (c *strictJSONCodec) Name() string { return c.name }
func (c *strictJSONCodec) Marshal(message any) ([]byte, error) {
	value, ok := message.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("selection JSON codec received %T", message)
	}
	return protojson.Marshal(value)
}
func (c *strictJSONCodec) Unmarshal(data []byte, message any) error {
	value, ok := message.(proto.Message)
	if !ok {
		return fmt.Errorf("selection JSON codec received %T", message)
	}
	return (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, value)
}

type connectHandler struct{ service internalselection.Service }

func NewConnectHandler(service internalselection.Service) selectionconnect.SelectionServiceHandler {
	return &connectHandler{service: service}
}

func (h *connectHandler) ListScenarios(ctx context.Context, _ *connect.Request[selectionv1.ListScenariosRequest]) (*connect.Response[selectionv1.ListScenariosResponse], error) {
	return unary(ctx, h.service.Scenarios)
}
func (h *connectHandler) GetCoreSet(ctx context.Context, req *connect.Request[selectionv1.GetCoreSetRequest]) (*connect.Response[selectionv1.GetCoreSetResponse], error) {
	return unary(ctx, func(ctx context.Context) (*selectionv1.GetCoreSetResponse, error) {
		return h.service.CoreSet(ctx, req.Msg.GetSeed())
	})
}
func (h *connectHandler) GetRecommendation(ctx context.Context, _ *connect.Request[selectionv1.GetRecommendationRequest]) (*connect.Response[selectionv1.GetRecommendationResponse], error) {
	return unary(ctx, h.service.Recommendation)
}
func (h *connectHandler) AcceptRecommendation(ctx context.Context, req *connect.Request[selectionv1.AcceptRecommendationRequest]) (*connect.Response[selectionv1.AcceptRecommendationResponse], error) {
	return unary(ctx, func(ctx context.Context) (*selectionv1.AcceptRecommendationResponse, error) {
		return h.service.Accept(ctx, req.Msg)
	})
}
func (h *connectHandler) GetClosure(ctx context.Context, _ *connect.Request[selectionv1.GetClosureRequest]) (*connect.Response[selectionv1.GetClosureResponse], error) {
	return unary(ctx, h.service.Closure)
}
func (h *connectHandler) GetUnion(ctx context.Context, _ *connect.Request[selectionv1.GetUnionRequest]) (*connect.Response[selectionv1.GetUnionResponse], error) {
	return unary(ctx, h.service.Union)
}
func (h *connectHandler) CreateHandoff(ctx context.Context, req *connect.Request[selectionv1.CreateHandoffRequest]) (*connect.Response[selectionv1.CreateHandoffResponse], error) {
	if req.Msg.GetDesiredSelection() == nil && strings.TrimSpace(req.Msg.GetNodeId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("node_id is required"))
	}
	return unary(ctx, func(ctx context.Context) (*selectionv1.CreateHandoffResponse, error) {
		return h.service.Handoff(ctx, req.Msg)
	})
}

func (h *connectHandler) GetHandoff(ctx context.Context, req *connect.Request[selectionv1.GetHandoffRequest]) (*connect.Response[selectionv1.GetHandoffResponse], error) {
	if strings.TrimSpace(req.Msg.GetReference()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("handoff reference is required"))
	}
	return unary(ctx, func(ctx context.Context) (*selectionv1.GetHandoffResponse, error) {
		return h.service.ResolveHandoff(ctx, req.Msg)
	})
}

func unary[T any](ctx context.Context, call func(context.Context) (*T, error)) (*connect.Response[T], error) {
	value, err := call(ctx)
	if err != nil {
		return nil, selectionError(err)
	}
	return connect.NewResponse(value), nil
}

func selectionError(err error) error {
	if strings.Contains(err.Error(), "unsupported setup selection schema version") {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	if strings.Contains(err.Error(), "stale or outside the authorized target revision") {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	return connect.NewError(connect.CodeInternal, fmt.Errorf("selection operation: %w", err))
}
