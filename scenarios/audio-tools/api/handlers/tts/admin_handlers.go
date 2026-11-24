package tts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"audio-tools/internal/audioformat"
	"audio-tools/internal/capabilities"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"
	inttts "audio-tools/internal/tts"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagnosticsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
)

// GetCache looks up audio bytes previously synthesized for the given
// (event_id, runtime, voice, speed, response_format, version, chunk_index)
// tuple. Runtime is resolved from the configured synthesis route. Content-hash-
// only lookups are unsupported because callers must provide an immutable event
// snapshot and version.
func (h *connectHandler) GetCache(ctx context.Context, req *connect.Request[ttsv1.GetCacheRequest]) (*connect.Response[ttsv1.GetCacheResponse], error) {
	if h.deps.Cache == nil || req.Msg.GetEventId() == "" {
		return connect.NewResponse(&ttsv1.GetCacheResponse{Hit: false}), nil
	}
	key := inttts.CacheKey{
		EventID:        req.Msg.GetEventId(),
		Backend:        h.deps.CacheBackend,
		Voice:          req.Msg.GetVoice(),
		Speed:          req.Msg.GetSpeed(),
		ResponseFormat: h.effectiveResponseFormat(ctx, protomap.ResponseFormatFromProto(req.Msg.GetResponseFormat())),
		Version:        req.Msg.GetVersion(),
		ChunkIndex:     req.Msg.GetChunkIndex(),
	}
	if key.Version == "" {
		key.Version = "active"
	}
	entry, ok := h.deps.Cache.Get(key)
	if !ok && key.Backend == "auto" {
		var backends []string
		if h.deps.Chain != nil {
			backends = h.deps.Chain.LocalCacheBackends()
		}
		for _, backend := range backends {
			key.Backend = backend
			if entry, ok = h.deps.Cache.Get(key); ok {
				break
			}
		}
	}
	if !ok {
		return connect.NewResponse(&ttsv1.GetCacheResponse{Hit: false}), nil
	}
	return connect.NewResponse(&ttsv1.GetCacheResponse{
		Audio: entry.Audio, ContentType: entry.ContentType,
		ContentHash: req.Msg.GetContentHash(), Hit: true,
	}), nil
}

func (h *connectHandler) GetConfig(ctx context.Context, _ *connect.Request[ttsv1.GetConfigRequest]) (*connect.Response[ttsv1.GetConfigResponse], error) {
	cfg := h.loadConfig(ctx)
	return connect.NewResponse(&ttsv1.GetConfigResponse{Config: configToProto(cfg)}), nil
}

var ttsConfigAllowedPaths = map[string]struct{}{
	"auto_enabled":            {},
	"default_voice":           {},
	"default_speed":           {},
	"default_response_format": {},
}

func (h *connectHandler) UpdateConfig(ctx context.Context, req *connect.Request[ttsv1.UpdateConfigRequest]) (*connect.Response[ttsv1.UpdateConfigResponse], error) {
	cfg := h.loadConfig(ctx)
	mask := req.Msg.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask required"))
	}
	if bad := protomap.MaskPathsOutsideAllowed(mask, ttsConfigAllowedPaths); len(bad) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown update_mask paths: %v", bad))
	}
	p := req.Msg.GetConfig()
	if protomap.MaskHas(mask, "auto_enabled") {
		cfg.AutoEnabled = p.GetAutoEnabled()
	}
	if protomap.MaskHas(mask, "default_voice") {
		cfg.KokoroVoice = p.GetDefaultVoice()
	}
	if protomap.MaskHas(mask, "default_speed") {
		cfg.KokoroSpeed = p.GetDefaultSpeed()
	}
	if protomap.MaskHas(mask, "default_response_format") {
		format := protomap.ResponseFormatFromProto(p.GetDefaultResponseFormat())
		if format == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("default_response_format required when updating this field"))
		}
		if _, ok := audioformat.OutputFormatFromString(format); !ok {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unsupported default_response_format %q", format))
		}
		cfg.DefaultResponseFormat = format
	}
	if h.deps.ConfigStore != nil {
		raw, _ := json.Marshal(cfg)
		// Preserve existing summarize-config blob untouched; SummarizeService
		// owns it now.
		_, existingSumm, _, _ := h.deps.ConfigStore.Get(ctx)
		if err := h.deps.ConfigStore.Set(ctx, string(raw), existingSumm); err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&ttsv1.UpdateConfigResponse{Config: configToProto(cfg)}), nil
}

