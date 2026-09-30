package planning

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilesystemMaterializerStripsScenarioPrefixAndRunsGenerate(t *testing.T) {
	root := t.TempDir()
	schemasRoot := filepath.Join(root, "schemas")
	require.NoError(t, os.MkdirAll(schemasRoot, 0o755))
	var ranDir string
	var ranArgs []string
	m := &FilesystemMaterializer{
		SchemasRoot: schemasRoot,
		ProtoRoot:   root,
		Command: func(_ context.Context, dir string, args ...string) error {
			ranDir = dir
			ranArgs = append([]string(nil), args...)
			return nil
		},
	}

	result, err := m.Materialize(context.Background(), Scenario{
		Slug: "planned-demo",
		Files: []ProtoFile{{
			Path: "planned-demo/v1/api/service.proto",
			Text: validProtoText(),
		}},
	})

	require.NoError(t, err)
	require.True(t, result.Generated)
	require.Equal(t, root, ranDir)
	require.Equal(t, []string{"generate"}, ranArgs)
	require.FileExists(t, filepath.Join(schemasRoot, "planned-demo", "v1", "api", "service.proto"))
	require.NoFileExists(t, filepath.Join(schemasRoot, "planned-demo", "planned-demo", "v1", "api", "service.proto"))
}
