package rules

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	CodeHealthCheckMalformed      = "HEALTH_CHECK_MALFORMED"
	CodeHealthEndpointMissing     = "HEALTH_ENDPOINT_MISSING"
	CodeAPIPreflightMissing       = "API_PREFLIGHT_MISSING"
	CodeAPIPreflightMalformed     = "API_PREFLIGHT_MALFORMED"
	CodeAPIServerRunNonconformant = "API_SERVER_RUN_NONCONFORMANT"
)

const canonicalAPIHealthTarget = "http://localhost:${API_PORT}/health"

func apiEntrypointRules(in Input) []Finding {
	if !apiDeclared(in.Model) {
		return nil
	}
	var out []Finding
	out = append(out, apiLifecycleHealthRules(in)...)

	mainPath := filepath.Join(in.Model.RootPath, "api", "main.go")
	mainFile, err := parseGoFile(mainPath)
	if err == nil {
		out = append(out, preflightRules(in, mainPath, mainFile)...)
		out = append(out, serverRunRules(in.Model.RootPath, mainPath, mainFile)...)
	}
	if !apiRootHealthEndpointExists(in.Model.RootPath) {
		out = append(out, Finding{
			Code:        CodeHealthEndpointMissing,
			Severity:    sevError,
			Title:       "API does not expose root /health",
			Message:     "The API lifecycle health check targets /health, but no root-level /health route registration was found in first-party API Go sources.",
			Location:    "api/",
			Remediation: "Mount a root /health route on the production API router. Keep versioned or prefixed health routes only as additional aliases.",
			Surface:     "api",
		})
	}
	return out
}

func apiLifecycleHealthRules(in Input) []Finding {
	health := in.Model.Intent.Lifecycle.Health
	if len(health.Checks) == 0 {
		return []Finding{{
			Code:        "HEALTH_CHECK_MISSING",
			Severity:    sevError,
			Title:       "no API lifecycle health check",
			Message:     "The scenario declares an API surface but no lifecycle.health check; the lifecycle cannot confirm readiness.",
			Location:    ".vrooli/service.json",
			Remediation: "Add a critical lifecycle.health http check targeting " + canonicalAPIHealthTarget + ".",
			Surface:     "api",
		}}
	}
	for _, c := range health.Checks {
		if !strings.EqualFold(strings.TrimSpace(c.Type), "http") {
			continue
		}
		if isCanonicalHealthTarget(c.Target) && c.Critical {
			return nil
		}
		return []Finding{{
			Code:        CodeHealthCheckMalformed,
			Severity:    sevError,
			Title:       "API lifecycle health check is malformed",
			Message:     "The API lifecycle health check must be a critical HTTP check targeting " + canonicalAPIHealthTarget + ".",
			Location:    ".vrooli/service.json",
			Remediation: "Normalize the API lifecycle health check to target " + canonicalAPIHealthTarget + " with critical=true.",
			Surface:     "api",
		}}
	}
	return []Finding{{
		Code:        "HEALTH_CHECK_MISSING",
		Severity:    sevError,
		Title:       "no API http health check",
		Message:     "The scenario declares an API surface but lifecycle.health has no HTTP check; the lifecycle cannot confirm readiness.",
		Location:    ".vrooli/service.json",
		Remediation: "Add a critical lifecycle.health http check targeting " + canonicalAPIHealthTarget + ".",
		Surface:     "api",
	}}
}

