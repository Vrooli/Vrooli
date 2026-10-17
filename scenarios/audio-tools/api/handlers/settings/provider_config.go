package settings

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"audio-tools/internal/ai/chains"
	"audio-tools/internal/protomap"
	"audio-tools/internal/store"

	"connectrpc.com/connect"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
)

func (h *connectHandler) GetProviderConfig(ctx context.Context, _ *connect.Request[settv1.GetProviderConfigRequest]) (*connect.Response[settv1.GetProviderConfigResponse], error) {
	cfg, err := h.deps.ProviderConfig.Get(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&settv1.GetProviderConfigResponse{Config: toProto(cfg)}), nil
}

var providerConfigAllowedPaths = map[string]struct{}{
	"byok_enabled":             {},
	"vrooli_enabled":           {},
	"local_enabled":            {},
	"whisper_url":              {},
	"kokoro_url":               {},
	"ollama_url":               {},
	"lpbs_base_url":            {},
	"lpbs_app_bundle_key":      {},
	"avail_ttl_byok_seconds":   {},
	"avail_ttl_vrooli_seconds": {},
}

func (h *connectHandler) UpdateProviderConfig(ctx context.Context, req *connect.Request[settv1.UpdateProviderConfigRequest]) (*connect.Response[settv1.UpdateProviderConfigResponse], error) {
	m := req.Msg
	mask := m.GetUpdateMask()
	if mask == nil || len(mask.GetPaths()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("update_mask required"))
	}
	if bad := protomap.MaskPathsOutsideAllowed(mask, providerConfigAllowedPaths); len(bad) > 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("unknown update_mask paths: %v", bad))
	}
	cfg := m.GetConfig()
	p := store.ProviderConfigPatch{}
	if protomap.MaskHas(mask, "byok_enabled") {
		v := cfg.GetByokEnabled()
		p.BYOKEnabled = &v
	}
	if protomap.MaskHas(mask, "vrooli_enabled") {
		v := cfg.GetVrooliEnabled()
		p.VrooliEnabled = &v
	}
	if protomap.MaskHas(mask, "local_enabled") {
		v := cfg.GetLocalEnabled()
		p.LocalEnabled = &v
	}
	if protomap.MaskHas(mask, "whisper_url") {
		v := strings.TrimSpace(cfg.GetWhisperUrl())
		p.WhisperURL = &v
	}
	if protomap.MaskHas(mask, "kokoro_url") {
		v := strings.TrimSpace(cfg.GetKokoroUrl())
		p.KokoroURL = &v
	}
	if protomap.MaskHas(mask, "ollama_url") {
		v := strings.TrimSpace(cfg.GetOllamaUrl())
		p.OllamaURL = &v
	}
	if protomap.MaskHas(mask, "lpbs_base_url") {
		v := strings.TrimSpace(cfg.GetLpbsBaseUrl())
		p.LPBSBaseURL = &v
	}
	if protomap.MaskHas(mask, "lpbs_app_bundle_key") {
		v := strings.TrimSpace(cfg.GetLpbsAppBundleKey())
		p.LPBSAppBundleKey = &v
	}
	if protomap.MaskHas(mask, "avail_ttl_byok_seconds") {
		v := cfg.GetAvailTtlByokSeconds()
		p.AvailTTLBYOKSeconds = &v
	}
	if protomap.MaskHas(mask, "avail_ttl_vrooli_seconds") {
		v := cfg.GetAvailTtlVrooliSeconds()
		p.AvailTTLVrooliSecs = &v
	}
	out, err := h.deps.ProviderConfig.Update(ctx, p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if h.deps.Coordinator != nil {
		h.deps.Coordinator.Reconfigure(chains.Config{
			BYOKEnabled:   out.BYOKEnabled,
			VrooliEnabled: out.VrooliEnabled,
			LocalEnabled:  out.LocalEnabled,
			TTLByOK:       time.Duration(out.AvailTTLBYOKSeconds) * time.Second,
			TTLVrooli:     time.Duration(out.AvailTTLVrooliSecs) * time.Second,
		})
	}
	return connect.NewResponse(&settv1.UpdateProviderConfigResponse{Config: toProto(out)}), nil
}
