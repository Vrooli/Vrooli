package hostcapability

import "context"

type Site struct {
	Name       string
	Invariants []Invariant
	Walked     bool
	Reason     string
}

type Coverage struct {
	SitesWalked         int      `json:"sites_walked"`
	InvariantsDeclared  int      `json:"invariants_declared"`
	InvariantsEvaluated int      `json:"invariants_evaluated"`
	Gaps                []string `json:"gaps,omitempty"`
}

func Aggregate(sites []Site, evaluated map[string]int) Coverage {
	coverage := Coverage{Gaps: []string{}}
	for _, site := range sites {
		if !site.Walked {
			reason := site.Reason
			if reason == "" {
				reason = "declaration site was not walked"
			}
			coverage.Gaps = append(coverage.Gaps, site.Name+": "+reason)
			continue
		}
		coverage.SitesWalked++
		coverage.InvariantsDeclared += len(site.Invariants)
		coverage.InvariantsEvaluated += evaluated[site.Name]
	}
	return coverage
}

// EvaluateSites walks the same registered sites as Aggregate and resolves
// every declaration in a walked site. It returns results separately so a
// caller can retain verdict evidence without weakening coverage accounting.
func EvaluateSites(ctx context.Context, registry *Registry, sites []Site, facts Facts) (Coverage, []Result) {
	coverage := Coverage{Gaps: []string{}}
	var results []Result
	for _, site := range sites {
		if !site.Walked {
			reason := site.Reason
			if reason == "" {
				reason = "declaration site was not walked"
			}
			coverage.Gaps = append(coverage.Gaps, site.Name+": "+reason)
			continue
		}
		coverage.SitesWalked++
		coverage.InvariantsDeclared += len(site.Invariants)
		resolved := Evaluate(ctx, registry, site.Invariants, facts)
		coverage.InvariantsEvaluated += len(resolved)
		results = append(results, resolved...)
	}
	return coverage, results
}
