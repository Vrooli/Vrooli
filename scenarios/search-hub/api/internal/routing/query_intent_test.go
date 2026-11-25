package routing

import (
	"fmt"
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

func TestIntentProvidersExposeOnlyUniquelyRankedSingleReserve(t *testing.T) {
	primary := &registryv1.ProviderDescriptor{ProviderId: "ui.primary", Type: "surface", Description: "design a mobile interface for a fitness tracker", Endpoint: &registryv1.Endpoint{}}
	reserve := &registryv1.ProviderDescriptor{ProviderId: "ui.reserve", Type: "page", Description: "design mobile interface", Endpoint: &registryv1.Endpoint{}}
	other := &registryv1.ProviderDescriptor{ProviderId: "records.owner", Type: "record", Description: "find project records", Endpoint: &registryv1.Endpoint{}}
	selected, added, byVariant, reasons := addIntentProviders(nil, []*registryv1.ProviderDescriptor{primary, reserve, other}, []string{"design a mobile interface for a fitness tracker"})
	if !slices.Equal(selected, []*registryv1.ProviderDescriptor{primary}) || !slices.Equal(added, []string{"ui.primary"}) {
		t.Fatalf("primary selection=%v added=%v", selected, added)
	}
	if len(byVariant) != 1 || !slices.Equal(byVariant[0], []*registryv1.ProviderDescriptor{primary, reserve}) || reasons[0] != "" {
		t.Fatalf("primary/reserve choices=%v reason=%q", byVariant, reasons[0])
	}

	other.Description = "design mobile interface"
	_, _, ambiguousReserve, _ := addIntentProviders(nil, []*registryv1.ProviderDescriptor{primary, reserve, other}, []string{"design a mobile interface for a fitness tracker"})
	if len(ambiguousReserve) != 1 || !slices.Equal(ambiguousReserve[0], []*registryv1.ProviderDescriptor{primary}) {
		t.Fatalf("equal reserve evidence must not be broken by ID: %v", ambiguousReserve)
	}
}

func TestUniqueAutomaticReserveExcludesSelectedOwnersAndAbstainsOnTies(t *testing.T) {
	primary := &registryv1.ProviderDescriptor{ProviderId: "ui.primary", Type: "surface", Description: "design mobile surface for a fitness tracker", Endpoint: &registryv1.Endpoint{}}
	reserve := &registryv1.ProviderDescriptor{ProviderId: "ui.reserve", Type: "page", Description: "design mobile surface", Endpoint: &registryv1.Endpoint{}}
	other := &registryv1.ProviderDescriptor{ProviderId: "ui.other", Type: "skill", Description: "surface", Endpoint: &registryv1.Endpoint{}}
	eligible := []*registryv1.ProviderDescriptor{primary, reserve, other}
	if got := uniqueAutomaticReserve("design a mobile surface for a fitness tracker", eligible, []*registryv1.ProviderDescriptor{primary}, ""); got != reserve {
		t.Fatalf("reserve = %v, want unique highest unselected owner %q", got, reserve.GetProviderId())
	}
	other.Description = reserve.GetDescription()
	if got := uniqueAutomaticReserve("design a mobile surface for a fitness tracker", eligible, []*registryv1.ProviderDescriptor{primary}, ""); got != nil {
		t.Fatalf("ambiguous reserve = %q, want abstention", got.GetProviderId())
	}
}

func TestIntentQueries(t *testing.T) {
	for _, tc := range []struct {
		query string
		want  []string
	}{
		{"design a UX", nil},
		{"create a UX design for a fitness tracker app", nil},
		{"design a fitness tracker and then build a clickable prototype", []string{"design a fitness tracker", "build a clickable prototype"}},
		{"design a surface and then run the command and then find project records", []string{"design a surface", "run the command", "find project records"}},
		{"please inspect cli-health.commands then configure ui-health.surfaces today", []string{"inspect cli-health.commands", "configure ui-health.surfaces today"}},
		{"search for \"design and then build\" tools and then run the command", []string{"search for \"design and then build\" tools", "run the command"}},
		{"create a user's view and then update it today", []string{"create a user's view", "update it today"}},
		{"create a design, run the tests, publish the report", []string{"create a design", "run the tests", "publish the report"}},
		{"read the docs; deploy the app", []string{"read the docs", "deploy the app"}},
		{"design a surface and then run the command and then find project records and then write the report", nil},
		{"design a screen for a fitness tracker and runners", nil},
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

func TestMergeIntentHitsPreservesDistinctAnonymousCandidates(t *testing.T) {
	first := &routingv1.SearchHit{Title: "unidentified result one"}
	second := &routingv1.SearchHit{Title: "unidentified result two"}
	original := &routingv1.ProviderResultGroup{ProviderId: "catalog", Hits: []*routingv1.SearchHit{first}}
	mergeIntentHits([]*routingv1.ProviderResultGroup{original}, []*routingv1.ProviderResultGroup{{
		ProviderId: "catalog", Hits: []*routingv1.SearchHit{second},
	}}, 3)
	if len(original.GetHits()) != 2 || original.GetHits()[0] != first || original.GetHits()[1] != second {
		t.Fatalf("anonymous candidates were merged or reordered: %#v", original.GetHits())
	}
}

func TestAutomaticSupplementalHitLimitRetainsBoundedPrimaryAndReserveEvidence(t *testing.T) {
	primaryLimit, parts := int32(10), 3
	limit := automaticSupplementalHitLimit(primaryLimit, parts)
	primaryHits := make([]*routingv1.SearchHit, 0, primaryLimit)
	for i := 0; i < int(primaryLimit); i++ {
		id := fmt.Sprintf("full-%d", i)
		primaryHits = append(primaryHits, &routingv1.SearchHit{Id: id, Type: "surface", Path: id})
	}
	groups := []*routingv1.ProviderResultGroup{{ProviderId: "ui.owner", Hits: primaryHits}}
	for part := 0; part < parts; part++ {
		for _, prefix := range []string{"primary", "reserve"} {
			extraHits := make([]*routingv1.SearchHit, 0, maxSupplementalHitsPerPart)
			for hit := 0; hit < maxSupplementalHitsPerPart; hit++ {
				id := fmt.Sprintf("%s-%d-%d", prefix, part, hit)
				extraHits = append(extraHits, &routingv1.SearchHit{Id: id, Type: "surface", Path: id})
			}
			// Both bounded attempts can merge into the same owner group.
			groups = mergeIntentHits(groups, []*routingv1.ProviderResultGroup{{ProviderId: "ui.owner", Hits: extraHits}}, limit)
		}
	}
	if got := len(groups[0].GetHits()); got != limit {
		t.Fatalf("merged hits = %d, want all %d full-query + primary/reserve candidates", got, limit)
	}
}
