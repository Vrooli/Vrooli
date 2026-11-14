package routing

import (
	"fmt"
	"sort"
	"strings"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// scenarioScopeOverfetch is how many times the requested limit each provider is
// asked for when a query is scoped to a scenario. Scenario leaves are a small
// slice of every shared corpus, so the filter needs a deeper candidate pool
// than the caller's display limit; the display limit is re-applied afterwards.
const (
	scenarioScopeOverfetch = 4
	scenarioScopeMinFetch  = 20
	scenarioScopeMaxFetch  = 50
)

// groupScope says how an explicit --group value is interpreted.
//
//   - provider group: the value names a registered provider_group
//     (for example "prompt-manager"); selection is by provider, as before.
//   - scenario scope: the value names no provider group, so it is read as a
//     scenario name and the query is scoped to that scenario's leaves, which
//     live inside shared corpora (skills, programs, commands, docs).
type groupScope struct {
	// Scenario is non-empty only in scenario scope.
	Scenario string
	// KnownGroups lists the registered provider groups, for diagnostics.
	KnownGroups []string
}

// resolveGroupScope classifies req.Group against the active registry.
func resolveGroupScope(active []*registryv1.ProviderDescriptor, group string) groupScope {
	group = strings.TrimSpace(group)
	if group == "" {
		return groupScope{}
	}
	seen := make(map[string]struct{})
	for _, p := range active {
		if g := p.GetProviderGroup(); g != "" {
			seen[g] = struct{}{}
		}
	}
	if _, ok := seen[group]; ok {
		return groupScope{}
	}
	known := make([]string, 0, len(seen))
	for g := range seen {
		known = append(known, g)
	}
	sort.Strings(known)
	return groupScope{Scenario: group, KnownGroups: known}
}

// scenarioScopeFetchLimit returns the per-provider fetch depth for a scoped query.
func scenarioScopeFetchLimit(limit int32) int32 {
	fetch := limit * scenarioScopeOverfetch
	if fetch < scenarioScopeMinFetch {
		fetch = scenarioScopeMinFetch
	}
	if fetch > scenarioScopeMaxFetch {
		fetch = scenarioScopeMaxFetch
	}
	return fetch
}

// selectScenarioTargets chooses the providers searched for a scenario-scoped
// query: the explicit --type set when given, otherwise every project-scope
// provider in the DO and REUSE buckets. Those buckets hold the leaves an agent
// acts with (skills, programs, commands, actions); KNOW and STATE corpora mention
// scenarios in prose and file names, which would drown the scope in documents
// that merely share a name. External corpora are never searched for a project
// scenario unless --all or --type asks for them.
func selectScenarioTargets(active []*registryv1.ProviderDescriptor, req *routingv1.QueryRequest) []*registryv1.ProviderDescriptor {
	unscoped := &routingv1.QueryRequest{Types: req.GetTypes(), All: req.GetAll()}
	if !unscoped.GetAll() && len(nonEmpty(unscoped.GetTypes())) == 0 {
		project, _ := partitionByScope(active)
		unscoped.All = true
		leafBearing := make([]*registryv1.ProviderDescriptor, 0, len(project))
		for _, p := range project {
			if p.GetBucket() == registryv1.Bucket_BUCKET_DO || p.GetBucket() == registryv1.Bucket_BUCKET_REUSE {
				leafBearing = append(leafBearing, p)
			}
		}
		return selectTargets(leafBearing, unscoped)
	}
	return selectTargets(active, unscoped)
}

// hitOwnedByScenario reports whether a result is a leaf of the named scenario.
// Ownership is read from identifiers every provider already returns: the
// provider group, a path under scenarios/<name>/, an id or title of the form
// "<name>", "<name>.<leaf>", or "<name> <leaf>", and for skills the
// "<name>-<suffix>" companions a scenario ships (for example "<name>-improve").
func hitOwnedByScenario(hit *routingv1.SearchHit, scenario string) bool {
	if hit == nil || scenario == "" {
		return false
	}
	if hit.GetProviderGroup() == scenario {
		return true
	}
	if strings.Contains(hit.GetPath(), "/scenarios/"+scenario+"/") {
		return true
	}
	for _, name := range []string{hit.GetId(), hit.GetTitle()} {
		if name == scenario || strings.HasPrefix(name, scenario+".") || strings.HasPrefix(name, scenario+" ") {
			return true
		}
	}
	return hit.GetType() == "skill" && strings.HasPrefix(hit.GetId(), scenario+"-")
}

// filterGroupsToScenario keeps only the scenario's hits in every group, trims
// each group to limit, and returns how many candidate hits were examined.
func filterGroupsToScenario(groups []*routingv1.ProviderResultGroup, scenario string, limit int32) (examined int) {
	for _, group := range groups {
		if group == nil {
			continue
		}
		examined += len(group.GetHits())
		kept := group.Hits[:0:0]
		for _, hit := range group.GetHits() {
			if hitOwnedByScenario(hit, scenario) && int32(len(kept)) < limit {
				kept = append(kept, hit)
			}
		}
		group.Hits = kept
		group.Count = int32(len(kept))
	}
	return examined
}

// scenarioScopeExplanation is always returned for a scenario-scoped query, not
// only under --explain: the scope is inferred, so the caller must be told how
// the value was read and, when nothing matched, what to try instead.
func scenarioScopeExplanation(scope groupScope, groups []*routingv1.ProviderResultGroup, examined int) []string {
	searched := make([]string, 0, len(groups))
	kept := 0
	for _, g := range groups {
		searched = append(searched, g.GetProviderId())
		kept += len(g.GetHits())
	}
	lines := []string{
		fmt.Sprintf("--group %q is not a provider group; read as scenario scope: kept only leaves owned by scenario %q (provider groups: %s)",
			scope.Scenario, scope.Scenario, strings.Join(scope.KnownGroups, ", ")),
	}
	if kept == 0 {
		lines = append(lines, fmt.Sprintf("no leaf of scenario %q matched among %d candidate hit(s) from %d provider(s); the scenario may not expose a leaf for this intent, or its names are not in vocabulary the query shares. Retry without --group, or widen with --type",
			scope.Scenario, examined, len(searched)))
	}
	return lines
}
