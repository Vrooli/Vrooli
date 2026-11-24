package facts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	controlPlaneModulePath = "github.com/vrooli/vrooli/"
	maxNestedPackageFiles  = 80
	maxNestedPackageBytes  = 4 << 20
	maxNestedCallDepth     = 12
)

type cliPackageFunction struct {
	unit cliSourceUnit
	node *ast.FuncDecl
}

type cliNestedRegistration struct {
	key      string
	expr     ast.Expr
	line     int
	path     string
	hash     string
	contract string
}

type cliNestedTable struct {
	function   string
	specName   string
	specs      []cliCommandSpec
	specPath   string
	specHash   string
	tablePath  string
	tableHash  string
	tableLine  int
	registries map[string][]cliNestedRegistration
}

type cliNestedChild struct {
	table          *cliNestedTable
	group          *cliNestedRegistration
	groupAmbiguous bool
}

// cliNestedGroupBindings follows direct local source calls from registered
// root handlers to commandtree.BindSpecs tables. It reports declared command
// specifications and explicit source registrations while leaving runtime
// invocation and argument predicates unknown.
func cliNestedGroupBindings(root string, app cliSourceUnit, appHash string, topSpecs []cliCommandSpec, handlers map[string]cliHandlerBinding) []CompositionBinding {
	imports := importedPackagePaths(app.file)
	var out []CompositionBinding
	for _, top := range topSpecs {
		binding := handlers[top.handler]
		alias, ok, ambiguous := cliRootHandlerAlias(binding.expression, app.file)
		if ambiguous {
			out = append(out, cliNestedDegradation(top.name, binding, app.path, appHash, "root handler factory reaches multiple imported package RootHandler functions; package ownership is ambiguous"))
			continue
		}
		if !ok {
			continue
		}
		packageDir, ok := cliInternalPackageDir(root, imports[alias])
		if !ok {
			out = append(out, cliNestedDegradation(top.name, binding, app.path, appHash, "root handler package is outside the supported in-repository scope"))
			continue
		}
		units, err := readCLIPackageUnits(root, packageDir)
		if err != nil {
			out = append(out, cliNestedDegradation(top.name, binding, app.path, appHash, "nested command package could not be parsed: "+err.Error()))
			continue
		}
		functions := cliPackageFunctions(units)
		rootFn, ok := functions["RootHandler"]
		if !ok {
			out = append(out, cliNestedDegradation(top.name, binding, app.path, appHash, "registered handler package has no parsed RootHandler"))
			continue
		}
		reachable := cliReachablePackageFunctions("RootHandler", functions)
		if _, capped := reachable["\x00analysis-cap"]; capped {
			out = append(out, cliNestedDegradation(top.name, binding, app.path, appHash, "nested direct-caller traversal hit its depth/function cap; reported rows are partial"))
			delete(reachable, "\x00analysis-cap")
		}
		tables := cliFindNestedTables(root, reachable)
		if len(tables) == 0 {
			out = append(out, cliNestedDegradation(top.name, binding, app.path, appHash, "no supported CommandSpecs→BindSpecs table found from RootHandler"))
			continue
		}
		out = append(out, cliRenderNestedTables(top.name, tables, functions, rootFn)...)
		out = append(out, CompositionBinding{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: top.name + ":nested-scan-boundary", Interface: "nested command inventory boundary", ProviderExpression: "PARTIAL: only reachable parsed CommandSpecs→BindSpecs tables and supported direct registration shapes are listed; other registration and dispatch forms remain UNKNOWN", Status: "unknown", SourcePath: app.path, Line: binding.line, SourceHash: appHash, Provenance: "nested-command-scan-boundary"})
	}
	return out
}

func cliNestedDegradation(topName string, binding cliHandlerBinding, path, hash, reason string) CompositionBinding {
	return CompositionBinding{
		Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: topName + ":nested-command-inventory",
		Interface: "nested command composition", Provider: binding.expression,
		ProviderExpression: "UNKNOWN: " + reason + "; nested command inventory may be incomplete",
		Status:             "unknown", SourcePath: path, Line: binding.line, SourceHash: hash,
		Provenance: "nested-command-scan-degraded",
	}
}

