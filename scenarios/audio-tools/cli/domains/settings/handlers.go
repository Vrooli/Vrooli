package settings

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"
	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client settconnect.SettingsServiceClient
	tts    ttsconnect.TTSServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: settconnect.NewSettingsServiceClient(httpClient, baseURL),
		tts:    ttsconnect.NewTTSServiceClient(httpClient, baseURL),
	}
}

func providerTierLabel(t commonv1.ProviderTier) string {
	switch t {
	case commonv1.ProviderTier_PROVIDER_TIER_LOCAL:
		return "local"
	case commonv1.ProviderTier_PROVIDER_TIER_BYOK:
		return "byok"
	case commonv1.ProviderTier_PROVIDER_TIER_VROOLI:
		return "vrooli"
	default:
		return "unknown"
	}
}

// providers prints the routing config plus the TTS-provider availability
// matrix. Folded in 2026-05-17 from the standalone `diagnose` CLI domain;
// see docs/internal/DECISIONS.md.
func (h *handlers) providers(ctx cliapp.RunContext) error {
	cfg, err := h.client.GetProviderConfig(context.Background(), connect.NewRequest(&settv1.GetProviderConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get provider config", err, nil)
	}
	status, err := h.tts.GetStatus(context.Background(), connect.NewRequest(&ttsv1.GetStatusRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get tts status", err, nil)
	}
	out := ctx.Stdout()
	fmt.Fprintln(out, "Provider routing")
	fmt.Fprintf(out, "  BYOK   enabled=%v\n", cfg.Msg.GetConfig().GetByokEnabled())
	fmt.Fprintf(out, "  Vrooli enabled=%v\n", cfg.Msg.GetConfig().GetVrooliEnabled())
	fmt.Fprintf(out, "  Local  enabled=%v\n", cfg.Msg.GetConfig().GetLocalEnabled())
	fmt.Fprintln(out)
	fmt.Fprintln(out, "TTS availability")
	for _, a := range status.Msg.GetStatus().GetAvailability() {
		state := "down"
		if a.GetState() == sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE {
			state = "up"
		}
		fmt.Fprintf(out, "  %-7s %-12s %s\n", providerTierLabel(a.GetTier()), a.GetProviderId(), state)
	}
	return nil
}

func (h *handlers) provider(ctx cliapp.RunContext) error {
	resp, err := h.client.GetProviderConfig(context.Background(), connect.NewRequest(&settv1.GetProviderConfigRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get provider config", err, nil)
	}
	c := resp.Msg.GetConfig()
	out := ctx.Stdout()
	fmt.Fprintf(out, "BYOK   enabled=%v\nVrooli enabled=%v\nLocal  enabled=%v\n", c.GetByokEnabled(), c.GetVrooliEnabled(), c.GetLocalEnabled())
	fmt.Fprintf(out, "Whisper URL: %s\nKokoro URL:  %s\nOllama URL:  %s\nLPBS Base:   %s\n",
		c.GetWhisperUrl(), c.GetKokoroUrl(), c.GetOllamaUrl(), c.GetLpbsBaseUrl())
	return nil
}

func (h *handlers) byokList(ctx cliapp.RunContext) error {
	resp, err := h.client.ListBYOKCredentials(context.Background(), connect.NewRequest(&settv1.ListBYOKCredentialsRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("list byok", err, nil)
	}
	out := ctx.Stdout()
	for _, c := range resp.Msg.GetCredentials() {
		fmt.Fprintf(out, "%-20s %-10s %s\n", c.GetProviderId(), c.GetCapability(), c.GetFingerprint())
	}
	if len(resp.Msg.GetCredentials()) == 0 {
		fmt.Fprintln(out, "(no credentials)")
	}
	return nil
}

func (h *handlers) byokUpsert(ctx cliapp.RunContext) error {
	resp, err := h.client.UpsertBYOKCredential(context.Background(), connect.NewRequest(&settv1.UpsertBYOKCredentialRequest{
		ProviderId: ctx.Flag("provider"),
		Capability: ctx.Flag("capability"),
		Secret:     &settv1.UpsertBYOKCredentialRequest_ApiKey{ApiKey: ctx.Flag("key")},
	}))
	if err != nil {
		return cliapp.WrapAPIError("upsert byok", err, nil)
	}
	fmt.Fprintf(ctx.Stdout(), "Stored %s/%s fingerprint=%s\n",
		resp.Msg.GetCredential().GetProviderId(), resp.Msg.GetCredential().GetCapability(), resp.Msg.GetCredential().GetFingerprint())
	return nil
}

func (h *handlers) byokDelete(ctx cliapp.RunContext) error {
	_, err := h.client.DeleteBYOKCredential(context.Background(), connect.NewRequest(&settv1.DeleteBYOKCredentialRequest{
		ProviderId: ctx.Flag("provider"), Capability: ctx.Flag("capability"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("delete byok", err, nil)
	}
	fmt.Fprintln(ctx.Stdout(), "Deleted.")
	return nil
}
