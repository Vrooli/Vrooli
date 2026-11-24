package lifecycleinference

import (
	"go/ast"
	"strings"
)

// filePersistenceParameters finds source-local helpers that serialize one of
// their parameters as JSON and write those serialized bytes to a file. Callers
// can then prove persistence for any candidate-containing value passed through
// that helper, without hard-coding scenario or helper names.
func filePersistenceParameters(files []parsedSource) map[string]map[int]bool {
	result := map[string]map[int]bool{}
	for _, source := range files {
		for _, declaration := range source.file.Decls {
			fn, ok := declaration.(*ast.FuncDecl)
			if !ok || fn.Body == nil || fn.Type.Params == nil {
				continue
			}
			params := map[string]map[int]bool{}
			paramIndex := 0
			for _, field := range fn.Type.Params.List {
				for _, name := range field.Names {
					params[name.Name] = map[int]bool{paramIndex: true}
					paramIndex++
				}
			}
			if len(params) == 0 {
				continue
			}

			// Carry parameter identity through simple aliases and conversions.
			for pass := 0; pass < 4; pass++ {
				changed := false
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					var names []ast.Expr
					var values []ast.Expr
					switch current := node.(type) {
					case *ast.ValueSpec:
						for _, name := range current.Names {
							names = append(names, name)
						}
						values = current.Values
					case *ast.AssignStmt:
						names, values = current.Lhs, current.Rhs
					}
					for i, nameExpr := range names {
						if i >= len(values) {
							continue
						}
						name, ok := nameExpr.(*ast.Ident)
						if !ok || name.Name == "_" {
							continue
						}
						origins := parameterOrigins(values[i], params)
						if len(origins) == 0 {
							continue
						}
						if params[name.Name] == nil {
							params[name.Name] = map[int]bool{}
						}
						for origin := range origins {
							if !params[name.Name][origin] {
								params[name.Name][origin] = true
								changed = true
							}
						}
					}
					return true
				})
				if !changed {
					break
				}
			}

			serialized := map[string]map[int]bool{}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || !isJSONEncoderCall(call, source) || len(call.Args) == 0 {
					return true
				}
				origins := parameterOrigins(call.Args[0], params)
				if len(origins) == 0 {
					return true
				}
				for _, name := range assignedNames(fn.Body, call) {
					if serialized[name] == nil {
						serialized[name] = map[int]bool{}
					}
					for origin := range origins {
						serialized[name][origin] = true
					}
				}
				return true
			})
			if len(serialized) == 0 {
				continue
			}

			fileVars := map[string]bool{}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || !isOSFileCreate(call, source) {
					return true
				}
				for _, name := range assignedNames(fn.Body, call) {
					fileVars[name] = true
				}
				return true
			})
			persisted := map[int]bool{}
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if isOSWriteFile(call, source) && len(call.Args) >= 2 {
					mergeOrigins(persisted, parameterOrigins(call.Args[1], serialized))
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if ok && selector.Sel.Name == "Write" && fileVars[exprIdent(selector.X)] {
					for _, arg := range call.Args {
						mergeOrigins(persisted, parameterOrigins(arg, serialized))
					}
				}
				return true
			})
			if len(persisted) > 0 {
				key := filePersistenceFunctionKey(source, fn)
				result[key] = persisted
			}
		}
	}
	return result
}

func parameterOrigins(expr ast.Expr, values map[string]map[int]bool) map[int]bool {
	origins := map[int]bool{}
	var walk func(ast.Expr)
	walk = func(expr ast.Expr) {
		switch node := expr.(type) {
		case *ast.Ident:
			mergeOrigins(origins, values[node.Name])
		case *ast.SelectorExpr:
			// A selector may serialize only one unrelated sibling field of a
			// candidate-containing aggregate. Treating its receiver as though the
			// whole record were persisted creates false durable-lifecycle proofs.
		case *ast.CallExpr:
			for _, arg := range node.Args {
				walk(arg)
			}
		case *ast.ParenExpr:
			walk(node.X)
		case *ast.StarExpr:
			walk(node.X)
		case *ast.UnaryExpr:
			walk(node.X)
		case *ast.BinaryExpr:
			walk(node.X)
			walk(node.Y)
		case *ast.KeyValueExpr:
			walk(node.Value)
		case *ast.CompositeLit:
			for _, elt := range node.Elts {
				walk(elt)
			}
		case *ast.IndexExpr:
			walk(node.X)
		case *ast.IndexListExpr:
			walk(node.X)
		}
	}
	walk(expr)
	return origins
}