func cliRootHandlerAlias(expression string, app *ast.File) (string, bool, bool) {
	if strings.HasSuffix(expression, ".RootHandler") {
		parts := strings.Split(expression, ".")
		if len(parts) == 2 {
			return parts[0], true, false
		}
	}
	// Some root entries are local factories whose body directly calls a
	// package RootHandler, such as topLevelPackageHandler.
	for _, decl := range app.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != expression || fn.Body == nil {
			continue
		}
		imports := importedPackagePaths(app)
		alias, packagePath := "", ""
		ambiguous := false
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "RootHandler" {
				return true
			}
			if pkg, ok := selector.X.(*ast.Ident); ok {
				candidatePath := imports[pkg.Name]
				if alias == "" {
					alias, packagePath = pkg.Name, candidatePath
				} else if candidatePath != packagePath {
					ambiguous = true
				}
			}
			return true
		})
		if ambiguous {
			return "", false, true
		}
		if alias != "" {
			return alias, true, false
		}
	}
	return "", false, false
}

func cliInternalPackageDir(root, importPath string) (string, bool) {
	rel := strings.TrimPrefix(importPath, controlPlaneModulePath)
	if rel == importPath || rel == "" || strings.HasPrefix(rel, "../") {
		return "", false
	}
	dir := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return dir, true
}

func readCLIPackageUnits(root, dir string) ([]cliSourceUnit, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	if len(entries) > maxNestedPackageFiles {
		return nil, fmt.Errorf("source file cap %d exceeded", maxNestedPackageFiles)
	}
	var total int64
	var units []cliSourceUnit
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		total += info.Size()
		if total > maxNestedPackageBytes {
			return nil, fmt.Errorf("source byte cap %d exceeded", maxNestedPackageBytes)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, content, parser.AllErrors)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(content)
		units = append(units, cliSourceUnit{path: filepath.ToSlash(rel), fset: fset, file: file, hash: hex.EncodeToString(sum[:])})
	}
	sort.Slice(units, func(i, j int) bool { return units[i].path < units[j].path })
	return units, nil
}

func cliPackageFunctions(units []cliSourceUnit) map[string]cliPackageFunction {
	functions := map[string]cliPackageFunction{}
	for _, unit := range units {
		for _, decl := range unit.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Body == nil {
				continue
			}
			functions[fn.Name.Name] = cliPackageFunction{unit: unit, node: fn}
		}
	}
	return functions
}

func cliReachablePackageFunctions(start string, functions map[string]cliPackageFunction) map[string]cliPackageFunction {
	reachable := map[string]cliPackageFunction{}
	type nextFunction struct {
		name  string
		depth int
	}
	queue := []nextFunction{{name: start}}
	for len(queue) > 0 && len(reachable) < 500 {
		item := queue[0]
		queue = queue[1:]
		if _, seen := reachable[item.name]; seen {
			continue
		}
		if item.depth > maxNestedCallDepth {
			reachable["\x00analysis-cap"] = cliPackageFunction{}
			continue
		}
		fn, ok := functions[item.name]
		if !ok {
			continue
		}
		reachable[item.name] = fn
		for _, called := range cliLocalCallNames(fn.node.Body) {
			if _, ok := functions[called]; ok {
				queue = append(queue, nextFunction{name: called, depth: item.depth + 1})
			}
		}
	}
	if len(queue) > 0 {
		reachable["\x00analysis-cap"] = cliPackageFunction{}
	}
	return reachable
}

func cliLocalCallNames(node ast.Node) []string {
	seen := map[string]bool{}
	var names []string
	ast.Inspect(node, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		var name string
		if id, ok := call.Fun.(*ast.Ident); ok {
			name = id.Name
		}
		if name != "" && !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
		return true
	})
	return names
}

