package facts

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CompositionReport is direct, source-backed evidence of Go dependency
// injection. It deliberately does not infer bindings from imports.
type CompositionReport struct {
	RootPath      string               `json:"root_path"`
	AnalysisScope string               `json:"analysis_scope"`
	Bindings      []CompositionBinding `json:"bindings"`
}

type CompositionBinding struct {
	Environment        string `json:"environment"`
	Consumer           string `json:"consumer"`
	Dependency         string `json:"dependency"`
	Interface          string `json:"interface"`
	Provider           string `json:"provider,omitempty"`
	ProviderExpression string `json:"provider_expression,omitempty"`
	Status             string `json:"status"` // concrete, unknown, or ambiguous
	SourcePath         string `json:"source_path"`
	Line               int    `json:"line"`
	SourceHash         string `json:"source_hash"`
	Provenance         string `json:"provenance"`
}

type compositionFunction struct {
	id    string
	name  string
	unit  compositionUnit
	node  *ast.FuncDecl
	calls []string
}

type compositionReturnLiteral struct {
	expr ast.Expr
	unit compositionUnit
	line int
}

type compositionReturnIndex struct {
	declarations int
	eligible     *compositionReturnLiteral
}

func compositionEntrypointChains(root string, units []compositionUnit) (map[string][]string, map[*ast.KeyValueExpr]string, map[string]*ast.FuncDecl) {
	functions := map[string]compositionFunction{}
	functionOwners := map[*ast.KeyValueExpr]string{}
	functionNodes := map[string]*ast.FuncDecl{}
	packageIDs := map[string]string{}
	for _, unit := range units {
		dir := filepath.ToSlash(filepath.Dir(unit.path))
		packageID := modulePackagePath(root, dir)
		packageIDs[dir] = packageID
	}
	for _, unit := range units {
		dir := filepath.ToSlash(filepath.Dir(unit.path))
		packageID := packageIDs[dir]
		imports := importedPackagePaths(unit.file)
		for _, decl := range unit.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			id := packageID + ":" + fn.Name.Name
			if fn.Recv != nil {
				id = packageID + ":" + receiverTypeName(fn.Recv.List[0].Type) + "." + fn.Name.Name
			}
			item := compositionFunction{id: id, name: fn.Name.Name, unit: unit, node: fn}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if lit, ok := node.(*ast.CompositeLit); ok {
					for _, element := range lit.Elts {
						if field, ok := element.(*ast.KeyValueExpr); ok {
							functionOwners[field] = id
						}
					}
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch target := call.Fun.(type) {
				case *ast.Ident:
					item.calls = append(item.calls, packageID+":"+target.Name)
				case *ast.SelectorExpr:
					if alias, ok := target.X.(*ast.Ident); ok {
						if importPath := imports[alias.Name]; importPath != "" {
							item.calls = append(item.calls, importPath+":"+target.Sel.Name)
						}
					}
				}
				return true
			})
			functions[id] = item
			functionNodes[id] = fn
		}
	}
	roots := []string{}
	for id, fn := range functions {
		if fn.name == "main" && fn.node.Recv == nil && fn.unit.file.Name.Name == "main" {
			roots = append(roots, id)
		}
	}
	chains := map[string][]string{}
	for _, entry := range roots {
		queue := []string{entry}
		paths := map[string][]string{entry: {entry}}
		rootFn := functions[entry]
		rootPos := rootFn.unit.fset.Position(rootFn.node.Pos())
		chains[entry] = []string{fmt.Sprintf("%s@%s:%d#sha256:%s", entry, rootFn.unit.path, rootPos.Line, rootFn.unit.hash)}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			fn := functions[current]
			for _, called := range fn.calls {
				if _, exists := paths[called]; exists {
					continue
				}
				target, exists := functions[called]
				if !exists {
					continue
				}
				paths[called] = append(append([]string{}, paths[current]...), called)
				queue = append(queue, called)
				pos := target.unit.fset.Position(target.node.Pos())
				described := fmt.Sprintf("%s@%s:%d#sha256:%s", called, target.unit.path, pos.Line, target.unit.hash)
				for i, step := range paths[called] {
					if i == len(paths[called])-1 {
						paths[called][i] = described
						break
					}
					if prior, ok := functions[step]; ok {
						priorPos := prior.unit.fset.Position(prior.node.Pos())
						paths[called][i] = fmt.Sprintf("%s@%s:%d#sha256:%s", step, prior.unit.path, priorPos.Line, prior.unit.hash)
					}
				}
				chains[called] = paths[called]
			}
		}
	}
	return chains, functionOwners, functionNodes
}

