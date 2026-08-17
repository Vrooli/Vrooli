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
	if err != nil || len(direct) != 90 || len(transitive) != 303 {
		t.Fatalf("foundations.tokens dependents = %d/%d, %v", len(direct), len(transitive), err)
	}
	closure, err := index.Closure("data-display.data-table")
	if err != nil || len(closure) != 28 {
		t.Fatalf("data table closure = %d, %v", len(closure), err)
	}
	bands := Bands(closure)
	want := []int{11, 7, 7, 3}
	if len(bands) != len(want) {
		t.Fatalf("data table bands = %+v", bands)
	}
	for i, band := range bands {
		if band.Count != want[i] {
			t.Fatalf("band %d = %d, want %d", i, band.Count, want[i])
		}
	}
	collection, err := index.Closure("templates.collection-page")
	if err != nil || len(collection) != 47 {
		t.Fatalf("collection page closure = %d, %v", len(collection), err)
	}
}