func cliFindNestedTables(root string, reachable map[string]cliPackageFunction) []*cliNestedTable {
	var tables []*cliNestedTable
	specUnitCache := map[string][]cliSourceUnit{}
	specUnitErrors := map[string]error{}
	for name, function := range reachable {
		if name == "\x00analysis-cap" || function.node == nil {
			continue
		}
		ast.Inspect(function.node.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || !cliIsBindSpecsCall(call) || len(call.Args) < 2 {
				return true
			}
			specCall, ok := call.Args[0].(*ast.CallExpr)
			if !ok {
				return true
			}
			specName, specAlias := cliCalledSelector(specCall.Fun)
			if specName == "" {
				return true
			}
			specPackagePath := importedPackagePaths(function.unit.file)[specAlias]
			specDir, ok := cliInternalPackageDir(root, specPackagePath)
			if !ok {
				return true
			}
			specUnits, loaded := specUnitCache[specPackagePath]
			err := specUnitErrors[specPackagePath]
			if !loaded {
				specUnits, err = readCLIPackageUnits(root, specDir)
				specUnitCache[specPackagePath] = specUnits
				specUnitErrors[specPackagePath] = err
			}
			if err != nil {
				tables = append(tables, &cliNestedTable{function: name, specName: specName, specPath: function.unit.path, specHash: function.unit.hash, registries: map[string][]cliNestedRegistration{}})
				return true
			}
			specs, specUnit := cliParseNamedCommandSpecs(specUnits, specName)
			if specUnit.path == "" {
				specUnit = function.unit
			}
			tables = append(tables, &cliNestedTable{function: name, specName: specName, specs: specs, specPath: specUnit.path, specHash: specUnit.hash, tablePath: function.unit.path, tableHash: function.unit.hash, tableLine: function.unit.fset.Position(call.Pos()).Line, registries: cliNestedHandlerRegistrations(function, call.Args[1], specCall, specs)})
			return true
		})
	}
	sort.Slice(tables, func(i, j int) bool {
		if tables[i].function == tables[j].function {
			return tables[i].specName < tables[j].specName
		}
		return tables[i].function < tables[j].function
	})
	return tables
}

func cliIsBindSpecsCall(call *ast.CallExpr) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "BindSpecs"
}

func cliCalledSelector(expr ast.Expr) (name, alias string) {
	switch fn := expr.(type) {
	case *ast.SelectorExpr:
		pkg, ok := fn.X.(*ast.Ident)
		if ok {
			return fn.Sel.Name, pkg.Name
		}
	case *ast.Ident:
		return fn.Name, ""
	}
	return "", ""
}

func cliParseNamedCommandSpecs(units []cliSourceUnit, name string) ([]cliCommandSpec, cliSourceUnit) {
	constants := map[string]string{}
	for _, unit := range units {
		for key, value := range cliCommandConstants(unit.file) {
			constants[key] = value
		}
	}
	for _, unit := range units {
		for _, decl := range unit.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != name || fn.Body == nil {
				continue
			}
			return cliCommandSpecsFromFunction(fn, constants, unit.fset), unit
		}
	}
	return nil, cliSourceUnit{}
}

func cliCommandSpecsFromFunction(fn *ast.FuncDecl, constants map[string]string, fset *token.FileSet) []cliCommandSpec {
	var specs []cliCommandSpec
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		lit, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if _, ok := lit.Type.(*ast.ArrayType); !ok {
			return true
		}
		for _, element := range lit.Elts {
			row, ok := element.(*ast.CompositeLit)
			if !ok {
				continue
			}
			spec := cliCommandSpec{line: fset.Position(row.Pos()).Line}
			for _, field := range row.Elts {
				kv, ok := field.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				if !ok {
					continue
				}
				switch key.Name {
				case "Name":
					spec.name = cliResolveCommandName(kv.Value, constants)
				case "Handler":
					spec.handler = resolveCommandID(kv.Value)
				case "Group":
					if value, ok := kv.Value.(*ast.BasicLit); ok {
						spec.group, _ = strconv.Unquote(value.Value)
					}
				case "Hidden":
					if value, ok := kv.Value.(*ast.Ident); ok {
						spec.hidden = value.Name == "true"
					}
				}
			}
			if spec.name != "" && spec.handler != "" {
				specs = append(specs, spec)
			}
		}
		return false
	})
	return specs
}

func cliResolveCommandName(expr ast.Expr, constants map[string]string) string {
	if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.STRING {
		value, _ := strconv.Unquote(literal.Value)
		return value
	}
	if call, ok := expr.(*ast.CallExpr); ok && len(call.Args) > 0 {
		expr = call.Args[0]
	}
	if id, ok := expr.(*ast.Ident); ok {
		return constants[id.Name]
	}
	if selector, ok := expr.(*ast.SelectorExpr); ok {
		return constants[selector.Sel.Name]
	}
	return ""
}

