package adapters

import (
	"context"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	domainadapters "network-manager/internal/adapters"
	"network-manager/internal/module"

	adaptersv1 "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters"
	adaptersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/network-manager/v1/adapters/adapters_v1connect"
)

type handler struct {
	service *domainadapters.Service
}

func Module(db domainadapters.SQLExecutor) module.Module {
	service := domainadapters.NewService(domainadapters.Config{
		Repo:     domainadapters.NewSQLiteRepository(db),
		Registry: domainadapters.NewStaticRegistry(),
	})
	path, h := adaptersconnect.NewAdapterServiceHandler(&handler{service: service})
	return module.Module{Name: "adapters", Mount: func(r *mux.Router) {
		connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: h})
	}, Endpoints: Endpoints}
}

func Schema() string { return domainadapters.Schema() }

func (h *handler) ListCapabilities(ctx context.Context, _ *connect.Request[adaptersv1.ListCapabilitiesRequest]) (*connect.Response[adaptersv1.ListCapabilitiesResponse], error) {
	caps, err := h.service.ListCapabilities(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*adaptersv1.Capability, 0, len(caps))
	for _, cap := range caps {
		out = append(out, toProtoCapability(cap))
	}
	return connect.NewResponse(&adaptersv1.ListCapabilitiesResponse{Capabilities: out}), nil
}

func (h *handler) ExplainUnsupportedAction(ctx context.Context, req *connect.Request[adaptersv1.ExplainUnsupportedActionRequest]) (*connect.Response[adaptersv1.ExplainUnsupportedActionResponse], error) {
	cap, steps, err := h.service.ExplainUnsupportedAction(ctx, req.Msg.GetAction())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adaptersv1.ExplainUnsupportedActionResponse{Capability: toProtoCapability(cap), ManualSteps: steps}), nil
}

func (h *handler) GetPlatformSummary(ctx context.Context, _ *connect.Request[adaptersv1.GetPlatformSummaryRequest]) (*connect.Response[adaptersv1.GetPlatformSummaryResponse], error) {
	summary, err := h.service.PlatformSummary(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&adaptersv1.GetPlatformSummaryResponse{Summary: &adaptersv1.PlatformSummary{Os: summary.OS, Arch: summary.Arch, Profile: summary.Profile, Notes: summary.Notes}}), nil
}

func toProtoCapability(cap domainadapters.Capability) *adaptersv1.Capability {
	return &adaptersv1.Capability{
		Adapter:           cap.Adapter,
		Action:            cap.Action,
		Supported:         cap.Supported,
		RequiresAdmin:     cap.RequiresAdmin,
		RollbackSupported: cap.RollbackSupported,
		Reason:            cap.Reason,
	}
}

var Endpoints = []module.EndpointDescriptor{
	connectEndpoint("adapters_capabilities", adaptersconnect.AdapterServiceListCapabilitiesProcedure, "List adapter capabilities"),
	connectEndpoint("adapters_explain_unsupported", adaptersconnect.AdapterServiceExplainUnsupportedActionProcedure, "Explain unsupported adapter action"),
	connectEndpoint("adapters_platform", adaptersconnect.AdapterServiceGetPlatformSummaryProcedure, "Get platform summary"),
}

func connectEndpoint(id, path, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: "POST", Summary: summary, Category: "adapters", Request: &module.Schema{Type: "object", Properties: map[string]string{}}, Response: &module.Schema{Type: "object", Properties: map[string]string{"capabilities": "array<Capability>"}}}
}
