package lint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"

	"react-vite-temporal-model/internal/layout"
	"react-vite-temporal-model/internal/model"
)

const goShape = `    1. imports the generated subpackage
    2. defines exactly one Test* function that calls <subpkg>.RunReplay(t, transition)
    3. supplies a transition function whose body is non-empty (not nil, not _ = ...).
  Example:
    func TestFooFormalReplay(t *testing.T) {
        foo.RunReplay(t, func(s foo.Status, e foo.Event) (foo.Status, error) {
            return transitionFoo(s, e)
        })
    }
`

func checkGo(root string, flow model.Flow) error {
	files, err := listFiles(root, flow.Layout.BaseDir, "_test.go")
	if err != nil {
		return fmt.Errorf("%s: read %s: %w", flow.FlowID, flow.Layout.BaseDir, err)
	}
	expectedImport := layout.SubpackageImportPath(flow.Layout)
	var failures []string
	matched := false
	for _, path := range files {
		ok, why, err := scanGoTestFile(path, expectedImport, flow.Layout.FolderName)
		if err != nil {
			failures = append(failures, fmt.Sprintf("    %s: %v", path, err))
			continue
		}
		if ok {
			matched = true
			continue
		}
		if why != "" {
			failures = append(failures, fmt.Sprintf("    %s: %s", path, why))
		}
	}
	if matched {
		return nil
	}
	msg := shape(flow, goShape)
	if len(failures) > 0 {
		msg += "  Scanned files reported:\n" + strings.Join(failures, "\n") + "\n"
	} else {
		msg += "  Scanned files: none with _test.go suffix found.\n"
	}
	return fmt.Errorf("%s", msg)
}

// scanGoTestFile parses a Go test file and returns (matched, why,
// err). matched is true only if the file imports the expected
// subpackage AND defines a Test* function that calls
// <subpkg>.RunReplay(t, transition) where transition is non-trivial.
// When the file imports the subpackage but the call shape is wrong,
// why explains the specific defect for the lint output.
func scanGoTestFile(path string, expectedImport string, pkgIdent string) (bool, string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
	if err != nil {
		return false, "", err
	}

	hasImport := false
	for _, imp := range file.Imports {
		value := strings.Trim(imp.Path.Value, "\"")
		if value == expectedImport {
			hasImport = true
			break
		}
	}
	if !hasImport {
		return false, "", nil
	}

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "Test") {
			continue
		}
		ok, why := bodyCallsRunReplay(fn.Body, pkgIdent)
		if ok {
			return true, "", nil
		}
		if why != "" {
			return false, fmt.Sprintf("%s: %s", fn.Name.Name, why), nil
		}
	}
	return false, "imports the generated subpackage but no Test* function calls RunReplay", nil
}

func bodyCallsRunReplay(body *ast.BlockStmt, pkgIdent string) (bool, string) {
	if body == nil {
		return false, "function body is nil"
	}
	for _, stmt := range body.List {
		expr, ok := stmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := expr.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok || ident.Name != pkgIdent {
			continue
		}
		if sel.Sel.Name != "RunReplay" {
			continue
		}
		if len(call.Args) != 2 {
			return false, fmt.Sprintf("RunReplay expects 2 arguments, got %d", len(call.Args))
		}
		if !isTIdent(call.Args[0]) {
			return false, "first argument to RunReplay must be t"
		}
		if reason := checkTransitionArg(call.Args[1]); reason != "" {
			return false, "transition argument: " + reason
		}
		return true, ""
	}
	return false, ""
}

func isTIdent(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == "t"
}

// checkTransitionArg rejects nil, empty function literals, and bodies
// that contain only blank assignments or trivial returns. A non-empty
// reason means the lint should fail.
func checkTransitionArg(expr ast.Expr) string {
	switch arg := expr.(type) {
	case *ast.Ident:
		if arg.Name == "nil" {
			return "nil is not a valid transition"
		}
		return ""
	case *ast.FuncLit:
		if arg.Body == nil || len(arg.Body.List) == 0 {
			return "function literal has an empty body"
		}
		if onlyBlankAssign(arg.Body) {
			return "function body contains only blank-identifier assignments"
		}
		return ""
	case *ast.CallExpr:
		return ""
	case *ast.SelectorExpr:
		return ""
	default:
		return "unsupported expression shape"
	}
}

func onlyBlankAssign(body *ast.BlockStmt) bool {
	for _, stmt := range body.List {
		assign, ok := stmt.(*ast.AssignStmt)
		if !ok {
			return false
		}
		if len(assign.Lhs) != 1 {
			return false
		}
		ident, ok := assign.Lhs[0].(*ast.Ident)
		if !ok || ident.Name != "_" {
			return false
		}
	}
	return len(body.List) > 0
}
