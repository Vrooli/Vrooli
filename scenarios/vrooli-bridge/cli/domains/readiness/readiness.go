// Package readiness exposes the host-readiness operational surface. These are
// REST exception wrappers because the underlying endpoint is an owner-only
// operational document rather than a workflow RPC; they still use cli-core's
// authenticated ScenarioApp transport and standard report renderers.
package readiness

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/vrooli/cli-core/cliapp"
)

// Register returns the `vrooli-bridge readiness status|configure` group.
func Register(core *cliapp.ScenarioApp, _ []byte) (cliapp.SubcommandGroup, error) {
	return cliapp.SubcommandGroup{Name: "readiness", Description: "Inspect and configure Bridge host readiness", NeedsAPI: true, Subcommands: []cliapp.Command{
		{Name: "status", Description: "Show canonical endpoint, local health, and latest candidate admission", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, RunCtx: status},
		{Name: "configure", Description: "Persist the default advertised endpoint for future onboarding", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, Args: cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "endpoint", Required: true, Description: "Absolute http(s) Bridge base URL"}, {Name: "reachability-mode", Required: true, Description: "lan, tunnel, or manual"}}}, RunCtx: configure},
		{Name: "firewall-inspect", Description: "Inspect exact UFW admission evidence through the setup-managed broker", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, Args: firewallArgs(false), RunCtx: firewallAction("inspect", false)},
		{Name: "firewall-preview", Description: "Preview the exact scoped admission before confirming a mutation", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, Args: firewallArgs(false), RunCtx: firewallAction("preview", false)},
		{Name: "firewall-verify", Description: "Verify exact UFW admission evidence through the setup-managed broker", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, Args: firewallArgs(false), RunCtx: firewallAction("verify", false)},
		{Name: "firewall-allow", Description: "Allow one candidate IP to Bridge port 18767 after explicit confirmation", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, Args: firewallArgs(true), RunCtx: firewallAction("allow", true)},
		{Name: "firewall-revoke", Description: "Remove the exact managed candidate admission rule after explicit confirmation", NeedsAPI: true, DryRun: cliapp.DryRunUnsupported, Args: firewallArgs(true), RunCtx: firewallAction("revoke", true)},
	}}, nil
}

func firewallArgs(mutation bool) cliapp.ArgSchema {
	flags := []cliapp.Flag{{Name: "candidate-ip", Required: true, Description: "Exact source IP recorded for the failed candidate admission"}}
	if mutation {
		flags = append(flags, cliapp.Flag{Name: "confirm", Required: true, Description: "Required literal confirmation: true"})
	}
	return cliapp.ArgSchema{Flags: flags}
}

type readinessResponse struct {
	Status         string `json:"status"`
	Endpoint       string `json:"endpoint"`
	Port           int    `json:"port"`
	EndpointSource string `json:"endpoint_source"`
	Mode           string `json:"reachability_mode"`
	LocalAPI       bool   `json:"local_api"`
	LastCandidate  *struct {
		Host     string `json:"host"`
		State    string `json:"state"`
		Category string `json:"category"`
		SourceIP string `json:"source_ip"`
	} `json:"last_candidate"`
}

func status(ctx cliapp.RunContext) error {
	data, err := ctx.Core().Get("/readiness", url.Values{})
	if err != nil {
		return fmt.Errorf("get Bridge readiness: %w", err)
	}
	var result readinessResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("decode Bridge readiness: %w", err)
	}
	lines := []string{fmt.Sprintf("%s (source: %s; mode: %s)", result.Endpoint, result.EndpointSource, result.Mode), fmt.Sprintf("fixed API port: %d", result.Port)}
	if result.LocalAPI {
		lines = append(lines, "local API: healthy")
	} else {
		lines = append(lines, "local API: unhealthy")
	}
	triage := []cliapp.TriageGroup{}
	if result.LastCandidate != nil {
		items := []string{fmt.Sprintf("host: %s", result.LastCandidate.Host), fmt.Sprintf("state: %s", result.LastCandidate.State)}
		if result.LastCandidate.Category != "" {
			items = append(items, "category: "+result.LastCandidate.Category)
		}
		if result.LastCandidate.SourceIP != "" {
			items = append(items, "candidate source: "+result.LastCandidate.SourceIP)
		}
		triage = append(triage, cliapp.TriageGroup{Heading: "Latest candidate admission", Items: items})
	}
	return ctx.RenderOperational(cliapp.OperationalReport{Status: []string{"Bridge readiness: " + result.Status, strings.Join(lines, "\n")}, Triage: triage, NextSteps: []string{"Use `readiness configure --endpoint <url> --reachability-mode <lan|tunnel|manual>` to set the default."}})
}

func firewallAction(action string, mutation bool) func(cliapp.RunContext) error {
	return func(ctx cliapp.RunContext) error {
		payload := map[string]any{"action": action, "candidate_ip": ctx.Flag("candidate-ip")}
		if mutation {
			payload["confirm"] = ctx.Flag("confirm") == "true"
		}
		data, err := ctx.Core().Request(http.MethodPost, "/readiness/firewall", nil, payload)
		if err != nil {
			return fmt.Errorf("%s Bridge firewall admission: %w", action, err)
		}
		var result struct {
			Status  string `json:"status"`
			Changed bool   `json:"changed"`
			Code    string `json:"code"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			return fmt.Errorf("decode firewall result: %w", err)
		}
		if mutation && ctx.Flag("confirm") != "true" {
			return fmt.Errorf("--confirm true is required for firewall mutation")
		}
		return ctx.RenderMutation(cliapp.MutationReport{Result: []string{fmt.Sprintf("Firewall %s: %s", action, result.Status)}, Changes: []string{fmt.Sprintf("candidate IP: %s", ctx.Flag("candidate-ip")), fmt.Sprintf("changed: %t", result.Changed)}, NextCommand: []string{"`readiness status` — refresh candidate admission evidence; retry onboarding after verified admission."}})
	}
}

func configure(ctx cliapp.RunContext) error {
	payload := map[string]string{"endpoint": ctx.Flag("endpoint"), "reachability_mode": ctx.Flag("reachability-mode")}
	if _, err := ctx.Core().Request(http.MethodPut, "/readiness/endpoint", nil, payload); err != nil {
		return fmt.Errorf("configure Bridge endpoint: %w", err)
	}
	return ctx.RenderMutation(cliapp.MutationReport{Result: []string{"Bridge advertised endpoint configured."}, Changes: []string{"endpoint: " + ctx.Flag("endpoint"), "reachability mode: " + ctx.Flag("reachability-mode")}, NextCommand: []string{"`readiness status` — verify the persisted endpoint and candidate evidence"}})
}
