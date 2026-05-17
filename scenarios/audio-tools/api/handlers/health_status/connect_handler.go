package health_status

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	"audio-tools/internal/capabilities"

	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	hsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
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
	states := h.deps.Registry.Resolve(ctx)
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
			states := h.deps.Registry.Resolve(ctx)
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
	return int32(d / time.Second)
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
	type key = diagv1.Capability
	byCap := make(map[key][]*hsv1.ProviderHealth)
	order := make([]key, 0, 4)

	for _, st := range states {
		// Skip the rollup pseudo-entry — it's a scenario advertisement,
		// not a real provider row.
		if st.Def.ID == "audio-tools" {
			continue
		}
		seen := make(map[key]struct{})
		for _, feat := range st.Def.Features {
			cap, ok := capabilities.CapabilityForFeature(feat)
			if !ok {
				continue
			}
			if _, dup := seen[cap]; dup {
				continue
			}
			seen[cap] = struct{}{}

			row := &hsv1.ProviderHealth{
				Capability:    cap,
				Tier:          capabilities.TierForProviderID(st.Def.ID),
				ProviderId:    st.Def.ID,
				State:         stateToProto(st.Status),
				LastCheckedAt: st.CheckedAt,
				// TODO(phase2): wire routing visibility for `serving`.
				Serving: false,
				// TODO(phase2): registry doesn't carry latency yet.
				LatencyMs: 0,
			}
			if st.Status == capabilities.StatusUnavailable {
				row.ErrorCode = "provider_unavailable"
				row.ErrorMessage = st.Message
			}

			if _, ok := byCap[cap]; !ok {
				order = append(order, cap)
			}
			byCap[cap] = append(byCap[cap], row)
		}
	}

	out := make([]*hsv1.CapabilityHealth, 0, len(order))
	for _, cap := range order {
		providers := byCap[cap]
		out = append(out, &hsv1.CapabilityHealth{
			Capability:     cap,
			Providers:      providers,
			EffectiveState: rollup(providers),
		})
	}
	return out
}

func rollup(providers []*hsv1.ProviderHealth) hsv1.State {
	hasUnknown := false
	hasUnavailable := false
	for _, p := range providers {
		switch p.GetState() {
		case hsv1.State_STATE_AVAILABLE:
			return hsv1.State_STATE_AVAILABLE
		case hsv1.State_STATE_UNAVAILABLE:
			hasUnavailable = true
		default:
			hasUnknown = true
		}
	}
	if hasUnavailable {
		return hsv1.State_STATE_UNAVAILABLE
	}
	if hasUnknown {
		return hsv1.State_STATE_UNKNOWN
	}
	return hsv1.State_STATE_UNSPECIFIED
}

func stateToProto(s capabilities.Status) hsv1.State {
	switch s {
	case capabilities.StatusAvailable:
		return hsv1.State_STATE_AVAILABLE
	case capabilities.StatusUnavailable:
		return hsv1.State_STATE_UNAVAILABLE
	case capabilities.StatusUnknown:
		return hsv1.State_STATE_UNKNOWN
	}
	return hsv1.State_STATE_UNSPECIFIED
}
