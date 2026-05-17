package diagnose

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"

	settv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings"
	settconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/settings/settings_v1connect"
	ttsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts"
	ttsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/tts/tts_v1connect"
)

type handlers struct {
	core     *cliapp.ScenarioApp
	settings settconnect.SettingsServiceClient
	tts      ttsconnect.TTSServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:     core,
		settings: settconnect.NewSettingsServiceClient(httpClient, baseURL),
		tts:      ttsconnect.NewTTSServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) providers(ctx cliapp.RunContext) error {
	cfg, err := h.settings.GetProviderConfig(context.Background(), connect.NewRequest(&settv1.GetProviderConfigRequest{}))
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
		if a.GetAvailable() {
			state = "up"
		}
		fmt.Fprintf(out, "  %-7s %-12s %s\n", a.GetTier(), a.GetProviderId(), state)
	}
	return nil
}
