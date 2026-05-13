package capabilities

import (
	"context"
	"log"

	"connectrpc.com/connect"

	capabilitiesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/capabilities"
)

// Deps wires the seams the Connect capabilities handler needs.
type Deps struct {
	Service Service
	Logger  *log.Logger
}

type connectHandler struct {
	deps Deps
}

// NewConnectHandler constructs the Connect-RPC handler implementing
// CapabilitiesServiceHandler.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

func (h *connectHandler) Get(ctx context.Context, _ *connect.Request[capabilitiesv1.GetRequest]) (*connect.Response[capabilitiesv1.GetResponse], error) {
	snap := h.deps.Service.Resolve(ctx)
	resp := &capabilitiesv1.GetResponse{
		Capabilities:    statesToProto(snap.Capabilities),
		Timestamp:       snap.Timestamp,
		SessionBackends: backendsToProto(snap.BackendOptions),
		DefaultBackend:  snap.DefaultBackend,
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) Liveness(ctx context.Context, _ *connect.Request[capabilitiesv1.LivenessRequest]) (*connect.Response[capabilitiesv1.LivenessResponse], error) {
	snap := h.deps.Service.Liveness(ctx)
	resp := &capabilitiesv1.LivenessResponse{
		Capabilities: statesToProto(snap.Capabilities),
		Timestamp:    snap.Timestamp,
	}
	return connect.NewResponse(resp), nil
}

func statesToProto(in []CapabilityState) []*capabilitiesv1.CapabilityState {
	out := make([]*capabilitiesv1.CapabilityState, len(in))
	for i, s := range in {
		out[i] = &capabilitiesv1.CapabilityState{
			Id:             s.ID,
			Name:           s.Name,
			Description:    s.Description,
			DependencyKind: s.DependencyKind,
			DependencySlug: s.DependencySlug,
			Features:       s.Features,
			Status:         s.Status,
			Message:        s.Message,
			CheckedAt:      s.CheckedAt,
		}
	}
	return out
}

func backendsToProto(in []BackendOption) []*capabilitiesv1.BackendOption {
	out := make([]*capabilitiesv1.BackendOption, len(in))
	for i, b := range in {
		out[i] = &capabilitiesv1.BackendOption{
			Id:              b.ID,
			DisplayName:     b.DisplayName,
			Description:     b.Description,
			SurvivesRestart: b.SurvivesRestart,
			Available:       b.Available,
			Reason:          b.Reason,
		}
	}
	return out
}
