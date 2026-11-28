package routing

import (
	"regexp"
	"sort"
	"strings"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

const maxSupplementalIntentParts = 3
const maxSupplementalHitsPerPart = 3

// automaticSupplementalHitLimit reserves room for the bounded primary and
// single-reserve retrievals for every compound part, in addition to the full
// query's provider limit. This keeps returned part evidence present in groups.
func automaticSupplementalHitLimit(primaryLimit int32, partCount int) int {
	if primaryLimit < 0 {
		primaryLimit = 0
	}
	if partCount < 0 {
		partCount = 0
	}
	perAttempt := min(int(primaryLimit), maxSupplementalHitsPerPart)
	return int(primaryLimit) + 2*perAttempt*partCount
}

// uniqueAutomaticReserve selects one positive, uniquely best unselected owner
// from the already eligible automatic pool. It makes no reachability claim;
// the caller must use it only after a selected owner's outcome warrants a
// bounded retry.
func uniqueAutomaticReserve(query string, eligible, selected []*registryv1.ProviderDescriptor, typoPolicy string) *registryv1.ProviderDescriptor {
	profiles := buildProfiles(eligible)
	if len(profiles) == 0 {
		return nil
	}
	excluded := make(map[string]struct{}, len(selected))
	byID := make(map[string]*registryv1.ProviderDescriptor, len(eligible))
	for _, provider := range selected {
		if provider != nil {
			excluded[provider.GetProviderId()] = struct{}{}
		}
	}
	for _, provider := range eligible {
		if provider != nil {
			byID[provider.GetProviderId()] = provider
		}
	}
	ranked, scores, _ := routeProviderShortlist(query, profiles, len(profiles), typoPolicy)
	var best *ProviderProfile
	for i := range ranked {
		candidate := &ranked[i]
		if _, skip := excluded[candidate.ProviderID]; skip {
			continue
		}
		score := scores[candidate.ProviderID]
		if score <= 0 {
			return nil
		}
		if best == nil {
			best = candidate
			continue
		}
		if score == scores[best.ProviderID] {
			return nil
		}
		break
	}
	if best == nil {
		return nil
	}
	return byID[best.ProviderID]
}

// Select one primary and at most one uniquely ranked reserve local provider
// per supplemental intent. The reserve is used only if the primary fails, so
// a variant cannot turn into an unrestricted provider sweep.
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
		ranked, scores, _ := routeProviderShortlist(query, profiles, len(profiles), typoPolicy)
		if len(ranked) == 0 || scores[ranked[0].ProviderID] <= 0 {
			reasons[variantIndex] = "no_positive_provider_evidence"
			continue
		}
		// Supplemental intent routing is allowed to add one owner only when
		// the evidence identifies that owner uniquely. The selector sorts ties
		// by provider ID for determinism, but that ordering is not relevance
		// evidence and must not resolve an ambiguous subtask.
		bestScore := scores[ranked[0].ProviderID]
		ambiguous := false
		for _, profile := range profiles {
			if profile.ProviderID != ranked[0].ProviderID && scores[profile.ProviderID] == bestScore {
				ambiguous = true
				break
			}
		}
		if ambiguous {
			reasons[variantIndex] = "ambiguous_provider_evidence"
			continue
		}
		id := ranked[0].ProviderID
		provider := byID[id]
		if provider == nil {
			continue
		}
		byVariant[variantIndex] = []*registryv1.ProviderDescriptor{provider}
		if len(ranked) > 1 {
			reserveScore := scores[ranked[1].ProviderID]
			reserveAmbiguous := reserveScore <= 0
			for _, profile := range profiles {
				if profile.ProviderID != ranked[0].ProviderID && profile.ProviderID != ranked[1].ProviderID && scores[profile.ProviderID] == reserveScore {
					reserveAmbiguous = true
					break
				}
			}
			if !reserveAmbiguous {
				if reserve := byID[ranked[1].ProviderID]; reserve != nil {
					byVariant[variantIndex] = append(byVariant[variantIndex], reserve)
				}
			}
		}
		if !seen[id] {
			seen[id] = true
			selected = append(selected, provider)
			added = append(added, id)
		}
	}
	return selected, added, byVariant, reasons
}

var nextAction = regexp.MustCompile(`(?i)(?:;\s*|,?\s+and\s+(?:then\s+)?|,\s+then\s+|\s+then\s+|\s+next\s+|,\s*)(?:please\s+)?(?:create|design|build|write|generate|find|search|test|deploy|publish|export|review|summarize|inspect|capture|record|run|compare|analyze|open|use|start|stop|check|fix|update|edit|implement|configure|execute|launch|explain|read|query|list|retrieve|evaluate|measure|identify|recommend|choose|plan|monitor)\b`)

