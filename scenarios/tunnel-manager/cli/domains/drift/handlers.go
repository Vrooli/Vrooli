package drift

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	configv1 "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config"
	configconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/config/config_v1connect"

	"github.com/vrooli/cli-core/cliapp"
)

type handlers struct {
	core   *cliapp.ScenarioApp
	client configconnect.ConfigServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{
		core:   core,
		client: configconnect.NewConfigServiceClient(httpClient, baseURL),
	}
}

func (h *handlers) list(ctx cliapp.RunContext) error {
	resp, err := h.client.GetDrift(context.Background(), connect.NewRequest(&configv1.GetDriftRequest{}))
	if err != nil {
		return cliapp.WrapAPIError("get drift", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no drift response")
	}
	results := make([]string, 0, len(resp.Msg.Entries))
	for _, e := range resp.Msg.Entries {
		results = append(results, formatEntry(e))
	}
	return cliapp.RenderProtoList(ctx, resp.Msg, cliapp.ListReport{
		Summary:        []string{driftSummary(resp.Msg)},
		ResultsHeading: "Ingress entries",
		Results:        results,
		RetrievalHints: []string{
			"`drift adopt <hostname> [--scenario <s> | --target <url>]` — bring drift under management",
			"`drift ignore <hostname> [--note <text>]` — acknowledge an external hostname",
			"`drift prune <hostname>` — remove a single hostname from live ingress",
		},
	})
}

func (h *handlers) adopt(ctx cliapp.RunContext) error {
	hostname := strings.TrimSpace(ctx.Positional("hostname"))
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	resp, err := h.client.AdoptIngress(context.Background(), connect.NewRequest(&configv1.AdoptIngressRequest{
		Hostname: hostname,
		Scenario: strings.TrimSpace(ctx.Flag("scenario")),
		Target:   strings.TrimSpace(ctx.Flag("target")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("adopt %q", hostname), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Entry == nil {
		return fmt.Errorf("server returned no adopt response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Adopted %s.", hostname)},
		Changes: []string{formatEntry(resp.Msg.Entry)},
		NextCommand: []string{
			"`drift list` — confirm the reclassified state",
			"`config sync` — publish the adopted route additively",
		},
	})
}

func (h *handlers) ignore(ctx cliapp.RunContext) error {
	hostname := strings.TrimSpace(ctx.Positional("hostname"))
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	resp, err := h.client.IgnoreIngress(context.Background(), connect.NewRequest(&configv1.IgnoreIngressRequest{
		Hostname: hostname,
		Note:     strings.TrimSpace(ctx.Flag("note")),
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("ignore %q", hostname), err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Entry == nil {
		return fmt.Errorf("server returned no ignore response")
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Ignored %s; reconcile will never push or prune it.", hostname)},
		Changes: []string{formatEntry(resp.Msg.Entry)},
	})
}

func (h *handlers) prune(ctx cliapp.RunContext) error {
	hostname := strings.TrimSpace(ctx.Positional("hostname"))
	if hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	resp, err := h.client.PruneIngress(context.Background(), connect.NewRequest(&configv1.PruneIngressRequest{
		Hostname: hostname,
	}))
	if err != nil {
		return cliapp.WrapAPIError(fmt.Sprintf("prune %q", hostname), err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no prune response")
	}
	msg := fmt.Sprintf("Pruned %s from live ingress and the ledger.", hostname)
	if !resp.Msg.Pruned {
		msg = fmt.Sprintf("%s was neither live nor tracked; nothing to prune.", hostname)
	}
	return cliapp.RenderProtoMutation(ctx, resp.Msg, cliapp.MutationReport{
		Result: []string{msg},
	})
}

func driftSummary(resp *configv1.GetDriftResponse) string {
	c := resp.Counts
	if c == nil {
		c = &configv1.DriftCounts{}
	}
	return fmt.Sprintf("Drift in %s mode: %d managed · %d missing · %d external · %d orphaned · %d ignored · %d unmanaged.",
		strings.ToLower(resp.Mode.String()), c.Managed, c.Missing, c.ExternalOk, c.Orphaned, c.Ignored, c.Unmanaged)
}

func formatEntry(e *configv1.IngressEntry) string {
	if e == nil {
		return "(nil)"
	}
	parts := []string{e.Hostname, "[" + stateLabel(e.State) + "]"}
	if e.ServiceTarget != "" {
		parts = append(parts, "→ "+e.ServiceTarget)
	}
	if e.Scenario != "" {
		parts = append(parts, "scenario="+e.Scenario)
	}
	if e.Note != "" {
		parts = append(parts, "note="+e.Note)
	}
	return strings.Join(parts, " ")
}

func stateLabel(s configv1.OwnershipState) string {
	switch s {
	case configv1.OwnershipState_OWNERSHIP_STATE_MANAGED:
		return "managed"
	case configv1.OwnershipState_OWNERSHIP_STATE_MISSING:
		return "missing"
	case configv1.OwnershipState_OWNERSHIP_STATE_EXTERNAL_OK:
		return "external"
	case configv1.OwnershipState_OWNERSHIP_STATE_ORPHANED:
		return "orphaned"
	case configv1.OwnershipState_OWNERSHIP_STATE_IGNORED:
		return "ignored"
	case configv1.OwnershipState_OWNERSHIP_STATE_UNMANAGED:
		return "unmanaged"
	default:
		return "unspecified"
	}
}
