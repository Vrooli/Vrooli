package resourcehandlers

import (
	"context"
	"strings"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
)

// codingAgentUpstreamEntries is the SSOT list of coding-agent resources the
// aggregate upstream-check spans. Each runs the resource CLI's own
// `upstream-check --json`, so per-resource source/pin config stays owned by
// the resource (resources/<name>/cli/main.go), never duplicated here.
var codingAgentUpstreamEntries = []upstreamcheck.AggregateEntry{
	{Name: "claude-code", CheckCmd: []string{"resource-claude-code", "upstream-check", "--json"}},
	{Name: "codex", CheckCmd: []string{"resource-codex", "upstream-check", "--json"}},
	{Name: "opencode", CheckCmd: []string{"resource-opencode", "upstream-check", "--json"}},
	{Name: "grok", CheckCmd: []string{"resource-grok", "upstream-check", "--json"}},
	{Name: "antigravity", CheckCmd: []string{"resource-antigravity", "upstream-check", "--json"}},
}

func knownCodingAgentResourceNames() string {
	names := make([]string, 0, len(codingAgentUpstreamEntries))
	for _, entry := range codingAgentUpstreamEntries {
		names = append(names, entry.Name)
	}
	return strings.Join(names, ", ")
}

func parseResourceUpstreamCheckRequest(args []string) (resourcecli.UpstreamCheckRequest, error) {
	req, err := resourcecli.ParseUpstreamCheckRequest(args)
	return req, mapResourceParseError("resource upstream-check", err)
}

// selectUpstreamEntries filters the SSOT list by an optional resource name.
// Empty name (or --all) selects every coding-agent resource.
func selectUpstreamEntries(req resourcecli.UpstreamCheckRequest) []upstreamcheck.AggregateEntry {
	if req.Name == "" || req.All {
		return codingAgentUpstreamEntries
	}
	for _, entry := range codingAgentUpstreamEntries {
		if entry.Name == req.Name {
			return []upstreamcheck.AggregateEntry{entry}
		}
	}
	return nil
}

// runUpstreamCheck executes the (filtered) aggregate check. It is read-only and
// agent-safe: it always returns a report and never errors on a missing binary
// or network failure (those degrade to StatusUnknown inside RunAggregate).
func runUpstreamCheck(req resourcecli.UpstreamCheckRequest) (upstreamcheck.AggregateReport, bool) {
	entries := selectUpstreamEntries(req)
	if len(entries) == 0 {
		return upstreamcheck.AggregateReport{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), tuning.ResourceControlTimeout())
	defer cancel()
	return upstreamcheck.RunAggregate(ctx, upstreamcheck.DefaultAggregateRunner, entries), true
}
