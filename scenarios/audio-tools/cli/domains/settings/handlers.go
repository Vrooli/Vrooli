package settings

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client settconnect.SettingsServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: settconnect.NewSettingsServiceClient(httpClient, baseURL),
	}
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
