package assetgraph

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"react-component-library/internal/assetrung"
	"react-component-library/internal/catalogcoverage"
)

func fixture() []catalogcoverage.Asset {
	return []catalogcoverage.Asset{
		{ID: "root", Name: "Root", Kind: "component", Requires: []string{"primitive", "runtime"}},
		{ID: "primitive", Name: "Primitive", Kind: "primitive", Requires: []string{"foundation"}},
		{ID: "runtime", Name: "Runtime", Kind: "runtime-hook"},
		{ID: "foundation", Name: "Foundation", Kind: "foundation"},
	}
}

func TestBuildAndTraversal(t *testing.T) {
	index, err := Build(fixture())
	if err != nil {
		t.Fatal(err)
	}
	closure, err := index.Closure("root")
	if err != nil || len(closure) != 4 {
		t.Fatalf("closure = %+v, %v", closure, err)
	}
	if closure[0].Rung != assetrung.RungComponent || closure[1].Rung != assetrung.RungPrimitive {
		t.Fatalf("closure order = %+v", closure)
	}
	bands := Bands(closure)
	if len(bands) != 4 || bands[0].Rung != assetrung.RungComponent {
		t.Fatalf("bands = %+v", bands)
	}
	direct, transitive, err := index.Dependents("foundation")
	if err != nil || len(direct) != 1 || len(transitive) != 2 {
		t.Fatalf("dependents = %v, %v, %v", direct, transitive, err)
	}
	repeated, err := index.Closure("root")
	if err != nil || !reflect.DeepEqual(closure, repeated) {
		t.Fatalf("traversal is not deterministic: %v vs %v", closure, repeated)
	}
}

func TestBuildUnknownEdgeAndUnknownID(t *testing.T) {
	_, err := Build([]catalogcoverage.Asset{{ID: "root", Kind: "component", Requires: []string{"missing"}}})
	var unknown UnknownAssetError
	if !errors.As(err, &unknown) || unknown.ID != "missing" {
		t.Fatalf("Build error = %v", err)
	}
	index, err := Build(nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = index.Closure("missing")
	if !errors.As(err, &unknown) {
		t.Fatalf("Closure error = %v", err)
	}
}

func TestClosureCycleReturnsNoPartialResult(t *testing.T) {
	index, err := Build([]catalogcoverage.Asset{{ID: "a", Kind: "component", Requires: []string{"b"}}, {ID: "b", Kind: "primitive", Requires: []string{"a"}}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := index.Closure("a")
	var cycle CycleError
	if got != nil || !errors.As(err, &cycle) || len(cycle.Path) != 3 {
		t.Fatalf("closure = %+v, error = %v", got, err)
	}
}

func TestOracleCatalogGraph(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	assets, err := catalogcoverage.LoadCatalog(filepath.Join(root, "scenarios", "react-component-library", "catalog"))
	if err != nil {
		t.Fatal(err)
	}
	index, err := Build(assets)
	if err != nil {
		t.Fatal(err)
	}
	direct, transitive, err := index.Dependents("foundations.tokens")
	if err != nil || len(direct) == 0 || len(transitive) < len(direct) {
		t.Fatalf("foundations.tokens dependents are not a valid closure: direct=%d transitive=%d err=%v", len(direct), len(transitive), err)
	}
	closure, err := index.Closure("data-display.data-table")
	if err != nil || len(closure) == 0 {
		t.Fatalf("data table closure is empty: %v", err)
	}
	seen := map[string]bool{}
	for _, node := range closure {
		if seen[node.ID] {
			t.Fatalf("data table closure repeats %q", node.ID)
		}
		seen[node.ID] = true
	}
	bands := Bands(closure)
	bandCount := 0
	for _, band := range bands {
		if band.Count != len(band.Assets) || band.Count == 0 {
			t.Fatalf("invalid data table band: %+v", band)
		}
		bandCount += band.Count
	}
	if bandCount != len(closure) {
		t.Fatalf("data table bands cover %d of %d closure nodes", bandCount, len(closure))
	}
	collection, err := index.Closure("templates.collection-page")
	if err != nil || len(collection) == 0 {
		t.Fatalf("collection page closure is invalid: %+v, %v", collection, err)
	}
	collectionRoot := false
	for _, node := range collection {
		collectionRoot = collectionRoot || node.ID == "templates.collection-page"
	}
	if !collectionRoot {
		t.Fatal("collection page closure does not contain its root")
	}
}
