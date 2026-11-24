package gates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOverlayWriteFileIsAtomicAndPreservesMode(t *testing.T) {
	rootPath := t.TempDir()
	root, err := os.OpenRoot(rootPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	path := filepath.Join(rootPath, "nested", "fixture.json")
	require.NoError(t, overlayWriteFile(root, path, []byte("first"), 0o640))
	require.NoError(t, overlayWriteFile(root, path, []byte("replacement"), 0o640))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "replacement", string(got))
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o640), info.Mode().Perm())
}

func TestOverlayWriteFileReplacesSymlinkWithoutWritingOutsideRoot(t *testing.T) {
	rootPath := t.TempDir()
	outsidePath := filepath.Join(t.TempDir(), "outside.txt")
	require.NoError(t, os.WriteFile(outsidePath, []byte("keep"), 0o600))
	root, err := os.OpenRoot(rootPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })
	require.NoError(t, root.Symlink(outsidePath, "escape"))

	err = overlayWriteFile(root, filepath.Join(rootPath, "escape"), []byte("changed"), 0o600)
	require.NoError(t, err)
	got, err := os.ReadFile(outsidePath)
	require.NoError(t, err)
	require.Equal(t, "keep", string(got))
	got, err = os.ReadFile(filepath.Join(rootPath, "escape"))
	require.NoError(t, err)
	require.Equal(t, "changed", string(got))
}

func TestOverlayRelativePathRejectsOutsideRoot(t *testing.T) {
	root, err := os.OpenRoot(t.TempDir())
	require.NoError(t, err)
	t.Cleanup(func() { _ = root.Close() })

	_, err = overlayRelativePath(root, filepath.Join(t.TempDir(), "outside"))
	require.Error(t, err)
}

func TestCalibrationOverlayTempCleanupDoesNotFollowOutsideSymlinks(t *testing.T) {
	tmp, parentRoot, err := createCalibrationOverlayTemp()
	require.NoError(t, err)
	t.Cleanup(func() { _ = parentRoot.Close() })

	root, err := os.OpenRoot(tmp)
	require.NoError(t, err)
	outsidePath := filepath.Join(t.TempDir(), "keep.txt")
	require.NoError(t, os.WriteFile(outsidePath, []byte("keep"), 0o600))
	require.NoError(t, root.Symlink(outsidePath, "escape"))
	require.NoError(t, root.Close())

	require.NoError(t, parentRoot.RemoveAll(filepath.Base(tmp)))
	_, err = os.Lstat(tmp)
	require.ErrorIs(t, err, os.ErrNotExist)
	got, err := os.ReadFile(outsidePath)
	require.NoError(t, err)
	require.Equal(t, "keep", string(got))
}

func TestPruneLibraryOverlayRemovesSymlinkWithoutTouchingTarget(t *testing.T) {
	rootPath := t.TempDir()
	libraryRoot := filepath.Join(rootPath, "scenarios", "react-component-library", "library")
	for _, kind := range []string{"foundations", "hooks", "services", "primitives", "components"} {
		require.NoError(t, os.MkdirAll(filepath.Join(libraryRoot, kind), 0o755))
	}
	kindDir := filepath.Join(libraryRoot, "components")
	target := filepath.Join(t.TempDir(), "keep.txt")
	require.NoError(t, os.WriteFile(target, []byte("keep"), 0o600))
	require.NoError(t, os.Symlink(target, filepath.Join(kindDir, "escape")))

	require.NoError(t, pruneLibraryOverlay(rootPath))
	_, err := os.Lstat(filepath.Join(kindDir, "escape"))
	require.ErrorIs(t, err, os.ErrNotExist)
	got, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, "keep", string(got))
}
