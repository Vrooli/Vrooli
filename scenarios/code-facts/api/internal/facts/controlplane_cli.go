package facts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type cliCommandSpec struct {
	name, handler, group string
	argsExpression       string
	argsStatus           string
	argsPath             string
	argsHash             string
	hidden               bool
	line                 int
	argsLine             int
}

// cliArgsDeclaration records the literal Args field expression without
// evaluating helpers, defaults, or validation behavior.
func cliArgsDeclaration(fields []ast.Expr, fset *token.FileSet) (string, string, int) {
	var args []ast.Expr
	for _, field := range fields {
		kv, ok := field.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := kv.Key.(*ast.Ident)
		if ok && key.Name == "Args" {
			args = append(args, kv.Value)
		}
	}
	if len(args) == 0 {
		return "", "NOT_DECLARED", 0
	}
	if len(args) != 1 {
		return "", "UNKNOWN_DUPLICATE_ARGS", fset.Position(args[0].Pos()).Line
	}
	var expression strings.Builder
	if err := format.Node(&expression, fset, args[0]); err != nil {
		return "", "UNKNOWN_UNRENDERABLE_ARGS", fset.Position(args[0].Pos()).Line
	}
	return expression.String(), "DECLARED_UNEVALUATED", fset.Position(args[0].Pos()).Line
}

func cliArgsEvidence(spec cliCommandSpec) string {
	if spec.argsStatus == "NOT_DECLARED" {
		return fmt.Sprintf("Args field NOT_DECLARED in command literal at %s:%d#sha256:%s; defaults and acceptance behavior unknown", spec.argsPath, spec.line, spec.argsHash)
	}
	if spec.argsStatus != "DECLARED_UNEVALUATED" {
		return fmt.Sprintf("Args declaration %s at %s:%d#sha256:%s; declaration could not be resolved", spec.argsStatus, spec.argsPath, spec.argsLine, spec.argsHash)
	}
	return fmt.Sprintf("Args declaration DECLARED_UNEVALUATED (%s) at %s:%d#sha256:%s; runtime parsing and acceptance behavior unknown", spec.argsExpression, spec.argsPath, spec.argsLine, spec.argsHash)
}

