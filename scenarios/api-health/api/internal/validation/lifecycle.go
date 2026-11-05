package validation

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

type LifecycleResult struct {
	MainPath             string
	MainReadable         bool
	ManifestHealthy      bool
	PreflightHealthy     bool
	ServerRunnerHealthy  bool
	DirectListenAndServe bool
	Diagnostics          []string
}

func validateLifecycle(target *Target) []Finding {
	var findings []Finding
	findings = append(findings, validateServiceHealthMetadata(target)...)

	mainPath := filepath.Join(target.APIDir, "main.go")
	target.Lifecycle.MainPath = mainPath
	file, fset, err := parseGoFile(mainPath)
	if err != nil {
		target.Lifecycle.Diagnostics = append(target.Lifecycle.Diagnostics, err.Error())
		if target.HasAPIDir {
			findings = append(findings, Finding{
				Severity:    SeverityError,
				Code:        CodePreflightMissingLate,
				Title:       "API main entrypoint unreadable",
				Location:    mainPath,
				Message:     "api/main.go could not be parsed for lifecycle preflight validation",
				Remediation: "provide a readable Go API entrypoint with api-core preflight wiring",
			})
		}
		return findings
	}
	target.Lifecycle.MainReadable = true

	if finding := validatePreflight(target, file, fset, mainPath); finding != nil {
		findings = append(findings, *finding)
	} else {
		target.Lifecycle.PreflightHealthy = true
	}
	if finding := validateServerRunner(target, file, fset, mainPath); finding != nil {
		target.Lifecycle.DirectListenAndServe = true
		findings = append(findings, *finding)
	} else {
		target.Lifecycle.ServerRunnerHealthy = true
	}
	return findings
}

func validateServiceHealthMetadata(target *Target) []Finding {
	var missing []string
	if !target.Service.PortsAPI {
		missing = append(missing, "ports.api")
	}
	if target.Service.HealthAPIPath != "/health" {
		missing = append(missing, "lifecycle.health.endpoints.api=/health")
	}
	if !target.Service.HealthAPICheck || !strings.Contains(target.Service.HealthAPICheckURL, "/health") {
		missing = append(missing, "lifecycle.health.checks http target for ${API_PORT}/health")
	}
	if len(missing) == 0 {
		target.Lifecycle.ManifestHealthy = true
		return nil
	}
	return []Finding{{
		Severity:    SeverityError,
		Code:        CodeServiceHealthMissing,
		Title:       "API lifecycle health metadata incomplete",
		Location:    nonEmpty(target.ServiceManifestPath, filepath.Join(target.RootPath, ".vrooli", "service.json")),
		Message:     "service manifest is missing required API lifecycle metadata: " + strings.Join(missing, ", "),
		Remediation: "declare ports.api, lifecycle.health.endpoints.api as /health, and an HTTP check targeting ${API_PORT}/health",
	}}
}

func parseGoFile(path string) (*ast.File, *token.FileSet, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, nil, err
	}
	return file, fset, nil
}

