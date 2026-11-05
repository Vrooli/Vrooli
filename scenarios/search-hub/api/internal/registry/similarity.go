package registry

import (
	"sort"
	"strings"
	"unicode"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// DescriptorSimilarityWarning is an advisory registration-time signal. It
// never blocks an upsert: adjacent providers may intentionally have similar
// descriptions, but the operator should know when automatic routing has little
// discriminative text to work with.
type DescriptorSimilarityWarning struct {
	ExistingProviderID string
	Similarity         float64
}

func SimilarDescriptorWarnings(candidate *registryv1.ProviderDescriptor, existing []*registryv1.ProviderDescriptor) []DescriptorSimilarityWarning {
	if candidate == nil {
		return nil
	}
	warnings := make([]DescriptorSimilarityWarning, 0)
	for _, other := range existing {
		if other == nil || other.GetProviderId() == candidate.GetProviderId() {
			continue
		}
		similarity := descriptorSimilarity(candidate, other)
		if similarity >= 0.60 {
			warnings = append(warnings, DescriptorSimilarityWarning{ExistingProviderID: other.GetProviderId(), Similarity: similarity})
		}
	}
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].Similarity == warnings[j].Similarity {
			return warnings[i].ExistingProviderID < warnings[j].ExistingProviderID
		}
		return warnings[i].Similarity > warnings[j].Similarity
	})
	return warnings
}

func descriptorSimilarity(a, b *registryv1.ProviderDescriptor) float64 {
	left := descriptorTerms(a)
	right := descriptorTerms(b)
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	intersection := 0
	union := make(map[string]struct{}, len(left)+len(right))
	for term := range left {
		union[term] = struct{}{}
	}
	for term := range right {
		if _, ok := left[term]; ok {
			intersection++
		}
		union[term] = struct{}{}
	}
	return float64(intersection) / float64(len(union))
}

func descriptorTerms(descriptor *registryv1.ProviderDescriptor) map[string]struct{} {
	// Provider groups are often scenario namespaces and should not dilute an
	// otherwise identical retrieval description. The router's scoring text still
	// includes the group; this advisory specifically measures descriptive
	// indistinguishability.
	text := strings.ToLower(strings.Join([]string{descriptor.GetType(), descriptor.GetDescription()}, " "))
	terms := make(map[string]struct{})
	for _, raw := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
		if len(raw) >= 3 {
			terms[raw] = struct{}{}
		}
	}
	return terms
}
