// Package settings hosts the `audio-tools settings ...` subtree.
package settings

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
)

func Register(core *cliapp.ScenarioApp) cliapp.SubcommandGroup {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	client := settconnect.NewSettingsServiceClient(httpClient, baseURL)
	return cliapp.SubcommandGroup{
		Name:        "settings",
		Description: "Provider routing, BYOK credentials, voice overrides",
		NeedsAPI:    true,
		Subcommands: []cliapp.Command{
			{
				Name:        "provider",
				Description: "Show the current provider routing config",
				RunCtx: func(ctx cliapp.RunContext) error {
					resp, err := client.GetProviderConfig(context.Background(), connect.NewRequest(&settv1.GetProviderConfigRequest{}))
					if err != nil {
						return cliapp.WrapAPIError("get provider config", err, nil)
					}
					c := resp.Msg.GetConfig()
					out := ctx.Stdout()
					fmt.Fprintf(out, "BYOK   enabled=%v\nVrooli enabled=%v\nLocal  enabled=%v\n", c.GetByokEnabled(), c.GetVrooliEnabled(), c.GetLocalEnabled())
					fmt.Fprintf(out, "Whisper URL: %s\nKokoro URL:  %s\nOllama URL:  %s\nLPBS Base:   %s\n",
						c.GetWhisperUrl(), c.GetKokoroUrl(), c.GetOllamaUrl(), c.GetLpbsBaseUrl())
					return nil
				},
			},
			{
				Name:        "byok-list",
				Description: "List stored BYOK credentials (redacted)",
				RunCtx: func(ctx cliapp.RunContext) error {
					resp, err := client.ListBYOKCredentials(context.Background(), connect.NewRequest(&settv1.ListBYOKCredentialsRequest{}))
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
				},
			},
			{
				Name:        "byok-upsert",
				Description: "Add or replace a BYOK credential",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "provider", Required: true, Description: "Provider id (e.g. openai-tts)"},
						{Name: "capability", Required: true, Description: "stt | tts | summarize"},
						{Name: "key", Required: true, Description: "API key value"},
					},
				},
				RunCtx: func(ctx cliapp.RunContext) error {
					resp, err := client.UpsertBYOKCredential(context.Background(), connect.NewRequest(&settv1.UpsertBYOKCredentialRequest{
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
				},
			},
			{
				Name:        "byok-delete",
				Description: "Delete a BYOK credential",
				Args: cliapp.ArgSchema{
					Flags: []cliapp.Flag{
						{Name: "provider", Required: true, Description: "Provider id"},
						{Name: "capability", Required: true, Description: "stt | tts | summarize"},
					},
				},
				RunCtx: func(ctx cliapp.RunContext) error {
					_, err := client.DeleteBYOKCredential(context.Background(), connect.NewRequest(&settv1.DeleteBYOKCredentialRequest{
						ProviderId: ctx.Flag("provider"), Capability: ctx.Flag("capability"),
					}))
					if err != nil {
						return cliapp.WrapAPIError("delete byok", err, nil)
					}
					fmt.Fprintln(ctx.Stdout(), "Deleted.")
					return nil
				},
			},
		},
	}
}