func validatePreflight(target *Target, file *ast.File, fset *token.FileSet, mainPath string) *Finding {
	mainFunc := findMain(file)
	if mainFunc == nil || mainFunc.Body == nil {
		return lifecycleFinding(CodePreflightMissingLate, "API preflight missing", mainPath, 1, "api/main.go has no inspectable main function", "call preflight.Run or preflight.MustRun before API initialization")
	}

	imports := importAliases(file, "github.com/vrooli/api-core/preflight")
	osImports := importAliases(file, "os")
	values := packageFunctionValues(file)
	info, firstBusinessLogic := inspectPreflightBody(mainFunc, imports, osImports, values, fset)
	if !info.found {
		// Some APIs keep main deliberately small and put startup in one same-file
		// helper. Follow only the first non-harmless call from main; this is a
		// structural startup check, not a general call-graph analysis.
		if firstBusinessLogic >= 0 {
			helper := initialLocalHelper(mainFunc.Body.List[firstBusinessLogic], values)
			if helper != nil && helper.Body != nil {
				info, firstBusinessLogic = inspectPreflightBody(helper, imports, osImports, values, fset)
			}
		}
	}

	expected := target.Scenario
	if !info.found {
		return lifecycleFinding(CodePreflightMissingLate, "API preflight missing", mainPath, fset.Position(mainFunc.Pos()).Line, "main does not call api-core preflight before startup", "add preflight.Run or preflight.MustRun with ScenarioName "+strconv.Quote(expected))
	}
	if firstBusinessLogic != -1 && firstBusinessLogic < info.position {
		return lifecycleFinding(CodePreflightMissingLate, "API preflight runs after startup work", mainPath, info.line, "business logic executes before api-core preflight", "move preflight to the first non-harmless statement in main")
	}
	if !info.isMustRun && !info.hasIfWrapper {
		return lifecycleFinding(CodePreflightMissingLate, "API preflight Run result ignored", mainPath, info.line, "preflight.Run must be wrapped in if preflight.Run(...) { return }", "return immediately when preflight.Run reports that it handled rebuild/re-exec work")
	}
	if !info.isMustRun && !info.hasReturn {
		return lifecycleFinding(CodePreflightMissingLate, "API preflight does not return", mainPath, info.line, "preflight.Run branch does not return from main", "return immediately from the preflight.Run branch")
	}
	if info.scenarioName == "" {
		return lifecycleFinding(CodePreflightMissingLate, "API preflight ScenarioName missing", mainPath, info.line, "preflight config does not declare ScenarioName", "set preflight.Config.ScenarioName to "+strconv.Quote(expected))
	}
	if info.scenarioName != expected {
		return lifecycleFinding(CodePreflightMissingLate, "API preflight ScenarioName mismatch", mainPath, info.line, fmt.Sprintf("preflight ScenarioName %q does not match target scenario %q", info.scenarioName, expected), "set preflight.Config.ScenarioName to "+strconv.Quote(expected))
	}
	return nil
}

func inspectPreflightBody(fn *ast.FuncDecl, imports, osImports map[string]bool, values map[string]packageFunctionValue, fset *token.FileSet) (preflightInfo, int) {
	var info preflightInfo
	firstBusinessLogic := -1
	if fn == nil || fn.Body == nil {
		return info, firstBusinessLogic
	}
	for i, stmt := range fn.Body.List {
		if ifStmt, ok := stmt.(*ast.IfStmt); ok {
			if call := preflightRunCall(ifStmt.Cond, imports, values); call != nil {
				info = preflightInfo{
					found:        true,
					position:     i,
					hasIfWrapper: true,
					hasReturn:    blockHasReturn(ifStmt.Body),
					scenarioName: scenarioNameFromPreflight(call),
					line:         fset.Position(ifStmt.Pos()).Line,
				}
				break
			}
		}
		if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
			if call, isMustRun := preflightCall(exprStmt.X, imports, values); call != nil {
				info = preflightInfo{
					found:        true,
					position:     i,
					isMustRun:    isMustRun,
					scenarioName: scenarioNameFromPreflight(call),
					line:         fset.Position(exprStmt.Pos()).Line,
				}
				break
			}
		}
		if firstBusinessLogic == -1 && !harmlessStartupStatement(stmt) && !harmlessPreflightConfigRead(stmt, osImports) {
			firstBusinessLogic = i
		}
	}
	return info, firstBusinessLogic
}

type preflightInfo struct {
	found        bool
	position     int
	isMustRun    bool
	hasIfWrapper bool
	hasReturn    bool
	scenarioName string
	line         int
}

func validateServerRunner(_ *Target, file *ast.File, fset *token.FileSet, mainPath string) *Finding {
	serverImports := importAliases(file, "github.com/vrooli/api-core/server")
	values := packageFunctionValues(file)
	usesServerRun := false
	var directLine int
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if resolvesImportedFunction(call.Fun, serverImports, "Run", values) {
				usesServerRun = true
				return true
			}
			if isListenAndServeCall(call) && directLine == 0 {
				directLine = fset.Position(call.Pos()).Line
			}
			return true
		})
	}
	// StartServer is the sanctioned api-core extension point. A callback may
	// legitimately call ListenAndServe (for example to configure h2c), so the
	// intent to use server.Run must win over an implementation detail inside it.
	if !usesServerRun && directLine != 0 {
		return lifecycleFinding(CodeServerRunnerMissing, "Direct ListenAndServe bypasses api-core server", mainPath, directLine, "API entrypoint starts an HTTP server directly instead of using api-core/server.Run", "use github.com/vrooli/api-core/server.Run so shutdown, signals, and cleanup are consistent")
	}
	if len(serverImports) > 0 && !usesServerRun {
		return lifecycleFinding(CodeServerRunnerMissing, "api-core server runner not called", mainPath, 1, "api-core/server is imported but server.Run is not called", "call server.Run with the API handler")
	}
	return nil
}

