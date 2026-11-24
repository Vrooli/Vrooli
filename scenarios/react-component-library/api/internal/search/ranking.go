package search

import (
	"sort"
	"strings"

	searchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/search"
)

// dedupeCandidates keeps the highest-scoring row per kind+subject. A subject can
// legitimately appear under two kinds (an asset and the page type it realizes),
// but never twice under the same kind.
func dedupeCandidates(candidates []candidate) []candidate {
	best := make(map[string]int, len(candidates))
	out := make([]candidate, 0, len(candidates))
	for _, cand := range candidates {
		key := dedupeKey(cand.doc.Kind, cand.doc.SubjectID)
		if index, ok := best[key]; ok {
			if cand.score > out[index].score {
				out[index] = cand
			}
			continue
		}
		best[key] = len(out)
		out = append(out, cand)
	}
	return out
}

func requestedKinds(kinds []searchv1.SearchKind) map[string]bool {
	if len(kinds) == 0 {
		return nil
	}
	out := map[string]bool{}
	for _, kind := range kinds {
		if text := KindText(kind); text != "" {
			out[text] = true
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterKinds(candidates []candidate, kinds map[string]bool) []candidate {
	out := candidates[:0]
	for _, cand := range candidates {
		if kinds[KindText(cand.doc.Kind)] {
			out = append(out, cand)
		}
	}
	return out
}

// defaultBrowseKinds is the curated empty-query set. It mirrors the program's
// five browse destinations; display/state/evidence stay reachable through their
// parent or through an explicit kind filter.
var defaultBrowseKinds = map[string]bool{
	kindWorkspace: true,
	kindPage:      true,
	kindPageType:  true,
	kindAsset:     true,
	kindCommand:   true,
}

func sortRanked(candidates []candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].intentRank != candidates[j].intentRank {
			return candidates[i].intentRank > candidates[j].intentRank
		}
		// Semantic-only hits make paraphrase discovery possible, but rank-only
		// RRF can tie one with a top lexical hit. Keep documents that satisfy
		// the lexical query ahead of semantic-only candidates so generic vector
		// neighbors cannot displace query-specific results at small limits.
		if candidates[i].lexicalMatch != candidates[j].lexicalMatch {
			return candidates[i].lexicalMatch
		}
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		left := strings.ToLower(candidates[i].doc.Title)
		right := strings.ToLower(candidates[j].doc.Title)
		if left != right {
			return left < right
		}
		return candidates[i].doc.ID < candidates[j].doc.ID
	})
}

// sortBrowse preserves the curated kind order. Availability and title break ties
// inside a kind; score is not the ordering key here because a planned workspace
// must still appear before an available command.
func sortBrowse(candidates []candidate) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i].doc, candidates[j].doc
		if l, r := browseKindRank(KindText(left.Kind)), browseKindRank(KindText(right.Kind)); l != r {
			return l < r
		}
		if l, r := availabilityRank(AvailabilityText(left.Availability)), availabilityRank(AvailabilityText(right.Availability)); l != r {
			return l < r
		}
		lt, rt := strings.ToLower(left.Title), strings.ToLower(right.Title)
		if lt != rt {
			return lt < rt
		}
		return left.ID < right.ID
	})
}

func buildFacets(candidates []candidate) []*searchv1.SearchFacet {
	kindCounts := map[string]int32{}
	availabilityCounts := map[string]int32{}
	for _, cand := range candidates {
		kindCounts[KindText(cand.doc.Kind)]++
		availabilityCounts[AvailabilityText(cand.doc.Availability)]++
	}
	return []*searchv1.SearchFacet{
		facet("kind", "Kind", kindCounts, kindFacetOrder),
		facet("availability", "Availability", availabilityCounts, availabilityFacetOrder),
	}
}

var (
	kindFacetOrder         = []string{kindWorkspace, kindPage, kindDisplay, kindState, kindPageType, kindAsset, kindCommand, kindEvidence}
	availabilityFacetOrder = []string{availabilityAvailable, availabilityDraft, availabilityPlanned, availabilityUnavailable}
)

func facet(key, label string, counts map[string]int32, order []string) *searchv1.SearchFacet {
	out := &searchv1.SearchFacet{Key: key, Label: label}
	for _, value := range order {
		count := counts[value]
		if count == 0 {
			continue
		}
		valueLabel := KindLabel(value)
		if key == "availability" {
			valueLabel = AvailabilityLabel(value)
		}
		out.Values = append(out.Values, &searchv1.SearchFacetValue{Value: value, Label: valueLabel, Count: count})
	}
	return out
}
