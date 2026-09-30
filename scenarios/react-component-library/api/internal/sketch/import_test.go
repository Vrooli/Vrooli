package sketch

import (
	"context"
	"react-component-library/internal/catalogsearch"
	"reflect"
	"testing"
)

type importSearchFixture struct {
	calls   []string
	results map[string][]catalogsearch.Result
}

func (f *importSearchFixture) SearchContext(_ context.Context, q string, _ int, kind, _, _ string) []catalogsearch.Result {
	f.calls = append(f.calls, q)
	if kind == "page-template" {
		return nil
	}
	return f.results[q]
}
func TestImportPreservesIntentAndRequiresBothSearchesForGap(t *testing.T) {
	source := Snapshot{PagePurpose: "Help operators resolve urgent requests.", DeclaredRegions: []Region{{ID: "inbox", Note: "Urgent requests", Elements: []string{"list"}}}, PageElements: []string{"list", "unmapped"}, Document: Document{Notes: []Note{{Scope: "page", Text: "Preserve keyboard flow"}}}}
	search := &importSearchFixture{}
	got, err := Import(context.Background(), source, search)
	if err != nil {
		t.Fatal(err)
	}
	if got.Document.Regions[0].Note != "Urgent requests" || len(got.Document.Notes) != 1 || len(got.Document.Unplaced) != 1 {
		t.Fatalf("lost authored intent: %+v", got)
	}
	if got.Items[0].Tier != 3 || len(got.Items[0].Checks) != 2 {
		t.Fatalf("gap bypassed reuse: %+v", got.Items)
	}
	if len(got.Document.Placements) != 0 {
		t.Fatal("import invented implementation")
	}
	source.Document = got.Document
	again, err := Import(context.Background(), source, search)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Document, again.Document) {
		t.Fatalf("repeat import changed composition: %+v", again.Document)
	}
}
func TestImportL0CreatesProposedRegionAndDoesNotPromoteSearchCandidates(t *testing.T) {
	source := Snapshot{PagePurpose: "Review and resolve urgent requests.", PageElements: []string{"list"}}
	search := &importSearchFixture{results: map[string][]catalogsearch.Result{"Review and resolve urgent requests. primary task": {{Document: catalogsearch.Document{CatalogID: "components.inbox", Kind: "component"}, Score: 3}}}}
	got, err := Import(context.Background(), source, search)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Document.Regions) != 1 || got.Document.Regions[0].Note != source.PagePurpose || got.Items[0].State != "candidate" || len(got.Document.Placements) != 0 {
		t.Fatalf("invalid L0 proposal: %+v", got)
	}
	if _, err := Import(context.Background(), source, nil); err == nil {
		t.Fatal("missing search classified as no reuse")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Import(ctx, source, search); err == nil {
		t.Fatal("cancelled import succeeded")
	}
}
