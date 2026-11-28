package facts

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type connectProcedureSource struct {
	path, service, method      string
	source, methodRegistration EndpointSource
}

type connectMountSource struct {
	alias, service, pathVar, handlerVar       string
	constructor, closure, mount, registration EndpointSource
}

func scanConnectEndpointAnatomy(root string, module *ast.File, fset *token.FileSet, moduleHash string, declarations []apiDeclaration, terms []string) ([]EndpointAnatomy, []string) {
	imports := map[string]string{}
	for _, item := range module.Imports {
		path, err := strconv.Unquote(item.Path.Value)
		if err != nil {
			continue
		}
		alias := filepath.Base(path)
		if item.Name != nil {
			alias = item.Name.Name
		}
		imports[alias] = path
	}
	var results []EndpointAnatomy
	var degradations []string
	for _, declaration := range declarations {
		if declaration.pathExpr == "" {
			continue
		}
		alias, symbol, selectorOK := splitSelectorName(declaration.pathExpr)
		supportedSymbol := selectorOK && strings.HasSuffix(symbol, "Procedure")
		procedure, found := connectProcedureSource{}, false
		if supportedSymbol {
			procedure, found = resolveConnectProcedure(root, imports[alias], symbol)
		}
		row := EndpointAnatomy{
			ID: "code-facts:endpoint:" + declaration.id, Title: declaration.method + " " + declaration.pathExpr,
			Summary: declaration.summary, Scenario: "code-facts", Method: declaration.method,
			Handler: "unknown", State: "CONNECT_ROUTE_UNKNOWN", Transport: "connect",
			Declaration: EndpointSource{Path: "api/handlers/facts/module.go", Line: declaration.line, SHA256: moduleHash, Basis: "EndpointDescriptor Path references generated Connect Procedure constant"},
			Limits:      []string{"This is Connect/protobuf source anatomy; it does not establish HTTP JSON behavior.", "Runtime Module.Mount invocation, router reachability, successful Connect request, and client compatibility remain unverified."},
		}
		if found {
			row.RoutePath = procedure.path
			row.Title = declaration.method + " " + procedure.path
			row.Handler = procedure.service
			row.ConnectService = procedure.service
			row.ConnectMethod = procedure.method
			row.SourcePath = procedure.source.Path
			row.SourceSHA256 = procedure.source.SHA256
			row.HandlerLine = procedure.source.Line
			row.ConnectChain = append(row.ConnectChain, procedure.source)
			row.ConnectChain = append(row.ConnectChain, procedure.methodRegistration)
			row.State = "CONNECT_MOUNT_PARTIAL"
		} else {
			row.RoutePath = declaration.pathExpr
			if supportedSymbol {
				row.Limits = append(row.Limits, "Generated Connect Procedure constant was absent or ambiguous in the bounded imported package source.")
			} else {
				row.Limits = append(row.Limits, "Endpoint path expression is outside the supported static Connect Procedure constant form.")
			}
		}
		if found {
			mounts := findConnectMounts(module, fset, moduleHash, alias, procedure.service)
			switch len(mounts) {
			case 1:
				mount := mounts[0]
				row.State = "CONNECT_STATIC_MOUNT_REGISTERED"
				row.Registration = mount.registration
				row.ConnectChain = append(row.ConnectChain, mount.constructor, mount.closure, mount.mount, mount.registration)
				row.Limits = append(row.Limits, "The static Mount closure contains this service registration; outer app invocation/runtime route mounting is not established.")
			case 0:
				row.Limits = append(row.Limits, "No supported direct service-handler constructor to connectx.ServiceMount to connectx.RegisterServices chain was proven.")
			default:
				row.State = "CONNECT_MOUNT_UNKNOWN"
				row.Limits = append(row.Limits, "Multiple matching service mount chains make the static registration ambiguous.")
			}
		}
		haystack := strings.ToLower(strings.Join([]string{row.ID, row.Title, row.Summary, row.Handler, row.State, declaration.pathExpr}, " "))
		row.Score = tokenScore(terms, haystack)
		if row.Score > 0 {
			results = append(results, row)
			if row.State != "CONNECT_STATIC_MOUNT_REGISTERED" {
				degradations = append(degradations, "connect_mount_partial:"+declaration.id)
			}
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	return results, degradations
}

func splitSelectorName(expr string) (string, string, bool) {
	parts := strings.Split(expr, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func resolveConnectProcedure(root, importPath, symbol string) (connectProcedureSource, bool) {
	const marker = "/packages/proto/gen/go/"
	index := strings.Index(importPath, marker)
	if index < 0 {
		return connectProcedureSource{}, false
	}
	packageSuffix := importPath[index+len(marker):]
	for _, part := range strings.Split(packageSuffix, "/") {
		if part == "" || part == "." || part == ".." {
			return connectProcedureSource{}, false
		}
	}
	repoRoot := filepath.Clean(filepath.Join(root, "..", ".."))
	packageDir := filepath.Join(repoRoot, "packages", "proto", "gen", "go", filepath.FromSlash(packageSuffix))
	dirInfo, err := os.Lstat(packageDir)
	if err != nil || !dirInfo.IsDir() || dirInfo.Mode()&os.ModeSymlink != 0 {
		return connectProcedureSource{}, false
	}
	entries, err := os.ReadDir(packageDir)
	if err != nil || len(entries) > 160 {
		return connectProcedureSource{}, false
	}
	var found []connectProcedureSource
	var total int64
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(entry.Name(), ".connect.go") {
			continue
		}
		path := filepath.Join(packageDir, entry.Name())
		info, err := entry.Info()
		if err != nil || info.Size() > 2<<20 || total+info.Size() > 8<<20 {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil || int64(len(data)) != info.Size() {
			continue
		}
		total += info.Size()
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, data, parser.AllErrors)
		if err != nil {
			continue
		}
		var generatedService string
		for _, decl := range file.Decls {
			if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name.Name == "New"+serviceNameFromProcedure(symbol)+"Handler" && fn.Body != nil {
				generatedService = fn.Name.Name
			}
		}
		var methodRegistrationLines []int
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name != generatedService || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || selector.Sel.Name != "NewUnaryHandler" {
					return true
				}
				if isIdentExpr(call.Args[0], symbol) {
					methodRegistrationLines = append(methodRegistrationLines, fset.Position(call.Pos()).Line)
				}
				return true
			})
		}
		if len(methodRegistrationLines) != 1 {
			continue
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				values, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, name := range values.Names {
					if name.Name != symbol || i >= len(values.Values) {
						continue
					}
					value, ok := values.Values[i].(*ast.BasicLit)
					if !ok || value.Kind != token.STRING {
						continue
					}
					route, err := strconv.Unquote(value.Value)
					if err != nil {
						continue
					}
					service, method, ok := parseConnectRoute(route)
					if !ok {
						continue
					}
					rel, _ := filepath.Rel(repoRoot, path)
					source := EndpointSource{Path: filepath.ToSlash(rel), Line: fset.Position(value.Pos()).Line, SHA256: hashBytes(data), Basis: "generated Connect Procedure path constant"}
					methodSource := EndpointSource{Path: filepath.ToSlash(rel), Line: methodRegistrationLines[0], SHA256: hashBytes(data), Basis: "generated service constructor registers this Procedure with NewUnaryHandler"}
					found = append(found, connectProcedureSource{path: route, service: service, method: method, source: source, methodRegistration: methodSource})
				}
			}
		}
	}
	if len(found) != 1 {
		return connectProcedureSource{}, false
	}
	return found[0], true
}

