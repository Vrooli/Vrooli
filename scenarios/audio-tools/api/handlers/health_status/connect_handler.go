package health_status

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	"audio-tools/internal/capabilities"
	"audio-tools/internal/protoint"

	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"
	hsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"
)

// minStreamTick clamps StreamProviderHealth's tick interval so a
// pathologically low cache TTL cannot pin a CPU.
const minStreamTick = 5 * time.Second

type connectHandler struct {
	deps Deps
}

// NewConnectHandler builds the Connect handler. Deps.Registry,
// Deps.Logger, and Deps.Clock are all required; no fallbacks.
func NewConnectHandler(d Deps) *connectHandler {
	return &connectHandler{deps: d}
}

func (h *connectHandler) GetProviderHealth(ctx context.Context, _ *connect.Request[hsv1.GetProviderHealthRequest]) (*connect.Response[hsv1.GetProviderHealthResponse], error) {
	if h.deps.Registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capabilities registry not configured"))
	}
	states := h.deps.Registry.ResolveLiveness(ctx)
	resp := &hsv1.GetProviderHealthResponse{
		Capabilities:    buildCapabilities(states),
		GeneratedAt:     h.deps.Clock.Now().UTC().Format(time.RFC3339),
		CacheTtlSeconds: ttlSeconds(h.deps.Registry.CacheTTL()),
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) RefreshProviderHealth(ctx context.Context, _ *connect.Request[hsv1.RefreshProviderHealthRequest]) (*connect.Response[hsv1.RefreshProviderHealthResponse], error) {
	if h.deps.Registry == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("capabilities registry not configured"))
	}
	states := h.deps.Registry.ResolveForce(ctx)
	resp := &hsv1.RefreshProviderHealthResponse{
		Capabilities:    buildCapabilities(states),
		GeneratedAt:     h.deps.Clock.Now().UTC().Format(time.RFC3339),
		CacheTtlSeconds: ttlSeconds(h.deps.Registry.CacheTTL()),
	}
	return connect.NewResponse(resp), nil
}

func (h *connectHandler) StreamProviderHealth(ctx context.Context, _ *connect.Request[hsv1.StreamProviderHealthRequest], stream *connect.ServerStream[hsv1.ProviderHealthEvent]) error {
	if h.deps.Registry == nil {
		return connect.NewError(connect.CodeUnavailable, errors.New("capabilities registry not configured"))
	}
	tick := h.deps.Registry.CacheTTL() / 2
	if tick < minStreamTick {
		tick = minStreamTick
	}
	t := time.NewTicker(tick)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			states := h.deps.Registry.ResolveLiveness(ctx)
			event := &hsv1.ProviderHealthEvent{
				GeneratedAt:  h.deps.Clock.Now().UTC().Format(time.RFC3339),
				Capabilities: buildCapabilities(states),
			}
			if err := stream.Send(event); err != nil {
				return err
			}
		}
	}
}

func ttlSeconds(d time.Duration) int32 {
	if d <= 0 {
		return 0
	}
	return protoint.FromInt64(int64(d / time.Second))
}

// buildCapabilities expands the registry State slice into per-capability
// rollups. Each State may map to zero (rollup entry / no feature
// mapping) or more capabilities — one ProviderHealth row per
// (provider, mapped capability) pair.
//
// Phase-1 fields TODO:
//   - serving: requires routing visibility, not yet plumbed.
//   - latency_ms: registry.State doesn't carry per-check latency yet.
func buildCapabilities(states []capabilities.State) []*hsv1.CapabilityHealth {
	groups := capabilities.Serviceability(states)
	out := make([]*hsv1.CapabilityHealth, 0, len(groups))
	for _, group := range groups {
		providers := make([]*sharedv1.ProviderHealth, 0, len(group.Providers))
		for _, st := range group.Providers {
			row := &sharedv1.ProviderHealth{
				Capability:    group.Capability,
				Tier:          capabilities.TierForProviderID(st.Def.ID),
				ProviderId:    st.Def.ID,
				State:         stateToProto(st.Status),
				LastCheckedAt: st.CheckedAt,
				Serving:       false,
				LatencyMs:     0,
			}
			if st.Status == capabilities.StatusUnavailable {
				row.ErrorCode = "provider_unavailable"
				row.ErrorMessage = st.Message
			}
			providers = append(providers, row)
		}
		out = append(out, &hsv1.CapabilityHealth{
			Capability:     group.Capability,
			Providers:      providers,
			EffectiveState: effectiveState(group.Providers),
		})
	}
	return out
}

func effectiveState(states []capabilities.State) sharedv1.ProviderState {
	if capabilityregistry.Serviceable(states) {
		return sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE
	}
	hasUnknown := false
	hasUnavailable := false
	for _, state := range states {
		switch state.Status {
		case capabilities.StatusUnavailable:
			hasUnavailable = true
		case capabilities.StatusUnknown:
			hasUnknown = true
		}
	}
	if hasUnavailable {
		return sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE
	}
	if hasUnknown {
		return sharedv1.ProviderState_PROVIDER_STATE_UNKNOWN
	}
	return sharedv1.ProviderState_PROVIDER_STATE_UNSPECIFIED
}

func stateToProto(s capabilities.Status) sharedv1.ProviderState {
	switch s {
	case capabilities.StatusAvailable:
		return sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE
	case capabilities.StatusUnavailable:
		return sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE
	case capabilities.StatusUnknown:
		return sharedv1.ProviderState_PROVIDER_STATE_UNKNOWN
	}
	return sharedv1.ProviderState_PROVIDER_STATE_UNSPECIFIED
}
