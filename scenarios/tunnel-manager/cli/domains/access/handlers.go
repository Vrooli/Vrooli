package access

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

// status reads GetAccessStatus and renders one of two read-only views: the
// default global-switch + per-host effective-bypass table, or — with
// --dry-run — the pending create/remove diff (mutating nothing). The CLI
// manifest contract binds exactly one command per RPC, so the view is chosen
// by a flag rather than by separate dry-run/list verbs.
func (h *handlers) statusCall(_ cliapp.OperationContext) (*configv1.GetAccessStatusResponse, error) {
	resp, err := h.client.GetAccessStatus(context.Background(), connect.NewRequest(&configv1.GetAccessStatusRequest{}))
	if err != nil {
		return nil, cliapp.WrapAPIError("get access status", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Status == nil {
		return nil, fmt.Errorf("server returned no access status")
	}
	return resp.Msg, nil
}

func (h *handlers) statusReport(ctx cliapp.OperationContext, msg *configv1.GetAccessStatusResponse) cliapp.ListReport {
	st := msg.Status
	if ctx.BoolFlag("dry-run") {
		return dryRunReport(st)
	}
	return statusReport(st)
}

func statusReport(st *configv1.AccessStatus) cliapp.ListReport {
	results := make([]string, 0, len(st.Hosts))
	for _, host := range st.Hosts {
		results = append(results, formatHostState(host))
	}
	if len(results) == 0 {
		results = []string{"(no exposed hosts)"}
	}
	return cliapp.ListReport{
		Summary:        []string{accessSummary(st)},
		ResultsHeading: "Hosts",
		Results:        results,
		RetrievalHints: []string{
			"`access status --dry-run` — preview the Bypass apps that would be created/removed",
			"`config public-exposure --on|--off` — flip the global /public exposure switch",
			"`routes update <id> --public-exposure inherit|enabled|disabled` — per-route override",
		},
	}
}

func dryRunReport(st *configv1.AccessStatus) cliapp.ListReport {
	changes := make([]string, 0, len(st.ToCreate)+len(st.ToRemove))
	for _, host := range st.ToCreate {
		changes = append(changes, fmt.Sprintf("+ %s — would create a Bypass-Everyone app scoped to %s/public", host, host))
	}
	for _, host := range st.ToRemove {
		changes = append(changes, fmt.Sprintf("- %s — would remove the TM-owned %s/public Bypass app", host, host))
	}
	summary := fmt.Sprintf("Dry-run (no changes applied): %d to create, %d to remove.", len(st.ToCreate), len(st.ToRemove))
	if len(changes) == 0 {
		summary = "Dry-run (no changes applied): Access bypass already in sync; nothing to create or remove."
		changes = []string{"(no pending changes)"}
	}
	return cliapp.ListReport{
		Summary:        []string{accessSummary(st), summary},
		ResultsHeading: "Pending Access-bypass changes",
		Results:        changes,
		RetrievalHints: []string{
			"`access status` — show effective per-host bypass state",
			"`config sync` — reconciliation applies these changes on the remote path",
		},
	}
}

func accessSummary(st *configv1.AccessStatus) string {
	enabled := "disabled"
	if st.Enabled {
		enabled = "enabled"
	}
	configured := "not configured"
	if st.Configured {
		configured = "configured"
	}
	return fmt.Sprintf("Global /public Access bypass: %s; capability %s (%d host(s)).",
		enabled, configured, len(st.Hosts))
}

func formatHostState(s *configv1.AccessHostState) string {
	if s == nil {
		return "(nil)"
	}
	override := s.Override
	if strings.TrimSpace(override) == "" {
		override = "inherit"
	}
	bypass := "gated"
	if s.EffectiveBypass {
		bypass = "bypass"
	}
	managed := "unmanaged"
	if s.Managed {
		managed = "managed"
	}
	parts := []string{s.Host, "override=" + override, bypass, managed}
	if s.AppId != "" {
		parts = append(parts, "app_id="+s.AppId)
	}
	return strings.Join(parts, " ")
}
