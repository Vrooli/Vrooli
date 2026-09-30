package gotest

import (
	"go/ast"
	"go/token"
	"strconv"
)

// Registration requires both sides of the repository's typed subprocess
// convention in the same package: a guarded helper entrypoint and a launcher
// payload with this test binary, its exact anchored test name, and the opt-in
// environment flag. A helper-looking name or getenv check alone is insufficient.
func registeredProcessHelpers(packages map[string]map[string][]function) map[string]bool {
	registered := map[string]bool{}
	for pkg, functions := range packages {
		for name, candidates := range functions {
			if len(candidates) != 1 || !processGuard(candidates[0]) {
				continue
			}
			for launcherName, launchers := range functions {
				if launcherName == name {
					continue
				}
				for _, launcher := range launchers {
					if typedLaunch(launcher, name) {
						registered[pkg+":"+name] = true
					}
				}
			}
		}
	}
	return registered
}
func literal(expr ast.Expr) string {
	value, ok := expr.(*ast.BasicLit)
	if !ok || value.Kind != token.STRING {
		return ""
	}
	text, _ := strconv.Unquote(value.Value)
	return text
}
func osSelector(f *source, expr ast.Expr, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != name {
		return false
	}
	id, ok := selector.X.(*ast.Ident)
	return ok && id.Obj == nil && f.imports[id.Name] == "os"
}
func processGuard(fn function) bool {
	if len(fn.decl.Body.List) == 0 {
		return false
	}
	guard, ok := fn.decl.Body.List[0].(*ast.IfStmt)
	if !ok || guard.Init != nil || guard.Else != nil || len(guard.Body.List) != 1 {
		return false
	}
	if _, ok := guard.Body.List[0].(*ast.ReturnStmt); !ok {
		return false
	}
	condition, ok := guard.Cond.(*ast.BinaryExpr)
	if !ok || condition.Op != token.NEQ || literal(condition.Y) != "1" {
		return false
	}
	call, ok := condition.X.(*ast.CallExpr)
	return ok && osSelector(fn.file, call.Fun, "Getenv") && len(call.Args) == 1 && literal(call.Args[0]) == "GO_WANT_HELPER_PROCESS"
}
func typedLaunch(fn function, name string) bool {
	found := false
	ast.Inspect(fn.decl.Body, func(node ast.Node) bool {
		object, ok := node.(*ast.CompositeLit)
		if !ok {
			return true
		}
		binary, args, environment := false, false, false
		for _, entry := range object.Elts {
			pair, ok := entry.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := pair.Key.(*ast.Ident)
			if !ok {
				continue
			}
			switch key.Name {
			case "Executable":
				index, ok := pair.Value.(*ast.IndexExpr)
				if ok && osSelector(fn.file, index.X, "Args") {
					if number, ok := index.Index.(*ast.BasicLit); ok && number.Kind == token.INT && number.Value == "0" {
						binary = true
					}
				}
			case "Args":
				if values, ok := pair.Value.(*ast.CompositeLit); ok {
					for _, value := range values.Elts {
						if literal(value) == "-test.run=^"+name+"$" {
							args = true
						}
					}
				}
			case "Env":
				if values, ok := pair.Value.(*ast.CompositeLit); ok {
					for _, value := range values.Elts {
						if item, ok := value.(*ast.KeyValueExpr); ok && literal(item.Key) == "GO_WANT_HELPER_PROCESS" && literal(item.Value) == "1" {
							environment = true
						}
					}
				}
			}
		}
		if binary && args && environment {
			found = true
		}
		return true
	})
	return found
}