func mergeOrigins(into, from map[int]bool) {
	for origin := range from {
		into[origin] = true
	}
}

func isJSONEncoderCall(call *ast.CallExpr, source parsedSource) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && (selector.Sel.Name == "Marshal" || selector.Sel.Name == "MarshalIndent") && source.imports[exprIdent(selector.X)] == "encoding/json"
}

func isOSFileCreate(call *ast.CallExpr, source parsedSource) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || source.imports[exprIdent(selector.X)] != "os" {
		return false
	}
	switch selector.Sel.Name {
	case "Create", "CreateTemp", "OpenFile":
		return true
	}
	return false
}

func isOSWriteFile(call *ast.CallExpr, source parsedSource) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "WriteFile" && source.imports[exprIdent(selector.X)] == "os"
}

func assignedNames(body *ast.BlockStmt, target *ast.CallExpr) []string {
	var names []string
	ast.Inspect(body, func(node ast.Node) bool {
		var lhs []ast.Expr
		var rhs []ast.Expr
		switch current := node.(type) {
		case *ast.ValueSpec:
			for _, name := range current.Names {
				lhs = append(lhs, name)
			}
			rhs = current.Values
		case *ast.AssignStmt:
			lhs, rhs = current.Lhs, current.Rhs
		default:
			return true
		}
		for i, value := range rhs {
			contains := false
			ast.Inspect(value, func(node ast.Node) bool {
				if node == target {
					contains = true
					return false
				}
				return !contains
			})
			if !contains || i >= len(lhs) {
				continue
			}
			if ident, ok := lhs[i].(*ast.Ident); ok {
				names = append(names, ident.Name)
			}
		}
		return true
	})
	return names
}

func filePersistenceFunctionKey(source parsedSource, fn *ast.FuncDecl) string {
	key := source.importPath
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		if receiver := receiverTypeName(fn.Recv.List[0].Type); receiver != "" {
			key += "." + receiver
		}
	}
	return key + "." + fn.Name.Name
}

func filePersistenceCallKey(call *ast.CallExpr, source parsedSource, caller *ast.FuncDecl) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return source.importPath + "." + fun.Name
	case *ast.SelectorExpr:
		importPath := source.imports[exprIdent(fun.X)]
		if strings.TrimSpace(importPath) != "" {
			return importPath + "." + fun.Sel.Name
		}
		// Resolve same-type receiver calls such as s.writeToDisk(...) from
		// within a method on the same receiver. We cannot safely infer local
		// variable or interface dispatch without type information.
		if caller != nil && caller.Recv != nil && len(caller.Recv.List) > 0 {
			for _, name := range caller.Recv.List[0].Names {
				if name.Name == exprIdent(fun.X) {
					if receiver := receiverTypeName(caller.Recv.List[0].Type); receiver != "" {
						return source.importPath + "." + receiver + "." + fun.Sel.Name
					}
				}
			}
		}
	}
	return ""
}

func receiverTypeName(expr ast.Expr) string {
	for {
		switch current := expr.(type) {
		case *ast.StarExpr:
			expr = current.X
		case *ast.IndexExpr:
			expr = current.X
		case *ast.IndexListExpr:
			expr = current.X
		case *ast.Ident:
			return current.Name
		default:
			return ""
		}
	}
}

func callCarriesPersistentJSON(call *ast.CallExpr, source parsedSource, caller *ast.FuncDecl, filesinks map[string]map[int]bool, carries func(ast.Expr) bool) bool {
	for index := range filesinks[filePersistenceCallKey(call, source, caller)] {
		if index < len(call.Args) && carries(call.Args[index]) {
			return true
		}
	}
	return false
}
