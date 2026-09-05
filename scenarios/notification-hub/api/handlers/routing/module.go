package routing

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/gorilla/mux"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/routing"
	connectv1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/routing/routing_v1connect"
	shared "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/shared"

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
	return module.Module{Name: "routing", Mount: func(r *mux.Router) {
		path, svc := connectv1.NewRoutingServiceHandler(h)
		r.PathPrefix(path).Handler(svc)
	}, Endpoints: Endpoints}
}

func (h *handler) ChannelsStatus(ctx context.Context, req *connect.Request[v1.ChannelsStatusRequest]) (*connect.Response[v1.ChannelsStatusResponse], error) {
	subject, err := identity.Subject(ctx, req.Header(), h.verifier)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("verified owner identity is required"))
	}
	items, err := h.service.ChannelsStatus(ctx, subject, req.Msg.GetMachineId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ChannelsStatusResponse{}
	for _, item := range items {
		d := shared.ChannelDisposition_CHANNEL_DISPOSITION_UNKNOWN
		if item.Disposition == "ready" {
			d = shared.ChannelDisposition_CHANNEL_DISPOSITION_READY
		} else if item.Disposition == "not_configured" {
			d = shared.ChannelDisposition_CHANNEL_DISPOSITION_NOT_CONFIGURED
		}
		out.Channels = append(out.Channels, &shared.ChannelStatus{Channel: item.Channel, MachineId: item.MachineID, Disposition: d, Reason: item.Reason, ObservedAt: item.ObservedAt})
	}
	return connect.NewResponse(out), nil
}

var Endpoints = []module.EndpointDescriptor{{ID: "routing_channels_status", Path: connectv1.RoutingServiceChannelsStatusProcedure, Method: http.MethodPost, Summary: "Report this machine's channel dispositions", Category: "routing"}}