func preflightRules(in Input, mainPath string, parsed *parsedGoFile) []Finding {
	mainFn := findMainFunc(parsed.file)
	if mainFn == nil || mainFn.Body == nil {
		return nil
	}
	alias, imported := importAlias(parsed.file, "github.com/vrooli/api-core/preflight", "preflight")
	info := analyzePreflight(parsed.fset, mainFn, alias)
	location := relPath(in.Model.RootPath, mainPath)
	scenario := firstNonEmpty(in.Model.Intent.Name, in.Model.Scenario)
	switch {
	case !imported || !info.found:
		return []Finding{{
			Code:        CodeAPIPreflightMissing,
			Severity:    sevError,
			Title:       "API main does not run lifecycle preflight",
			Message:     "api/main.go must run api-core preflight before opening listeners or initializing service state.",
			Location:    location,
			Remediation: "Import github.com/vrooli/api-core/preflight and call if preflight.Run(preflight.Config{ScenarioName: \"" + scenario + "\"}) { return } at the start of main.",
			Surface:     "api",
		}}
	case !info.validRunShape:
		return []Finding{{
			Code:        CodeAPIPreflightMalformed,
			Severity:    sevError,
			Title:       "preflight.Run result is not handled",
			Message:     "preflight.Run returns true after managing a stale-source re-exec; main must return immediately in that branch.",
			Location:    location + ":" + strconv.Itoa(info.line),
			Remediation: "Wrap preflight.Run in an if statement whose body returns, or use preflight.MustRun.",
			Surface:     "api",
		}}
	case info.businessBefore:
		return []Finding{{
			Code:        CodeAPIPreflightMalformed,
			Severity:    sevError,
			Title:       "lifecycle preflight runs too late",
			Message:     "main executes service setup before preflight; stale-source re-exec must happen before listeners, storage, or dependency setup.",
			Location:    location + ":" + strconv.Itoa(info.line),
			Remediation: "Move the preflight call to the start of main, before service initialization. Logging flag setup may remain before it.",
			Surface:     "api",
		}}
	case info.scenarioName == "":
		return []Finding{{
			Code:        CodeAPIPreflightMalformed,
			Severity:    sevError,
			Title:       "preflight ScenarioName is missing",
			Message:     "preflight.Config must include ScenarioName so lifecycle errors identify the target scenario.",
			Location:    location + ":" + strconv.Itoa(info.line),
			Remediation: "Set preflight.Config{ScenarioName: \"" + scenario + "\"}.",
			Surface:     "api",
		}}
	case scenario != "" && info.scenarioName != scenario:
		return []Finding{{
			Code:        CodeAPIPreflightMalformed,
			Severity:    sevError,
			Title:       "preflight ScenarioName does not match service.name",
			Message:     "preflight ScenarioName is " + info.scenarioName + " but the scenario identity is " + scenario + ".",
			Location:    location + ":" + strconv.Itoa(info.line),
			Remediation: "Set preflight.Config{ScenarioName: \"" + scenario + "\"}.",
			Surface:     "api",
		}}
	default:
		return nil
	}
}

func serverRunRules(root, mainPath string, parsed *parsedGoFile) []Finding {
	alias, imported := importAlias(parsed.file, "github.com/vrooli/api-core/server", "server")
	hasServerRun := imported && fileCallsSelector(parsed.file, alias, "Run")
	if hasServerRun {
		return nil
	}
	if !fileHasListenAndServe(parsed.file) {
		return nil
	}
	return []Finding{{
		Code:        CodeAPIServerRunNonconformant,
		Severity:    sevError,
		Title:       "API server bypasses api-core/server.Run",
		Message:     "api/main.go starts an HTTP listener directly instead of using api-core/server.Run, so lifecycle-managed API_PORT binding and graceful shutdown may drift.",
		Location:    relPath(root, mainPath),
		Remediation: "Run the production handler with github.com/vrooli/api-core/server.Run(server.Config{Handler: ...}) instead of direct ListenAndServe calls.",
		Surface:     "api",
	}}
}

type parsedGoFile struct {
	fset *token.FileSet
	file *ast.File
}

func parseGoFile(path string) (*parsedGoFile, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	return &parsedGoFile{fset: fset, file: file}, nil
}

func apiRootHealthEndpointExists(root string) bool {
	apiDir := filepath.Join(root, "api")
	found := false
	_ = filepath.WalkDir(apiDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "node_modules", ".git":
				return filepath.SkipDir
			default:
				return nil
			}
		}
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		parsed, err := parseGoFile(path)
		if err != nil {
			return nil
		}
		if fileRegistersRootHealth(parsed.file) {
			found = true
		}
		return nil
	})
	return found
}

func fileRegistersRootHealth(file *ast.File) bool {
	prefixed := prefixedRouterVars(file)
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 || stringLiteral(call.Args[0]) != "/health" {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !isRouteRegistration(sel.Sel.Name) {
			return true
		}
		if selectorReceiverName(sel.X) == "http" {
			found = true
			return false
		}
		if receiverContainsPrefixedRouter(sel.X) {
			return true
		}
		recv := selectorReceiverName(sel.X)
		if recv == "" || prefixed[recv] {
			return true
		}
		found = true
		return false
	})
	return found
}

func prefixedRouterVars(file *ast.File) map[string]bool {
	out := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok {
			return true
		}
		for i, rhs := range assign.Rhs {
			call, ok := rhs.(*ast.CallExpr)
			if !ok || !callBuildsPrefixedRouter(call) || i >= len(assign.Lhs) {
				continue
			}
			if ident, ok := assign.Lhs[i].(*ast.Ident); ok {
				out[ident.Name] = true
			}
		}
		return true
	})
	return out
}

func callBuildsPrefixedRouter(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		switch sel.Sel.Name {
		case "Group", "Route":
			return len(call.Args) > 0 && strings.HasPrefix(stringLiteral(call.Args[0]), "/")
		case "Subrouter":
			return receiverContainsPrefixedRouter(sel.X)
		}
	}
	return false
}

func receiverContainsPrefixedRouter(expr ast.Expr) bool {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name == "PathPrefix" && len(call.Args) > 0 && strings.HasPrefix(stringLiteral(call.Args[0]), "/") {
		return true
	}
	return receiverContainsPrefixedRouter(sel.X)
}

