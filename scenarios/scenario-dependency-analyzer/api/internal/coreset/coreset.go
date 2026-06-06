// Package coreset computes the live reflexive core set of scenarios that the
// Baseline Modes self-improvement machinery depends on (P3 stage (a)).
//
// The core set is the hardcoded seed (owned by github.com/vrooli/api-core/coreset
// so it cannot drift) UNIONED with the fresh transitive Required-closure read
// directly from each scenario's .vrooli/service.json on disk. The seed is always
// included, so unreliable `required` flags can only ADD members, never drop one.
//
// This path is deliberately DATABASE-FREE: it reuses deployment.BuildDependencyNodeList
// (fresh from disk) and never touches the stale postgres-backed dependency store,
// so the core set can be computed even when postgres is down. Over-inclusion is
// safe; under-inclusion is dangerous, hence seed-always-included.
package coreset

import (
	"path/filepath"
	"scenario-dependency-analyzer/internal/config"
	"scenario-dependency-analyzer/internal/deployment"
	"sort"
	"strings"

	apicoreset "github.com/vrooli/api-core/coreset"
)

// Result is the computed core-set report returned by the API/CLI.
type Result struct {
	// Source is "computed" when the closure ran over a real scenarios
	// directory, or "fallback" when the directory was unusable and only the
	// hardcoded seed could be returned.
	Source string `json:"source"`
	// CoreSet is the full reflexive set (seed ∪ Required-closure), sorted.
	CoreSet []string `json:"core_set"`
	// Seed is the always-included hardcoded authority (the 9-seed).
	Seed []string `json:"seed"`
	// AddedByClosure lists the members contributed by the transitive Required
	// closure beyond the seed (sorted; empty when the seed already covers it).
	AddedByClosure []string `json:"added_by_closure"`
	// TrustedBase is the subset of the core set that must never be shadowed.
	TrustedBase []string `json:"trusted_base"`
	// LoadErrors maps a scenario name to a non-fatal load error (its
	// service.json was missing or invalid). Seed members with a load error
	// remain in the core set; only their outbound Required edges are skipped.
	LoadErrors map[string]string `json:"load_errors,omitempty"`
}

// Compute returns the live core set rooted at scenariosDir. It never fails: the
// worst case is Source=="fallback" with CoreSet equal to the hardcoded seed.
func Compute(scenariosDir string) Result {
	seed := apicoreset.CoreSeedScenarios()

	scenariosDir = strings.TrimSpace(scenariosDir)
	if scenariosDir == "" {
		// No directory to traverse — return the safe over-inclusive seed.
		return Result{
			Source:      "fallback",
			CoreSet:     seed,
			Seed:        seed,
			TrustedBase: trustedBaseSubset(seed),
		}
	}

	members := make(map[string]struct{}, len(seed))
	for _, name := range seed {
		members[config.NormalizeName(name)] = struct{}{}
	}

	loadErrors := map[string]string{}
	added := map[string]struct{}{}

	// Worklist BFS over Required==true scenario edges. The worklist (rather than
	// walking each node's recursively-built Children) keeps the closure logic in
	// one place and makes transitivity obvious: every newly added scenario is
	// re-loaded fresh and its own Required edges expanded.
	queue := append([]string(nil), seed...)
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]

		scenarioPath := filepath.Join(scenariosDir, name)
		cfg, err := config.LoadServiceConfig(scenarioPath)
		if err != nil {
			// Non-fatal: the scenario stays in the set (if it was a seed/closure
			// member) but contributes no outbound edges.
			loadErrors[name] = err.Error()
			continue
		}

		nodes := deployment.BuildDependencyNodeList(scenariosDir, name, cfg, map[string]struct{}{})
		for _, node := range nodes {
			if node.Type != "scenario" || node.Required == nil || !*node.Required {
				continue
			}
			norm := config.NormalizeName(node.Name)
			if norm == "" {
				continue
			}
			if _, seen := members[norm]; seen {
				continue
			}
			members[norm] = struct{}{}
			added[norm] = struct{}{}
			queue = append(queue, norm)
		}
	}

	coreSet := sortedKeys(members)
	result := Result{
		Source:         "computed",
		CoreSet:        coreSet,
		Seed:           seed,
		AddedByClosure: sortedKeys(added),
		TrustedBase:    trustedBaseSubset(coreSet),
	}
	if len(loadErrors) > 0 {
		result.LoadErrors = loadErrors
	}
	return result
}

// trustedBaseSubset returns the trusted-base members that are present in the
// given set, preserving the authoritative subset from api-core.
func trustedBaseSubset(set []string) []string {
	present := make(map[string]struct{}, len(set))
	for _, name := range set {
		present[config.NormalizeName(name)] = struct{}{}
	}
	out := make([]string, 0, len(present))
	for _, name := range apicoreset.TrustedBaseScenarios() {
		if _, ok := present[config.NormalizeName(name)]; ok {
			out = append(out, config.NormalizeName(name))
		}
	}
	sort.Strings(out)
	return out
}

func sortedKeys(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
