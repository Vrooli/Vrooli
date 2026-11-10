package capabilities

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	internal "client-rehearsal/internal/capabilities"
	"client-rehearsal/internal/module"
	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/shared"
	sharedconnect "github.com/vrooli/vrooli/packages/proto/gen/go/client-rehearsal/v1/shared/shared_v1connect"
	"google.golang.org/protobuf/types/known/structpb"
)

type ConnectHandler struct {
	registry *internal.Registry
	logger   *log.Logger
}

func NewConnectHandler(registry *internal.Registry, logger *log.Logger) *ConnectHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &ConnectHandler{registry: registry, logger: logger}
}
func (h *ConnectHandler) DescribeCapabilities(ctx context.Context, _ *connect.Request[sharedv1.DescribeCapabilitiesRequest]) (*connect.Response[sharedv1.DescribeCapabilitiesResponse], error) {
	if h.registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capability registry is not configured"))
	}
	data, err := h.registry.Describe(ctx)
	if err != nil {
		h.logger.Printf("capabilities.DescribeCapabilities: %v", err)
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capability status is unavailable"))
	}
	var values map[string]any
	if err = json.Unmarshal(data, &values); err != nil {
		h.logger.Printf("capabilities registry returned invalid JSON: %v", err)
		return nil, connect.NewError(connect.CodeInternal, errors.New("capability description could not be encoded"))
	}
	description, err := structpb.NewStruct(values)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("capability description could not be encoded"))
	}
	return connect.NewResponse(&sharedv1.DescribeCapabilitiesResponse{Description: description}), nil
}

func Module(registry *internal.Registry) module.Module {
	path, handler := sharedconnect.NewCapabilitiesServiceHandler(NewConnectHandler(registry, log.Default()))
	return module.Module{Name: "capabilities", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) }, Endpoints: Endpoints}
}
