package surfaces

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultLocatorResolvesPathTarget(t *testing.T) {
	root := t.TempDir()
	locator := DefaultLocator{}

	scenario, kind, path, err := locator.Locate(context.Background(), "", root)

	require.NoError(t, err)
	require.Equal(t, filepath.Base(root), scenario)
	require.Equal(t, "path", kind)
	require.Equal(t, root, path)
}

func TestFallbackInventoryDetectsGeneratedSurfaces(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "ui"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "ui", "package.json"), []byte(`{"dependencies":{"react":"latest","vite":"latest"},"devDependencies":{}}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "ui", "tsconfig.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "api"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module example.test/api\n"), 0o644))

	inv := fallbackInventory("demo", "path", root)

	require.Len(t, inv.Surfaces, 2)
	require.Equal(t, "go", inv.Surfaces[0].Language)
	require.Equal(t, "typescript", inv.Surfaces[1].Language)
	require.Equal(t, "react-vite", inv.Surfaces[1].Framework)
}
