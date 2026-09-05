package settings

import (
	"context"
	"errors"
	"log"

	"connectrpc.com/connect"

	settingsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/settings"
)

// Deps wires the seams the Connect settings handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// SettingsServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

// ErrInvalidArgument is the sentinel the Service implementation returns
// for caller-visible validation failures (unknown backend, malformed
// policy). The handler maps it to connect.CodeInvalidArgument.
var ErrInvalidArgument = errors.New("invalid argument")

func (h *connectHandler) GetSessionDefaults(_ context.Context, _ *connect.Request[settingsv1.GetSessionDefaultsRequest]) (*connect.Response[settingsv1.GetSessionDefaultsResponse], error) {
	d := h.deps.Service.GetDefaults()
	return connect.NewResponse(&settingsv1.GetSessionDefaultsResponse{
		Defaults: defaultsToProto(d),
	}), nil
}

func (h *connectHandler) UpdateSessionDefaults(_ context.Context, req *connect.Request[settingsv1.UpdateSessionDefaultsRequest]) (*connect.Response[settingsv1.UpdateSessionDefaultsResponse], error) {
	upd := UpdateDefaultsRequest{}
	if req.Msg.DefaultBackend != nil {
		v := *req.Msg.DefaultBackend
		upd.DefaultBackend = &v
	}
	if req.Msg.DefaultPolicy != nil {
		p := Policy{
			Mode:     req.Msg.DefaultPolicy.Mode,
			Duration: req.Msg.DefaultPolicy.Duration,
		}
		upd.DefaultPolicy = &p
	}
	out, err := h.deps.Service.UpdateDefaults(upd)
	if err != nil {
		if errors.Is(err, ErrInvalidArgument) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		h.deps.Logger.Printf("settings.UpdateSessionDefaults: %v", err)
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&settingsv1.UpdateSessionDefaultsResponse{
		Defaults: defaultsToProto(out),
	}), nil
}

func defaultsToProto(d Defaults) *settingsv1.SessionDefaults {
	return &settingsv1.SessionDefaults{
		DefaultBackend: d.DefaultBackend,
		DefaultPolicy: &settingsv1.ExpirationPolicy{
			Mode:     d.DefaultPolicy.Mode,
			Duration: d.DefaultPolicy.Duration,
		},
	}
}