// intentQueries produces supplemental retrieval text. The original request
// still drives primary retrieval and final relevance ranking. A purpose
// phrase ("for a fitness tracker") is a constraint on that request, not a
// separate intent; splitting there would discard the target from supplemental
// owner selection. Only explicit second actions create more intents. The
// fixed part cap bounds supplemental fan-out for unusually long requests. If a
// request exceeds the cap, keep it whole as the primary query instead of
// presenting a partial decomposition as complete.
func intentQueries(query string) []string {
	if len(strings.Fields(query)) < 6 || len(query) > 2048 {
		return nil
	}
	boundaries := unquotedActionBoundaries(query, maxSupplementalIntentParts)
	if len(boundaries) == 0 {
		return nil
	}
	if len(boundaries) >= maxSupplementalIntentParts {
		return nil
	}

	parts := make([]string, 0, len(boundaries)+1)
	start := 0
	for _, boundary := range boundaries {
		parts = append(parts, cleanIntentSegment(query[start:boundary[0]]))
		start = boundary[0]
	}
	parts = append(parts, cleanIntentSegment(query[start:]))
	for _, part := range parts {
		if len(strings.Fields(part)) < 2 {
			return nil
		}
	}
	return parts
}

func unquotedActionBoundaries(query string, limit int) [][]int {
	if limit <= 0 {
		return nil
	}
	allMatches := nextAction.FindAllStringIndex(query, -1)
	boundaries := make([][]int, 0, min(len(allMatches), limit))
	var quote byte
	escaped := false
	position := 0
	for _, match := range allMatches {
		for position < match[0] {
			character := query[position]
			if escaped {
				escaped = false
				position++
				continue
			}
			if quote != 0 && character == '\\' {
				escaped = true
				position++
				continue
			}
			if quote != 0 {
				if character == quote {
					quote = 0
				}
				position++
				continue
			}
			if character == '"' || character == '`' || (character == '\'' && (position == 0 || !isRouteWordByte(query[position-1]))) {
				quote = character
			}
			position++
		}
		if quote == 0 {
			boundaries = append(boundaries, match)
			if len(boundaries) >= limit {
				return boundaries
			}
		}
		position = match[1]
	}
	return boundaries
}

func isRouteWordByte(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9' || character >= 0x80
}

func cleanIntentSegment(segment string) string {
	segment = strings.TrimLeft(segment, ";, ")
	for {
		trimmed := false
		for _, prefix := range []string{"and ", "then ", "please "} {
			if strings.HasPrefix(strings.ToLower(segment), prefix) {
				segment = strings.TrimSpace(segment[len(prefix):])
				trimmed = true
				break
			}
		}
		if !trimmed {
			return strings.TrimSpace(segment)
		}
	}
}

// mergeIntentHits keeps one provider group and preserves its primary health
// report when present. A supplemental-only provider retains its own group so
// its hits or degradation remain visible. Scores across queries are never
// compared.
func mergeIntentHits(groups, supplemental []*routingv1.ProviderResultGroup, limit int) []*routingv1.ProviderResultGroup {
	byProvider := make(map[string]*routingv1.ProviderResultGroup, len(groups))
	for _, group := range groups {
		byProvider[group.GetProviderId()] = group
	}
	for _, extra := range supplemental {
		if extra == nil {
			continue
		}
		group := byProvider[extra.GetProviderId()]
		if group == nil {
			if limit > 0 && len(extra.GetHits()) > limit {
				extra.Hits = extra.GetHits()[:limit]
			}
			extra.Count = max(extra.GetCount(), int32(len(extra.GetHits())))
			groups = append(groups, extra)
			byProvider[extra.GetProviderId()] = extra
			continue
		}
		if len(extra.GetHits()) == 0 {
			continue
		}
		seen := make(map[[3]string]*routingv1.SearchHit)
		seenAnonymous := make(map[*routingv1.SearchHit]struct{})
		key := func(hit *routingv1.SearchHit) [3]string {
			return [3]string{hit.GetType(), hit.GetId(), hit.GetPath()}
		}
		for _, hit := range group.GetHits() {
			if !hasStableHitIdentity(hit) {
				seenAnonymous[hit] = struct{}{}
				continue
			}
			seen[key(hit)] = hit
		}
		for _, hit := range extra.GetHits() {
			if !hasStableHitIdentity(hit) {
				if _, exists := seenAnonymous[hit]; exists {
					continue
				}
			} else if existing := seen[key(hit)]; existing != nil {
				continue
			}
			if len(group.Hits) >= limit {
				continue
			}
			if hasStableHitIdentity(hit) {
				seen[key(hit)] = hit
			} else {
				seenAnonymous[hit] = struct{}{}
			}
			group.Hits = append(group.Hits, hit)
		}
		group.Count = max(group.Count, int32(len(group.Hits)))
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return groups[i].GetProviderId() < groups[j].GetProviderId()
	})
	return groups
}
