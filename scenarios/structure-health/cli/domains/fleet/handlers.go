package fleet

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/structure-health/v1/fleet/fleet_v1connect"
)

type handlers struct {
	client fleetconnect.FleetServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: fleetconnect.NewFleetServiceClient(httpClient, baseURL)}
}

// scan grades the fleet (or a requested subset) and renders the rollup.
func (h *handlers) scan(ctx cliapp.RunContext) error {
	resp, err := h.client.ScanFleet(context.Background(), connect.NewRequest(&fleetv1.ScanFleetRequest{
		Scenarios: splitCSV(ctx.FlagValues("scenario")),
	}))
	if err != nil {
		return cliapp.WrapAPIError("scan fleet structure conformance", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fleet response")
	}
	msg := resp.Msg

	summary := []string{
		fmt.Sprintf("%d scenario(s): %d passing, %d with structure errors, %d missing a freshness check.",
			msg.GetScenarioCount(), msg.GetPassingCount(),
			msg.GetScenarioCount()-msg.GetPassingCount(), msg.GetMissingFreshnessCount()),
		fmt.Sprintf("%d auto-fixable finding(s) across the fleet.", msg.GetAutofixableTotal()),
	}
	summary = append(summary, profileLines(msg.GetProfileDistribution())...)
	summary = append(summary, ruleLines(msg.GetRuleConformance())...)

	// Offenders: scenarios with errors or a missing freshness check, worst first.
	results := make([]string, 0, len(msg.GetEntries()))
	for _, e := range msg.GetEntries() {
		if e.GetErrorCount() == 0 && !e.GetMissingFreshnessCheck() {
			continue
		}
		flags := make([]string, 0, 2)
		if e.GetMissingFreshnessCheck() {
			flags = append(flags, "no-freshness-check")
		}
		if e.GetAutofixableCount() > 0 {
			flags = append(flags, fmt.Sprintf("%d auto-fixable", e.GetAutofixableCount()))
		}
		suffix := ""
		if len(flags) > 0 {
			suffix = " [" + strings.Join(flags, ", ") + "]"
		}
		results = append(results, fmt.Sprintf("%s (%s): %d error(s), %d warning(s)%s",
			e.GetScenario(), e.GetProfileId(), e.GetErrorCount(), e.GetWarningCount(), suffix))
	}
	if len(results) == 0 {
		results = append(results, "No structure offenders — every scanned scenario is conformant.")
	}

	hints := []string{
		"`structure-health validate scenario <name> --json` - drill into one scenario's findings",
		"`structure-health fix-config run <name>` - preview auto-fixable remediations",
	}
	for _, e := range msg.GetErrors() {
		hints = append(hints, fmt.Sprintf("could not grade %s: %s", e.GetScenario(), e.GetReason()))
	}

	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: "Offenders",
		Results:        results,
		RetrievalHints: hints,
	})
}

func profileLines(dist []*fleetv1.ProfileDistribution) []string {
	if len(dist) == 0 {
		return nil
	}
	lines := []string{"Profiles:"}
	for _, p := range dist {
		recognized := "recognized"
		if !p.GetRecognized() {
			recognized = "unrecognized"
		}
		lines = append(lines, fmt.Sprintf("  • %s [%s]: %d", p.GetProfileId(), recognized, p.GetScenarioCount()))
	}
	return lines
}

func ruleLines(rules []*fleetv1.RuleConformance) []string {
	if len(rules) == 0 {
		return nil
	}
	limit := len(rules)
	if limit > 10 {
		limit = 10
	}
	lines := []string{fmt.Sprintf("Top offending rules (%d shown of %d):", limit, len(rules))}
	for _, r := range rules[:limit] {
		fixable := ""
		if r.GetAutofixable() > 0 {
			fixable = fmt.Sprintf(", %d auto-fixable", r.GetAutofixable())
		}
		lines = append(lines, fmt.Sprintf("  • %s [%s]: %d scenario(s), %d finding(s)%s",
			r.GetCode(), r.GetWorstSeverity(), r.GetOffendingScenarios(), r.GetTotalFindings(), fixable))
	}
	return lines
}

func splitCSV(values []string) []string {
	var out []string
	for _, v := range values {
		for _, part := range strings.Split(v, ",") {
			if p := strings.TrimSpace(part); p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}
