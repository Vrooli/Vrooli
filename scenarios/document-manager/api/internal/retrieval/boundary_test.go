package retrieval

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetrievalCannotImportLedger(t *testing.T) {
	root, err := filepath.Abs(".")
	require.NoError(t, err)
	err = filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "boundary_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		require.NoError(t, err)
		for _, imp := range file.Imports {
			require.NotContains(t, imp.Path.Value, "source-ledger")
		}
		return nil
	})
	require.NoError(t, err)
}
