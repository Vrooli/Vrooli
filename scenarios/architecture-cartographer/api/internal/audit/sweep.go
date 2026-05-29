package audit

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"

	"architecture-cartographer/internal/conflicts"
)

// ScenarioLister enumerates the scenarios the sweep should touch.
// Production wires a filesystem lister over <repo>/scenarios/; tests
// pass an in-memory fake.
type ScenarioLister interface {
	List(ctx context.Context) ([]string, error)
}

// RunAll executes Run against every scenario matched by the filters and
// aggregates the result. Implemented on the same orchestrator type so
// the per-scenario semantics (snapshot freshness, authority axis,
// suppression accounting) are identical.
func (s *service) RunAll(ctx context.Context, in RunAllInput) (SweepReport, error) {
	if s.scenarios == nil {
		return SweepReport{}, errors.New("audit: no scenario lister configured for RunAll")
	}
	all, err := s.scenarios.List(ctx)
	if err != nil {
		return SweepReport{}, err
	}
	keep := filterScenarios(all, in.IncludeScenarios, in.ExcludeScenarios)
	if len(keep) == 0 {
		return SweepReport{
			BySeverity: map[string]int{},
			ByOutcome:  map[string]int{},
		}, nil
	}

	conc := in.Concurrency
	if conc <= 0 {
		conc = 4
	}
	if conc > len(keep) {
		conc = len(keep)
	}

	start := s.clock.Now()
	reports := make([]Report, len(keep))
	sem := make(chan struct{}, conc)
	var wg sync.WaitGroup
	for i, name := range keep {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, scenario string) {
			defer wg.Done()
			defer func() { <-sem }()
			rep, runErr := s.Run(ctx, RunInput{
				Scenario:          scenario,
				FailOn:            in.FailOn,
				IncludeTypes:      in.IncludeTypes,
				ExcludeTypes:      in.ExcludeTypes,
				AllowLowAuthority: in.AllowLowAuthority,
			})
			if runErr != nil {
				// Per-scenario invalid input shouldn't sink the sweep;
				// capture the tool-error report so the operator sees it.
				rep = Report{Scenario: scenario, Outcome: OutcomeToolError, Error: runErr.Error()}
			}
			reports[idx] = rep
		}(i, name)
	}
	wg.Wait()

	sweep := SweepReport{
		Reports:        reports,
		TotalScenarios: len(reports),
		BySeverity:     map[string]int{},
		ByOutcome:      map[string]int{},
		Duration:       s.clock.Now().Sub(start),
	}
	for _, r := range reports {
		sweep.TotalFindings += r.TotalFindings
		sweep.TotalSuppressed += r.SuppressedFindings
		for sev, n := range r.BySeverity {
			sweep.BySeverity[sev] += n
		}
		sweep.ByOutcome[string(r.Outcome)]++
	}
	return sweep, nil
}

func filterScenarios(all, include, exclude []string) []string {
	includeSet := newSet(include)
	excludeSet := newSet(exclude)
	out := make([]string, 0, len(all))
	for _, name := range all {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if len(includeSet) > 0 {
			if _, ok := includeSet[name]; !ok {
				continue
			}
		}
		if _, ok := excludeSet[name]; ok {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// guard so the unused conflicts import (for severity transitions) sticks
// even if the body simplifies later.
var _ = conflicts.SeverityWarn
