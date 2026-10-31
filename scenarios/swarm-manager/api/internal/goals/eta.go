package goals

import "swarm-manager/internal/eta"

// EstimatorFactory builds a fresh ETA estimator from the current duration
// samples and execute-lane capacity. It may return (nil, nil) when estimation
// is unavailable (e.g. the event log is not wired), in which case the ETA band
// is simply omitted from the goal response.
type EstimatorFactory func() (*eta.Estimator, error)

// closureInput maps a computed scope and the scope input it was built from into
// the pure ETA rollup input: per-item effort class, done/gated flags, and the
// dependency edges that stay within the closure. "Gated" mirrors the scope's
// blocked set — an item still waiting on a prerequisite or gate — so the rollup
// adds gate-wait latency to exactly those items.
func closureInput(scope Scope, in ScopeInput) eta.GoalClosureInput {
	inClosure := make(map[string]bool, len(scope.Closure))
	for _, ref := range scope.Closure {
		inClosure[ref] = true
	}
	done := make(map[string]bool, len(scope.Completed))
	for _, ref := range scope.Completed {
		done[ref] = true
	}
	blocked := make(map[string]bool, len(scope.Blocked))
	for _, ref := range scope.Blocked {
		blocked[ref] = true
	}

	out := eta.GoalClosureInput{Deps: make(map[string][]string, len(scope.Closure))}
	for _, ref := range scope.Closure {
		out.Items = append(out.Items, eta.ClosureItem{
			Ref:         ref,
			EffortClass: eta.NormalizeEffort(in.ItemEffort[ref]),
			Done:        done[ref],
			Gated:       blocked[ref],
		})
		var deps []string
		for _, d := range in.ItemDeps[ref] {
			if inClosure[d] {
				deps = append(deps, d)
			}
		}
		out.Deps[ref] = deps
	}
	return out
}