func isRouteRegistration(name string) bool {
	switch name {
	case "Handle", "HandleFunc", "Get", "GET", "Post", "POST", "Put", "PUT", "Delete", "DELETE", "Patch", "PATCH", "Head", "HEAD", "Options", "OPTIONS":
		return true
	default:
		return false
	}
}

type preflightInfo struct {
	found          bool
	validRunShape  bool
	businessBefore bool
	scenarioName   string
	line           int
}

func analyzePreflight(fset *token.FileSet, mainFn *ast.FuncDecl, alias string) preflightInfo {
	info := preflightInfo{validRunShape: true}
	for i, stmt := range mainFn.Body.List {
		if call, ok := preflightRunFromIf(stmt, alias); ok {
			info.found = true
			info.validRunShape = blockHasTopLevelReturn(stmt.(*ast.IfStmt).Body)
			info.scenarioName = preflightScenarioName(call)
			info.line = fset.Position(stmt.Pos()).Line
			info.businessBefore = hasBusinessBefore(mainFn.Body.List[:i])
			return info
		}
		if call, mustRun, ok := preflightCallFromExpr(stmt, alias); ok {
			info.found = true
			info.validRunShape = mustRun
			info.scenarioName = preflightScenarioName(call)
			info.line = fset.Position(stmt.Pos()).Line
			info.businessBefore = hasBusinessBefore(mainFn.Body.List[:i])
			return info
		}
	}
	return info
}

func preflightRunFromIf(stmt ast.Stmt, alias string) (*ast.CallExpr, bool) {
	ifs, ok := stmt.(*ast.IfStmt)
	if !ok {
		return nil, false
	}
	call, ok := ifs.Cond.(*ast.CallExpr)
	if !ok || !isSelectorCall(call, alias, "Run") {
		return nil, false
	}
	return call, true
}

func preflightCallFromExpr(stmt ast.Stmt, alias string) (*ast.CallExpr, bool, bool) {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return nil, false, false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return nil, false, false
	}
	switch {
	case isSelectorCall(call, alias, "MustRun"):
		return call, true, true
	case isSelectorCall(call, alias, "Run"):
		return call, false, true
	default:
		return nil, false, false
	}
}

func preflightScenarioName(call *ast.CallExpr) string {
	if len(call.Args) == 0 {
		return ""
	}
	lit, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return ""
	}
	for _, elt := range lit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		if ident, ok := kv.Key.(*ast.Ident); !ok || ident.Name != "ScenarioName" {
			continue
		}
		return stringLiteral(kv.Value)
	}
	return ""
}

func hasBusinessBefore(stmts []ast.Stmt) bool {
	for _, stmt := range stmts {
		if !isHarmlessBeforePreflight(stmt) {
			return true
		}
	}
	return false
}

func isHarmlessBeforePreflight(stmt ast.Stmt) bool {
	expr, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := expr.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	return isSelectorCall(call, "log", "SetFlags") ||
		isSelectorCall(call, "log", "SetOutput") ||
		isSelectorCall(call, "log", "SetPrefix")
}

func blockHasTopLevelReturn(block *ast.BlockStmt) bool {
	if block == nil {
		return false
	}
	for _, stmt := range block.List {
		if _, ok := stmt.(*ast.ReturnStmt); ok {
			return true
		}
	}
	return false
}

func fileCallsSelector(file *ast.File, receiver, method string) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if ok && isSelectorCall(call, receiver, method) {
			found = true
			return false
		}
		return true
	})
	return found
}

func fileHasListenAndServe(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "ListenAndServe" || sel.Sel.Name == "ListenAndServeTLS" {
			found = true
			return false
		}
		return true
	})
	return found
}

func importAlias(file *ast.File, importPath, defaultName string) (string, bool) {
	for _, imp := range file.Imports {
		if strings.Trim(imp.Path.Value, `"`) != importPath {
			continue
		}
		if imp.Name != nil && imp.Name.Name != "" && imp.Name.Name != "." && imp.Name.Name != "_" {
			return imp.Name.Name, true
		}
		return defaultName, true
	}
	return defaultName, false
}

func isSelectorCall(call *ast.CallExpr, receiver, method string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != method {
		return false
	}
	return selectorReceiverName(sel.X) == receiver
}

func selectorReceiverName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		return selectorReceiverName(x.X)
	default:
		return ""
	}
}

func stringLiteral(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	v, err := strconv.Unquote(lit.Value)
	if err != nil {
		return strings.Trim(lit.Value, `"`)
	}
	return v
}

func findMainFunc(file *ast.File) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "main" {
			return fn
		}
	}
	return nil
}

func isCanonicalHealthTarget(target string) bool {
	target = strings.TrimSpace(target)
	if target == canonicalAPIHealthTarget {
		return true
	}
	return strings.HasSuffix(target, "${API_PORT}/health") || strings.HasSuffix(target, ":${API_PORT}/health")
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}