func (h *connectHandler) GetStatus(ctx context.Context, _ *connect.Request[ttsv1.GetStatusRequest]) (*connect.Response[ttsv1.GetStatusResponse], error) {
	cfg := h.loadConfig(ctx)
	avail := []*sharedv1.ProviderHealth{}
	localProviderID := ""
	if h.deps.Chain != nil {
		p := h.deps.Chain.Probe(ctx)
		localProviderID = capabilities.CanonicalProviderID(h.deps.Chain.LocalProviderID())
		ts := h.deps.Clock.Now().UTC().Format(time.RFC3339Nano)
		avail = []*sharedv1.ProviderHealth{
			ttsProviderHealth(protomap.ProviderTierToProto("local"), localProviderID, p.Local, ts),
			h.ttsBYOKProviderHealth(ctx, p.BYOK, h.deps.Chain.BYOKAdapterConfigured(), ts),
			ttsProviderHealth(protomap.ProviderTierToProto("vrooli"), "", p.Vrooli, ts),
		}
	}
	capLabel := "unavailable"
	if len(avail) > 0 && avail[0].GetState() == sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE {
		capLabel = "available"
	}
	return connect.NewResponse(&ttsv1.GetStatusResponse{Status: &ttsv1.Status{
		Config:          configToProto(cfg),
		Availability:    avail,
		Capability:      capLabel,
		CapabilityLabel: "TTS (Local)",
	}}), nil
}

func (h *connectHandler) ttsBYOKProviderHealth(ctx context.Context, routeEnabled, adapterConfigured bool, checkedAt string) *sharedv1.ProviderHealth {
	item := &sharedv1.ProviderHealth{
		Capability:    diagnosticsv1.Capability_CAPABILITY_TTS,
		Tier:          protomap.ProviderTierToProto("byok"),
		State:         sharedv1.ProviderState_PROVIDER_STATE_UNKNOWN,
		LastCheckedAt: checkedAt,
	}
	if !adapterConfigured {
		item.ErrorCode = "adapter_not_configured"
		item.ErrorMessage = "no TTS BYOK adapter is registered"
		return item
	}
	if !routeEnabled {
		item.State = sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE
		item.ErrorCode = "routing_disabled"
		item.ErrorMessage = "TTS BYOK adapters are registered, but the BYOK routing tier is disabled"
		return item
	}
	if h.deps.DefaultCredential == nil {
		item.ErrorCode = "credential_status_unknown"
		item.ErrorMessage = "TTS BYOK adapters are registered; credential presence could not be checked"
		return item
	}
	providerID, key, configured := h.deps.DefaultCredential(ctx, "tts")
	if !configured || key == "" {
		item.State = sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE
		item.ErrorCode = "credential_not_configured"
		item.ErrorMessage = "TTS BYOK adapters are registered, but no TTS credential is configured"
		return item
	}
	item.ProviderId = providerID
	item.ErrorCode = "credential_present_unverified"
	item.ErrorMessage = "TTS credential is configured; provider usability was not checked"
	return item
}

// ttsProviderHealth builds a ProviderHealth row for the TTS Status
// surface. capability is always CAPABILITY_TTS here; latency/error/
// serving fields are left at their zero values since the legacy
// chain.Probe shape does not surface them.
func ttsProviderHealth(tier commonv1.ProviderTier, providerID string, available bool, checkedAt string) *sharedv1.ProviderHealth {
	state := sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE
	if available {
		state = sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE
	}
	return &sharedv1.ProviderHealth{
		Capability:    diagnosticsv1.Capability_CAPABILITY_TTS,
		Tier:          tier,
		ProviderId:    providerID,
		State:         state,
		LastCheckedAt: checkedAt,
	}
}

func (h *connectHandler) RecordPlaybackEvent(ctx context.Context, req *connect.Request[ttsv1.RecordPlaybackEventRequest]) (*connect.Response[ttsv1.RecordPlaybackEventResponse], error) {
	if h.deps.Playback == nil {
		return connect.NewResponse(&ttsv1.RecordPlaybackEventResponse{Status: "noop"}), nil
	}
	ev := req.Msg.GetEvent()
	id := ev.GetEventId()
	if id == "" {
		id = uuid.NewString()
	}
	if err := h.deps.Playback.Insert(ctx, store.PlaybackEvent{
		EventID: id, Kind: ev.GetStage(), Voice: ev.GetBackend(),
	}); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&ttsv1.RecordPlaybackEventResponse{Status: "recorded"}), nil
}

// loadConfig prefers the persisted config; falls back to defaults.
func (h *connectHandler) loadConfig(ctx context.Context) inttts.Config {
	cfg := inttts.DefaultConfig()
	if h.deps.ConfigStore == nil {
		return cfg
	}
	rawCfg, _, ok, err := h.deps.ConfigStore.Get(ctx)
	if err != nil || !ok {
		return cfg
	}
	_ = json.Unmarshal([]byte(rawCfg), &cfg)
	return cfg
}

func (h *connectHandler) effectiveResponseFormat(ctx context.Context, requested string) string {
	if requested != "" {
		return requested
	}
	if configured := h.loadConfig(ctx).DefaultResponseFormat; configured != "" {
		return configured
	}
	return inttts.DefaultResponseFormat
}