func cliNestedHandlerRegistrations(function cliPackageFunction, mapExpr ast.Expr, specCall *ast.CallExpr, specs []cliCommandSpec) map[string][]cliNestedRegistration {
	registries := map[string][]cliNestedRegistration{}
	mapName := ""
	if id, ok := mapExpr.(*ast.Ident); ok {
		mapName = id.Name
	}
	if mapName == "" {
		return registries
	}
	add := func(key string, expression ast.Expr, pos token.Pos, contract string) {
		if key == "" {
			return
		}
		registries[key] = append(registries[key], cliNestedRegistration{expr: expression, line: function.unit.fset.Position(pos).Line, path: function.unit.path, hash: function.unit.hash, contract: contract})
	}
	ast.Inspect(function.node.Body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			return true
		}
		lhs, ok := assignment.Lhs[0].(*ast.Ident)
		if !ok || lhs.Name != mapName {
			return true
		}
		literal, ok := assignment.Rhs[0].(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, element := range literal.Elts {
			field, ok := element.(*ast.KeyValueExpr)
			if ok {
				add(cliStaticMapKey(field.Key, function.unit.file), field.Value, field.Pos(), "direct map-literal registration")
			}
		}
		return true
	})
	ast.Inspect(function.node.Body, func(node ast.Node) bool {
		assignment, ok := node.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
			return true
		}
		index, ok := assignment.Lhs[0].(*ast.IndexExpr)
		if !ok {
			return true
		}
		base, ok := index.X.(*ast.Ident)
		if ok && base.Name == mapName {
			add(cliStaticMapKey(index.Index, function.unit.file), assignment.Rhs[0], assignment.Pos(), "constant-index map assignment")
		}
		return true
	})
	// Recognize the exact shared-table idiom used by resource handlers: the
	// same CommandSpecs function is ranged, each row's Handler is assigned to a
	// local key, and that key is used for the handler-map assignment.
	specFunction, specPackage := cliCalledSelector(specCall.Fun)
	ast.Inspect(function.node.Body, func(node ast.Node) bool {
		rangeStmt, ok := node.(*ast.RangeStmt)
		if !ok || rangeStmt.Value == nil {
			return true
		}
		rangeCall, ok := rangeStmt.X.(*ast.CallExpr)
		if !ok {
			return true
		}
		rangeFunction, rangePackage := cliCalledSelector(rangeCall.Fun)
		if rangeFunction != specFunction || rangePackage != specPackage {
			return true
		}
		row, ok := rangeStmt.Value.(*ast.Ident)
		if !ok {
			return true
		}
		actionName := ""
		var indexedAssignments []*ast.AssignStmt
		ast.Inspect(rangeStmt.Body, func(bodyNode ast.Node) bool {
			assignment, ok := bodyNode.(*ast.AssignStmt)
			if !ok || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
				return true
			}
			if identifier, ok := assignment.Lhs[0].(*ast.Ident); ok {
				if selector, ok := assignment.Rhs[0].(*ast.SelectorExpr); ok {
					if receiver, ok := selector.X.(*ast.Ident); ok && receiver.Name == row.Name && selector.Sel.Name == "Handler" {
						actionName = identifier.Name
					}
				}
			}
			if index, ok := assignment.Lhs[0].(*ast.IndexExpr); ok {
				if base, ok := index.X.(*ast.Ident); ok && base.Name == mapName {
					if key, ok := index.Index.(*ast.Ident); ok && actionName != "" && key.Name == actionName {
						indexedAssignments = append(indexedAssignments, assignment)
					}
				}
			}
			return true
		})
		if actionName == "" || len(indexedAssignments) != 1 {
			return true
		}
		for _, spec := range specs {
			registries[spec.handler] = append(registries[spec.handler], cliNestedRegistration{expr: indexedAssignments[0].Rhs[0], line: function.unit.fset.Position(indexedAssignments[0].Pos()).Line, path: function.unit.path, hash: function.unit.hash, contract: "handler map key derives from the same command spec row's Handler field; loop execution remains unknown"})
		}
		return true
	})
	return registries
}