// DescribeControlPlaneCLIComposition binds the root CLI's declared command
// specs to the handler map. It intentionally does not infer runtime reachability
// through special pre-run dispatch or evaluate argument-schema expressions.
func DescribeControlPlaneCLIComposition(root string) (*CompositionReport, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	topPath := filepath.Join(root, "internal/cli/topcli/topcli.go")
	appPath := filepath.Join(root, "internal/cli/vroolicli/app.go")
	top, topHash, err := parseCLIUnit(root, topPath)
	if err != nil {
		return nil, err
	}
	app, appHash, err := parseCLIUnit(root, appPath)
	if err != nil {
		return nil, err
	}
	main, mainHash, err := parseCLIUnit(root, filepath.Join(root, "cmd/vrooli/main.go"))
	if err != nil {
		return nil, err
	}
	constants := cliCommandConstants(top.file)
	specs := cliCommandSpecs(top.file, constants, top.fset)
	for i := range specs {
		specs[i].argsPath, specs[i].argsHash = top.path, topHash
	}
	handlers := cliHandlerBindings(app.file, app.fset)
	report := &CompositionReport{RootPath: root, AnalysisScope: "go-syntax-v1; top-level and supported nested command specs are joined to explicit source registrations (nested expansion follows only registered package RootHandler source calls); Args field expressions are reported with declaration status, source path/line/hash and are never evaluated; main conditional early-dispatch call sites are listed without resolving predicates; unsupported forms degrade to UNKNOWN; nested/runtime argument dispatch and execution are not analyzed"}
	for _, spec := range specs {
		handler := handlers[spec.handler]
		status, provider, registrationPath, sourceHash, line := "unknown", "", top.path, topHash, spec.line
		if handler.expression != "" {
			status, provider = "concrete", handler.expression
			registrationPath, sourceHash, line = app.path, appHash, handler.line
		}
		binding := CompositionBinding{
			Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: spec.name,
			Interface: fmt.Sprintf("commandtree.Spec group=%q handler=%s → rootcli.Handler", spec.group, spec.handler),
			Provider:  provider, ProviderExpression: fmt.Sprintf("declared handler id %s; group=%q; hidden=%t; %s", spec.handler, spec.group, spec.hidden, cliArgsEvidence(spec)),
			Status: status, SourcePath: registrationPath, Line: line, SourceHash: sourceHash,
			Provenance: "topcli-command-spec+vroolicli-handler-map; " + cliArgsEvidence(spec),
		}
		report.Bindings = append(report.Bindings, binding)
	}
	report.Bindings = append(report.Bindings, cliNestedGroupBindings(root, app, appHash, specs, handlers)...)
	if registryLine := cliCallLine(app.file, app.fset, "New", "rootcli", "NewRegistry"); registryLine > 0 {
		report.Bindings = append(report.Bindings, CompositionBinding{
			Environment: "control-plane", Consumer: "vrooli root CLI", Dependency: "command registry",
			Interface: "rootcli.Registry", Provider: "rootcli.NewRegistry",
			ProviderExpression: "NewRegistry(buildTopLevelHandlerMap(), buildScenarioHandlerMap())",
			Status:             "concrete", SourcePath: app.path, Line: registryLine, SourceHash: appHash,
			Provenance: "vroolicli-runner-registry-composition",
		})
	}
	if dispatchLine := cliCallLine(main.file, main.fset, "run", "configuredRunner", "Run"); dispatchLine > 0 {
		report.Bindings = append(report.Bindings, CompositionBinding{
			Environment: "control-plane", Consumer: "vrooli process", Dependency: "normal CLI dispatch",
			Interface: "rootcli.Runner", Provider: "configuredRunner().Run",
			ProviderExpression: "run(args) delegates to configuredRunner().Run(args, stdout, stderr)",
			Status:             "concrete", SourcePath: main.path, Line: dispatchLine, SourceHash: mainHash,
			Provenance: "main-run-configured-runner-call",
		})
	}
	for _, hook := range cliEarlyDispatchHooks(main.file, main.fset) {
		report.Bindings = append(report.Bindings, CompositionBinding{
			Environment: "control-plane", Consumer: "vrooli process", Dependency: "early-dispatch:" + hook.name,
			Interface: "conditional pre-run dispatch probe; argument predicate and handled outcome are unknown",
			Provider:  hook.name, ProviderExpression: "main conditionally invokes " + hook.name + " before normal CLI dispatch; matching args and runtime outcome are unknown",
			Status: "concrete", SourcePath: main.path, Line: hook.line, SourceHash: mainHash,
			Provenance: "main-conditional-early-dispatch-call-site",
		})
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("no top-level command specs found in %s", top.path)
	}
	return report, nil
}

type cliEarlyDispatchHook struct {
	name string
	line int
}

// cliEarlyDispatchHooks reports only direct main-level if-init calls that bind
// two results and whose handled branch exits the process. This identifies an
// early dispatch seam without claiming that a particular argument matched.
func cliEarlyDispatchHooks(file *ast.File, fset *token.FileSet) []cliEarlyDispatchHook {
	var main *ast.FuncDecl
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == "main" && fn.Recv == nil {
			main = fn
			break
		}
	}
	if main == nil || main.Body == nil {
		return nil
	}
	var hooks []cliEarlyDispatchHook
	for _, stmt := range main.Body.List {
		conditional, ok := stmt.(*ast.IfStmt)
		if !ok {
			continue
		}
		assignment, ok := conditional.Init.(*ast.AssignStmt)
		if !ok || len(assignment.Lhs) != 2 || len(assignment.Rhs) != 1 {
			continue
		}
		call, ok := assignment.Rhs[0].(*ast.CallExpr)
		if !ok {
			continue
		}
		name, ok := call.Fun.(*ast.Ident)
		if !ok {
			continue
		}
		first, firstOK := assignment.Lhs[0].(*ast.Ident)
		second, secondOK := assignment.Lhs[1].(*ast.Ident)
		condition, conditionOK := conditional.Cond.(*ast.Ident)
		if !firstOK || !secondOK || !conditionOK || first.Name != "handled" || second.Name != "code" || condition.Name != "handled" {
			continue
		}
		exits := false
		ast.Inspect(conditional.Body, func(node ast.Node) bool {
			if _, closure := node.(*ast.FuncLit); closure {
				return false
			}
			invocation, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := invocation.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, pkgOK := selector.X.(*ast.Ident)
			exits = exits || (pkgOK && pkg.Name == "os" && selector.Sel.Name == "Exit")
			return true
		})
		if exits {
			hooks = append(hooks, cliEarlyDispatchHook{name: name.Name, line: fset.Position(call.Pos()).Line})
		}
	}
	return hooks
}

type cliSourceUnit struct {
	path string
	fset *token.FileSet
	file *ast.File
	hash string
}

