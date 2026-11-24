package routing

import (
	"slices"
	"testing"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/routing"
)

func TestIntentProvidersAddOnlyRelevantEligibleOwners(t *testing.T) {
	design := &registryv1.ProviderDescriptor{ProviderId: "design.skills", Type: "skill", Description: "design layouts and user interfaces"}
	prototype := &registryv1.ProviderDescriptor{ProviderId: "prototype.programs", Type: "library", Description: "build clickable prototypes and interactions"}
	// A declared endpoint makes a registry profile callable.
	for _, provider := range []*registryv1.ProviderDescriptor{design, prototype} {
		provider.Endpoint = &registryv1.Endpoint{}
	}
	_, added := addIntentProviders([]*registryv1.ProviderDescriptor{design}, []*registryv1.ProviderDescriptor{design, prototype},
		[]string{"design layouts", "build clickable prototypes", "build clickable prototypes", "unrelatedzzzz"})
	if !slices.Equal(added, []string{"prototype.programs"}) {
		t.Fatalf("additional providers = %v", added)
	}
}

func TestIntentQueries(t *testing.T) {
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"design a UX", nil},
		{"create a UX design for a fitness tracker app", []string{"create a UX design"}},
		{"design a fitness tracker and then build a clickable prototype", []string{"design a fitness tracker", "build a clickable prototype"}},
		{"design a screen for a fitness tracker and runners", []string{"design a screen"}},
	} {
		if got := intentQueries(tc.query); !slices.Equal(got, tc.want) {
			t.Errorf("intentQueries(%q) = %v, want %v", tc.query, got, tc.want)
		}
	}
}

func TestMergeIntentHitsPreservesOriginalAndAddsDistinctCandidates(t *testing.T) {
	original := &routingv1.ProviderResultGroup{ProviderId: "skills", Count: 1,
		Hits: []*routingv1.SearchHit{{Id: "specific", Score: 0.2}}}
	mergeIntentHits([]*routingv1.ProviderResultGroup{original}, []*routingv1.ProviderResultGroup{
		{ProviderId: "skills", Hits: []*routingv1.SearchHit{{Id: "specific", Score: 0.9}, {Id: "general"}, {Id: "overflow"}}},
	}, 2, "build a prototype")
	if original.GetCount() != 2 || original.Hits[0].GetScore() != 0.2 || original.Hits[1].GetId() != "general" {
		t.Fatalf("merged group = %v", original)
	}
	if got := original.Hits[1].GetMetadata().GetFields()[intentMatchMetadata].GetStringValue(); got != "build a prototype" {
		t.Fatalf("supplemental match origin = %q", got)
	}
}
