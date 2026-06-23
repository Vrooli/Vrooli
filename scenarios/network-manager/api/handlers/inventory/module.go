package inventory

import (
	"context"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	"network-manager/internal/module"

	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory"
	inventoryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/inventory/inventory_v1connect"
)

type handler struct{}

func Module() module.Module {
	path, h := inventoryconnect.NewInventoryServiceHandler(&handler{})
	return module.Module{Name: "inventory", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return "" }

func (h *handler) RefreshInventory(context.Context, *connect.Request[inventoryv1.RefreshInventoryRequest]) (*connect.Response[inventoryv1.RefreshInventoryResponse], error) {
	return connect.NewResponse(&inventoryv1.RefreshInventoryResponse{Devices: []*inventoryv1.Device{sampleDevice()}, Findings: []string{"Discovery adapter not implemented yet; returned sample device."}}), nil
}

func (h *handler) ListDevices(context.Context, *connect.Request[inventoryv1.ListDevicesRequest]) (*connect.Response[inventoryv1.ListDevicesResponse], error) {
	return connect.NewResponse(&inventoryv1.ListDevicesResponse{Devices: []*inventoryv1.Device{sampleDevice()}}), nil
}

func (h *handler) UpdateDeviceGroup(_ context.Context, req *connect.Request[inventoryv1.UpdateDeviceGroupRequest]) (*connect.Response[inventoryv1.UpdateDeviceGroupResponse], error) {
	d := sampleDevice()
	if req.Msg.GetId() != "" {
		d.Id = req.Msg.GetId()
	}
	if req.Msg.GetGroup() != "" {
		d.Group = req.Msg.GetGroup()
	}
	return connect.NewResponse(&inventoryv1.UpdateDeviceGroupResponse{Device: d}), nil
}

func (h *handler) ExplainDeviceIdentity(_ context.Context, req *connect.Request[inventoryv1.ExplainDeviceIdentityRequest]) (*connect.Response[inventoryv1.ExplainDeviceIdentityResponse], error) {
	d := sampleDevice()
	if req.Msg.GetId() != "" {
		d.Id = req.Msg.GetId()
	}
	return connect.NewResponse(&inventoryv1.ExplainDeviceIdentityResponse{Device: d, Evidence: []string{"Identity confidence is scaffolded until LAN discovery is implemented."}}), nil
}

func sampleDevice() *inventoryv1.Device {
	return &inventoryv1.Device{Id: "device-preview", Hostname: "unknown", Group: "unassigned", IdentityConfidence: "low", Notes: []string{"Sample inventory row; no LAN scan performed."}}
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
