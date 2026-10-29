package wiring

import (
	"context"
	"fmt"
	"strings"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/findings"
	"agent-manager/internal/pricing"
	"agent-manager/internal/promptmanager"
	"agent-manager/internal/usageattribution"
)

// frictionBugFiler files a friction problem in scenario-qa's bug inbox through
// the report-bug writer path (the same one the model-policy drift check uses).
type frictionBugFiler struct {
	client promptmanager.ScenarioQAReporter
}

func (f frictionBugFiler) FileFrictionBug(ctx context.Context, p usageattribution.Problem) (string, error) {
	context := map[string]string{"fingerprint": p.Fingerprint, "pattern": p.Pattern, "owner": p.Owner, "owner_kind": p.OwnerKind}
	if p.OwnerScenario != "" {
		context["scenario"] = p.OwnerScenario
	}
	if p.OwnerCommand != "" {
		context["command"] = p.OwnerCommand
	}
	cost := fmt.Sprintf("wasted %s in total", usageattribution.FormatDuration(p.WallClockMS))
	if !p.TimeLost {
		cost = fmt.Sprintf("made %d tool calls (%d tokens) of unrequested work", p.EpisodeTurns, p.Tokens)
	}
	return f.client.PublishScenarioQABug(ctx, promptmanager.ScenarioQABug{
		Title:      "agent friction: " + strings.TrimSuffix(p.Description, "."),
		SignalType: "unknown", Severity: "minor",
		Repro: []string{
			"Run `agent-manager usage friction`",
			fmt.Sprintf("Find the problem with fingerprint %s", p.Fingerprint),
			"Open the runs listed under it with `agent-manager run episodes <run-id>`",
		},
		Expected: "an agent finishes the work without repeating this pattern",
		Actual:   fmt.Sprintf("%s It recurred in %d episodes across %d runs, used about %.1f quota points, and %s.", p.Description, p.Episodes, p.Runs, p.Points, cost),
		Description: fmt.Sprintf("Agent Manager grouped these episodes by fingerprint %s (pattern %s; suspected owner: %s). Evidence runs: %s.",
			p.Fingerprint, p.Pattern, p.Owner, strings.Join(p.RunIDs, ", ")),
		Context:      context,
		HonestyFlags: []string{"speculative-cause", "repro-not-attempted"},
		// One bug per problem: the bug inbox deduplicates on this key.
		IdempotencyKey: "agent-manager-friction/" + p.Fingerprint,
	})
}

// newFrictionService builds the usage service that lists, reports and
// dismisses friction problems, or reports false when its inputs are absent.
func newFrictionService(deps RouteDependencies) (usageattribution.Service, bool) {
	readings, ok := deps.PricingRepository.(pricing.QuotaReadingSource)
	if !ok || deps.DB == nil {
		return usageattribution.Service{}, false
	}
	return usageattribution.Service{
		Store:    database.NewUsageAttributionRepository(deps.DB),
		Readings: readings,
		Pricer:   usageattribution.NewPriceBook(deps.PricingService),
		Statuses: findings.NewFrictionStatusStore(deps.DB),
		Bugs:     frictionBugFiler{client: promptmanager.NewHTTPClient()},
	}, true
}