func cliStaticMapKey(expr ast.Expr, file *ast.File) string {
	if selector, ok := expr.(*ast.SelectorExpr); ok {
		return selector.Sel.Name
	}
	if id, ok := expr.(*ast.Ident); ok {
		if _, declared := cliCommandConstants(file)[id.Name]; declared {
			return id.Name
		}
		return ""
	}
	if literal, ok := expr.(*ast.BasicLit); ok && literal.Kind == token.STRING {
		key, _ := strconv.Unquote(literal.Value)
		return key
	}
	return ""
}

func cliRenderNestedTables(topName string, tables []*cliNestedTable, functions map[string]cliPackageFunction, rootFn cliPackageFunction) []CompositionBinding {
	var out []CompositionBinding
	childTables := func(table *cliNestedTable, spec cliCommandSpec) []cliNestedChild {
		registrations := table.registries[spec.handler]
		if len(registrations) != 1 || registrations[0].expr == nil {
			return nil
		}
		called := map[string]bool{}
		for _, name := range cliLocalCallNames(registrations[0].expr) {
			for functionName := range cliReachablePackageFunctions(name, functions) {
				if functionName != table.function {
					called[functionName] = true
				}
			}
		}
		var children []cliNestedChild
		for _, candidate := range tables {
			if candidate.function != table.function && called[candidate.function] {
				group, groupAmbiguous := cliNamedIntermediateGroup(registrations[0].expr, candidate.function, functions)
				children = append(children, cliNestedChild{table: candidate, group: group, groupAmbiguous: groupAmbiguous})
			}
		}
		return children
	}
	incoming := map[*cliNestedTable]bool{}
	for _, table := range tables {
		for _, spec := range table.specs {
			for _, child := range childTables(table, spec) {
				incoming[child.table] = true
			}
		}
	}
	var roots []*cliNestedTable
	for _, table := range tables {
		if !incoming[table] {
			roots = append(roots, table)
		}
	}
	if len(roots) == 0 {
		roots = tables
	}
	visited := map[string]bool{}
	var walk func(*cliNestedTable, string, int)
	walk = func(table *cliNestedTable, parent string, depth int) {
		visitKey := table.function + "\x00" + parent
		if visited[visitKey] {
			return
		}
		visited[visitKey] = true
		if depth > maxNestedCallDepth {
			out = append(out, CompositionBinding{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: parent + ":nested-depth-cap", Interface: "nested command composition", ProviderExpression: "UNKNOWN: nested table traversal cap reached", Status: "unknown", SourcePath: rootFn.unit.path, Line: rootFn.unit.fset.Position(rootFn.node.Pos()).Line, SourceHash: rootFn.unit.hash, Provenance: "nested-command-depth-cap"})
			return
		}
		if len(table.specs) == 0 {
			out = append(out, CompositionBinding{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: parent + ":command-specs-unknown", Interface: "nested command specification table", ProviderExpression: "UNKNOWN: no literal command rows were extracted from the referenced CommandSpecs function", Status: "unknown", SourcePath: table.specPath, SourceHash: table.specHash, Provenance: "nested-command-spec-parse-degraded"})
			return
		}
		for _, spec := range table.specs {
			registrations := table.registries[spec.handler]
			status, provider := "unknown", "UNKNOWN: no supported direct registration found"
			registration := cliNestedRegistration{}
			switch len(registrations) {
			case 1:
				status = "concrete"
				registration = registrations[0]
				provider = handlerExpression(registration.expr)
			case 0:
			default:
				status = "ambiguous"
				provider = "AMBIGUOUS: multiple source registrations for declared handler id"
			}
			line, path, hash := spec.line, table.specPath, table.specHash
			if registration.line > 0 {
				line, path, hash = registration.line, registration.path, registration.hash
			}
			provenance := "nested-command-spec-handler-table; spec=" + table.specPath + ":" + strconv.Itoa(spec.line) + "#sha256:" + table.specHash
			text := fmt.Sprintf("nested command %q declares handler id %s; %s; runtime dispatch and invocation are unknown", spec.name, spec.handler, provider)
			if table.tablePath != "" {
				provenance += fmt.Sprintf("; BindSpecs table=%s:%d#sha256:%s", table.tablePath, table.tableLine, table.tableHash)
				text += fmt.Sprintf("; BindSpecs table source=%s:%d#sha256:%s", table.tablePath, table.tableLine, table.tableHash)
			}
			if registration.contract != "" {
				provenance += "; " + registration.contract
				text += fmt.Sprintf("; registration source=%s:%d#sha256:%s", registration.path, registration.line, registration.hash)
			}
			if len(registrations) > 1 {
				var sources []string
				for _, candidate := range registrations {
					sources = append(sources, fmt.Sprintf("%s:%d#sha256:%s (%s)", candidate.path, candidate.line, candidate.hash, candidate.contract))
				}
				text += "; competing source registrations: " + strings.Join(sources, ", ")
				provenance += "; competing registration evidence: " + strings.Join(sources, ", ")
			}
			dependency := parent + ":" + spec.name
			out = append(out, CompositionBinding{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: dependency, Interface: fmt.Sprintf("nested commandtree.Spec group=%q handler=%s under %s", spec.group, spec.handler, parent), Provider: provider, ProviderExpression: text, Status: status, SourcePath: path, Line: line, SourceHash: hash, Provenance: provenance})
			children := childTables(table, spec)
			if len(children) == 1 {
				childParent := dependency
				if children[0].groupAmbiguous {
					out = append(out, CompositionBinding{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: dependency + ":nested-group-ambiguity", Interface: "nested string-key command group", ProviderExpression: "UNKNOWN: multiple intermediate string-key registrations lead to this table; hierarchy was not selected", Status: "unknown", SourcePath: path, Line: line, SourceHash: hash, Provenance: "nested-command-ambiguous-intermediate-map"})
					continue
				}
				if children[0].group != nil {
					group := children[0].group
					childParent += ":" + group.key
					out = append(out, CompositionBinding{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: childParent, Interface: "nested string-key command group below " + dependency, Provider: handlerExpression(group.expr), ProviderExpression: fmt.Sprintf("source declares nested group key %q; runtime dispatch and invocation are unknown; registration source=%s:%d#sha256:%s", group.key, group.path, group.line, group.hash), Status: "concrete", SourcePath: group.path, Line: group.line, SourceHash: group.hash, Provenance: "direct-nested-string-map-registration"})
				}
				walk(children[0].table, childParent, depth+1)
			} else if len(children) > 1 {
				out = append(out, CompositionBinding{Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: dependency + ":nested-table-ambiguity", Interface: "nested handler references multiple command tables", ProviderExpression: "UNKNOWN: multiple statically reachable child tables were found; hierarchy was not selected", Status: "unknown", SourcePath: path, Line: line, SourceHash: hash, Provenance: "nested-command-ambiguous-hierarchy"})
			}
		}
	}
	for _, table := range roots {
		parent := topName
		if len(roots) > 1 || table.specName != "CommandSpecs" {
			parent += ":table:" + table.specName
		}
		walk(table, parent, 0)
	}
	return out
}