func parseCLIUnit(root, path string) (cliSourceUnit, string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return cliSourceUnit{}, "", err
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.AllErrors)
	if err != nil {
		return cliSourceUnit{}, "", fmt.Errorf("parse %s: %w", path, err)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return cliSourceUnit{}, "", err
	}
	sum := sha256.Sum256(content)
	hash := hex.EncodeToString(sum[:])
	return cliSourceUnit{path: filepath.ToSlash(rel), fset: fset, file: file, hash: hash}, hash, nil
}

func cliCommandConstants(file *ast.File) map[string]string {
	out := map[string]string{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			values := spec.(*ast.ValueSpec)
			for i, name := range values.Names {
				if i >= len(values.Values) {
					continue
				}
				literal, ok := values.Values[i].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					continue
				}
				value, err := strconv.Unquote(literal.Value)
				if err == nil {
					out[name.Name] = value
				}
			}
		}
	}
	return out
}

func cliCommandSpecs(file *ast.File, constants map[string]string, fset *token.FileSet) []cliCommandSpec {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "CommandSpecs" {
			continue
		}
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
						spec.name = resolveCommandName(kv.Value, constants)
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
				spec.argsExpression, spec.argsStatus, spec.argsLine = cliArgsDeclaration(row.Elts, fset)
				if spec.name != "" && spec.handler != "" {
					specs = append(specs, spec)
				}
			}
			return false
		})
		return specs
	}
	return nil
}

type cliHandlerBinding struct {
	expression string
	line       int
}

func cliHandlerBindings(file *ast.File, fset *token.FileSet) map[string]cliHandlerBinding {
	out := map[string]cliHandlerBinding{}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		switch fn.Name.Name {
		case "topLevelHandlerDefinitions":
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				lit, ok := node.(*ast.CompositeLit)
				if !ok {
					return true
				}
				if _, ok := lit.Type.(*ast.MapType); !ok {
					return true
				}
				for _, element := range lit.Elts {
					kv, ok := element.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key := resolveCommandID(kv.Key)
					if key != "" {
						out[key] = cliHandlerBinding{expression: handlerExpression(kv.Value), line: fset.Position(kv.Pos()).Line}
					}
				}
				return false
			})
		case "buildTopLevelHandlerMap":
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				assignment, ok := node.(*ast.AssignStmt)
				if !ok {
					return true
				}
				for i, lhs := range assignment.Lhs {
					index, ok := lhs.(*ast.IndexExpr)
					if !ok || i >= len(assignment.Rhs) {
						continue
					}
					key := resolveCommandID(index.Index)
					if key != "" {
						out[key] = cliHandlerBinding{expression: handlerExpression(assignment.Rhs[i]), line: fset.Position(assignment.Pos()).Line}
					}
				}
				return true
			})
		}
	}
	return out
}

func resolveCommandName(expr ast.Expr, constants map[string]string) string {
	if call, ok := expr.(*ast.CallExpr); ok {
		expr = call.Args[0]
	}
	if id, ok := expr.(*ast.Ident); ok {
		return constants[id.Name]
	}
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		return constants[sel.Sel.Name]
	}
	return ""
}

func resolveCommandID(expr ast.Expr) string {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		return sel.Sel.Name
	}
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

func handlerExpression(expr ast.Expr) string {
	if _, ok := expr.(*ast.FuncLit); ok {
		return "inline handler closure"
	}
	if call, ok := expr.(*ast.CallExpr); ok {
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if pkg, ok := sel.X.(*ast.Ident); ok {
				return pkg.Name + "." + sel.Sel.Name
			}
		}
		if id, ok := call.Fun.(*ast.Ident); ok {
			return id.Name
		}
	}
	var out strings.Builder
	if err := format.Node(&out, token.NewFileSet(), expr); err == nil {
		return out.String()
	}
	return "explicit handler expression"
}

func cliCallLine(file *ast.File, fset *token.FileSet, funcName, receiverName, methodName string) int {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != funcName {
			continue
		}
		var line int
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != methodName {
				return true
			}
			if receiverName == "configuredRunner" {
				receiver, ok := selector.X.(*ast.CallExpr)
				if !ok {
					return true
				}
				name, ok := receiver.Fun.(*ast.Ident)
				if !ok || name.Name != receiverName {
					return true
				}
			} else {
				receiver, ok := selector.X.(*ast.Ident)
				if !ok || receiver.Name != receiverName {
					return true
				}
			}
			line = fset.Position(call.Pos()).Line
			return false
		})
		return line
	}
	return 0
}
