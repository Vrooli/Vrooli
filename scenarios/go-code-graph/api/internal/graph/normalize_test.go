package graph_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"

	intgraph "go-code-graph/internal/graph"
)

// TestNormalizeEmpty exercises the trivial path: nil package slice
// yields a Graph with no nodes/edges and no warnings.
func TestNormalizeEmpty(t *testing.T) {
	t.Parallel()
	g, w := intgraph.Normalize(nil, "/scenario")
	if len(g.Nodes) != 0 {
		t.Fatalf("empty input: want 0 nodes, got %d", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Fatalf("empty input: want 0 edges, got %d", len(g.Edges))
	}
	if len(w) != 0 {
		t.Fatalf("empty input: want 0 warnings, got %d", len(w))
	}
}

// TestNormalizeProducesPackageAndFileNodes asserts that a single
// minimal package contributes one package node + one file node + the
// declared symbols, with stable IDs.
func TestNormalizeProducesPackageAndFileNodes(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	src := "package alpha\n\ntype Widget struct{}\n\nfunc Hello() {}\n"
	file, err := parser.ParseFile(fset, "/scenario/alpha/alpha.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	pkg := &packages.Package{
		PkgPath: "example.com/m/alpha",
		Name:    "alpha",
		GoFiles: []string{"/scenario/alpha/alpha.go"},
		Fset:    fset,
		Syntax:  []*ast.File{file},
	}

	g, _ := intgraph.Normalize([]*packages.Package{pkg}, "/scenario")

	// Build a quick id->node lookup.
	byID := map[string]intgraph.Node{}
	for _, n := range g.Nodes {
		byID[n.ID] = n
	}
	if _, ok := byID["package:example.com/m/alpha"]; !ok {
		t.Errorf("missing package node; got ids: %v", ids(g.Nodes))
	}
	if _, ok := byID["file:alpha/alpha.go"]; !ok {
		t.Errorf("missing file node; got ids: %v", ids(g.Nodes))
	}
	if _, ok := byID["go_type:package:example.com/m/alpha:Widget"]; !ok {
		t.Errorf("missing type node; got ids: %v", ids(g.Nodes))
	}
	if _, ok := byID["go_func:package:example.com/m/alpha:Hello"]; !ok {
		t.Errorf("missing func node; got ids: %v", ids(g.Nodes))
	}
}

// TestNormalizeSortsNodesAndEdges asserts deterministic output ordering
// regardless of the loader's iteration order.
func TestNormalizeSortsNodesAndEdges(t *testing.T) {
	t.Parallel()
	pkgs := []*packages.Package{
		{PkgPath: "z/last", Name: "last", GoFiles: nil, Imports: map[string]*packages.Package{}},
		{PkgPath: "a/first", Name: "first", GoFiles: nil, Imports: map[string]*packages.Package{
			"z/last": {PkgPath: "z/last"},
		}},
	}
	g, _ := intgraph.Normalize(pkgs, "/scenario")

	// Verify nodes are sorted by ID.
	for i := 1; i < len(g.Nodes); i++ {
		if g.Nodes[i-1].ID > g.Nodes[i].ID {
			t.Fatalf("nodes not sorted at %d: %q > %q", i, g.Nodes[i-1].ID, g.Nodes[i].ID)
		}
	}
	for i := 1; i < len(g.Edges); i++ {
		prev, cur := g.Edges[i-1], g.Edges[i]
		if prev.From > cur.From || (prev.From == cur.From && prev.To > cur.To) {
			t.Fatalf("edges not sorted at %d: %+v > %+v", i, prev, cur)
		}
	}
}

// TestNormalizeEmitsImportEdges asserts that one package importing
// another yields exactly one IMPORT edge in stable form.
func TestNormalizeEmitsImportEdges(t *testing.T) {
	t.Parallel()
	to := &packages.Package{PkgPath: "example.com/m/bravo", Name: "bravo"}
	from := &packages.Package{
		PkgPath: "example.com/m/alpha",
		Name:    "alpha",
		Imports: map[string]*packages.Package{"example.com/m/bravo": to},
	}
	g, _ := intgraph.Normalize([]*packages.Package{from, to}, "/scenario")
	if len(g.Edges) != 1 {
		t.Fatalf("want 1 edge, got %d (%+v)", len(g.Edges), g.Edges)
	}
	e := g.Edges[0]
	if e.From != "package:example.com/m/alpha" || e.To != "package:example.com/m/bravo" {
		t.Fatalf("unexpected edge: %+v", e)
	}
	if e.Kind != intgraph.EdgeKindImport {
		t.Fatalf("want kind=import, got %q", e.Kind)
	}
}

func TestNormalizeAnnotatesImportEdgesWithSymbolKinds(t *testing.T) {
	t.Parallel()
	fset := token.NewFileSet()
	src := `package alpha

import "fmt"

type widget struct{}

func Hello(s fmt.Stringer) string {
	return fmt.Sprintf("%s", s)
}
`
	file, err := parser.ParseFile(fset, "/scenario/alpha/alpha.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
	}
	conf := types.Config{Importer: importer.Default()}
	if _, err := conf.Check("example.com/m/alpha", fset, []*ast.File{file}, info); err != nil {
		t.Fatalf("typecheck: %v", err)
	}
	pkg := &packages.Package{
		PkgPath:   "example.com/m/alpha",
		Name:      "alpha",
		GoFiles:   []string{"/scenario/alpha/alpha.go"},
		Imports:   map[string]*packages.Package{"fmt": {PkgPath: "fmt", Name: "fmt"}},
		Fset:      fset,
		Syntax:    []*ast.File{file},
		TypesInfo: info,
	}

	g, _ := intgraph.Normalize([]*packages.Package{pkg}, "/scenario")
	if len(g.Edges) != 1 {
		t.Fatalf("want one edge, got %+v", g.Edges)
	}
	kinds := g.Edges[0].Attributes["symbol_kinds"]
	if kinds == "" {
		t.Fatalf("want symbol_kinds on import edge, attrs=%+v", g.Edges[0].Attributes)
	}
	if kinds != "go_func,go_interface" && kinds != "go_interface,go_func" {
		t.Fatalf("unexpected symbol_kinds %q", kinds)
	}
	if g.Edges[0].Attributes["symbol_ids"] == "" {
		t.Fatalf("want symbol_ids on import edge, attrs=%+v", g.Edges[0].Attributes)
	}
}

func ids(nodes []intgraph.Node) []string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.ID)
	}
	return out
}
