package routing

import (
	"regexp"
	"strings"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

// Reserve at most one additional local provider per supplemental intent.
// Only the best positive lexical match may enter, so a variant cannot turn
// into an unrestricted provider sweep.
func addIntentProviders(selected, eligible []*registryv1.ProviderDescriptor, variants []string, typoPolicies ...string) ([]*registryv1.ProviderDescriptor, []string, [][]*registryv1.ProviderDescriptor, []string) {
	if len(variants) == 0 {
		return selected, nil, nil, nil
	}
	typoPolicy := ""
	if len(typoPolicies) > 0 {
		typoPolicy = typoPolicies[0]
	}
	profiles := buildProfiles(eligible)
	byID := make(map[string]*registryv1.ProviderDescriptor, len(eligible))
	for _, provider := range eligible {
		byID[provider.GetProviderId()] = provider
	}
	seen := make(map[string]bool, len(selected))
	for _, provider := range selected {
		seen[provider.GetProviderId()] = true
	}
	var added []string
	byVariant := make([][]*registryv1.ProviderDescriptor, len(variants))
	reasons := make([]string, len(variants))
	for variantIndex, variant := range variants {
		query := variant
		best, scores, _ := routeProviderShortlist(query, profiles, 1, typoPolicy)
		if len(best) == 0 || scores[best[0].ProviderID] <= 0 {
			reasons[variantIndex] = "no_positive_provider_evidence"
			continue
		}
		// Supplemental intent routing is allowed to add one owner only when
		// the evidence identifies that owner uniquely. The selector sorts ties
		// by provider ID for determinism, but that ordering is not relevance
		// evidence and must not resolve an ambiguous subtask.
		bestScore := scores[best[0].ProviderID]
		ambiguous := false
		for _, profile := range profiles {
			if profile.ProviderID != best[0].ProviderID && scores[profile.ProviderID] == bestScore {
				ambiguous = true
				break
			}
		}
		if ambiguous {
			reasons[variantIndex] = "ambiguous_provider_evidence"
			continue
		}
		id := best[0].ProviderID
		provider := byID[id]
		if provider == nil {
			continue
		}
		byVariant[variantIndex] = []*registryv1.ProviderDescriptor{provider}
		if !seen[id] {
			seen[id] = true
			selected = append(selected, provider)
			added = append(added, id)
		}
	}
	return selected, added, byVariant, reasons
}

var nextAction = regexp.MustCompile(`(?i)(?:;\s*|,?\s+and\s+(?:then\s+)?|,\s+then\s+)(?:please\s+)?(?:create|design|build|write|generate|find|search|test|deploy|publish|export|review|summarize|inspect|capture|record|run|compare|analyze)\b`)
var purposeBoundary = regexp.MustCompile(`(?i)\s+for\s+`)

// intentQueries produces supplemental retrieval text. The original request
// still drives primary retrieval and final relevance ranking.
func intentQueries(query string) []string {
	if len(strings.Fields(query)) < 6 || len(query) > 2048 {
		return nil
	}
	if boundary := nextAction.FindStringIndex(query); boundary != nil {
		first := strings.TrimSpace(query[:boundary[0]])
		second := strings.TrimLeft(query[boundary[0]:], ";, ")
		for _, prefix := range []string{"and ", "then ", "please "} {
			if strings.HasPrefix(strings.ToLower(second), prefix) {
				second = second[len(prefix):]
			}
		}
		if len(strings.Fields(first)) >= 2 && len(strings.Fields(second)) >= 2 {
			return []string{first, second}
		}
	}
	if boundary := purposeBoundary.FindStringIndex(query); boundary != nil && len(strings.Fields(query[:boundary[0]])) >= 3 {
		return []string{query[:boundary[0]]}
	}
	return nil
}

// mergeIntentHits keeps one provider group and preserves its original health
// report. Supplemental results contribute candidates without replacing
// original hits or treating scores across queries as comparable.
func mergeIntentHits(groups, supplemental []*routingv1.ProviderResultGroup, limit int) {
	byProvider := make(map[string]*routingv1.ProviderResultGroup, len(groups))
	for _, group := range groups {
		byProvider[group.GetProviderId()] = group
	}
	for _, extra := range supplemental {
		group := byProvider[extra.GetProviderId()]
		if group == nil || len(extra.GetHits()) == 0 {
			continue
		}
		seen := make(map[[3]string]*routingv1.SearchHit)
		key := func(hit *routingv1.SearchHit) [3]string {
			return [3]string{hit.GetType(), hit.GetId(), hit.GetPath()}
		}
		for _, hit := range group.GetHits() {
			seen[key(hit)] = hit
		}
		for _, hit := range extra.GetHits() {
			if existing := seen[key(hit)]; existing != nil {
				continue
			}
			if len(group.Hits) >= limit {
				continue
			}
			seen[key(hit)] = hit
			group.Hits = append(group.Hits, hit)
		}
		group.Count = max(group.Count, int32(len(group.Hits)))
	}
}
