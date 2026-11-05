package settings

import (
	"context"
	"errors"
	"strings"

	"audio-tools/internal/store"

	"connectrpc.com/connect"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
)

func (h *connectHandler) GetVoiceOverrides(ctx context.Context, _ *connect.Request[settv1.GetVoiceOverridesRequest]) (*connect.Response[settv1.GetVoiceOverridesResponse], error) {
	if h.deps.VoiceOverrides == nil {
		return connect.NewResponse(&settv1.GetVoiceOverridesResponse{}), nil
	}
	rows, err := h.deps.VoiceOverrides.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&settv1.GetVoiceOverridesResponse{Overrides: voiceOverridesToProto(rows)}), nil
}

func (h *connectHandler) SetVoiceOverride(ctx context.Context, req *connect.Request[settv1.SetVoiceOverrideRequest]) (*connect.Response[settv1.SetVoiceOverrideResponse], error) {
	if h.deps.VoiceOverrides == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("voice override store not configured"))
	}
	o := req.Msg.GetOverride()
	if o == nil || strings.TrimSpace(o.GetCanonicalVoice()) == "" || strings.TrimSpace(o.GetTierProvider()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("canonical_voice and tier_provider required"))
	}
	if err := h.deps.VoiceOverrides.Set(ctx, store.VoiceOverride{
		CanonicalVoice: o.GetCanonicalVoice(),
		TierProvider:   o.GetTierProvider(),
		AdapterVoice:   strings.TrimSpace(o.GetAdapterVoice()),
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	rows, err := h.deps.VoiceOverrides.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&settv1.SetVoiceOverrideResponse{Overrides: voiceOverridesToProto(rows)}), nil
}