func serviceNameFromProcedure(symbol string) string {
	const suffix = "Procedure"
	without := strings.TrimSuffix(symbol, suffix)
	index := strings.Index(without, "Service")
	if index < 0 {
		return ""
	}
	return without[:index+len("Service")]
}

func parseConnectRoute(route string) (string, string, bool) {
	if !strings.HasPrefix(route, "/") {
		return "", "", false
	}
	parts := strings.Split(strings.TrimPrefix(route, "/"), "/")
	if len(parts) != 2 {
		return "", "", false
	}
	serviceParts := strings.Split(parts[0], ".")
	service := serviceParts[len(serviceParts)-1]
	if service == "" || parts[1] == "" {
		return "", "", false
	}
	return service, parts[1], true
}

func findConnectMounts(file *ast.File, fset *token.FileSet, moduleHash, alias, service string) []connectMountSource {
	var mounts []connectMountSource
	ast.Inspect(file, func(node ast.Node) bool {
		fn, ok := node.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "Module" || fn.Body == nil {
			return true
		}
		var mountBody *ast.BlockStmt
		var closureSource EndpointSource
		ast.Inspect(fn.Body, func(child ast.Node) bool {
			lit, ok := child.(*ast.CompositeLit)
			if !ok {
				return true
			}
			typ, ok := lit.Type.(*ast.SelectorExpr)
			if !ok || typ.Sel.Name != "Module" {
				return true
			}
			for _, elt := range lit.Elts {
				kv, ok := elt.(*ast.KeyValueExpr)
				if !ok || !isIdentExpr(kv.Key, "Mount") {
					continue
				}
				if mount, ok := kv.Value.(*ast.FuncLit); ok {
					mountBody = mount.Body
					closureSource = EndpointSource{Path: "api/handlers/facts/module.go", Line: fset.Position(kv.Pos()).Line, SHA256: moduleHash, Basis: "module.Module returned with this Mount closure"}
				}
			}
			return true
		})
		if mountBody == nil {
			return false
		}
		stmts := mountBody.List
		for i, stmt := range stmts {
			assign, ok := stmt.(*ast.AssignStmt)
			if !ok || len(assign.Lhs) < 2 || len(assign.Rhs) != 1 {
				continue
			}
			call, ok := assign.Rhs[0].(*ast.CallExpr)
			if !ok {
				continue
			}
			ctor, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !isIdentExpr(ctor.X, alias) || ctor.Sel.Name != "New"+service+"Handler" {
				continue
			}
			pathVar, pathOK := assign.Lhs[0].(*ast.Ident)
			handlerVar, handlerOK := assign.Lhs[1].(*ast.Ident)
			if !pathOK || !handlerOK {
				continue
			}
			for j := i + 1; j < len(stmts); j++ {
				if reassignsConnectPair(stmts[j], pathVar.Name, handlerVar.Name) {
					break
				}
				registration, ok := registerServicesStatement(stmts[j], fset, moduleHash, pathVar.Name, handlerVar.Name)
				if ok {
					line := fset.Position(assign.Pos()).Line
					constructor := EndpointSource{Path: "api/handlers/facts/module.go", Line: line, SHA256: moduleHash, Basis: "generated service handler constructor returns path/handler pair"}
					mountSource := registration.mount
					mounts = append(mounts, connectMountSource{alias: alias, service: service, pathVar: pathVar.Name, handlerVar: handlerVar.Name, constructor: constructor, closure: closureSource, mount: mountSource, registration: registration.registration})
				}
			}
		}
		return false
	})
	return mounts
}

