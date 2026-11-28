package settings

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"audio-tools/cli/domains/internal/providerlabels"
	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client settconnect.SettingsServiceClient
	tts    ttsconnect.TTSServiceClient
}

const maxBYOKSecretBytes = 8 * 1024

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: settconnect.NewSettingsServiceClient(httpClient, baseURL),
		tts:    ttsconnect.NewTTSServiceClient(httpClient, baseURL),
	}
}

// providers prints the routing config plus the TTS-provider availability
// matrix. Folded in 2026-05-17 from the standalone `diagnose` CLI domain;
// see docs/internal/DECISIONS.md.
func (h *handlers) providers(ctx cliapp.RunContext) error {
	return h.providersPrimitive().Run(ctx)
}

type providersSnapshot struct {
	Config       *settv1.ProviderConfig
	Availability []*sharedv1.ProviderHealth
}

func (h *handlers) providersPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ActionCustomRenderers(func(_ cliapp.OperationContext) (providersSnapshot, error) {
		cfg, err := h.client.GetProviderConfig(context.Background(), connect.NewRequest(&settv1.GetProviderConfigRequest{}))
		if err != nil {
			return providersSnapshot{}, cliapp.WrapAPIError("get provider config", err, nil)
		}
		status, err := h.tts.GetStatus(context.Background(), connect.NewRequest(&ttsv1.GetStatusRequest{}))
		if err != nil {
			return providersSnapshot{}, cliapp.WrapAPIError("get tts status", err, nil)
		}
		return providersSnapshot{Config: cfg.Msg.GetConfig(), Availability: status.Msg.GetStatus().GetAvailability()}, nil
	}, renderProviders, renderProviders)
}

func renderProviders(ctx cliapp.RunContext, snapshot providersSnapshot) error {
	cfg := snapshot.Config
	out := ctx.Stdout()
	fmt.Fprintln(out, "Provider routing")
	fmt.Fprintf(out, "  BYOK   enabled=%v\n", cfg.GetByokEnabled())
	fmt.Fprintf(out, "  Vrooli enabled=%v\n", cfg.GetVrooliEnabled())
	fmt.Fprintf(out, "  Local  enabled=%v\n", cfg.GetLocalEnabled())
	fmt.Fprintln(out)
	fmt.Fprintln(out, "TTS availability")
	for _, a := range snapshot.Availability {
		state := "unknown"
		switch a.GetState() {
		case sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE:
			state = "up"
		case sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE:
			state = "down"
		}
		providerID := strings.TrimSpace(a.GetProviderId())
		if providerID == "" {
			providerID = "not-configured"
		}
		fmt.Fprintf(out, "  %-7s %-12s %s\n", providerlabels.TierLabel(a.GetTier()), providerID, state)
	}
	return nil
}

func (h *handlers) provider(ctx cliapp.RunContext) error {
	return h.providerPrimitive().Run(ctx)
}

func (h *handlers) providerPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(_ cliapp.OperationContext) (*settv1.GetProviderConfigResponse, error) {
		resp, err := h.client.GetProviderConfig(context.Background(), connect.NewRequest(&settv1.GetProviderConfigRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("get provider config", err, nil)
		}
		return resp.Msg, nil
	}, renderProvider, renderProvider)
}

func renderProvider(ctx cliapp.RunContext, resp *settv1.GetProviderConfigResponse) error {
	c := resp.GetConfig()
	out := ctx.Stdout()
	fmt.Fprintf(out, "BYOK   enabled=%v\nVrooli enabled=%v\nLocal  enabled=%v\n", c.GetByokEnabled(), c.GetVrooliEnabled(), c.GetLocalEnabled())
	fmt.Fprintf(out, "Whisper URL: %s\nKokoro URL:  %s\nOllama URL:  %s\nLPBS Base:   %s\n",
		c.GetWhisperUrl(), c.GetKokoroUrl(), c.GetOllamaUrl(), c.GetLpbsBaseUrl())
	return nil
}

func (h *handlers) byokList(ctx cliapp.RunContext) error {
	return h.byokListPrimitive().Run(ctx)
}

func (h *handlers) byokListPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(_ cliapp.OperationContext) (*settv1.ListBYOKCredentialsResponse, error) {
		resp, err := h.client.ListBYOKCredentials(context.Background(), connect.NewRequest(&settv1.ListBYOKCredentialsRequest{}))
		if err != nil {
			return nil, cliapp.WrapAPIError("list byok", err, nil)
		}
		return resp.Msg, nil
	}, renderBYOKList, renderBYOKList)
}