func cliNamedIntermediateGroup(expression ast.Expr, childFunction string, functions map[string]cliPackageFunction) (*cliNestedRegistration, bool) {
	var matches []cliNestedRegistration
	for _, called := range cliLocalCallNames(expression) {
		for _, function := range cliReachablePackageFunctions(called, functions) {
			if function.node == nil {
				continue
			}
			ast.Inspect(function.node.Body, func(node ast.Node) bool {
				literal, ok := node.(*ast.CompositeLit)
				if !ok {
					return true
				}
				mapType, ok := literal.Type.(*ast.MapType)
				if !ok {
					return true
				}
				keyType, ok := mapType.Key.(*ast.Ident)
				if !ok || keyType.Name != "string" {
					return true
				}
				for _, element := range literal.Elts {
					field, ok := element.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := field.Key.(*ast.BasicLit)
					if !ok || key.Kind != token.STRING {
						continue
					}
					calls := map[string]bool{}
					for _, name := range cliLocalCallNames(field.Value) {
						for reachable := range cliReachablePackageFunctions(name, functions) {
							calls[reachable] = true
						}
					}
					if calls[childFunction] {
						groupName, _ := strconv.Unquote(key.Value)
						matches = append(matches, cliNestedRegistration{key: groupName, expr: field.Value, line: function.unit.fset.Position(field.Pos()).Line, path: function.unit.path, hash: function.unit.hash, contract: "direct string-key map value reaches nested command table"})
					}
				}
				return true
			})
		}
	}
	if len(matches) == 1 {
		return &matches[0], false
	}
	return nil, len(matches) > 1
}