type registrationSources struct{ mount, registration EndpointSource }

func registerServicesStatement(stmt ast.Stmt, fset *token.FileSet, moduleHash, pathVar, handlerVar string) (registrationSources, bool) {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return registrationSources{}, false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return registrationSources{}, false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "RegisterServices" || len(call.Args) != 2 {
		return registrationSources{}, false
	}
	pkg, ok := selector.X.(*ast.Ident)
	if !ok || pkg.Name != "connectx" {
		return registrationSources{}, false
	}
	mountLit, ok := call.Args[1].(*ast.CompositeLit)
	if !ok {
		return registrationSources{}, false
	}
	mountType, ok := mountLit.Type.(*ast.SelectorExpr)
	if !ok || mountType.Sel.Name != "ServiceMount" || !isIdentExpr(mountType.X, "connectx") {
		return registrationSources{}, false
	}
	pathFound, handlerFound := false, false
	for _, elt := range mountLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return registrationSources{}, false
		}
		switch {
		case isIdentExpr(kv.Key, "Path"):
			pathFound = isIdentExpr(kv.Value, pathVar)
		case isIdentExpr(kv.Key, "Handler"):
			handlerFound = isIdentExpr(kv.Value, handlerVar)
		}
	}
	if !pathFound || !handlerFound {
		return registrationSources{}, false
	}
	return registrationSources{
		mount:        EndpointSource{Path: "api/handlers/facts/module.go", Line: fset.Position(mountLit.Pos()).Line, SHA256: moduleHash, Basis: "explicit connectx.ServiceMount Path and Handler fields"},
		registration: EndpointSource{Path: "api/handlers/facts/module.go", Line: fset.Position(call.Pos()).Line, SHA256: moduleHash, Basis: "connectx.RegisterServices call in Module.Mount closure"},
	}, true
}

func isIdentExpr(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func reassignsConnectPair(stmt ast.Stmt, pathVar, handlerVar string) bool {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok {
		return false
	}
	for _, lhs := range assign.Lhs {
		if isIdentExpr(lhs, pathVar) || isIdentExpr(lhs, handlerVar) {
			return true
		}
	}
	return false
}

func expressionName(expr ast.Expr) string {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name + "." + selector.Sel.Name
}
