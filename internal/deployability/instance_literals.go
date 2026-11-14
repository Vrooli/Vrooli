package deployability

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

// InstanceLiteral identifies a forbidden instance name embedded in a source
// string literal. It is used by structural validation at the control-plane
// boundary; the resolver itself remains independent of the fleet.
type InstanceLiteral struct {
	Path  string
	Line  int
	Value string
}

// FindInstanceLiterals scans Go source for exact string literals that name a
// concrete fleet object. Callers load the known names from manifests, so the
// validator does not maintain a second instance catalog.
func FindInstanceLiterals(path string, source []byte, knownNames []string) ([]InstanceLiteral, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, 0)
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(knownNames))
	for _, name := range knownNames {
		name = strings.ToLower(strings.TrimSpace(name))
		if name != "" {
			known[name] = struct{}{}
		}
	}
	result := make([]InstanceLiteral, 0)
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		if _, exists := known[strings.ToLower(strings.TrimSpace(value))]; exists {
			result = append(result, InstanceLiteral{Path: path, Line: fset.Position(literal.Pos()).Line, Value: value})
		}
		return true
	})
	return result, nil
}
