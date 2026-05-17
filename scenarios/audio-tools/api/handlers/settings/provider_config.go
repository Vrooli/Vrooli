package settings

import (
	"context"
	"strings"
	"time"

	"audio-tools/internal/ai/chains"
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

func (h *connectHandler) UpdateProviderConfig(ctx context.Context, req *connect.Request[settv1.UpdateProviderConfigRequest]) (*connect.Response[settv1.UpdateProviderConfigResponse], error) {
	m := req.Msg
	p := store.ProviderConfigPatch{}
	if m.GetHasByokEnabled() {
		v := m.GetByokEnabled()
		p.BYOKEnabled = &v
	}
	if m.GetHasVrooliEnabled() {
		v := m.GetVrooliEnabled()
		p.VrooliEnabled = &v
	}
	if m.GetHasLocalEnabled() {
		v := m.GetLocalEnabled()
		p.LocalEnabled = &v
	}
	if m.GetHasWhisperUrl() {
		v := strings.TrimSpace(m.GetWhisperUrl())
		p.WhisperURL = &v
	}
	if m.GetHasKokoroUrl() {
		v := strings.TrimSpace(m.GetKokoroUrl())
		p.KokoroURL = &v
	}
	if m.GetHasOllamaUrl() {
		v := strings.TrimSpace(m.GetOllamaUrl())
		p.OllamaURL = &v
	}
	if m.GetHasLpbsBaseUrl() {
		v := strings.TrimSpace(m.GetLpbsBaseUrl())
		p.LPBSBaseURL = &v
	}
	if m.GetHasLpbsAppBundleKey() {
		v := strings.TrimSpace(m.GetLpbsAppBundleKey())
		p.LPBSAppBundleKey = &v
	}
	if m.GetHasAvailTtlByokSeconds() {
		v := m.GetAvailTtlByokSeconds()
		p.AvailTTLBYOKSeconds = &v
	}
	if m.GetHasAvailTtlVrooliSeconds() {
		v := m.GetAvailTtlVrooliSeconds()
		p.AvailTTLVrooliSecs = &v
	}
	cfg, err := h.deps.ProviderConfig.Update(ctx, p)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if h.deps.Coordinator != nil {
		h.deps.Coordinator.Reconfigure(chains.Config{
			BYOKEnabled:   cfg.BYOKEnabled,
			VrooliEnabled: cfg.VrooliEnabled,
			LocalEnabled:  cfg.LocalEnabled,
			TTLByOK:       time.Duration(cfg.AvailTTLBYOKSeconds) * time.Second,
			TTLVrooli:     time.Duration(cfg.AvailTTLVrooliSecs) * time.Second,
		})
	}
	return connect.NewResponse(&settv1.UpdateProviderConfigResponse{Config: toProto(cfg)}), nil
}
