package resourcehandlers

import (
	"context"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/cli-core/agentcatalog/harness"
	"github.com/vrooli/cli-core/upstreamcheck"
	"github.com/vrooli/vrooli/internal/cli/resourcecli"
	"github.com/vrooli/vrooli/internal/resources"
)

// codingAgentUpstreamEntries lists the coding-agent resources the aggregate
// upstream-check spans, one per declared harness, by resource name. Each runs the
// resource CLI's own `upstream-check --json`, so per-resource source/pin config
// stays owned by the resource (resources/<name>/cli/main.go).
var codingAgentUpstreamEntries = declaredUpstreamEntries()

func declaredUpstreamEntries() []upstreamcheck.AggregateEntry {
	var entries []upstreamcheck.AggregateEntry
	for _, descriptor := range harness.Builtin().All() {
		entries = append(entries, upstreamcheck.AggregateEntry{
			Name: descriptor.Resource, CheckCmd: []string{"resource-" + descriptor.Resource, "upstream-check", "--json"},
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
	return entries
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
func runUpstreamCheck(parent context.Context, req resourcecli.UpstreamCheckRequest) (upstreamcheck.AggregateReport, bool) {
	entries := selectUpstreamEntries(req)
	if len(entries) == 0 {
		return upstreamcheck.AggregateReport{}, false
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, tuning.ResourceControlTimeout())
	defer cancel()
	return upstreamcheck.RunAggregate(ctx, upstreamcheck.DefaultAggregateRunner, entries), true
}

func runResourceLivenessCheck(parent context.Context, controller *resources.Controller, req resourcecli.UpstreamCheckRequest) (upstreamcheck.AggregateReport, error) {
	if parent == nil {
		parent = context.Background()
	}
	statePath := filepath.Join(controller.Home, "state", "resource-upstream-liveness.json")
	report, err := resources.CheckUpstream(parent, controller.Root, statePath, req.Name, nil, time.Now().UTC())
	if err != nil {
		return upstreamcheck.AggregateReport{}, err
	}
	out := upstreamcheck.AggregateReport{}
	for _, finding := range report.Findings {
		out.Artifacts = append(out.Artifacts, upstreamcheck.ArtifactFinding{
			Resource: finding.Resource, Target: finding.Target, Kind: finding.Kind, Reference: finding.Reference,
			Predicate: finding.Predicate, CheckedAt: finding.CheckedAt.Format(time.RFC3339), FirstFailedAt: finding.FirstFailedAt.Format(time.RFC3339),
			Status: finding.Status, Reachable: finding.Reachable, Stale: finding.Stale, Note: finding.Note,
		})
	}
	return out, nil
}
