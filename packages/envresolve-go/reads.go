package envresolve

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
)

type EnvRead struct {
	Variable string
	Line     int
}

// FindEnvReads returns actual os.Getenv/os.LookupEnv calls from Go syntax.
// Parsing instead of matching source text prevents examples in comments and
// raw documentation strings from entering the producer census.
func FindEnvReads(payload []byte) ([]EnvRead, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", payload, 0)
	if err != nil {
		return nil, err
	}
	reads := make([]EnvRead, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != "Getenv" && selector.Sel.Name != "LookupEnv" {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		if !ok || ident.Name != "os" {
			return true
		}
		literal, ok := call.Args[0].(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err == nil {
			reads = append(reads, EnvRead{Variable: value, Line: fset.Position(call.Pos()).Line})
		}
		return true
	})
	return reads, nil
}
