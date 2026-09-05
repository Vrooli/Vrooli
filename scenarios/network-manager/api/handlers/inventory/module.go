package inventory

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	domaininventory "network-manager/internal/inventory"
	"network-manager/internal/module"
	domainresolver "network-manager/internal/resolver"

	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory/inventory_v1connect"
)

type handler struct {
	service *domaininventory.Service
}

func Module(db domaininventory.SQLExecutor) module.Module {
	resolverRepo := domainresolver.NewSQLiteRepository(db)
	service := domaininventory.NewService(domaininventory.Config{
		Repo:   domaininventory.NewSQLiteRepository(db),
		Source: domaininventory.NewAdGuardClientDiscoverySource(resolverRepo, nil),
	})
	path, h := inventoryconnect.NewInventoryServiceHandler(&handler{service: service})
	return module.Module{Name: "inventory", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return domaininventory.Schema() }

func (h *handler) RefreshInventory(ctx context.Context, req *connect.Request[inventoryv1.RefreshInventoryRequest]) (*connect.Response[inventoryv1.RefreshInventoryResponse], error) {
	devices, findings, err := h.service.Refresh(ctx, req.Msg.GetDryRun())
	if err != nil {
		return nil, inventoryError(err)
	}
	return connect.NewResponse(&inventoryv1.RefreshInventoryResponse{Devices: toProtoDevices(devices), Findings: findings}), nil
}

func (h *handler) ListDevices(ctx context.Context, req *connect.Request[inventoryv1.ListDevicesRequest]) (*connect.Response[inventoryv1.ListDevicesResponse], error) {
	devices, err := h.service.List(ctx, req.Msg.GetGroup())
	if err != nil {
		return nil, inventoryError(err)
	}
	return connect.NewResponse(&inventoryv1.ListDevicesResponse{Devices: toProtoDevices(devices)}), nil
}

func (h *handler) UpdateDeviceGroup(ctx context.Context, req *connect.Request[inventoryv1.UpdateDeviceGroupRequest]) (*connect.Response[inventoryv1.UpdateDeviceGroupResponse], error) {
	device, err := h.service.UpdateGroup(ctx, req.Msg.GetId(), req.Msg.GetGroup())
	if err != nil {
		return nil, inventoryError(err)
	}
	return connect.NewResponse(&inventoryv1.UpdateDeviceGroupResponse{Device: toProtoDevice(device)}), nil
}

func (h *handler) ExplainDeviceIdentity(ctx context.Context, req *connect.Request[inventoryv1.ExplainDeviceIdentityRequest]) (*connect.Response[inventoryv1.ExplainDeviceIdentityResponse], error) {
	device, evidence, err := h.service.Explain(ctx, req.Msg.GetId())
	if err != nil {
		return nil, inventoryError(err)
	}
	return connect.NewResponse(&inventoryv1.ExplainDeviceIdentityResponse{Device: toProtoDevice(device), Evidence: evidence}), nil
}

func inventoryError(err error) error {
	switch {
	case errors.Is(err, domaininventory.ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	default:
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
}

func toProtoDevices(devices []domaininventory.Device) []*inventoryv1.Device {
	out := make([]*inventoryv1.Device, 0, len(devices))
	for _, device := range devices {
		out = append(out, toProtoDevice(device))
	}
	return out
}

func toProtoDevice(device domaininventory.Device) *inventoryv1.Device {
	return &inventoryv1.Device{
		Id:                 device.ID,
		Hostname:           device.Hostname,
		IpAddress:          device.IPAddress,
		MacAddress:         device.MACAddress,
		Group:              device.Group,
		IdentityConfidence: device.IdentityConfidence,
		Notes:              device.Notes,
	}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("inventory_refresh", inventoryconnect.InventoryServiceRefreshInventoryProcedure, "Refresh device inventory"),
	connectEndpoint("inventory_list", inventoryconnect.InventoryServiceListDevicesProcedure, "List devices"),
	connectEndpoint("inventory_group_update", inventoryconnect.InventoryServiceUpdateDeviceGroupProcedure, "Update device group"),
	connectEndpoint("inventory_identity_explain", inventoryconnect.InventoryServiceExplainDeviceIdentityProcedure, "Explain device identity"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "inventory", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"devices": "array<Device>"}}}
}
