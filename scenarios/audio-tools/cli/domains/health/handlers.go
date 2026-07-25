package health

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliapp"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/common"
	diagv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/diagnostics"
	hsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status"
	hsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health_status/health_status_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/shared"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client hsconnect.HealthStatusServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: hsconnect.NewHealthStatusServiceClient(httpClient, baseURL),
	}
}

// show prints one snapshot of the provider rollup. --refresh forces a
// bypass of the registry cache; --json switches to proto JSON.
func (h *handlers) show(ctx cliapp.RunContext) error {
	refresh := ctx.Flag("refresh") == "true"

	if refresh {
		resp, err := h.client.RefreshProviderHealth(context.Background(), connect.NewRequest(&hsv1.RefreshProviderHealthRequest{}))
		if err != nil {
			return cliapp.WrapAPIError("health show --refresh", err, nil)
		}
		return cliapp.RenderProtoMutation(ctx, resp.Msg, mutationReport(resp.Msg.GetCapabilities(), resp.Msg.GetGeneratedAt(), resp.Msg.GetCacheTtlSeconds(), true))
	}

	resp, err := h.client.GetProviderHealth(context.Background(), connect.NewRequest(&hsv1.GetProviderHealthRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("health show", err, nil)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, mutationReport(resp.Msg.GetCapabilities(), resp.Msg.GetGeneratedAt(), resp.Msg.GetCacheTtlSeconds(), false))
}

// watch streams events from StreamProviderHealth until the user
// cancels (Ctrl-C) or the server closes the stream.
func (h *handlers) watch(ctx cliapp.RunContext) error {
	stream, err := h.client.StreamProviderHealth(context.Background(), connect.NewRequest(&hsv1.StreamProviderHealthRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("health watch", err, nil)
	}
	defer stream.Close()

	out := ctx.Stdout()
	for stream.Receive() {
		event := stream.Msg()
		fmt.Fprintf(out, "\n--- %s ---\n", event.GetGeneratedAt())
		for _, line := range formatCapabilities(event.GetCapabilities()) {
			fmt.Fprintln(out, line)
		}
	}
	if err := stream.Err(); err != nil {
		return cliapp.WrapAPIError("health watch", err, nil)
	}
	return nil
}

func mutationReport(caps []*hsv1.CapabilityHealth, generatedAt string, ttl int32, refreshed bool) cliapp.MutationReport {
	header := fmt.Sprintf("Provider health  (generated_at=%s  cache_ttl=%ds)", generatedAt, ttl)
	if refreshed {
		header = "[refreshed] " + header
	}
	rep := cliapp.MutationReport{Result: []string{header}}
	rep.Changes = append(rep.Changes, formatCapabilities(caps)...)
	rep.NextCommand = []string{
		"audio-tools health show --refresh    # bypass cache",
		"audio-tools health watch             # stream events",
		"audio-tools health show --json       # proto JSON",
	}
	return rep
}

// formatCapabilities renders a flat capability | provider_id | tier |
// state | last_check | error table sorted by capability.
func formatCapabilities(caps []*hsv1.CapabilityHealth) []string {
	if len(caps) == 0 {
		return []string{"(no capabilities reported)"}
	}
	var out []string
	out = append(out, fmt.Sprintf("%-10s %-22s %-7s %-12s %-22s %s", "CAP", "PROVIDER", "TIER", "STATE", "LAST_CHECK", "ERROR"))
	for _, c := range caps {
		for _, p := range c.GetProviders() {
			out = append(out, fmt.Sprintf("%-10s %-22s %-7s %-12s %-22s %s",
				capabilityLabel(c.GetCapability()),
				truncate(p.GetProviderId(), 22),
				tierLabel(p.GetTier()),
				stateLabel(p.GetState()),
				truncate(p.GetLastCheckedAt(), 22),
				truncate(p.GetErrorMessage(), 80),
			))
		}
		out = append(out, fmt.Sprintf("  effective: %s -> %s", capabilityLabel(c.GetCapability()), stateLabel(c.GetEffectiveState())))
	}
	return out
}

func capabilityLabel(c diagv1.Capability) string {
	switch c {
	case diagv1.Capability_CAPABILITY_STT:
		return "stt"
	case diagv1.Capability_CAPABILITY_TTS:
		return "tts"
	case diagv1.Capability_CAPABILITY_SUMMARIZE:
		return "summarize"
	case diagv1.Capability_CAPABILITY_TRANSCODE:
		return "transcode"
	}
	return "unknown"
}

func tierLabel(t commonv1.ProviderTier) string {
	switch t {
	case commonv1.ProviderTier_PROVIDER_TIER_LOCAL:
		return "local"
	case commonv1.ProviderTier_PROVIDER_TIER_BYOK:
		return "byok"
	case commonv1.ProviderTier_PROVIDER_TIER_VROOLI:
		return "vrooli"
	}
	return "-"
}

func stateLabel(s sharedv1.ProviderState) string {
	switch s {
	case sharedv1.ProviderState_PROVIDER_STATE_AVAILABLE:
		return "AVAILABLE"
	case sharedv1.ProviderState_PROVIDER_STATE_UNAVAILABLE:
		return "UNAVAILABLE"
	case sharedv1.ProviderState_PROVIDER_STATE_UNKNOWN:
		return "UNKNOWN"
	}
	return "-"
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
