package fleet

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/fleet"
	fleetconnect "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/fleet/fleet_v1connect"
)

type handlers struct {
	client fleetconnect.FleetServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: fleetconnect.NewFleetServiceClient(httpClient, baseURL)}
}

// scan grades the requested (or all) scenarios and renders the selected
// deterministic offender view. --view selects which structured query to show:
//
//	all (default) — every graded scenario
//	no-budget     — scenarios with no declared performance budget
//	slowest       — slowest combined (go+ui) builds (bounded by --limit)
//	regressed     — scenarios whose latest sample breaches its budget
//	tiers         — the capture-tier distribution
//
// These are exact/structured queries computed from the scan result — never a
// semantic ranking.
func (h *handlers) scan(ctx cliapp.RunContext) error {
	resp, err := h.client.ScanFleet(context.Background(), connect.NewRequest(&fleetv1.ScanFleetRequest{
		Scenarios: ctx.FlagValues("scenario"),
	}))
	if err != nil {
		return cliapp.WrapAPIError("scan fleet", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return fmt.Errorf("server returned no fleet response")
	}
	msg := resp.Msg
	view := strings.ToLower(strings.TrimSpace(firstFlag(ctx.FlagValues("view"))))
	if view == "" {
		view = "all"
	}
	limit := parseInt(firstFlag(ctx.FlagValues("limit")))

	summary := []string{fmt.Sprintf("%d scenario(s): %d without a budget, %d regressed.", msg.GetScenarioCount(), msg.GetNoBudgetCount(), msg.GetRegressedCount())}

	var heading string
	var results []string
	switch view {
	case "no-budget":
		heading = "Scenarios with no perf budget"
		for _, e := range sortedEntries(filter(msg.GetEntries(), func(e *fleetv1.FleetScenarioEntry) bool { return !e.GetHasBudget() })) {
			results = append(results, fmt.Sprintf("%s — tier=%s", e.GetScenario(), e.GetTier()))
		}
	case "regressed":
		heading = "Recently regressed scenarios"
		for _, e := range sortedEntries(filter(msg.GetEntries(), func(e *fleetv1.FleetScenarioEntry) bool { return e.GetRegressed() })) {
			results = append(results, fmt.Sprintf("%s — %s", e.GetScenario(), e.GetDegradedReason()))
		}
	case "slowest":
		heading = "Slowest builds"
		for _, e := range slowestBuilds(msg.GetEntries(), limit) {
			results = append(results, fmt.Sprintf("%s — total=%dms (go=%dms ui=%dms)", e.GetScenario(), e.GetGoBuildMs()+e.GetUiBuildMs(), e.GetGoBuildMs(), e.GetUiBuildMs()))
		}
	case "tiers":
		heading = "Capture-tier distribution"
		for _, d := range msg.GetTierDistribution() {
			results = append(results, fmt.Sprintf("tier %s — %d scenario(s)", d.GetTier(), d.GetScenarioCount()))
		}
	default: // "all"
		heading = "Fleet performance"
		for _, e := range msg.GetEntries() {
			budget := "no-budget"
			if e.GetHasBudget() {
				budget = "budgeted"
			}
			flag := ""
			if e.GetRegressed() {
				flag = " REGRESSED"
			}
			results = append(results, fmt.Sprintf("%s — tier=%s %s go=%dms ui=%dms%s", e.GetScenario(), e.GetTier(), budget, e.GetGoBuildMs(), e.GetUiBuildMs(), flag))
		}
	}
	if len(results) == 0 {
		results = append(results, "No scenarios match this view.")
	}
	return cliapp.RenderProtoList(ctx, msg, cliapp.ListReport{
		Summary:        summary,
		ResultsHeading: heading,
		Results:        results,
	})
}

func filter(entries []*fleetv1.FleetScenarioEntry, keep func(*fleetv1.FleetScenarioEntry) bool) []*fleetv1.FleetScenarioEntry {
	var out []*fleetv1.FleetScenarioEntry
	for _, e := range entries {
		if keep(e) {
			out = append(out, e)
		}
	}
	return out
}

func sortedEntries(entries []*fleetv1.FleetScenarioEntry) []*fleetv1.FleetScenarioEntry {
	sort.Slice(entries, func(i, j int) bool { return entries[i].GetScenario() < entries[j].GetScenario() })
	return entries
}

// slowestBuilds returns scenarios with a measured build, slowest combined build
// first, bounded by limit (limit <= 0 returns all).
func slowestBuilds(entries []*fleetv1.FleetScenarioEntry, limit int) []*fleetv1.FleetScenarioEntry {
	out := filter(entries, func(e *fleetv1.FleetScenarioEntry) bool { return e.GetGoBuildMs()+e.GetUiBuildMs() > 0 })
	sort.Slice(out, func(i, j int) bool {
		li, lj := out[i].GetGoBuildMs()+out[i].GetUiBuildMs(), out[j].GetGoBuildMs()+out[j].GetUiBuildMs()
		if li != lj {
			return li > lj
		}
		return out[i].GetScenario() < out[j].GetScenario()
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func firstFlag(values []string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
