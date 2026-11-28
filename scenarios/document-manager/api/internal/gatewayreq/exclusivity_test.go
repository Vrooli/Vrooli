package gatewayreq

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestGatewayRequestHasOneConstructionSite is intentionally structural. The
// policy is not protected if a future caller can construct the provider-neutral
// envelope beside the builder and bypass its fail-closed check.
func TestGatewayRequestHasOneConstructionSite(t *testing.T) { // [REQ:DOC-P0-010] [REQ:DOC-P0-026]
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	require.NoError(t, err)
	fset := token.NewFileSet()
	count := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(path) != ".go" || filepath.Base(path) == "exclusivity_test.go" {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return parseErr
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}
			selector, ok := literal.Type.(*ast.SelectorExpr)
			if ok && selector.Sel.Name == "GatewayRequest" {
				count++
			}
			return true
		})
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, count, "[REQ:DOC-P0-026] GatewayRequest must have one construction site")
}
