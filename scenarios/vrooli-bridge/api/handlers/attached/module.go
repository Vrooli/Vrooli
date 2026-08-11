package attached

import (
	"context"
	"database/sql"
	"log"
	"time"

	"connectrpc.com/connect"
	internal "vrooli-bridge/internal/attached"
	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/module"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	attachedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices"
	attachedconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/attached_devices/attached_devices_v1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type handler struct {
	service *internal.Service
	logger  *log.Logger
}

func Module(db *sql.DB, logger *log.Logger) module.Module {
	service, err := internal.NewServiceWithDB(db)
	if err != nil {
		panic(err)
	}
	h := &handler{service: service, logger: logger}
	path, svc := attachedconnect.NewAttachedDeviceServiceHandler(h)
	return module.Module{Name: "attached-devices", Mount: func(r *mux.Router) { connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: svc}) }, Endpoints: Endpoints}
}

func (h *handler) PairAttachedDevice(ctx context.Context, req *connect.Request[attachedv1.PairAttachedDeviceRequest]) (*connect.Response[attachedv1.AttachedDeviceResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	d, err := h.service.Pair(ctx, internal.PairInput{Name: req.Msg.Name, HostNodeID: req.Msg.HostNodeId, Kind: req.Msg.Kind, Transport: req.Msg.Transport, Serial: req.Msg.Serial, OSVersion: req.Msg.OsVersion, HostNodeOnline: req.Msg.HostNodeOnline})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&attachedv1.AttachedDeviceResponse{Device: toProto(d)}), nil
}

func (h *handler) ListAttachedDevices(ctx context.Context, _ *connect.Request[attachedv1.ListAttachedDevicesRequest]) (*connect.Response[attachedv1.ListAttachedDevicesResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	items := h.service.List(ctx)
	out := make([]*attachedv1.AttachedDevice, 0, len(items))
	for _, d := range items {
		out = append(out, toProto(d))
	}
	return connect.NewResponse(&attachedv1.ListAttachedDevicesResponse{Devices: out}), nil
}

func (h *handler) RevokeAttachedDevice(ctx context.Context, req *connect.Request[attachedv1.RevokeAttachedDeviceRequest]) (*connect.Response[attachedv1.AttachedDeviceResponse], error) {
	if _, err := auth.RequireOwner(ctx); err != nil {
		return nil, auth.ToConnectError(err)
	}
	d, err := h.service.Revoke(ctx, req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewResponse(&attachedv1.AttachedDeviceResponse{Device: toProto(d)}), nil
}

func toProto(d internal.Device) *attachedv1.AttachedDevice {
	return &attachedv1.AttachedDevice{Id: d.ID, Name: d.Name, HostNodeId: d.HostNodeID, Kind: d.Kind, Transport: d.Transport, Serial: d.Serial, OsVersion: d.OSVersion, TrustState: d.TrustState, Reachability: d.Reachability, HealthReason: d.HealthReason, CreatedAt: timestamppb.New(d.CreatedAt), RevokedAt: timestampOrNil(d.RevokedAt)}
}

func timestampOrNil(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}
