package recipients

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/recipients/recipients_v1connect"
	"notification-hub/internal/hub"
	identity "notification-hub/internal/identity"
	"notification-hub/internal/module"
)

type handler struct {
	service  *hub.Service
	verifier identity.Verifier
}

func Module(service *hub.Service) module.Module {
	return ModuleWithVerifier(service, nil)
}

func ModuleWithVerifier(service *hub.Service, verifier identity.Verifier) module.Module {
	h := &handler{service: service, verifier: verifier}
	return module.Module{Name: "recipients", Mount: func(r *mux.Router) {
		path, svc := connectv1.NewRecipientsServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func subject(ctx context.Context, headers http.Header, verifier identity.Verifier) (string, error) {
	return identity.Subject(ctx, headers, verifier)
}

func (h *handler) RegisterPushSubscription(ctx context.Context, req *connect.Request[v1.RegisterPushSubscriptionRequest]) (*connect.Response[v1.RegisterPushSubscriptionResponse], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	p := hub.PushSubscription{Endpoint: req.Msg.GetEndpoint(), P256DH: req.Msg.GetP256Dh(), Auth: req.Msg.GetAuth(), Origin: req.Msg.GetOrigin()}
	if err := h.service.RegisterPushSubscription(ctx, s, p); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.RegisterPushSubscriptionResponse{SubscriptionId: p.Endpoint}), nil
}

func (h *handler) GetRecipient(ctx context.Context, req *connect.Request[v1.GetRecipientRequest]) (*connect.Response[v1.Recipient], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	recipient, err := h.service.GetRecipient(ctx, s)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.Recipient{Id: recipient.ID, Subject: recipient.Subject, TrustPosture: recipient.TrustPosture, CreatedAt: recipient.CreatedAt, UpdatedAt: recipient.UpdatedAt}), nil
}

func (h *handler) RemovePushSubscription(ctx context.Context, req *connect.Request[v1.RemovePushSubscriptionRequest]) (*connect.Response[v1.RemovePushSubscriptionResponse], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if err := h.service.RemovePushSubscription(ctx, s, req.Msg.GetEndpoint()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.RemovePushSubscriptionResponse{}), nil
}

func (h *handler) ListDevices(ctx context.Context, req *connect.Request[v1.ListDevicesRequest]) (*connect.Response[v1.ListDevicesResponse], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	devices, err := h.service.ListDevices(ctx, s)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListDevicesResponse{VapidPublicKey: h.service.WebPushPublicKey()}
	for _, device := range devices {
		out.Devices = append(out.Devices, &v1.Device{Id: device.ID, Name: device.Name, MachineId: device.MachineID, Channels: device.Channels})
	}
	return connect.NewResponse(out), nil
}

func (h *handler) SetQuietWindow(ctx context.Context, req *connect.Request[v1.SetQuietWindowRequest]) (*connect.Response[v1.QuietWindow], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	window, err := h.service.SetQuietWindow(ctx, s, hub.QuietWindow{Weekday: int(req.Msg.GetWeekday()), Start: req.Msg.GetStart(), End: req.Msg.GetEnd(), Timezone: req.Msg.GetTimezone(), CriticalOverride: req.Msg.GetCriticalOverride()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(&v1.QuietWindow{Id: window.ID, Weekday: int32(window.Weekday), Start: window.Start, End: window.End, Timezone: window.Timezone, CriticalOverride: window.CriticalOverride}), nil
}

func (h *handler) UpsertDevice(ctx context.Context, req *connect.Request[v1.UpsertDeviceRequest]) (*connect.Response[v1.Device], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	device, err := h.service.UpsertDevice(ctx, s, hub.Device{ID: req.Msg.GetId(), Name: req.Msg.GetName(), MachineID: req.Msg.GetMachineId()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(deviceProto(device)), nil
}

func (h *handler) RemoveDevice(ctx context.Context, req *connect.Request[v1.RemoveDeviceRequest]) (*connect.Response[v1.RemoveDeviceResponse], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if err := h.service.RemoveDevice(ctx, s, req.Msg.GetId()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.RemoveDeviceResponse{}), nil
}

func (h *handler) UpsertChannelAddress(ctx context.Context, req *connect.Request[v1.UpsertChannelAddressRequest]) (*connect.Response[v1.ChannelAddress], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	address, err := h.service.UpsertChannelAddress(ctx, s, hub.ChannelAddress{ID: req.Msg.GetId(), DeviceID: req.Msg.GetDeviceId(), Channel: req.Msg.GetChannel(), Address: req.Msg.GetAddress(), ApprovedLabels: req.Msg.GetApprovedLabels()})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(channelAddressProto(address)), nil
}

func (h *handler) RemoveChannelAddress(ctx context.Context, req *connect.Request[v1.RemoveChannelAddressRequest]) (*connect.Response[v1.RemoveChannelAddressResponse], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if err := h.service.RemoveChannelAddress(ctx, s, req.Msg.GetId()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.RemoveChannelAddressResponse{}), nil
}

func (h *handler) ListQuietWindows(ctx context.Context, req *connect.Request[v1.ListQuietWindowsRequest]) (*connect.Response[v1.ListQuietWindowsResponse], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	windows, err := h.service.ListQuietWindows(ctx, s)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListQuietWindowsResponse{}
	for _, window := range windows {
		out.Windows = append(out.Windows, quietWindowProto(window))
	}
	return connect.NewResponse(out), nil
}

func (h *handler) DeleteQuietWindow(ctx context.Context, req *connect.Request[v1.DeleteQuietWindowRequest]) (*connect.Response[v1.DeleteQuietWindowResponse], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	if err := h.service.DeleteQuietWindow(ctx, s, req.Msg.GetId()); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.DeleteQuietWindowResponse{}), nil
}

func (h *handler) SetEscalationChain(ctx context.Context, req *connect.Request[v1.SetEscalationChainRequest]) (*connect.Response[v1.EscalationChain], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	chain, err := h.service.SetEscalationChain(ctx, s, req.Msg.GetChannels())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(escalationProto(chain)), nil
}

func (h *handler) GetEscalationChain(ctx context.Context, req *connect.Request[v1.GetEscalationChainRequest]) (*connect.Response[v1.EscalationChain], error) {
	s, err := subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	chain, err := h.service.GetEscalationChain(ctx, s)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(escalationProto(chain)), nil
}

func deviceProto(device hub.Device) *v1.Device {
	return &v1.Device{Id: device.ID, Name: device.Name, MachineId: device.MachineID, Channels: device.Channels}
}

func channelAddressProto(address hub.ChannelAddress) *v1.ChannelAddress {
	return &v1.ChannelAddress{Id: address.ID, DeviceId: address.DeviceID, Channel: address.Channel, Address: address.Address, ApprovedLabels: address.ApprovedLabels}
}

func quietWindowProto(window hub.QuietWindow) *v1.QuietWindow {
	return &v1.QuietWindow{Id: window.ID, Weekday: int32(window.Weekday), Start: window.Start, End: window.End, Timezone: window.Timezone, CriticalOverride: window.CriticalOverride}
}

func escalationProto(chain hub.EscalationChain) *v1.EscalationChain {
	out := &v1.EscalationChain{RecipientId: chain.RecipientID}
	for _, step := range chain.Steps {
		out.Steps = append(out.Steps, &v1.EscalationStep{Ordinal: int32(step.Ordinal), Channel: step.Channel})
	}
	return out
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "recipients_get", Path: connectv1.RecipientsServiceGetRecipientProcedure, Method: http.MethodPost, Summary: "Read the authenticated recipient projection", Category: "recipients"},
	{ID: "recipients_push_subscribe", Path: connectv1.RecipientsServiceRegisterPushSubscriptionProcedure, Method: http.MethodPost, Summary: "Register a browser push subscription", Category: "recipients"},
	{ID: "recipients_push_remove", Path: connectv1.RecipientsServiceRemovePushSubscriptionProcedure, Method: http.MethodPost, Summary: "Remove a dead browser push subscription", Category: "recipients"},
	{ID: "recipients_devices_list", Path: connectv1.RecipientsServiceListDevicesProcedure, Method: http.MethodPost, Summary: "List recipient devices", Category: "recipients"},
	{ID: "recipients_device_upsert", Path: connectv1.RecipientsServiceUpsertDeviceProcedure, Method: http.MethodPost, Summary: "Register or update a device", Category: "recipients"},
	{ID: "recipients_device_remove", Path: connectv1.RecipientsServiceRemoveDeviceProcedure, Method: http.MethodPost, Summary: "Remove a device", Category: "recipients"},
	{ID: "recipients_address_upsert", Path: connectv1.RecipientsServiceUpsertChannelAddressProcedure, Method: http.MethodPost, Summary: "Register or update a channel address", Category: "recipients"},
	{ID: "recipients_address_remove", Path: connectv1.RecipientsServiceRemoveChannelAddressProcedure, Method: http.MethodPost, Summary: "Remove a channel address", Category: "recipients"},
	{ID: "recipients_quiet_window_set", Path: connectv1.RecipientsServiceSetQuietWindowProcedure, Method: http.MethodPost, Summary: "Configure quiet hours", Category: "recipients"},
	{ID: "recipients_quiet_window_list", Path: connectv1.RecipientsServiceListQuietWindowsProcedure, Method: http.MethodPost, Summary: "List quiet hours", Category: "recipients"},
	{ID: "recipients_quiet_window_delete", Path: connectv1.RecipientsServiceDeleteQuietWindowProcedure, Method: http.MethodPost, Summary: "Delete quiet hours", Category: "recipients"},
	{ID: "recipients_escalation_set", Path: connectv1.RecipientsServiceSetEscalationChainProcedure, Method: http.MethodPost, Summary: "Set ordered escalation channels", Category: "recipients"},
	{ID: "recipients_escalation_get", Path: connectv1.RecipientsServiceGetEscalationChainProcedure, Method: http.MethodPost, Summary: "Read ordered escalation channels", Category: "recipients"},
}
