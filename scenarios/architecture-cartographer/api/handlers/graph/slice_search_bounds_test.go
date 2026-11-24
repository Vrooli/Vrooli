package graph

import (
	"testing"

	graphv1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture-cartographer/v1/graph"
)

func TestSliceEvidenceBoundsReportsReturnedAndOmittedCounts(t *testing.T) {
	slice := &graphv1.DomainSlice{
		Rungs: []*graphv1.SliceRung{{
			Name:     "api",
			Files:    makeFiles(6),
			Symbols:  makeSymbols(4),
			Evidence: makeEvidence(5),
		}},
		LayerEdges: makeEdges(7),
	}
	bounds := sliceEvidenceBoundsFor(slice)
	if len(bounds.Rungs) != 1 || bounds.Rungs[0].Name != "api" {
		t.Fatalf("rung identity missing: %#v", bounds)
	}
	if got := bounds.Rungs[0].Files; got.Total != 6 || got.Returned != 4 || got.Omitted != 2 {
		t.Fatalf("file bounds = %#v", got)
	}
	if got := bounds.Rungs[0].Symbols; got.Total != 4 || got.Returned != 4 || got.Omitted != 0 {
		t.Fatalf("symbol bounds = %#v", got)
	}
	if got := bounds.Rungs[0].Evidence; got.Total != 5 || got.Returned != 4 || got.Omitted != 1 {
		t.Fatalf("evidence bounds = %#v", got)
	}
	if got := bounds.LayerEdges; got.Total != 7 || got.Returned != 4 || got.Omitted != 3 {
		t.Fatalf("edge bounds = %#v", got)
	}
}

func TestSliceEvidenceBoundsShowsCompleteEvidenceWithoutOmission(t *testing.T) {
	bounds := sliceEvidenceBoundsFor(&graphv1.DomainSlice{Rungs: []*graphv1.SliceRung{{Name: "cli", Files: makeFiles(2)}}})
	if len(bounds.Rungs) != 1 || bounds.Rungs[0].Files != (sliceEvidenceCount{Total: 2, Returned: 2, Omitted: 0}) {
		t.Fatalf("complete file bounds = %#v", bounds)
	}
	if bounds.LayerEdges != (sliceEvidenceCount{Total: 0, Returned: 0, Omitted: 0}) {
		t.Fatalf("empty edge bounds = %#v", bounds.LayerEdges)
	}
}

func makeFiles(count int) []*graphv1.SliceFile {
	items := make([]*graphv1.SliceFile, count)
	for i := range items {
		items[i] = &graphv1.SliceFile{Path: "file"}
	}
	return items
}

func makeSymbols(count int) []*graphv1.SliceSymbol {
	items := make([]*graphv1.SliceSymbol, count)
	for i := range items {
		items[i] = &graphv1.SliceSymbol{Name: "symbol"}
	}
	return items
}

func makeEvidence(count int) []*graphv1.SliceEvidence {
	items := make([]*graphv1.SliceEvidence, count)
	for i := range items {
		items[i] = &graphv1.SliceEvidence{Value: "locator"}
	}
	return items
}

func makeEdges(count int) []*graphv1.SliceEdge {
	items := make([]*graphv1.SliceEdge, count)
	for i := range items {
		items[i] = &graphv1.SliceEdge{Kind: "import"}
	}
	return items
}
