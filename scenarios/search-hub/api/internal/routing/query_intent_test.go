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
	_, added, byVariant, reasons := addIntentProviders([]*registryv1.ProviderDescriptor{design}, []*registryv1.ProviderDescriptor{design, prototype},
		[]string{"design layouts", "build clickable prototypes", "build clickable prototypes", "unrelatedzzzz"})
	if !slices.Equal(added, []string{"prototype.programs"}) {
		t.Fatalf("additional providers = %v", added)
	}
	if len(byVariant) != 4 || !slices.Equal(byVariant[0], []*registryv1.ProviderDescriptor{design}) ||
		!slices.Equal(byVariant[1], []*registryv1.ProviderDescriptor{prototype}) || len(byVariant[3]) != 0 {
		t.Fatalf("per-intent owners = %#v", byVariant)
	}
	if reasons[3] != "no_positive_provider_evidence" {
		t.Fatalf("no-match reason = %q", reasons[3])
	}
}

func TestIntentProvidersCanUseUniqueBoundedTypoEvidenceOnCandidatePolicy(t *testing.T) {
	command := &registryv1.ProviderDescriptor{ProviderId: "cli-health.commands", Type: "command", Description: "run a command"}
	command.Endpoint = &registryv1.Endpoint{}
	selected, added, byVariant, _ := addIntentProviders(nil, []*registryv1.ProviderDescriptor{command}, []string{"commnad"}, routeTypoPolicyOneEditUnique)
	if !slices.Equal(added, []string{"cli-health.commands"}) || !slices.Equal(selected, []*registryv1.ProviderDescriptor{command}) {
		t.Fatalf("typo candidate selected=%v added=%v", selected, added)
	}
	if len(byVariant) != 1 || !slices.Equal(byVariant[0], []*registryv1.ProviderDescriptor{command}) {
		t.Fatalf("typo intent owners = %#v", byVariant)
	}
	selected, added, byVariant, _ = addIntentProviders(nil, []*registryv1.ProviderDescriptor{command, {
		ProviderId: "another.commands", Type: "workflow", Description: "run a command",
		Endpoint: &registryv1.Endpoint{},
	}}, []string{"commnad"}, routeTypoPolicyOneEditUnique)
	if len(selected) != 0 || len(added) != 0 {
		t.Fatalf("ambiguous typo candidates selected=%v added=%v", selected, added)
	}
	if len(byVariant) != 1 || len(byVariant[0]) != 0 {
		t.Fatalf("ambiguous typo intent owners = %#v", byVariant)
	}
}

func TestIntentProvidersDoNotBreakEqualEvidenceTiesByProviderID(t *testing.T) {
	first := &registryv1.ProviderDescriptor{
		ProviderId: "owner-alpha", Type: "skill", Description: "build clickable prototypes",
		Endpoint: &registryv1.Endpoint{},
	}
	second := &registryv1.ProviderDescriptor{
		ProviderId: "owner-beta", Type: "program", Description: "build clickable prototypes",
		Endpoint: &registryv1.Endpoint{},
	}

	selected, added, byVariant, reasons := addIntentProviders(nil, []*registryv1.ProviderDescriptor{first, second}, []string{"build clickable prototypes"})
	if len(selected) != 0 || len(added) != 0 {
		t.Fatalf("ambiguous intent selected=%v added=%v", selected, added)
	}
	if len(byVariant) != 1 || len(byVariant[0]) != 0 {
		t.Fatalf("ambiguous intent owners = %#v", byVariant)
	}
	if reasons[0] != "ambiguous_provider_evidence" {
		t.Fatalf("ambiguous intent reason = %q", reasons[0])
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
	}, 2)
	if original.GetCount() != 2 || original.Hits[0].GetScore() != 0.2 || original.Hits[1].GetId() != "general" {
		t.Fatalf("merged group = %v", original)
	}
}