func modulePackagePath(root, relDir string) string {
	dir := filepath.Join(root, filepath.FromSlash(relDir))
	for {
		modulePath := modulePathFromFile(filepath.Join(dir, "go.mod"))
		if modulePath != "" {
			rel, err := filepath.Rel(dir, filepath.Join(root, filepath.FromSlash(relDir)))
			if err != nil || rel == "." {
				return modulePath
			}
			return modulePath + "/" + filepath.ToSlash(rel)
		}
		if dir == root || filepath.Dir(dir) == dir {
			break
		}
		dir = filepath.Dir(dir)
	}
	return "scenario/" + relDir
}

func modulePathFromFile(path string) string {
	file, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	return ""
}

func receiverTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	return typeName(expr)
}

type compositionUnit struct {
	path string
	fset *token.FileSet
	file *ast.File
	hash string
	test bool
}

// DescribeComposition scans one bounded Go target and reports explicit
// constructor bindings for interface-typed fields in *Deps structs. It
// resolves direct literals, one safe same-function local form, and one narrow
// direct same-package call-return form. Unsupported expressions remain
// unknown; multiple distinct providers become ambiguous.
func DescribeComposition(root string) (*CompositionReport, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var units []compositionUnit
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		fset := token.NewFileSet()
		file, parseErr := parser.ParseFile(fset, path, content, parser.AllErrors)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", path, parseErr)
		}
		sum := sha256.Sum256(content)
		rel, _ := filepath.Rel(root, path)
		units = append(units, compositionUnit{path: filepath.ToSlash(rel), fset: fset, file: file, hash: hex.EncodeToString(sum[:]), test: strings.HasSuffix(path, "_test.go")})
		return nil
	})
	if err != nil {
		return nil, err
	}
	callChains, functionOwners, functionNodes := compositionEntrypointChains(root, units)
	returnLiterals := indexSingleReturnLiterals(units)
	// Interface names are collected per source package so named local and
	// in-target imported interfaces can be distinguished from concrete fields.
	interfaces := map[string]map[string]bool{}
	for _, unit := range units {
		pkg := filepath.ToSlash(filepath.Dir(unit.path))
		if interfaces[pkg] == nil {
			interfaces[pkg] = map[string]bool{}
		}
		for _, decl := range unit.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts := spec.(*ast.TypeSpec)
				if _, ok := ts.Type.(*ast.InterfaceType); ok {
					interfaces[pkg][ts.Name.Name] = true
				}
			}
		}
	}
	type depKey struct{ pkg, consumer, field, contract string }
	deps := map[depKey]CompositionBinding{}
	for _, unit := range units {
		pkg := filepath.ToSlash(filepath.Dir(unit.path))
		imports := importedPackagePaths(unit.file)
		for _, decl := range unit.file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				ts := spec.(*ast.TypeSpec)
				st, ok := ts.Type.(*ast.StructType)
				if !ok || !strings.HasSuffix(ts.Name.Name, "Deps") {
					continue
				}
				for _, field := range st.Fields.List {
					contract := exprText(field.Type)
					isIface := false
					switch t := field.Type.(type) {
					case *ast.InterfaceType:
						isIface = true
					case *ast.Ident:
						isIface = interfaces[pkg][t.Name]
					case *ast.SelectorExpr:
						if alias, ok := t.X.(*ast.Ident); ok {
							isIface = importedInterface(interfaces, imports[alias.Name], t.Sel.Name)
						}
					}
					if !isIface {
						continue
					}
					for _, name := range field.Names {
						pos := unit.fset.Position(field.Pos())
						key := depKey{pkg, ts.Name.Name, name.Name, contract}
						deps[key] = CompositionBinding{Consumer: ts.Name.Name, Dependency: name.Name, Interface: contract, SourcePath: unit.path, Line: pos.Line, SourceHash: unit.hash, Provenance: "deps-interface-field-declaration"}
					}
				}
			}
		}
	}
	type observation struct{ binding CompositionBinding }
	observed := map[depKey]map[string][]observation{}
	for _, unit := range units {
		pkg := filepath.ToSlash(filepath.Dir(unit.path))
		imports := importedPackagePaths(unit.file)
		ast.Inspect(unit.file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			name := typeName(lit.Type)
			declPackage := pkg
			if sel, ok := lit.Type.(*ast.SelectorExpr); ok {
				if alias, ok := sel.X.(*ast.Ident); ok {
					if importPath := imports[alias.Name]; importPath != "" {
						declPackage = importPath
					}
				}
			}
			if !strings.HasSuffix(name, "Deps") {
				return true
			}
			for _, element := range lit.Elts {
				kv, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				for dep := range deps {
					if dep.consumer != name || !(dep.pkg == declPackage || strings.HasSuffix(declPackage, "/"+dep.pkg)) {
						continue
					}
					if key.Name != dep.field {
						continue
					}
					provider, expression, status := concreteType(kv.Value)
					pos := unit.fset.Position(kv.Pos())
					env := "production"
					if unit.test {
						env = "test"
					}
					provenance := "go-composite-literal-field; type_check=UNKNOWN(package type checking is not performed); constructor_resolution=direct_literal_or_unique_same-function_local_literal_or_unique_zero-arg_same-package_single-return_literal_only; unsupported_calls=UNKNOWN"
					if function := functionOwners[kv]; function != "" {
						if ident, ok := kv.Value.(*ast.Ident); ok {
							if resolved, initializerPos, ok := resolveSingleLocalLiteral(functionNodes[function], ident.Name, kv.Pos()); ok {
								provider, _, status = concreteType(resolved)
								provenance += fmt.Sprintf("; resolved_local_initializer=%s:%d#sha256:%s", unit.path, unit.fset.Position(initializerPos).Line, unit.hash)
							}
						}
						if call, ok := kv.Value.(*ast.CallExpr); ok {
							if resolved, ok := resolveSinglePackageReturnLiteral(call, unit, returnLiterals); ok {
								provider, _, status = concreteType(resolved.expr)
								provenance += fmt.Sprintf("; resolved_call_return=%s:%d#sha256:%s", resolved.unit.path, resolved.line, resolved.unit.hash)
							}
						}
						if chain := callChains[function]; len(chain) > 0 {
							provenance += "; entrypoint_reachability=DERIVED; entrypoint_chain=" + strings.Join(chain, " -> ")
						} else {
							provenance += "; entrypoint_reachability=PARTIAL(no supported static main-to-constructor chain; dynamic registration, method dispatch and reflection unresolved)"
						}
					} else {
						provenance += "; entrypoint_reachability=UNKNOWN(composite literal is outside a supported function body)"
					}
					b := CompositionBinding{Environment: env, Consumer: dep.consumer, Dependency: dep.field, Interface: dep.contract, Provider: provider, ProviderExpression: expression, Status: status, SourcePath: unit.path, Line: pos.Line, SourceHash: unit.hash, Provenance: provenance}
					if observed[dep] == nil {
						observed[dep] = map[string][]observation{}
					}
					observed[dep][env] = append(observed[dep][env], observation{b})
				}
			}
			return true
		})
	}
	report := &CompositionReport{RootPath: root, AnalysisScope: "go-syntax-v4; direct Go composite literals, one unique same-function local initialized directly by T{} or &T{}, and one zero-argument same-package call to a unique function whose only statement directly returns T{} or &T{} are source-backed with path/line/hash; supported static main-to-constructor call chains are derived; package type checking, parameterized/conditional/indirect call returns, alias-chain, dynamic-dispatch, reflection, build-variant, non-Go, and out-of-target composition remain UNKNOWN/PARTIAL"}
	keys := make([]depKey, 0, len(deps))
	for key := range deps {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		return a.pkg+"."+a.consumer+"."+a.field < b.pkg+"."+b.consumer+"."+b.field
	})
	for _, key := range keys {
		for _, env := range []string{"production", "test"} {
			items := observed[key][env]
			if len(items) == 0 {
				missing := deps[key]
				missing.Environment = env
				missing.Status = "unknown"
				missing.Provenance = "deps-interface-field-no-matching-composite-literal"
				report.Bindings = append(report.Bindings, missing)
				continue
			}
			providers := map[string]bool{}
			for _, item := range items {
				if item.binding.Provider != "" {
					providers[item.binding.Provider] = true
				}
			}
			if len(providers) > 1 {
				for _, item := range items {
					item.binding.Status = "ambiguous"
					report.Bindings = append(report.Bindings, item.binding)
				}
				continue
			}
			for _, item := range items {
				report.Bindings = append(report.Bindings, item.binding)
			}
		}
	}
	return report, nil
}

