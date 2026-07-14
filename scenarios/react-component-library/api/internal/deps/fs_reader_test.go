package deps

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFSPackageJSONReader_PrefersUIManifestAndFallsBackToRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "ui-scenario", "ui"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "ui-scenario", "ui", "package.json"), []byte(`{"name":"ui"}`), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "root-scenario"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "root-scenario", "package.json"), []byte(`{"name":"root"}`), 0o600))
	reader := NewFSPackageJSONReader(root)
	ui, err := reader.Read(context.Background(), "ui-scenario")
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"ui"}`, string(ui))
	rootManifest, err := reader.Read(context.Background(), "root-scenario")
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"root"}`, string(rootManifest))
}