func findMain(file *ast.File) *ast.FuncDecl {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "main" {
			return fn
		}
	}
	return nil
}

func importAliases(file *ast.File, importPath string) map[string]bool {
	aliases := map[string]bool{}
	for _, imp := range file.Imports {
		if imp.Path == nil || strings.Trim(imp.Path.Value, `"`) != importPath {
			continue
		}
		switch {
		case imp.Name == nil:
			aliases[filepath.Base(importPath)] = true
		case imp.Name.Name != "." && imp.Name.Name != "_":
			aliases[imp.Name.Name] = true
		}
	}
	return aliases
}

func preflightRunCall(expr ast.Expr, imports map[string]bool, values map[string]packageFunctionValue) *ast.CallExpr {
	call, ok := expr.(*ast.CallExpr)
	if !ok || !resolvesImportedFunction(call.Fun, imports, "Run", values) {
		return nil
	}
	return call
}

func preflightCall(expr ast.Expr, imports map[string]bool, values map[string]packageFunctionValue) (*ast.CallExpr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	if resolvesImportedFunction(call.Fun, imports, "MustRun", values) {
		return call, true
	}
	if resolvesImportedFunction(call.Fun, imports, "Run", values) {
		return call, false
	}
	return nil, false
}

// packageFunctionValues resolves only immutable-looking package-level function
// values declared directly from an identifier or imported function selector.
// Package aliases assigned elsewhere in this file are excluded so a mutable
// alias cannot satisfy the lifecycle contract by name alone.
type packageFunctionValue struct {
	expr        ast.Expr
	declaration *ast.ValueSpec
}

func packageFunctionValues(file *ast.File) map[string]packageFunctionValue {
	values := map[string]packageFunctionValue{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 {
				continue
			}
			name := valueSpec.Names[0].Name
			if name == "_" {
				continue
			}
			values[name] = packageFunctionValue{expr: valueSpec.Values[0], declaration: valueSpec}
		}
	}
	assigned := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			if stmt.Tok == token.DEFINE {
				return true
			}
			for _, lhs := range stmt.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok {
					if value, exists := values[ident.Name]; exists && ident.Obj != nil && ident.Obj.Decl == value.declaration {
						assigned[ident.Name] = true
					}
				}
			}
		case *ast.IncDecStmt:
			if ident, ok := stmt.X.(*ast.Ident); ok {
				if value, exists := values[ident.Name]; exists && ident.Obj != nil && ident.Obj.Decl == value.declaration {
					assigned[ident.Name] = true
				}
			}
		}
		return true
	})
	for name := range assigned {
		delete(values, name)
	}
	return values
}

func resolvesImportedFunction(expr ast.Expr, imports map[string]bool, method string, values map[string]packageFunctionValue) bool {
	if selector, ok := expr.(*ast.SelectorExpr); ok {
		base, ok := selector.X.(*ast.Ident)
		// The parser leaves imported package names unresolved, while local
		// shadowing receives an Obj binding. Require the former.
		return ok && base.Obj == nil && selector.Sel.Name == method && imports[base.Name]
	}
	ident, ok := expr.(*ast.Ident)
	if !ok || ident.Obj == nil {
		return false
	}
	value, ok := values[ident.Name]
	if !ok || ident.Obj.Decl != value.declaration {
		return false
	}
	return selectorUsesAlias(value.expr, imports, method)
}

