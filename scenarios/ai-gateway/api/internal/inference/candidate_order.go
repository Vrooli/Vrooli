package inference

import (
	"sort"
	"strings"

	"ai-gateway/internal/providers"
	profilepolicy "github.com/vrooli/ai-provider-go/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

// orderedCandidates applies the same profile locality ordering used by generic
// routing while retaining catalog order within equal locality/provider ranks.
// The returned slice is a copy, so a request never mutates the loaded catalog.
func orderedCandidates(candidates []Candidate, adapters map[string]providers.Adapter, profile sharedv1.Profile) []Candidate {
	ordered := append([]Candidate(nil), candidates...)
	rank := func(candidate Candidate) int {
		adapter, ok := adapters[strings.ToLower(strings.TrimSpace(candidate.Provider))]
		if !ok {
			return 1000
		}
		return profilepolicy.Rank(profile.String(), adapter.Locality, candidate.Provider)
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return rank(ordered[i]) < rank(ordered[j])
	})
	return ordered
}
