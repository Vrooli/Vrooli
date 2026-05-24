package graph

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/packages"
)

// Normalize converts the loader's []*packages.Package into a
// deterministic Graph plus a slice of non-fatal Warnings. The function
// is pure (no I/O, no clocks) and stable: same input list always
// produces the same output bytes.
//
// Sorting strategy:
//
//   - Packages are visited in import-path order.
//   - For each package, files are visited in basename order; symbols
//     are visited in (file_basename, name) order.
//   - The final Nodes slice is sorted by ID; Edges likewise.
//
// scenarioRoot is the absolute scenario path; file paths in the graph
// are rebased relative to it when possible so the output is host-path
// independent.
func Normalize(pkgs []*packages.Package, scenarioRoot string) (Graph, []Warning) {
	var (
		nodes    []Node
		edges    []Edge
		warnings []Warning
		seenPkg  = make(map[string]bool)
		seenEdge = make(map[string]bool)
	)

	// Visit packages in stable import-path order. packages.Load is
	// notoriously non-deterministic across runs.
	sorted := make([]*packages.Package, 0, len(pkgs))
	for _, p := range pkgs {
		if p == nil {
			continue
		}
		sorted = append(sorted, p)
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].PkgPath < sorted[j].PkgPath })

	for _, p := range sorted {
		warnings = append(warnings, packageWarnings(p)...)

		pkgID := "package:" + p.PkgPath
		if !seenPkg[pkgID] {
			seenPkg[pkgID] = true
			nodes = append(nodes, Node{
				ID:   pkgID,
				Kind: NodeKindPackage,
				Name: p.Name,
				Path: p.PkgPath,
				Attributes: map[string]string{
					"language":    string(LanguageGo),
					"import_path": p.PkgPath,
				},
			})
		}

		// File nodes, sorted by basename.
		files := append([]string(nil), p.GoFiles...)
		sort.Strings(files)
		for _, abs := range files {
			rel := relPath(scenarioRoot, abs)
			fileID := "file:" + rel
			nodes = append(nodes, Node{
				ID:   fileID,
				Kind: NodeKindFile,
				Name: filepath.Base(abs),
				Path: rel,
				Attributes: map[string]string{
					"language":   string(LanguageGo),
					"package_id": pkgID,
				},
			})
		}

		// Symbol nodes, sorted by (file_basename, name).
		nodes = append(nodes, packageSymbols(p, pkgID, scenarioRoot)...)

		// Import edges in stable to-path order.
		importPaths := make([]string, 0, len(p.Imports))
		for imp := range p.Imports {
			importPaths = append(importPaths, imp)
		}
		sort.Strings(importPaths)
		for _, imp := range importPaths {
			toID := "package:" + imp
			edgeID := "import:" + pkgID + "->" + toID
			if seenEdge[edgeID] {
				continue
			}
			seenEdge[edgeID] = true
			edges = append(edges, Edge{
				ID:   edgeID,
				Kind: EdgeKindImport,
				From: pkgID,
				To:   toID,
			})
		}
	}

	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		return edges[i].To < edges[j].To
	})
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].File != warnings[j].File {
			return warnings[i].File < warnings[j].File
		}
		if warnings[i].Kind != warnings[j].Kind {
			return warnings[i].Kind < warnings[j].Kind
		}
		return warnings[i].Message < warnings[j].Message
	})

	return Graph{Nodes: nodes, Edges: edges}, warnings
}

// packageSymbols emits the symbol nodes declared at file scope in p.
// It walks p.Syntax (already sorted by the caller) but emits a sorted
// slice keyed by (file_basename, name) so output is deterministic.
func packageSymbols(p *packages.Package, pkgID, scenarioRoot string) []Node {
	type pending struct {
		fileBase string
		name     string
		node     Node
	}
	var out []pending

	if p.Fset == nil {
		return nil
	}

	for _, syn := range p.Syntax {
		if syn == nil {
			continue
		}
		fileAbs := p.Fset.Position(syn.Pos()).Filename
		fileBase := filepath.Base(fileAbs)
		rel := relPath(scenarioRoot, fileAbs)
		fileID := "file:" + rel

		for _, decl := range syn.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Name == nil {
					continue
				}
				kind := NodeKindFunc
				if d.Recv != nil && len(d.Recv.List) > 0 {
					kind = NodeKindMethod
				}
				out = append(out, pending{
					fileBase: fileBase,
					name:     d.Name.Name,
					node: Node{
						ID:   string(kind) + ":" + pkgID + ":" + d.Name.Name,
						Kind: kind,
						Name: d.Name.Name,
						Path: rel,
						Attributes: map[string]string{
							"language":   string(LanguageGo),
							"package_id": pkgID,
							"file_id":    fileID,
							"exported":   boolAttr(d.Name.IsExported()),
						},
					},
				})
			case *ast.GenDecl:
				kind := genDeclKind(d.Tok)
				if kind == "" {
					continue
				}
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						if s.Name == nil {
							continue
						}
						nk := kind
						if _, isIface := s.Type.(*ast.InterfaceType); isIface {
							nk = NodeKindInterface
						}
						out = append(out, pending{
							fileBase: fileBase,
							name:     s.Name.Name,
							node: Node{
								ID:   string(nk) + ":" + pkgID + ":" + s.Name.Name,
								Kind: nk,
								Name: s.Name.Name,
								Path: rel,
								Attributes: map[string]string{
									"language":   string(LanguageGo),
									"package_id": pkgID,
									"file_id":    fileID,
									"exported":   boolAttr(s.Name.IsExported()),
								},
							},
						})
					case *ast.ValueSpec:
						for _, ident := range s.Names {
							if ident == nil || ident.Name == "_" {
								continue
							}
							out = append(out, pending{
								fileBase: fileBase,
								name:     ident.Name,
								node: Node{
									ID:   string(kind) + ":" + pkgID + ":" + ident.Name,
									Kind: kind,
									Name: ident.Name,
									Path: rel,
									Attributes: map[string]string{
										"language":   string(LanguageGo),
										"package_id": pkgID,
										"file_id":    fileID,
										"exported":   boolAttr(ident.IsExported()),
									},
								},
							})
						}
					}
				}
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].fileBase != out[j].fileBase {
			return out[i].fileBase < out[j].fileBase
		}
		return out[i].name < out[j].name
	})

	nodes := make([]Node, 0, len(out))
	for _, p := range out {
		nodes = append(nodes, p.node)
	}
	return nodes
}

// genDeclKind returns the SymbolNode kind for a GenDecl token, or "" if
// the token doesn't map to a symbol family this scenario emits.
func genDeclKind(t token.Token) NodeKind {
	switch t {
	case token.TYPE:
		return NodeKindType
	case token.VAR:
		return NodeKindVar
	case token.CONST:
		return NodeKindConst
	default:
		return ""
	}
}

// boolAttr renders a Go bool as the canonical "true"/"false" string so
// attribute maps serialize deterministically.
func boolAttr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// relPath returns the scenario-rooted relative path when scenarioRoot
// is a prefix of abs, otherwise abs unchanged. Both inputs should be
// cleaned absolute paths.
func relPath(scenarioRoot, abs string) string {
	if scenarioRoot == "" {
		return abs
	}
	if rel, err := filepath.Rel(scenarioRoot, abs); err == nil && !filepath.IsAbs(rel) {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(abs)
}