func initialLocalHelper(stmt ast.Stmt, values map[string]packageFunctionValue) *ast.FuncDecl {
	var call *ast.CallExpr
	switch statement := stmt.(type) {
	case *ast.ExprStmt:
		call, _ = statement.X.(*ast.CallExpr)
	case *ast.AssignStmt:
		if len(statement.Rhs) == 1 {
			call, _ = statement.Rhs[0].(*ast.CallExpr)
		}
	case *ast.IfStmt:
		if assignment, ok := statement.Init.(*ast.AssignStmt); ok && len(assignment.Rhs) == 1 {
			call, _ = assignment.Rhs[0].(*ast.CallExpr)
		}
	}
	if call == nil || len(call.Args) != 0 {
		return nil
	}
	name, ok := call.Fun.(*ast.Ident)
	if !ok {
		return nil
	}
	return resolveLocalFunction(name, values, map[string]bool{})
}

func resolveLocalFunction(identifier *ast.Ident, values map[string]packageFunctionValue, seen map[string]bool) *ast.FuncDecl {
	if identifier == nil || identifier.Obj == nil || seen[identifier.Name] {
		return nil
	}
	seen[identifier.Name] = true
	switch decl := identifier.Obj.Decl.(type) {
	case *ast.FuncDecl:
		if decl.Recv == nil {
			return decl
		}
	case *ast.ValueSpec:
		value, ok := values[identifier.Name]
		if !ok || value.declaration != decl {
			return nil
		}
		target, ok := value.expr.(*ast.Ident)
		if !ok {
			return nil
		}
		return resolveLocalFunction(target, values, seen)
	}
	return nil
}

func selectorUsesAlias(expr ast.Expr, aliases map[string]bool, method string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != method {
		return false
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && aliases[ident.Name]
}

func scenarioNameFromPreflight(call *ast.CallExpr) string {
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
		key, ok := kv.Key.(*ast.Ident)
		if !ok || key.Name != "ScenarioName" {
			continue
		}
		value, ok := kv.Value.(*ast.BasicLit)
		if ok && value.Kind == token.STRING {
			unquoted, err := strconv.Unquote(value.Value)
			if err == nil {
				return unquoted
			}
		}
	}
	return ""
}

func blockHasReturn(block *ast.BlockStmt) bool {
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

func harmlessStartupStatement(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.EmptyStmt:
		return true
	case *ast.ExprStmt:
		call, ok := s.X.(*ast.CallExpr)
		return ok && selectorChain(call.Fun) == "log.SetFlags"
	case *ast.AssignStmt:
		return false
	default:
		return false
	}
}

// harmlessPreflightConfigRead accepts only a local boolean declaration derived
// from a direct os.Getenv string comparison. Preflight Config values sometimes
// depend on a process-mode flag; reading that value does not initialize the API.
// Other assignments remain startup work, including calls that happen to return
// configuration-shaped values.
func harmlessPreflightConfigRead(stmt ast.Stmt, osImports map[string]bool) bool {
	assignment, ok := stmt.(*ast.AssignStmt)
	if !ok || assignment.Tok != token.DEFINE || len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return false
	}
	name, ok := assignment.Lhs[0].(*ast.Ident)
	if !ok || name.Name == "_" {
		return false
	}
	comparison, ok := assignment.Rhs[0].(*ast.BinaryExpr)
	if !ok || comparison.Op != token.EQL {
		return false
	}
	if _, ok := stringLiteral(comparison.Y); !ok {
		return false
	}
	call, ok := comparison.X.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	if _, ok := stringLiteral(call.Args[0]); !ok {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Getenv" {
		return false
	}
	base, ok := selector.X.(*ast.Ident)
	return ok && base.Obj == nil && osImports[base.Name]
}

func selectorChain(expr ast.Expr) string {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return ""
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok {
		return ""
	}
	return ident.Name + "." + sel.Sel.Name
}

func isListenAndServeCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if sel.Sel.Name != "ListenAndServe" && sel.Sel.Name != "ListenAndServeTLS" {
		return false
	}
	if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == "http" {
		return true
	}
	return true
}

func lifecycleFinding(code, title, path string, line int, message, remediation string) *Finding {
	location := path
	if line > 0 {
		location = fmt.Sprintf("%s:%d", path, line)
	}
	return &Finding{
		Severity:    SeverityError,
		Code:        code,
		Title:       title,
		Location:    location,
		Message:     message,
		Remediation: remediation,
	}
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
