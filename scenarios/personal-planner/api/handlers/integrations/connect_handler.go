package integrations

import (
	"context"
	"log"

	"connectrpc.com/connect"
	v "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/integrations"
	"google.golang.org/protobuf/types/known/timestamppb"

	d "personal-planner/internal/integrations"
)

type (
	Deps struct {
		Service d.Service
		Logger  *log.Logger
	}
	connectHandler struct{ deps Deps }
)

func NewConnectHandler(x Deps) *connectHandler {
	if x.Logger == nil {
		x.Logger = log.Default()
	}
	return &connectHandler{deps: x}
}

func (h *connectHandler) ListConnections(ctx context.Context, _ *connect.Request[v.ListConnectionsRequest]) (*connect.Response[v.ListConnectionsResponse], error) {
	items, err := h.deps.Service.List(ctx)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	out := &v.ListConnectionsResponse{}
	for _, item := range items {
		out.Connections = append(out.Connections, toProto(item))
	}
	return connect.NewResponse(out), nil
}

func (h *connectHandler) CreateFixtureConnection(ctx context.Context, req *connect.Request[v.CreateFixtureConnectionRequest]) (*connect.Response[v.CreateFixtureConnectionResponse], error) {
	item, err := h.deps.Service.CreateFixture(ctx, req.Msg.DisplayName)
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.CreateFixtureConnectionResponse{Connection: toProto(item)}), nil
}

func (h *connectHandler) SyncConnection(ctx context.Context, req *connect.Request[v.SyncConnectionRequest]) (*connect.Response[v.SyncConnectionResponse], error) {
	item, err := h.deps.Service.Sync(ctx, d.SyncInput{ID: req.Msg.Id, ExpectedRevision: req.Msg.ExpectedRevision})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.SyncConnectionResponse{Connection: toProto(item), ImportedEventCount: int32(item.ImportedEventCount), BusyMinutes: int64(item.BusyMinutes)}), nil
}

func (h *connectHandler) DisconnectConnection(ctx context.Context, req *connect.Request[v.DisconnectConnectionRequest]) (*connect.Response[v.DisconnectConnectionResponse], error) {
	item, err := h.deps.Service.Disconnect(ctx, d.SyncInput{ID: req.Msg.Id, ExpectedRevision: req.Msg.ExpectedRevision})
	if err != nil {
		return nil, d.ToConnectError(err)
	}
	return connect.NewResponse(&v.DisconnectConnectionResponse{Connection: toProto(item)}), nil
}

func toProto(item d.Connection) *v.ProviderConnection {
	out := &v.ProviderConnection{Id: item.ID, Provider: item.Provider, DisplayName: item.DisplayName, SourceKind: item.SourceKind, Status: item.Status, HealthMessage: item.HealthMessage, ReadOnly: item.ReadOnly, CalendarCount: int32(item.CalendarCount), ImportedEventCount: int32(item.ImportedEventCount), BusyMinutes: int64(item.BusyMinutes), Revision: item.Revision}
	if !item.LastSyncAt.IsZero() {
		out.LastSyncAt = timestamppb.New(item.LastSyncAt)
	}
	return out
}
