package graph_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	intgraph "go-code-graph/internal/graph"
	"golang.org/x/tools/go/packages"
)

func TestNormalizeSourceFactsPreserveMethodIdentityAndBoundedEvidence(t *testing.T) {
	source := `package demo
 type A struct { Status string; Complete bool }
 type B struct{}
 func (a *A) run() int { if a.Complete { return 1 }; return 0 }
 func (b B) run() int { return 2 }
 func unused() int { if true { return 1 }; return 0 }
 func clone() int { if true { return 1 }; return 0 }
 func invoke(a *A, b B) { a.run(); b.run() }
 `
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "/repo/demo.go", source, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	info := &types.Info{Defs: map[*ast.Ident]types.Object{}, Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	checked, err := new(types.Config).Check("demo", fset, []*ast.File{file}, info)
	if err != nil {
		t.Fatal(err)
	}
	pkg := &packages.Package{PkgPath: "demo", Name: "demo", GoFiles: []string{"/repo/demo.go"}, Fset: fset, Syntax: []*ast.File{file}, Types: checked, TypesInfo: info}
	result, _ := intgraph.Normalize([]*packages.Package{pkg}, "/repo")
	nodes := map[string]intgraph.Node{}
	references := map[string]bool{}
	for _, node := range result.Nodes {
		nodes[node.ID] = node
		if node.Kind == intgraph.NodeKindCall {
			references[node.Attributes["callee_symbol"]] = true
		}
	}
	for _, receiver := range []string{"A", "B"} {
		id := "go_method:package:demo:" + receiver + ".run"
		node, ok := nodes[id]
		if !ok || node.Attributes["receiver"] != receiver {
			t.Fatalf("receiver identity lost: %s %+v", id, node)
		}
	}
	unused := nodes["go_func:package:demo:unused"].Attributes
	clone := nodes["go_func:package:demo:clone"].Attributes
	if unused["package_name_occurrences"] != "1" || unused["signature"] != "func() int" || unused["cyclomatic_complexity"] != "2" || unused["start_line"] == "" || unused["body_lines"] != "1" {
		t.Fatalf("source facts: %+v", unused)
	}
	if unused["body_hash"] == "" || unused["body_hash"] != clone["body_hash"] {
		t.Fatalf("identical bodies need same identity: %v / %v", unused, clone)
	}
	// Resolved calls must use the same receiver-qualified identities as declarations.
	found := map[string]bool{}
	for _, node := range result.Nodes {
		for _, v := range node.Attributes {
			if v == "go_method:package:demo:A.run" || v == "go_method:package:demo:B.run" {
				found[v] = true
			}
		}
	}
	if len(found) != 2 {
		t.Fatalf("resolved method references detached: %+v", found)
	}
	if nodes["go_type:package:demo:A"].Attributes["type_expression"] == "" {
		t.Fatal("struct shape missing")
	}
}
