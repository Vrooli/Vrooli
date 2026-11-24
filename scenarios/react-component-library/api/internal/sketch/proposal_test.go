package sketch

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"react-component-library/internal/availability"
	"react-component-library/internal/search"
)

type proposalSearch []search.CatalogResult

func (p proposalSearch) SearchCatalog(context.Context, string, int, string, string, string) ([]search.CatalogResult, error) {
	return p, nil
}

func (p proposalSearch) LookupCatalogID(context.Context, string) (search.CatalogResult, bool, error) {
	return search.CatalogResult{}, false, nil
}

func TestProposalFiltersCompatibilityAndPreservesAuthoredIntent(t *testing.T) {
	var results proposalSearch
	for i := 0; i < 6; i++ {
		id := fmt.Sprintf("templates.page-%d", i)
		results = append(results, search.CatalogResult{
			CatalogMetadata: search.CatalogMetadata{CatalogID: id, Name: id, Kind: "page-template", Regions: []string{"main"}, Targets: []string{"react-vite"}, Kits: []string{"vrooli-default"}, RequiredCapabilities: []string{"keyboard"}},
			Availability:    availability.Result{CatalogID: id, State: availability.Built, Version: "1.2.3"},
		})
	}
	results[0].Availability.State = availability.Declared
	results[1].Kits = []string{"other"}
	intent := DesignIntent{Intent: "Browse records", Users: []string{"operator"}, PrimaryTasks: []string{"find a record"}, Target: "react-vite", Kit: "vrooli-default", Viewports: []string{"phone", "desktop"}, RequiredCapabilities: []string{"keyboard"}}
	snapshot := Snapshot{DeclaredRegions: []Region{{ID: "authored", Note: "Preserve context"}}, Document: Document{Notes: []Note{{Scope: "page", Text: "Keep routes"}}}}
	got, err := Propose(context.Background(), snapshot, intent, 3, results)
	if err != nil || len(got) != 3 {
		t.Fatal("bounded compatible proposals missing", err, len(got))
	}
	if got[0].Document.Template.Asset != "templates.page-2" || got[0].Document.Template.Version != "1.2.3" {
		t.Fatal("selected incompatible or invented identity")
	}
	if len(got[0].Document.Regions) != 2 || got[0].Document.Intent.Intent != intent.Intent || len(got[0].Document.Notes) != 1 || got[0].Document.Render != nil || len(got[0].Obligations) == 0 {
		t.Fatal("candidate lost intent or claimed completed rendering")
	}
	defaults := got[0].Document.Intent.Constraints
	if defaults == nil || defaults.PreserveRoutes == nil || !*defaults.PreserveRoutes || defaults.PreserveBusinessBehavior == nil || !*defaults.PreserveBusinessBehavior || intent.Constraints != nil {
		t.Fatal("missing preservation defaults or mutated caller")
	}
	disabled := false
	intent.Constraints = &DesignConstraints{DesignSource: "path:docs/DESIGN.md", PreserveRoutes: &disabled, PreserveBusinessBehavior: &disabled}
	constrained, err := Propose(context.Background(), snapshot, intent, 1, results)
	if err != nil {
		t.Fatal(err)
	}
	c := constrained[0].Document.Intent.Constraints
	obligations := strings.Join(constrained[0].Obligations, " ")
	if c.DesignSource != "docs/DESIGN.md" || *c.PreserveRoutes || *c.PreserveBusinessBehavior ||
		!strings.Contains(obligations, "not proof of source conformance") ||
		!strings.Contains(obligations, "Route preservation is explicitly disabled") ||
		!strings.Contains(obligations, "Business behavior preservation is explicitly disabled") {
		t.Fatalf("constraints or obligations lost: %+v %s", c, obligations)
	}
	if intent.Constraints.DesignSource != "path:docs/DESIGN.md" {
		t.Fatal("caller source mutated")
	}
	for _, source := range []string{"../DESIGN.md", "/DESIGN.md", "docs/../../DESIGN.md", "https://example.com", ".", "docs\\DESIGN.md"} {
		intent.Constraints = &DesignConstraints{DesignSource: source}
		if _, err := Propose(context.Background(), snapshot, intent, 1, results); err == nil {
			t.Errorf("accepted invalid source %q", source)
		}
	}
	intent.Constraints = nil
	if snapshot.Document.Template != nil || snapshot.Document.Intent != nil {
		t.Fatal("proposal mutated current page")
	}
	if _, err := Propose(context.Background(), snapshot, intent, 4, results); err == nil {
		t.Fatal("unbounded candidates accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Propose(ctx, snapshot, intent, 3, results); err != context.Canceled {
		t.Fatal("cancellation lost", err)
	}
}
