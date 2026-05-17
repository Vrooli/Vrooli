// Package settings hosts the SettingsService Connect-RPC handler.
package settings

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"audio-tools/internal/ai/chains"
	"audio-tools/internal/byokstore"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/store"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
)

type Deps struct {
	Logger         *log.Logger
	ProviderConfig *store.ProviderConfigStore
	BYOK           *byokstore.Store
	VoiceOverrides *store.VoiceOverrideStore
	Coordinator    *chains.Coordinator
}

type connectHandler struct{ deps Deps }

// NewConnectHandler returns the live Connect handler. Caller is
// responsible for wiring the dependencies.
func NewConnectHandler(d Deps) *connectHandler {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	return &connectHandler{deps: d}
}

var Endpoints = []modulekit.EndpointDescriptor{
	{ID: "settings.get_provider_config", Path: "/vrooli.audio_tools.v1.settings.SettingsService/GetProviderConfig", Method: "POST", Category: "settings"},
	{ID: "settings.update_provider_config", Path: "/vrooli.audio_tools.v1.settings.SettingsService/UpdateProviderConfig", Method: "POST", Category: "settings"},
	{ID: "settings.list_byok_credentials", Path: "/vrooli.audio_tools.v1.settings.SettingsService/ListBYOKCredentials", Method: "POST", Category: "settings"},
	{ID: "settings.upsert_byok_credential", Path: "/vrooli.audio_tools.v1.settings.SettingsService/UpsertBYOKCredential", Method: "POST", Category: "settings"},
	{ID: "settings.delete_byok_credential", Path: "/vrooli.audio_tools.v1.settings.SettingsService/DeleteBYOKCredential", Method: "POST", Category: "settings"},
	{ID: "settings.get_voice_overrides", Path: "/vrooli.audio_tools.v1.settings.SettingsService/GetVoiceOverrides", Method: "POST", Category: "settings"},
	{ID: "settings.set_voice_override", Path: "/vrooli.audio_tools.v1.settings.SettingsService/SetVoiceOverride", Method: "POST", Category: "settings"},
}

func Module(d Deps) modulekit.Module {
	if d.Logger == nil {
		d.Logger = log.Default()
	}
	connectPath, h := settconnect.NewSettingsServiceHandler(NewConnectHandler(d))
	return modulekit.Module{
		Name: "settings",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: h})
		},
		Endpoints: Endpoints,
	}
}

func Schema() string { return "" }

// --------------------------------------------------------------------
// Provider config
// --------------------------------------------------------------------

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

func toProto(c store.ProviderConfig) *settv1.ProviderConfig {
	return &settv1.ProviderConfig{
		ByokEnabled:           c.BYOKEnabled,
		VrooliEnabled:         c.VrooliEnabled,
		LocalEnabled:          c.LocalEnabled,
		WhisperUrl:            c.WhisperURL,
		KokoroUrl:             c.KokoroURL,
		OllamaUrl:             c.OllamaURL,
		LpbsBaseUrl:           c.LPBSBaseURL,
		LpbsAppBundleKey:      c.LPBSAppBundleKey,
		AvailTtlByokSeconds:   c.AvailTTLBYOKSeconds,
		AvailTtlVrooliSeconds: c.AvailTTLVrooliSecs,
	}
}

// --------------------------------------------------------------------
// BYOK credentials
// --------------------------------------------------------------------

func (h *connectHandler) ListBYOKCredentials(ctx context.Context, _ *connect.Request[settv1.ListBYOKCredentialsRequest]) (*connect.Response[settv1.ListBYOKCredentialsResponse], error) {
	if h.deps.BYOK == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("byok store not configured"))
	}
	creds, err := h.deps.BYOK.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*settv1.BYOKCredentialSummary, 0, len(creds))
	for _, c := range creds {
		out = append(out, credToProto(c))
	}
	return connect.NewResponse(&settv1.ListBYOKCredentialsResponse{Credentials: out}), nil
}

