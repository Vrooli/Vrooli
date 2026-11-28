package health

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapp"

	"audio-tools/cli/domains/labels"

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
	return h.showPrimitive().Run(ctx)
}

type healthSnapshot struct {
	payload proto.Message
	report  cliapp.MutationReport
}

func (h *handlers) showPrimitive() cliapp.PrimitiveHandler {
	return cliapp.ActionCustomRenderers(func(ctx cliapp.OperationContext) (healthSnapshot, error) {
		if ctx.Flag("refresh") == "true" {
			resp, err := h.client.RefreshProviderHealth(context.Background(), connect.NewRequest(&hsv1.RefreshProviderHealthRequest{}))
			if err != nil {
				return healthSnapshot{}, cliapp.WrapAPIError("health show --refresh", err, nil)
			}
			return healthSnapshot{payload: resp.Msg, report: mutationReport(resp.Msg.GetCapabilities(), resp.Msg.GetGeneratedAt(), resp.Msg.GetCacheTtlSeconds(), true)}, nil
		}
		resp, err := h.client.GetProviderHealth(context.Background(), connect.NewRequest(&hsv1.GetProviderHealthRequest{}))
		if err != nil {
			return healthSnapshot{}, cliapp.WrapAPIError("health show", err, nil)
		}
		return healthSnapshot{payload: resp.Msg, report: mutationReport(resp.Msg.GetCapabilities(), resp.Msg.GetGeneratedAt(), resp.Msg.GetCacheTtlSeconds(), false)}, nil
	}, renderHealthHuman, renderHealthJSON)
}

func renderHealthHuman(ctx cliapp.RunContext, snapshot healthSnapshot) error {
	return ctx.RenderMutation(snapshot.report)
}

func renderHealthJSON(ctx cliapp.RunContext, snapshot healthSnapshot) error {
	return cliapp.PrintProtoJSON(ctx.Stdout(), snapshot.payload)
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
	out = append(out, fmt.Sprintf("%-10s %-22s %-7s %-12s %-9s %-22s %s", "CAP", "PROVIDER", "TIER", "STATE", "LATENCY", "LAST_CHECK", "ERROR"))
	for _, c := range caps {
		for _, p := range c.GetProviders() {
			out = append(out, fmt.Sprintf("%-10s %-22s %-7s %-12s %-9s %-22s %s",
				capabilityLabel(c.GetCapability()),
				truncate(p.GetProviderId(), 22),
				tierLabel(p.GetTier()),
				stateLabel(p.GetState()),
				latencyLabel(p.GetLatencyMs()),
				truncate(p.GetLastCheckedAt(), 22),
				truncate(p.GetErrorMessage(), 80),
			))
		}
		out = append(out, fmt.Sprintf("  effective: %s -> %s", capabilityLabel(c.GetCapability()), stateLabel(c.GetEffectiveState())))
	}
	return out
}

func capabilityLabel(c diagv1.Capability) string {
	return labels.Enum(c, "CAPABILITY_", "unknown", labels.LowerWords)
}

func tierLabel(t commonv1.ProviderTier) string {
	return labels.Enum(t, "PROVIDER_TIER_", "-", labels.LowerWords)
}

// latencyLabel renders the last measured liveness-check duration. 0 means the
// provider has not been measured yet, not "instant".
func latencyLabel(ms float64) string {
	if ms <= 0 {
		return "-"
	}
	if ms < 10 {
		return fmt.Sprintf("%.1fms", ms)
	}
	return fmt.Sprintf("%.0fms", ms)
}

func stateLabel(s sharedv1.ProviderState) string {
	return labels.Enum(s, "PROVIDER_STATE_", "-", strings.ToUpper)
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
