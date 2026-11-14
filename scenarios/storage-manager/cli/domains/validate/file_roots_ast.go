package validate

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"strconv"
)

const fileroutingImportPath = "github.com/vrooli/api-core/filerouting"

type parsedGoFile struct {
	dir     string
	file    *ast.File
	aliases map[string]bool
}

type sourceFile struct {
	dir     string
	content string
}

// hasContextualFileRootPick detects the API-free source seam by matching AST
// declarations and imports for api-core/filerouting.RoutedRoots, then looking
// for Pick/PickRequired calls whose first argument is named ctx or context.
// This does not type-check Go or prove that a runtime request uses the route.
func hasContextualFileRootPick(sources []sourceFile) bool {
	files := make([]parsedGoFile, 0, len(sources))
	routedStructFields := map[string]map[string]map[string]bool{}

	for _, source := range sources {
		file, err := parser.ParseFile(token.NewFileSet(), source.dir+"/scenario.go", source.content, 0)
		if err != nil {
			continue
		}
		aliases := fileroutingAliases(file)
		files = append(files, parsedGoFile{dir: source.dir, file: file, aliases: aliases})
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok || structure.Fields == nil {
					continue
				}
				for _, field := range structure.Fields.List {
					if !isRoutedRootsType(field.Type, aliases) {
						continue
					}
					for _, name := range field.Names {
						if routedStructFields[source.dir] == nil {
							routedStructFields[source.dir] = map[string]map[string]bool{}
						}
						if routedStructFields[source.dir][typeSpec.Name.Name] == nil {
							routedStructFields[source.dir][typeSpec.Name.Name] = map[string]bool{}
						}
						routedStructFields[source.dir][typeSpec.Name.Name][name.Name] = true
					}
				}
			}
		}
	}

	for _, parsed := range files {
		for _, decl := range parsed.file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			rootVars := map[string]bool{}
			if fn.Type.Params != nil {
				for _, field := range fn.Type.Params.List {
					if isRoutedRootsType(field.Type, parsed.aliases) {
						for _, name := range field.Names {
							rootVars[name.Name] = true
						}
					}
				}
			}
			rootReceiverFields := map[string]bool{}
			if fn.Recv != nil {
				for _, receiver := range fn.Recv.List {
					for _, name := range receiver.Names {
						for field := range routedStructFields[parsed.dir][namedType(receiver.Type)] {
							rootReceiverFields[name.Name+"."+field] = true
						}
					}
				}
			}

			ast.Inspect(fn.Body, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.ValueSpec:
					if isRoutedRootsType(typed.Type, parsed.aliases) {
						for _, name := range typed.Names {
							rootVars[name.Name] = true
						}
					}
				case *ast.AssignStmt:
					for lhsIndex, rhs := range typed.Rhs {
						call, ok := rhs.(*ast.CallExpr)
						if !ok || !isFileroutingNew(call, parsed.aliases) || len(typed.Rhs) != len(typed.Lhs) {
							continue
						}
						if id, ok := typed.Lhs[lhsIndex].(*ast.Ident); ok {
							rootVars[id.Name] = true
						}
					}
				}
				return true
			})
			found := false
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 || !contextArgument(call) {
					return true
				}
				selector, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || (selector.Sel.Name != "Pick" && selector.Sel.Name != "PickRequired") {
					return true
				}
				if routedRootReceiver(selector.X, rootVars, rootReceiverFields) {
					found = true
					return false
				}
				return true
			})
			if found {
				return true
			}
		}
	}
	return false
}

func fileroutingAliases(file *ast.File) map[string]bool {
	aliases := map[string]bool{}
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil || importPath != fileroutingImportPath {
			continue
		}
		alias := path.Base(importPath)
		if spec.Name != nil {
			alias = spec.Name.Name
		}
		aliases[alias] = true
	}
	return aliases
}

func isRoutedRootsType(expr ast.Expr, aliases map[string]bool) bool {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "RoutedRoots" {
		return false
	}
	packageName, ok := selector.X.(*ast.Ident)
	return ok && aliases[packageName.Name]
}

func isFileroutingNew(call *ast.CallExpr, aliases map[string]bool) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "New" {
		return false
	}
	packageName, ok := selector.X.(*ast.Ident)
	return ok && aliases[packageName.Name]
}

func namedType(expr ast.Expr) string {
	for {
		switch typed := expr.(type) {
		case *ast.StarExpr:
			expr = typed.X
		case *ast.Ident:
			return typed.Name
		case *ast.SelectorExpr:
			return typed.Sel.Name
		default:
			return ""
		}
	}
}

func contextArgument(call *ast.CallExpr) bool {
	if len(call.Args) == 0 {
		return false
	}
	ident, ok := call.Args[0].(*ast.Ident)
	return ok && (ident.Name == "ctx" || ident.Name == "context")
}

func routedRootReceiver(receiver ast.Expr, rootVars, rootReceiverFields map[string]bool) bool {
	switch typed := receiver.(type) {
	case *ast.Ident:
		return rootVars[typed.Name]
	case *ast.SelectorExpr:
		base, ok := typed.X.(*ast.Ident)
		return ok && rootReceiverFields[base.Name+"."+typed.Sel.Name]
	default:
		return false
	}
}
