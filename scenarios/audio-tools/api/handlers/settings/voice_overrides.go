package settings

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"audio-tools/internal/providercatalog"
	"audio-tools/internal/store"
	"audio-tools/internal/tts"

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

// validateVoiceOverridePair rejects overrides that can never be applied: an
// unknown canonical voice, or a tier_provider that is not a declared TTS
// adapter. Stored rows with a bad pair would look configured in the UI while
// every synthesis silently fell back to the adapter default.
func validateVoiceOverridePair(canonicalVoice, tierProvider string) error {
	knownVoice := false
	for _, id := range tts.CanonicalVoiceIDs {
		if id == canonicalVoice {
			knownVoice = true
			break
		}
	}
	if !knownVoice {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown canonical voice %q", canonicalVoice))
	}
	for _, provider := range providercatalog.ByCapability(providercatalog.CapabilityTTS) {
		if provider.AdapterID == tierProvider {
			return nil
		}
	}
	return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown TTS adapter %q", tierProvider))
}

func (h *connectHandler) SetVoiceOverride(ctx context.Context, req *connect.Request[settv1.SetVoiceOverrideRequest]) (*connect.Response[settv1.SetVoiceOverrideResponse], error) {
	if h.deps.VoiceOverrides == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("voice override store not configured"))
	}
	o := req.Msg.GetOverride()
	if o == nil || strings.TrimSpace(o.GetCanonicalVoice()) == "" || strings.TrimSpace(o.GetTierProvider()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("canonical_voice and tier_provider required"))
	}
	if err := validateVoiceOverridePair(o.GetCanonicalVoice(), o.GetTierProvider()); err != nil {
		return nil, err
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