func indexSingleReturnLiterals(units []compositionUnit) map[string]compositionReturnIndex {
	index := map[string]compositionReturnIndex{}
	for _, unit := range units {
		packageKey := filepath.ToSlash(filepath.Dir(unit.path)) + "\x00" + unit.file.Name.Name
		for _, decl := range unit.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue
			}
			key := packageKey + "\x00" + fn.Name.Name
			item := index[key]
			item.declarations++
			if len(fn.Type.Params.List) == 0 && fn.Body != nil && len(fn.Body.List) == 1 {
				if returned, ok := fn.Body.List[0].(*ast.ReturnStmt); ok && len(returned.Results) == 1 {
					if _, _, status := concreteType(returned.Results[0]); status == "concrete" {
						item.eligible = &compositionReturnLiteral{expr: returned.Results[0], unit: unit, line: unit.fset.Position(returned.Pos()).Line}
					}
				}
			}
			index[key] = item
		}
	}
	return index
}

func resolveSinglePackageReturnLiteral(call *ast.CallExpr, unit compositionUnit, index map[string]compositionReturnIndex) (compositionReturnLiteral, bool) {
	if call == nil || len(call.Args) != 0 {
		return compositionReturnLiteral{}, false
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok {
		return compositionReturnLiteral{}, false
	}
	packageKey := filepath.ToSlash(filepath.Dir(unit.path)) + "\x00" + unit.file.Name.Name + "\x00" + name.Name
	item, ok := index[packageKey]
	if !ok || item.declarations != 1 || item.eligible == nil {
		return compositionReturnLiteral{}, false
	}
	return *item.eligible, true
}

// resolveSingleLocalLiteral supports one narrow constructor form: a top-level
// short declaration in the same function, initialized directly by T{} or
// &T{}, and never reassigned. It does not follow calls, parameters, aliases,
// nested-block declarations, or values from another function.
func resolveSingleLocalLiteral(fn *ast.FuncDecl, name string, use token.Pos) (ast.Expr, token.Pos, bool) {
	if fn == nil || fn.Body == nil || name == "" {
		return nil, token.NoPos, false
	}
	var definition *ast.AssignStmt
	for _, stmt := range fn.Body.List {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) != len(assign.Rhs) {
			continue
		}
		for i, lhs := range assign.Lhs {
			ident, ok := lhs.(*ast.Ident)
			if ok && ident.Name == name && assign.Rhs[i].Pos() < use {
				if definition != nil {
					return nil, token.NoPos, false
				}
				definition = assign
			}
		}
	}
	if definition == nil {
		return nil, token.NoPos, false
	}
	var resolved ast.Expr
	for i, lhs := range definition.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if ok && ident.Name == name {
			resolved = definition.Rhs[i]
			break
		}
	}
	if resolved == nil {
		return nil, token.NoPos, false
	}
	// Reject any additional assignment/declaration to the name, including
	// nested writes that could invalidate the direct value before the use.
	writes := 0
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		switch stmt := node.(type) {
		case *ast.AssignStmt:
			for _, lhs := range stmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Name == name {
					writes++
				}
			}
		case *ast.ValueSpec:
			for _, declared := range stmt.Names {
				if declared.Name == name {
					writes++
				}
			}
		}
		return true
	})
	if writes != 1 {
		return nil, token.NoPos, false
	}
	if _, _, status := concreteType(resolved); status != "concrete" {
		return nil, token.NoPos, false
	}
	return resolved, definition.Pos(), true
}

func exprText(expr ast.Expr) string {
	var b strings.Builder
	_ = format.Node(&b, token.NewFileSet(), expr)
	return b.String()
}
func typeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	}
	return ""
}

func importedPackagePaths(file *ast.File) map[string]string {
	paths := map[string]string{}
	for _, spec := range file.Imports {
		path := strings.Trim(spec.Path.Value, "\"")
		base := filepath.Base(path)
		if spec.Name != nil {
			base = spec.Name.Name
		}
		paths[base] = path
	}
	return paths
}

func importedInterface(interfaces map[string]map[string]bool, importPath, name string) bool {
	if importPath == "" {
		return false
	}
	for pkg, types := range interfaces {
		if (importPath == pkg || strings.HasSuffix(importPath, "/"+pkg)) && types[name] {
			return true
		}
	}
	return false
}
func concreteType(expr ast.Expr) (string, string, string) {
	switch t := expr.(type) {
	case *ast.UnaryExpr:
		if t.Op == token.AND {
			provider, _, status := concreteType(t.X)
			return provider, exprText(expr), status
		}
	case *ast.CompositeLit:
		return exprText(t.Type), exprText(expr), "concrete"
	}
	return "", exprText(expr), "unknown"
}
