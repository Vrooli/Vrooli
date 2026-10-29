// Package appdelivery mounts AppDeliveryService: a scenario's own installed
// app registering to show that scenario's notifications on one device
// instead of Chimesail. Every method needs the verified owner. The
// scenario-scoped methods read the calling scenario from the
// x-vrooli-app-scenario header that the shared scenario web server sets.
package appdelivery

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"notification-hub/internal/hub"
	identity "notification-hub/internal/identity"
	"notification-hub/internal/module"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/appdelivery"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/appdelivery/appdelivery_v1connect"
)

// ScenarioHeader names the calling scenario. The shared scenario web server
// sets it from its own configuration and drops any browser-supplied value.
const ScenarioHeader = "X-Vrooli-App-Scenario"

type handler struct {
	service  *hub.Service
	verifier identity.Verifier
}

func ModuleWithVerifier(service *hub.Service, verifier identity.Verifier) module.Module {
	h := &handler{service: service, verifier: verifier}
	return module.Module{Name: "appdelivery", Mount: func(r *mux.Router) {
		path, svc := connectv1.NewAppDeliveryServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) owner(ctx context.Context, headers http.Header) (string, error) {
	subject, err := identity.Subject(ctx, headers, h.verifier)
	if err != nil {
		return "", connect.NewError(connect.CodeUnauthenticated, err)
	}
	return subject, nil
}

func scenarioFrom(headers http.Header) (string, error) {
	scenario := strings.TrimSpace(headers.Get(ScenarioHeader))
	if scenario == "" {
		return "", connect.NewError(connect.CodeInvalidArgument, errors.New("the calling scenario is unknown; app notifications must be reached through the scenario's own web server"))
	}
	return scenario, nil
}

func mapErr(err error) error {
	switch {
	case errors.Is(err, hub.ErrInvalidArgument):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, hub.ErrAppIneligible):
		return connect.NewError(connect.CodeFailedPrecondition, err)
	case errors.Is(err, hub.ErrAppNotRegistered):
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}

func (h *handler) GetAppState(ctx context.Context, req *connect.Request[v1.GetAppStateRequest]) (*connect.Response[v1.AppState], error) {
	subject, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	scenario, err := scenarioFrom(req.Header())
	if err != nil {
		return nil, err
	}
	state, err := h.service.GetAppState(ctx, subject, scenario, req.Msg.GetEndpoint(), req.Msg.GetUserAgent())
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(appStateProto(state)), nil
}

func (h *handler) EnableApp(ctx context.Context, req *connect.Request[v1.EnableAppRequest]) (*connect.Response[v1.AppState], error) {
	subject, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	scenario, err := scenarioFrom(req.Header())
	if err != nil {
		return nil, err
	}
	m := req.Msg
	state, err := h.service.RegisterApp(ctx, subject, hub.AppRegistration{
		Scenario: scenario, Endpoint: m.GetEndpoint(), P256DH: m.GetP256Dh(), Auth: m.GetAuth(), Origin: m.GetOrigin(),
		UserAgent: m.GetUserAgent(), Permission: m.GetPermission(), PairWithGroup: m.GetPairWithGroup(), Standalone: m.GetStandalone(),
	})
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(appStateProto(state)), nil
}

func (h *handler) ConfirmApp(ctx context.Context, req *connect.Request[v1.ConfirmAppRequest]) (*connect.Response[v1.ConfirmAppResponse], error) {
	subject, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	scenario, err := scenarioFrom(req.Header())
	if err != nil {
		return nil, err
	}
	healthy, err := h.service.ConfirmApp(ctx, subject, scenario, req.Msg.GetEndpoint(), req.Msg.GetPermission(), req.Msg.GetUserAgent())
	if errors.Is(err, hub.ErrAppNotRegistered) {
		return connect.NewResponse(&v1.ConfirmAppResponse{}), nil
	}
	if err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&v1.ConfirmAppResponse{Registered: true, Healthy: healthy}), nil
}

func (h *handler) DisableApp(ctx context.Context, req *connect.Request[v1.DisableAppRequest]) (*connect.Response[v1.DisableAppResponse], error) {
	subject, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	scenario, err := scenarioFrom(req.Header())
	if err != nil {
		return nil, err
	}
	if err := h.service.DisableApp(ctx, subject, scenario, req.Msg.GetEndpoint()); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&v1.DisableAppResponse{}), nil
}

