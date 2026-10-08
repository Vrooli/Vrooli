// Package devices owns the operator-facing connected-device roster contract.
package devices

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"google.golang.org/protobuf/types/known/timestamppb"

	devicesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/devices"
	devicesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/devices/devices_v1connect"

	"web-console/internal/module"
)

type Device struct {
	ID, Label, Class string
	ConnectionCount  int
	FirstSeenUnix    int64
	Sessions         []SessionAttachment
	IsSelf           bool
	Reconnecting     bool
}

type SessionAttachment struct {
	SessionID   string
	SessionName string
	HoldsLease  bool
}

type Service interface {
	List(ctx context.Context, selfDeviceID string) ([]Device, error)
	Disconnect(ctx context.Context, deviceID, connectionID string) (int, error)
	GiveControl(ctx context.Context, deviceID, sessionID string) (bool, error)
}

type Deps struct{ Service Service }

type connectHandler struct{ service Service }

func unixTimestamp(unix int64) *timestamppb.Timestamp {
	if unix <= 0 {
		return nil
	}
	return timestamppb.New(time.Unix(unix, 0).UTC())
}

func NewConnectHandler(d Deps) *connectHandler { return &connectHandler{service: d.Service} }

func Module(service Service) module.Module {
	path, handler := devicesconnect.NewDeviceServiceHandler(NewConnectHandler(Deps{Service: service}))
	return module.Module{
		Name:      "devices",
		Mount:     func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler}) },
		Endpoints: Endpoints,
	}
}

func (h *connectHandler) List(ctx context.Context, req *connect.Request[devicesv1.ListRequest]) (*connect.Response[devicesv1.ListResponse], error) {
	rows, err := h.service.List(ctx, req.Msg.GetSelfDeviceId())
	if err != nil {
		return nil, err
	}
	out := make([]*devicesv1.Device, 0, len(rows))
	for _, row := range rows {
		sessions := make([]*devicesv1.SessionAttachment, 0, len(row.Sessions))
		for _, session := range row.Sessions {
			sessions = append(sessions, &devicesv1.SessionAttachment{SessionId: session.SessionID, SessionName: session.SessionName, HoldsLease: session.HoldsLease})
		}
		out = append(out, &devicesv1.Device{
			DeviceId: row.ID, DeviceLabel: row.Label, DeviceClass: row.Class,
			ConnectionCount: int32(row.ConnectionCount), FirstSeenAt: unixTimestamp(row.FirstSeenUnix),
			Sessions: sessions, IsSelf: row.IsSelf, Reconnecting: row.Reconnecting,
		})
	}
	return connect.NewResponse(&devicesv1.ListResponse{Devices: out}), nil
}

func (h *connectHandler) Disconnect(ctx context.Context, req *connect.Request[devicesv1.DisconnectRequest]) (*connect.Response[devicesv1.DisconnectResponse], error) {
	if self := req.Header().Get("X-Vrooli-Device-Id"); self != "" && self == req.Msg.GetDeviceId() {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("the current device cannot disconnect itself"))
	}
	closed, err := h.service.Disconnect(ctx, req.Msg.GetDeviceId(), req.Msg.GetConnectionId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&devicesv1.DisconnectResponse{ClosedConnections: int32(closed)}), nil
}

func (h *connectHandler) GiveControl(ctx context.Context, req *connect.Request[devicesv1.GiveControlRequest]) (*connect.Response[devicesv1.GiveControlResponse], error) {
	deviceID := req.Msg.GetDeviceId()
	if deviceID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("a device id is required"))
	}
	transferred, err := h.service.GiveControl(ctx, deviceID, req.Msg.GetSessionId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&devicesv1.GiveControlResponse{Transferred: transferred}), nil
}
