package validation

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

// usesSharedSecurityHeaders recognizes the typed api-core middleware adoption.
// A matching comment, unused import, or same-named local package is not evidence.
func usesSharedSecurityHeaders(source string) bool {
	file, err := parser.ParseFile(token.NewFileSet(), "server.go", source, 0)
	if err != nil {
		return false
	}
	alias := ""
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != "github.com/vrooli/api-core/apihttp" {
			continue
		}
		alias = "apihttp"
		if spec.Name != nil {
			alias = spec.Name.Name
		}
	}
	if alias == "" || alias == "_" || alias == "." {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "SecurityHeaders" || len(call.Args) != 1 {
			return true
		}
		object, ok := selector.X.(*ast.Ident)
		if ok && object.Name == alias {
			found = true
		}
		return true
	})
	return found
}
