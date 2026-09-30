// Package fleet answers deterministic, structured offender queries about
// scenario performance across the whole fleet: scenarios with no perf budget,
// slowest builds, recent regressions, and capture-tier distribution. These are
// exact/structured queries (not semantic), so performance-health is NOT a
// search-hub data provider; the CLI verbs are discoverable through cli-health's
// command index.
//
// Per-scenario grading sits behind the Grader seam; the production grader
// (composing readiness tier + budget config + persisted trend) lives in
// handlers/fleet and is injected here, so this package stays testable with a
// fake grader.
package fleet

import (
	"context"
	"errors"
	"sort"
)

// ScenarioEntry is one scenario's performance rollup.
type ScenarioEntry struct {
	Scenario       string
	Tier           string
	HasBudget      bool
	GoBuildMs      int64
	UIBuildMs      int64
	Regressed      bool
	DegradedReason string
}

// TierDistribution counts scenarios per reachable capture tier.
type TierDistribution struct {
	Tier          string
	ScenarioCount int
}

// ScanError records a scenario enumerated but not gradable.
type ScanError struct {
	Scenario string
	Reason   string
}

// Result is a fleet scan rollup.
type Result struct {
	Entries          []ScenarioEntry
	TierDistribution []TierDistribution
	ScenarioCount    int
	NoBudgetCount    int
	RegressedCount   int
	Errors           []ScanError
}

// Grader produces one ScenarioEntry per scenario. The real grader composes the
// readiness, budgets, and trend engines; tests drive a fake.
type Grader interface {
	Grade(ctx context.Context, scenario string) (ScenarioEntry, error)
}

// Enumerator lists the scenarios to scan when none are requested.
type Enumerator interface {
	List(ctx context.Context) ([]string, error)
}

// Service is the fleet engine.
type Service struct {
	grader     Grader
	enumerator Enumerator
}

// NewService wires a fleet Service over the grader + enumerator seams.
func NewService(grader Grader, enumerator Enumerator) *Service {
	return &Service{grader: grader, enumerator: enumerator}
}

// Scan grades the requested scenarios (or every enumerated scenario) and rolls
// the entries up into offender counts and the tier distribution.
func (s *Service) Scan(ctx context.Context, scenarios []string) (Result, error) {
	if s == nil || s.grader == nil {
		return Result{}, errors.New("fleet: service not wired")
	}
	targets := scenarios
	if len(targets) == 0 {
		if s.enumerator == nil {
			return Result{}, errors.New("fleet: no scenarios requested and no enumerator wired")
		}
		listed, err := s.enumerator.List(ctx)
		if err != nil {
			return Result{}, err
		}
		targets = listed
	}
	sort.Strings(targets)

	res := Result{}
	tierCounts := map[string]int{}
	for _, scenario := range targets {
		entry, err := s.grader.Grade(ctx, scenario)
		if err != nil {
			res.Errors = append(res.Errors, ScanError{Scenario: scenario, Reason: err.Error()})
			continue
		}
		res.Entries = append(res.Entries, entry)
		res.ScenarioCount++
		if !entry.HasBudget {
			res.NoBudgetCount++
		}
		if entry.Regressed {
			res.RegressedCount++
		}
		tierCounts[entry.Tier]++
	}
	res.TierDistribution = distribution(tierCounts)
	return res, nil
}

// NoBudget returns the scenarios with no declared performance budget, sorted by
// scenario name. Deterministic offender query — no AI, no semantic ranking.
func (r Result) NoBudget() []ScenarioEntry {
	var out []ScenarioEntry
	for _, e := range r.Entries {
		if !e.HasBudget {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out
}

// Regressed returns the scenarios that regressed vs. their budget/trend in the
// latest sample, sorted by scenario name.
func (r Result) Regressed() []ScenarioEntry {
	var out []ScenarioEntry
	for _, e := range r.Entries {
		if e.Regressed {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Scenario < out[j].Scenario })
	return out
}

// SlowestBuilds returns the scenarios with the highest combined (go+ui) build
// time, slowest first, bounded by limit (limit <= 0 returns all with a measured
// build). Scenarios with no measured build time are excluded.
func (r Result) SlowestBuilds(limit int) []ScenarioEntry {
	var out []ScenarioEntry
	for _, e := range r.Entries {
		if e.GoBuildMs+e.UIBuildMs > 0 {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		li, lj := out[i].GoBuildMs+out[i].UIBuildMs, out[j].GoBuildMs+out[j].UIBuildMs
		if li != lj {
			return li > lj
		}
		return out[i].Scenario < out[j].Scenario
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

func distribution(counts map[string]int) []TierDistribution {
	out := make([]TierDistribution, 0, len(counts))
	for tier, n := range counts {
		out = append(out, TierDistribution{Tier: tier, ScenarioCount: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ScenarioCount != out[j].ScenarioCount {
			return out[i].ScenarioCount > out[j].ScenarioCount
		}
		return out[i].Tier < out[j].Tier
	})
	return out
}
