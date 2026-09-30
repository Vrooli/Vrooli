package graph

import (
	"bytes"
	"go/ast"
	"go/constant"
	"go/printer"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

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
// moduleRoot is the absolute module path; file paths in the graph
// are rebased relative to it when possible so the output is host-path
// independent.
func Normalize(pkgs []*packages.Package, moduleRoot string) (Graph, []Warning) {
	var (
		nodes    []Node
		edges    []Edge
		warnings []Warning
		seenPkg  = make(map[string]bool)
		seenEdge = make(map[string]bool)
	)

	// Visit packages in stable import-path order. packages.Load is
	// notoriously non-deterministic across runs. Tests:true also returns
	// production, test-variant, external-test, and synthetic test-binary
	// packages; canonicalize those variants before graph emission.
	sorted, testOnlyImports := mergePackageTestVariants(pkgs)
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
					"internal":    boolAttr(isInternalImportPath(p.PkgPath)),
				},
			})
		}

		// Index syntax files by absolute path so we can count source lines
		// without a second filesystem read.
		linesByAbs := fileLineCounts(p)

		// File nodes, sorted by basename.
		files := append([]string(nil), p.GoFiles...)
		sort.Strings(files)
		for _, abs := range files {
			rel := relPath(moduleRoot, abs)
			fileID := "file:" + rel
			nodes = append(nodes, Node{
				ID:   fileID,
				Kind: NodeKindFile,
				Name: filepath.Base(abs),
				Path: rel,
				Attributes: map[string]string{
					"language":   string(LanguageGo),
					"package_id": pkgID,
					"is_test":    boolAttr(isTestFile(abs)),
					"lines":      intAttr(linesByAbs[abs]),
				},
			})
		}

		// Symbol nodes, sorted by (file_basename, name).
		nodes = append(nodes, packageSymbols(p, pkgID, moduleRoot)...)
		nodes = append(nodes, packageUsageFacts(p, pkgID, moduleRoot)...)

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
			symbolIDs, symbolKinds := importSymbolFacts(p, imp)
			attrs := map[string]string{
				"test_only": boolAttr(testOnlyImports[p.PkgPath][imp]),
			}
			if len(symbolIDs) > 0 {
				attrs["symbol_ids"] = strings.Join(symbolIDs, ",")
			}
			if len(symbolKinds) > 0 {
				attrs["symbol_kinds"] = strings.Join(symbolKinds, ",")
			}
			edges = append(edges, Edge{
				ID:         edgeID,
				Kind:       EdgeKindImport,
				From:       pkgID,
				To:         toID,
				Attributes: attrs,
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

func mergePackageTestVariants(pkgs []*packages.Package) ([]*packages.Package, map[string]map[string]bool) {
	groups := map[string][]*packages.Package{}
	for _, p := range pkgs {
		if p == nil || isSyntheticTestBinaryPackage(p) {
			continue
		}
		groups[p.PkgPath] = append(groups[p.PkgPath], p)
	}

	paths := make([]string, 0, len(groups))
	for path := range groups {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	out := make([]*packages.Package, 0, len(paths))
	testOnly := make(map[string]map[string]bool, len(paths))
	for _, path := range paths {
		group := groups[path]
		sort.Slice(group, func(i, j int) bool { return packageVariantKey(group[i]) < packageVariantKey(group[j]) })

		prod := productionPackageVariant(group)
		merged := clonePackage(richestPackageVariant(group))
		merged.Imports = mergedImports(group)
		merged.GoFiles = mergedFiles(group, func(p *packages.Package) []string { return p.GoFiles })
		merged.CompiledGoFiles = mergedFiles(group, func(p *packages.Package) []string { return p.CompiledGoFiles })

		prodImports := map[string]bool{}
		if prod != nil && !isExternalTestPackage(prod) {
			for imp := range prod.Imports {
				prodImports[imp] = true
			}
		}
		testOnly[path] = map[string]bool{}
		allTestPackage := prod == nil || isExternalTestPackage(merged)
		for imp := range merged.Imports {
			testOnly[path][imp] = allTestPackage || !prodImports[imp]
		}
		out = append(out, merged)
	}
	return out, testOnly
}

func packageVariantKey(p *packages.Package) string {
	if p == nil {
		return ""
	}
	return p.PkgPath + "\x00" + p.ID
}

func productionPackageVariant(group []*packages.Package) *packages.Package {
	for _, p := range group {
		if p != nil && p.ID == p.PkgPath && !isExternalTestPackage(p) {
			return p
		}
	}
	for _, p := range group {
		if p != nil && !strings.Contains(p.ID, " [") && !isExternalTestPackage(p) {
			return p
		}
	}
	return nil
}

func richestPackageVariant(group []*packages.Package) *packages.Package {
	var best *packages.Package
	for _, p := range group {
		if p == nil {
			continue
		}
		if best == nil || packageVariantWeight(p) > packageVariantWeight(best) ||
			(packageVariantWeight(p) == packageVariantWeight(best) && packageVariantKey(p) < packageVariantKey(best)) {
			best = p
		}
	}
	if best == nil {
		return &packages.Package{}
	}
	return best
}

func packageVariantWeight(p *packages.Package) int {
	if p == nil {
		return 0
	}
	weight := len(p.GoFiles)*2 + len(p.Syntax)
	if strings.Contains(p.ID, " [") {
		weight++
	}
	return weight
}

func clonePackage(p *packages.Package) *packages.Package {
	cp := *p
	return &cp
}

func mergedImports(group []*packages.Package) map[string]*packages.Package {
	out := map[string]*packages.Package{}
	for _, p := range group {
		if p == nil {
			continue
		}
		keys := make([]string, 0, len(p.Imports))
		for imp := range p.Imports {
			keys = append(keys, imp)
		}
		sort.Strings(keys)
		for _, imp := range keys {
			if _, ok := out[imp]; !ok {
				out[imp] = p.Imports[imp]
			}
		}
	}
	return out
}

func mergedFiles(group []*packages.Package, files func(*packages.Package) []string) []string {
	seen := map[string]bool{}
	for _, p := range group {
		if p == nil {
			continue
		}
		for _, file := range files(p) {
			seen[file] = true
		}
	}
	out := make([]string, 0, len(seen))
	for file := range seen {
		out = append(out, file)
	}
	sort.Strings(out)
	return out
}

func isSyntheticTestBinaryPackage(p *packages.Package) bool {
	return p != nil && strings.HasSuffix(p.PkgPath, ".test")
}

func isExternalTestPackage(p *packages.Package) bool {
	return p != nil && strings.HasSuffix(p.Name, "_test")
}

func importSymbolFacts(p *packages.Package, importPath string) ([]string, []string) {
	if p == nil || p.TypesInfo == nil {
		return nil, nil
	}
	ids := map[string]struct{}{}
	kinds := map[string]struct{}{}
	for _, syn := range p.Syntax {
		if syn == nil {
			continue
		}
		ast.Inspect(syn, func(n ast.Node) bool {
			ident, ok := n.(*ast.Ident)
			if !ok {
				return true
			}
			obj := usedObject(p, ident)
			if obj == nil || objectPackagePath(obj) != importPath {
				return true
			}
			id := objectSymbolID(obj)
			kind := objectKind(obj)
			if id != "" {
				ids[id] = struct{}{}
			}
			if kind != "" {
				kinds[kind] = struct{}{}
			}
			return true
		})
	}
	return sortedSet(ids), sortedSet(kinds)
}

func sortedSet(in map[string]struct{}) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for item := range in {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

// packageUsageFacts emits generic language facts used by downstream proof
// systems: import specs, symbol references, call expressions, and type usages.
// The facts are intentionally policy-free; Vrooli-specific interpretation lives
// in code-facts and consumers.
func packageUsageFacts(p *packages.Package, pkgID, moduleRoot string) []Node {
	if p.Fset == nil {
		return nil
	}
	type pending struct {
		key  string
		node Node
	}
	var out []pending
	seqByFile := map[string]int{}

	nextSeq := func(rel string) int {
		seqByFile[rel]++
		return seqByFile[rel]
	}

	for _, syn := range p.Syntax {
		if syn == nil {
			continue
		}
		fileAbs := p.Fset.Position(syn.Pos()).Filename
		rel := relPath(moduleRoot, fileAbs)
		fileID := "file:" + rel

		for _, spec := range syn.Imports {
			if spec == nil {
				continue
			}
			importPath, _ := strconv.Unquote(spec.Path.Value)
			alias := ""
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			attrs := usageAttributes(LanguageGo, pkgID, fileID, p.Fset, spec.Pos(), spec.End())
			attrs["import_path"] = importPath
			attrs["alias"] = alias
			attrs["is_blank"] = boolAttr(alias == "_")
			attrs["is_dot"] = boolAttr(alias == ".")
			out = append(out, pending{
				key: rel + ":import:" + importPath + ":" + alias + ":" + positionKey(p.Fset, spec.Pos()),
				node: Node{
					ID:         factID(NodeKindImportSpec, pkgID, rel, nextSeq(rel)),
					Kind:       NodeKindImportSpec,
					Name:       importPath,
					Path:       rel,
					Attributes: attrs,
				},
			})
		}

		ast.Inspect(syn, func(n ast.Node) bool {
			switch x := n.(type) {
			case nil:
				return true
			case *ast.FuncDecl:
				return true
			case *ast.Ident:
				obj := usedObject(p, x)
				kind := objectKind(obj)
				if obj == nil || obj.Pkg() == nil || kind == "" {
					return true
				}
				attrs := usageAttributes(LanguageGo, pkgID, fileID, p.Fset, x.Pos(), x.End())
				attrs["referenced_symbol"] = objectSymbolID(obj)
				attrs["referenced_name"] = obj.Name()
				attrs["referenced_package"] = obj.Pkg().Path()
				attrs["referenced_kind"] = kind
				attrs["enclosing_symbol"] = enclosingFunctionName(p.Fset, syn, x.Pos())
				out = append(out, pending{
					key: rel + ":ref:" + positionKey(p.Fset, x.Pos()) + ":" + attrs["referenced_symbol"],
					node: Node{
						ID:         factID(NodeKindReference, pkgID, rel, nextSeq(rel)),
						Kind:       NodeKindReference,
						Name:       x.Name,
						Path:       rel,
						Attributes: attrs,
					},
				})
			case *ast.CallExpr:
				attrs := usageAttributes(LanguageGo, pkgID, fileID, p.Fset, x.Pos(), x.End())
				attrs["callee"] = exprString(p.Fset, x.Fun)
				if obj := calledObject(p, x.Fun); obj != nil {
					if kind := objectKind(obj); kind != "" {
						attrs["callee_symbol"] = objectSymbolID(obj)
						attrs["callee_package"] = objectPackagePath(obj)
						attrs["callee_kind"] = kind
					}
				}
				if recv := receiverType(p, x.Fun); recv != "" {
					attrs["receiver_type"] = recv
				}
				attrs["argument_types"] = argumentTypes(p, x.Args)
				attrs["enclosing_symbol"] = enclosingFunctionName(p.Fset, syn, x.Pos())
				out = append(out, pending{
					key: rel + ":call:" + positionKey(p.Fset, x.Pos()) + ":" + attrs["callee"],
					node: Node{
						ID:         factID(NodeKindCall, pkgID, rel, nextSeq(rel)),
						Kind:       NodeKindCall,
						Name:       attrs["callee"],
						Path:       rel,
						Attributes: attrs,
					},
				})
				for _, route := range routeRegistrationsForCall(p, syn, pkgID, fileID, x) {
					out = append(out, pending{
						key: rel + ":route:" + positionKey(p.Fset, x.Pos()) + ":" + route.key,
						node: Node{
							ID:         factID(NodeKindRouteRegistration, pkgID, rel, nextSeq(rel)),
							Kind:       NodeKindRouteRegistration,
							Name:       route.name,
							Path:       rel,
							Attributes: route.attrs,
						},
					})
				}
			case *ast.CompositeLit:
				attrs := usageAttributes(LanguageGo, pkgID, fileID, p.Fset, x.Pos(), x.End())
				attrs["type"] = exprString(p.Fset, x.Type)
				attrs["usage"] = "composite_literal"
				if p.TypesInfo != nil {
					if typ := p.TypesInfo.TypeOf(x.Type); typ != nil {
						attrs["resolved_type"] = typ.String()
					}
				}
				attrs["address_of"] = boolAttr(isAddressOfComposite(syn, x))
				attrs["enclosing_symbol"] = enclosingFunctionName(p.Fset, syn, x.Pos())
				out = append(out, pending{
					key: rel + ":type:" + positionKey(p.Fset, x.Pos()) + ":" + attrs["type"],
					node: Node{
						ID:         factID(NodeKindTypeUsage, pkgID, rel, nextSeq(rel)),
						Kind:       NodeKindTypeUsage,
						Name:       attrs["type"],
						Path:       rel,
						Attributes: attrs,
					},
				})
			}
			return true
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].key < out[j].key })
	nodes := make([]Node, 0, len(out))
	for i := range out {
		nodes = append(nodes, out[i].node)
	}
	return nodes
}

// packageSymbols emits the symbol nodes declared at file scope in p.
// It walks p.Syntax (already sorted by the caller) but emits a sorted
// slice keyed by (file_basename, name) so output is deterministic.
func packageSymbols(p *packages.Package, pkgID, moduleRoot string) []Node {
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
		rel := relPath(moduleRoot, fileAbs)
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

func factID(kind NodeKind, pkgID, rel string, seq int) string {
	return string(kind) + ":" + pkgID + ":" + rel + ":" + strconv.Itoa(seq)
}

func usageAttributes(lang Language, pkgID, fileID string, fset *token.FileSet, start, end token.Pos) map[string]string {
	attrs := map[string]string{
		"language":   string(lang),
		"package_id": pkgID,
		"file_id":    fileID,
	}
	if fset == nil {
		return attrs
	}
	sp := fset.Position(start)
	ep := fset.Position(end)
	attrs["start_line"] = intAttr(sp.Line)
	attrs["start_column"] = intAttr(sp.Column)
	attrs["end_line"] = intAttr(ep.Line)
	attrs["end_column"] = intAttr(ep.Column)
	return attrs
}

func positionKey(fset *token.FileSet, pos token.Pos) string {
	if fset == nil {
		return intAttr(int(pos))
	}
	p := fset.Position(pos)
	return intAttr(p.Line) + ":" + intAttr(p.Column)
}

func usedObject(p *packages.Package, ident *ast.Ident) types.Object {
	if p == nil || p.TypesInfo == nil || ident == nil {
		return nil
	}
	return p.TypesInfo.Uses[ident]
}

func calledObject(p *packages.Package, expr ast.Expr) types.Object {
	if p == nil || p.TypesInfo == nil || expr == nil {
		return nil
	}
	switch x := expr.(type) {
	case *ast.Ident:
		return p.TypesInfo.Uses[x]
	case *ast.SelectorExpr:
		if sel := p.TypesInfo.Selections[x]; sel != nil {
			return sel.Obj()
		}
		return p.TypesInfo.Uses[x.Sel]
	default:
		return nil
	}
}

func objectSymbolID(obj types.Object) string {
	if obj == nil || obj.Pkg() == nil {
		return ""
	}
	return string(objectNodeKind(obj)) + ":package:" + obj.Pkg().Path() + ":" + obj.Name()
}

func objectPackagePath(obj types.Object) string {
	if obj == nil || obj.Pkg() == nil {
		return ""
	}
	return obj.Pkg().Path()
}

func objectKind(obj types.Object) string {
	return string(objectNodeKind(obj))
}

func objectNodeKind(obj types.Object) NodeKind {
	switch o := obj.(type) {
	case *types.TypeName:
		if named, ok := o.Type().(*types.Named); ok {
			if _, ok := named.Underlying().(*types.Interface); ok {
				return NodeKindInterface
			}
		}
		return NodeKindType
	case *types.Func:
		if o.Type() != nil {
			if sig, ok := o.Type().(*types.Signature); ok && sig.Recv() != nil {
				return NodeKindMethod
			}
		}
		return NodeKindFunc
	case *types.Var:
		return NodeKindVar
	case *types.Const:
		return NodeKindConst
	default:
		return ""
	}
}

func receiverType(p *packages.Package, expr ast.Expr) string {
	if p == nil || p.TypesInfo == nil {
		return ""
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.X == nil {
		return ""
	}
	if selection := p.TypesInfo.Selections[sel]; selection != nil && selection.Recv() != nil {
		return validTypeString(selection.Recv().String())
	}
	if typ := p.TypesInfo.TypeOf(sel.X); typ != nil {
		return validTypeString(typ.String())
	}
	return ""
}

func argumentTypes(p *packages.Package, args []ast.Expr) string {
	if p == nil || p.TypesInfo == nil || len(args) == 0 {
		return ""
	}
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		if typ := p.TypesInfo.TypeOf(arg); typ != nil {
			parts = append(parts, validTypeString(typ.String()))
			continue
		}
		parts = append(parts, "unknown")
	}
	return strings.Join(parts, ",")
}

type routeRegistrationFact struct {
	key   string
	name  string
	attrs map[string]string
}

func routeRegistrationsForCall(p *packages.Package, file *ast.File, pkgID, fileID string, call *ast.CallExpr) []routeRegistrationFact {
	if p == nil || p.Fset == nil || call == nil {
		return nil
	}
	if facts := gorillaMuxRouteRegistrations(p, file, pkgID, fileID, call); len(facts) > 0 {
		return facts
	}
	if fact, ok := netHTTPRouteRegistration(p, file, pkgID, fileID, call); ok {
		return []routeRegistrationFact{fact}
	}
	return nil
}

func gorillaMuxRouteRegistrations(p *packages.Package, file *ast.File, pkgID, fileID string, call *ast.CallExpr) []routeRegistrationFact {
	methodsSel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || methodsSel.Sel == nil || methodsSel.Sel.Name != "Methods" {
		return nil
	}
	handleCall, ok := unparen(methodsSel.X).(*ast.CallExpr)
	if !ok {
		return nil
	}
	handleSel, ok := handleCall.Fun.(*ast.SelectorExpr)
	if !ok || handleSel.Sel == nil {
		return nil
	}
	handleName := handleSel.Sel.Name
	if handleName != "HandleFunc" && handleName != "Handle" {
		return nil
	}
	if len(handleCall.Args) < 2 {
		return nil
	}

	attrs := usageAttributes(LanguageGo, pkgID, fileID, p.Fset, call.Pos(), call.End())
	attrs["router_framework"] = "gorilla/mux"
	attrs["route_source"] = handleName + ".Methods"
	attrs["handler_expr"] = exprString(p.Fset, handleCall.Args[1])
	attrs["enclosing_symbol"] = enclosingFunctionName(p.Fset, file, call.Pos())
	attrs["callee"] = exprString(p.Fset, call.Fun)
	if obj := calledObject(p, handleCall.Args[1]); obj != nil {
		attrs["handler_symbol"] = objectSymbolID(obj)
	}
	if path, ok := staticStringValue(p, handleCall.Args[0]); ok {
		attrs["route_path"] = path
		attrs["route_path_status"] = "known"
	} else {
		attrs["route_path_status"] = "unknown"
	}

	methods := staticHTTPMethods(p, call.Args)
	if len(methods) == 0 {
		attrs["http_method_status"] = "unknown"
		return []routeRegistrationFact{{
			key:   attrs["route_path"] + ":unknown:" + positionKey(p.Fset, call.Pos()),
			name:  firstNonEmptyGraph(attrs["route_path"], attrs["handler_expr"], "route registration"),
			attrs: attrs,
		}}
	}

	out := make([]routeRegistrationFact, 0, len(methods))
	for _, method := range methods {
		methodAttrs := copyStringMap(attrs)
		methodAttrs["http_method"] = method
		methodAttrs["http_method_status"] = "known"
		name := method
		if methodAttrs["route_path"] != "" {
			name += " " + methodAttrs["route_path"]
		}
		out = append(out, routeRegistrationFact{
			key:   methodAttrs["route_path"] + ":" + method + ":" + positionKey(p.Fset, call.Pos()),
			name:  name,
			attrs: methodAttrs,
		})
	}
	return out
}

func netHTTPRouteRegistration(p *packages.Package, file *ast.File, pkgID, fileID string, call *ast.CallExpr) (routeRegistrationFact, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "HandleFunc" || len(call.Args) < 2 {
		return routeRegistrationFact{}, false
	}
	obj := calledObject(p, call.Fun)
	if obj == nil || objectPackagePath(obj) != "net/http" {
		return routeRegistrationFact{}, false
	}

	attrs := usageAttributes(LanguageGo, pkgID, fileID, p.Fset, call.Pos(), call.End())
	attrs["router_framework"] = "net/http"
	attrs["route_source"] = "http.HandleFunc"
	attrs["handler_expr"] = exprString(p.Fset, call.Args[1])
	attrs["http_method_status"] = "unknown"
	attrs["enclosing_symbol"] = enclosingFunctionName(p.Fset, file, call.Pos())
	attrs["callee"] = exprString(p.Fset, call.Fun)
	if obj := calledObject(p, call.Args[1]); obj != nil {
		attrs["handler_symbol"] = objectSymbolID(obj)
	}
	if path, ok := staticStringValue(p, call.Args[0]); ok {
		attrs["route_path"] = path
		attrs["route_path_status"] = "known"
	} else {
		attrs["route_path_status"] = "unknown"
	}
	return routeRegistrationFact{
		key:   attrs["route_path"] + ":unknown:" + positionKey(p.Fset, call.Pos()),
		name:  firstNonEmptyGraph(attrs["route_path"], attrs["handler_expr"], "route registration"),
		attrs: attrs,
	}, true
}

func staticHTTPMethods(p *packages.Package, args []ast.Expr) []string {
	if len(args) == 0 {
		return nil
	}
	out := make([]string, 0, len(args))
	for _, arg := range args {
		method, ok := staticStringValue(p, arg)
		if !ok || strings.TrimSpace(method) == "" {
			return nil
		}
		out = append(out, strings.ToUpper(method))
	}
	return out
}

func staticStringValue(p *packages.Package, expr ast.Expr) (string, bool) {
	switch x := unparen(expr).(type) {
	case *ast.BasicLit:
		if x.Kind != token.STRING {
			return "", false
		}
		value, err := strconv.Unquote(x.Value)
		return value, err == nil
	case *ast.Ident:
		return constStringValue(usedObject(p, x))
	case *ast.SelectorExpr:
		return constStringValue(calledObject(p, x))
	default:
		return "", false
	}
}

func constStringValue(obj types.Object) (string, bool) {
	c, ok := obj.(*types.Const)
	if !ok || c.Val().Kind() != constant.String {
		return "", false
	}
	return constant.StringVal(c.Val()), true
}

func unparen(expr ast.Expr) ast.Expr {
	for {
		p, ok := expr.(*ast.ParenExpr)
		if !ok || p.X == nil {
			return expr
		}
		expr = p.X
	}
}

func copyStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func firstNonEmptyGraph(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func validTypeString(s string) string {
	if s == "<nil>" || s == "invalid type" {
		return ""
	}
	return s
}

func exprString(fset *token.FileSet, expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, expr); err != nil {
		return ""
	}
	return buf.String()
}

func enclosingFunctionName(fset *token.FileSet, file *ast.File, pos token.Pos) string {
	if file == nil {
		return ""
	}
	var name string
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		if fn.Pos() <= pos && pos <= fn.End() {
			name = fn.Name.Name
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				name = receiverSummary(fset, fn.Recv) + "." + name
			}
			break
		}
	}
	return name
}

func receiverSummary(fset *token.FileSet, recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 || recv.List[0] == nil {
		return ""
	}
	return exprString(fset, recv.List[0].Type)
}

func isAddressOfComposite(file *ast.File, lit *ast.CompositeLit) bool {
	if file == nil || lit == nil {
		return false
	}
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found || n == nil {
			return false
		}
		unary, ok := n.(*ast.UnaryExpr)
		if !ok || unary.Op != token.AND {
			return true
		}
		if child, ok := unary.X.(*ast.CompositeLit); ok && child == lit {
			found = true
			return false
		}
		return true
	})
	return found
}

// boolAttr renders a Go bool as the canonical "true"/"false" string so
// attribute maps serialize deterministically.
func boolAttr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// intAttr renders a non-negative int as its decimal string.
func intAttr(n int) string {
	if n < 0 {
		n = 0
	}
	return strconv.Itoa(n)
}

// isTestFile returns true when the file basename ends in "_test.go", the
// Go convention for test files. Today the loader is configured with
// Tests:false, so test files do not appear in p.GoFiles and this returns
// false for every emitted file. The attribute is still emitted so the
// wire shape is stable for consumers and a future Tests:true switch
// produces meaningful values without a fixture re-cut.
func isTestFile(absPath string) bool {
	return strings.HasSuffix(filepath.Base(absPath), "_test.go")
}

// isInternalImportPath returns true when path matches the Go convention
// for internal-only packages: either starts with "internal/" or contains
// "/internal/" or "/internal" as a trailing segment.
func isInternalImportPath(path string) bool {
	if path == "internal" || strings.HasPrefix(path, "internal/") {
		return true
	}
	if strings.Contains(path, "/internal/") || strings.HasSuffix(path, "/internal") {
		return true
	}
	return false
}

// fileLineCounts returns a map from absolute file path to the file's
// last line number, computed from p.Fset. Files whose syntax was not
// parsed (or whose Fset entry is missing) get a zero count.
func fileLineCounts(p *packages.Package) map[string]int {
	out := make(map[string]int, len(p.GoFiles))
	if p.Fset == nil {
		return out
	}
	for _, syn := range p.Syntax {
		if syn == nil {
			continue
		}
		pos := p.Fset.Position(syn.End())
		if pos.Filename == "" {
			continue
		}
		// End() points at the byte after the last token; Line is the
		// physical line of that position. For well-formed Go files this
		// is the line of the trailing newline.
		out[pos.Filename] = pos.Line
	}
	return out
}

// relPath returns the module-rooted relative path when moduleRoot
// is a prefix of abs, otherwise abs unchanged. Both inputs should be
// cleaned absolute paths.
func relPath(moduleRoot, abs string) string {
	if moduleRoot == "" {
		return abs
	}
	if rel, err := filepath.Rel(moduleRoot, abs); err == nil && !filepath.IsAbs(rel) {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(abs)
}