func (h *handler) ListDeviceApps(ctx context.Context, req *connect.Request[v1.ListDeviceAppsRequest]) (*connect.Response[v1.ListDeviceAppsResponse], error) {
	subject, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	devices, err := h.service.ListDeviceApps(ctx, subject)
	if err != nil {
		return nil, mapErr(err)
	}
	out := &v1.ListDeviceAppsResponse{}
	for _, d := range devices {
		dev := &v1.DeviceApps{Group: d.Group, Label: d.Label, LastUsed: d.LastUsed}
		for _, a := range d.Apps {
			dev.Apps = append(dev.Apps, &v1.DeviceAppEntry{SubscriptionId: a.SubscriptionID, Scenario: a.Scenario, DisplayName: a.DisplayName, Healthy: a.Healthy, Status: a.Status, ConfirmedAt: a.ConfirmedAt})
		}
		out.Devices = append(out.Devices, dev)
	}
	return connect.NewResponse(out), nil
}

func (h *handler) RemoveDeviceApp(ctx context.Context, req *connect.Request[v1.RemoveDeviceAppRequest]) (*connect.Response[v1.RemoveDeviceAppResponse], error) {
	subject, err := h.owner(ctx, req.Header())
	if err != nil {
		return nil, err
	}
	if err := h.service.RemoveDeviceApp(ctx, subject, req.Msg.GetSubscriptionId()); err != nil {
		return nil, mapErr(err)
	}
	return connect.NewResponse(&v1.RemoveDeviceAppResponse{}), nil
}

func appStateProto(state hub.AppState) *v1.AppState {
	out := &v1.AppState{
		Scenario: state.Scenario, DisplayName: state.DisplayName, VapidPublicKey: state.VAPIDPublicKey,
		Eligible: state.Eligible, IneligibleReason: state.IneligibleReason, PublicUrl: state.PublicURL,
	}
	if sub := state.Subscription; sub != nil {
		out.Subscription = &v1.AppSubscription{
			Id: sub.ID, DeviceGroup: sub.DeviceGroup, DeviceLabel: sub.DeviceLabel, Permission: sub.Permission,
			ConfirmedAt: sub.ConfirmedAt, Healthy: sub.Healthy, UnhealthyReason: sub.UnhealthyReason, Standalone: sub.Standalone,
		}
	}
	for _, d := range state.Devices {
		out.Devices = append(out.Devices, &v1.DeviceChoice{Group: d.Group, Label: d.Label, LastUsed: d.LastUsed, BestMatch: d.BestMatch, Apps: d.Apps})
	}
	return out
}

var Endpoints = []module.EndpointDescriptor{
	{ID: "app_delivery_state", Path: connectv1.AppDeliveryServiceGetAppStateProcedure, Method: http.MethodPost, Summary: "Read a scenario app's notification state and pairable devices", Category: "appdelivery"},
	{ID: "app_delivery_enable", Path: connectv1.AppDeliveryServiceEnableAppProcedure, Method: http.MethodPost, Summary: "Pair a scenario app with a device so it shows that scenario's notifications", Category: "appdelivery"},
	{ID: "app_delivery_confirm", Path: connectv1.AppDeliveryServiceConfirmAppProcedure, Method: http.MethodPost, Summary: "Record that a scenario app was opened and its notification permission", Category: "appdelivery"},
	{ID: "app_delivery_disable", Path: connectv1.AppDeliveryServiceDisableAppProcedure, Method: http.MethodPost, Summary: "Return a device to Chimesail for the calling scenario", Category: "appdelivery"},
	{ID: "app_delivery_devices", Path: connectv1.AppDeliveryServiceListDeviceAppsProcedure, Method: http.MethodPost, Summary: "List each device and the apps that show notifications on it", Category: "appdelivery"},
	{ID: "app_delivery_remove", Path: connectv1.AppDeliveryServiceRemoveDeviceAppProcedure, Method: http.MethodPost, Summary: "Remove one scenario app from a device", Category: "appdelivery"},
}