func renderBYOKList(ctx cliapp.RunContext, resp *settv1.ListBYOKCredentialsResponse) error {
	out := ctx.Stdout()
	for _, c := range resp.GetCredentials() {
		fmt.Fprintf(out, "%-20s %-10s %s\n", c.GetProviderId(), c.GetCapability(), c.GetFingerprint())
	}
	if len(resp.GetCredentials()) == 0 {
		fmt.Fprintln(out, "(no credentials)")
	}
	return nil
}

func (h *handlers) byokUpsert(ctx cliapp.RunContext) error {
	return h.byokUpsertFrom(ctx, os.Stdin)
}

// byokUpsertFrom reads the secret only from stdin so it never appears in argv,
// environment variables, or CLI output. The explicit flag prevents an
// accidental hang when a terminal is attached but no secret was intended.
func (h *handlers) byokUpsertFrom(ctx cliapp.RunContext, input io.Reader) error {
	resp, err := h.upsertBYOK(ctx, input)
	if err != nil {
		return err
	}
	return renderBYOKUpsert(ctx, resp)
}

func (h *handlers) upsertBYOK(ctx cliapp.OperationContext, input io.Reader) (*settv1.UpsertBYOKCredentialResponse, error) {
	if !ctx.BoolFlag("key-stdin") {
		return nil, fmt.Errorf("provide the API key through stdin with --key-stdin")
	}
	secretText, err := readBYOKSecret(input)
	if err != nil {
		return nil, err
	}
	resp, err := h.client.UpsertBYOKCredential(context.Background(), connect.NewRequest(&settv1.UpsertBYOKCredentialRequest{
		ProviderId: ctx.Flag("provider"),
		Capability: ctx.Flag("capability"),
		Secret:     &settv1.UpsertBYOKCredentialRequest_ApiKey{ApiKey: secretText},
	}))
	if err != nil {
		return nil, cliapp.WrapAPIError("upsert byok", err, nil)
	}
	return resp.Msg, nil
}

func (h *handlers) byokUpsertPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*settv1.UpsertBYOKCredentialResponse, error) {
		return h.upsertBYOK(ctx, os.Stdin)
	}, renderBYOKUpsert, renderBYOKUpsert)
}

func readBYOKSecret(input io.Reader) (string, error) {
	secret, err := io.ReadAll(io.LimitReader(input, maxBYOKSecretBytes+1))
	if err != nil {
		return "", fmt.Errorf("could not read API key from stdin")
	}
	if len(secret) > maxBYOKSecretBytes {
		return "", fmt.Errorf("API key from stdin exceeds the 8 KiB limit")
	}
	secretText := strings.TrimSuffix(strings.TrimSuffix(string(secret), "\n"), "\r")
	if secretText == "" {
		return "", fmt.Errorf("API key stdin is empty")
	}
	return secretText, nil
}

func renderBYOKUpsert(ctx cliapp.RunContext, resp *settv1.UpsertBYOKCredentialResponse) error {
	c := resp.GetCredential()
	fmt.Fprintf(ctx.Stdout(), "Stored %s/%s fingerprint=%s\n", c.GetProviderId(), c.GetCapability(), c.GetFingerprint())
	return nil
}

func (h *handlers) byokDelete(ctx cliapp.RunContext) error {
	return h.byokDeletePrimitive().Run(ctx)
}

func (h *handlers) byokDeletePrimitive() cliapp.PrimitiveHandler {
	return cliapp.ProtoCustomRenderers(func(ctx cliapp.OperationContext) (*settv1.DeleteBYOKCredentialResponse, error) {
		resp, err := h.client.DeleteBYOKCredential(context.Background(), connect.NewRequest(&settv1.DeleteBYOKCredentialRequest{
			ProviderId: ctx.Flag("provider"), Capability: ctx.Flag("capability"),
		}))
		if err != nil {
			return nil, cliapp.WrapAPIError("delete byok", err, nil)
		}
		return resp.Msg, nil
	}, renderBYOKDelete, renderBYOKDelete)
}

func renderBYOKDelete(ctx cliapp.RunContext, _ *settv1.DeleteBYOKCredentialResponse) error {
	fmt.Fprintln(ctx.Stdout(), "Deleted.")
	return nil
}