func (h *connectHandler) UpsertBYOKCredential(ctx context.Context, req *connect.Request[settv1.UpsertBYOKCredentialRequest]) (*connect.Response[settv1.UpsertBYOKCredentialResponse], error) {
	if h.deps.BYOK == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("byok store not configured"))
	}
	m := req.Msg
	if err := validateCapability(m.GetCapability()); err != nil {
		return nil, err
	}
	if strings.TrimSpace(m.GetProviderId()) == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("provider_id required"))
	}
	key := strings.TrimSpace(m.GetApiKey())
	if key == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("api_key required"))
	}
	c, err := h.deps.BYOK.Upsert(ctx, m.GetProviderId(), m.GetCapability(), key)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&settv1.UpsertBYOKCredentialResponse{Credential: credToProto(byokstore.Credential{
		ProviderID:  c.ProviderID,
		Capability:  c.Capability,
		Fingerprint: c.Fingerprint,
		CreatedAt:   c.CreatedAt,
		LastUsedAt:  c.LastUsedAt,
	})}), nil
}

func (h *connectHandler) DeleteBYOKCredential(ctx context.Context, req *connect.Request[settv1.DeleteBYOKCredentialRequest]) (*connect.Response[settv1.DeleteBYOKCredentialResponse], error) {
	if h.deps.BYOK == nil {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("byok store not configured"))
	}
	m := req.Msg
	if err := validateCapability(m.GetCapability()); err != nil {
		return nil, err
	}
	if _, err := h.deps.BYOK.Delete(ctx, m.GetProviderId(), m.GetCapability()); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&settv1.DeleteBYOKCredentialResponse{}), nil
}

func validateCapability(cap string) error {
	switch cap {
	case "stt", "tts", "summarize":
		return nil
	default:
		return connect.NewError(connect.CodeInvalidArgument, errors.New("capability must be one of stt|tts|summarize"))
	}
}

func credToProto(c byokstore.Credential) *settv1.BYOKCredentialSummary {
	out := &settv1.BYOKCredentialSummary{
		ProviderId:  c.ProviderID,
		Capability:  c.Capability,
		Fingerprint: c.Fingerprint,
	}
	if !c.CreatedAt.IsZero() {
		out.CreatedAt = c.CreatedAt.UTC().Format(time.RFC3339)
	}
	if c.LastUsedAt != nil && !c.LastUsedAt.IsZero() {
		out.LastUsedAt = c.LastUsedAt.UTC().Format(time.RFC3339)
	}
	return out
}

// --------------------------------------------------------------------
// Voice overrides
// --------------------------------------------------------------------

func (h *connectHandler) GetVoiceOverrides(ctx context.Context, _ *connect.Request[settv1.GetVoiceOverridesRequest]) (*connect.Response[settv1.GetVoiceOverridesResponse], error) {
	if h.deps.VoiceOverrides == nil {
		return connect.NewResponse(&settv1.GetVoiceOverridesResponse{}), nil
	}
	rows, err := h.deps.VoiceOverrides.List(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*settv1.VoiceOverride, 0, len(rows))
	for _, v := range rows {
		out = append(out, &settv1.VoiceOverride{
			CanonicalVoice: v.CanonicalVoice,
			TierProvider:   v.TierProvider,
			AdapterVoice:   v.AdapterVoice,
		})
	}
	return connect.NewResponse(&settv1.GetVoiceOverridesResponse{Overrides: out}), nil
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
	out := make([]*settv1.VoiceOverride, 0, len(rows))
	for _, v := range rows {
		out = append(out, &settv1.VoiceOverride{
			CanonicalVoice: v.CanonicalVoice,
			TierProvider:   v.TierProvider,
			AdapterVoice:   v.AdapterVoice,
		})
	}
	return connect.NewResponse(&settv1.SetVoiceOverrideResponse{Overrides: out}), nil
}
