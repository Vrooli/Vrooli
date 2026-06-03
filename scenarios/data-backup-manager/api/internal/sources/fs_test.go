package sources_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"data-backup-manager/internal/sources"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFilesystemSource_RoundTrip creates a nested directory tree, Captures it
// into a stage directory, Restores into a fresh directory, and asserts that
// every file's content is byte-identical to the original.
func TestFilesystemSource_RoundTrip(t *testing.T) {
	t.Parallel()

	// --- build a source tree ---
	srcDir := t.TempDir()
	files := map[string][]byte{
		"readme.txt":          []byte("hello world\n"),
		"subdir/data.bin":     {0x00, 0x01, 0x02, 0xAB, 0xCD},
		"subdir/nested/a.txt": []byte("nested a\n"),
		"subdir/nested/b.txt": []byte("nested b\n"),
		"another/empty.txt":   {},
	}
	for rel, content := range files {
		full := filepath.Join(srcDir, rel)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
		require.NoError(t, os.WriteFile(full, content, 0o640))
	}

	ctx := context.Background()
	reg := sources.NewProductionRegistry(sources.ExecRunner{})

	cap, err := reg.Capturer(sources.KindFilesystem)
	require.NoError(t, err)

	// --- Capture ---
	stageDir := t.TempDir()
	art, err := cap.Capture(ctx, sources.CaptureSpec{
		Locator:  srcDir,
		StageDir: stageDir,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, art.Path)

	// --- Restore ---
	restoreDir := t.TempDir()
	err = cap.Restore(ctx, sources.RestoreSpec{
		Locator:      srcDir,
		ArtifactPath: art.Path,
		Target:       restoreDir,
	})
	require.NoError(t, err)

	// --- Assert byte-identical ---
	wantChecksums := checksumTree(t, srcDir)
	gotChecksums := checksumTree(t, restoreDir)
	assert.Equal(t, wantChecksums, gotChecksums, "restored tree must be byte-identical to source")
}

// TestFilesystemSource_DirectoryInPlace proves a directory locator is captured
// in place: the artifact points AT the source (no staging copy), the stage dir
// stays empty, and the reported bytes are the logical sum of the tree's regular
// files. This is the I/O win — kopia snapshots the live tree directly.
func TestFilesystemSource_DirectoryInPlace(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(srcDir, "sub"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("12345"), 0o640))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "sub/b.txt"), []byte("678"), 0o640))

	ctx := context.Background()
	cap, err := sources.NewProductionRegistry(sources.ExecRunner{}).Capturer(sources.KindFilesystem)
	require.NoError(t, err)

	stageDir := t.TempDir()
	art, err := cap.Capture(ctx, sources.CaptureSpec{Locator: srcDir, StageDir: stageDir})
	require.NoError(t, err)

	assert.Equal(t, srcDir, art.Path, "directory capture must be in place (artifact path == source)")
	assert.Equal(t, int64(8), art.Bytes, "bytes must be the logical sum of regular files")

	entries, err := os.ReadDir(stageDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "no staging copy must be made for a directory locator")
}

// TestFilesystemSource_SingleFile captures a single-file locator (not a
// directory) and asserts it round-trips as fs/<basename>. This is the shape of
// most default coverage targets (config.toml, history.jsonl, secrets.json).
func TestFilesystemSource_SingleFile(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, "config.toml")
	content := []byte("model = \"opus\"\n")
	require.NoError(t, os.WriteFile(srcFile, content, 0o640))

	ctx := context.Background()
	cap, err := sources.NewProductionRegistry(sources.ExecRunner{}).Capturer(sources.KindFilesystem)
	require.NoError(t, err)

	stageDir := t.TempDir()
	art, err := cap.Capture(ctx, sources.CaptureSpec{Locator: srcFile, StageDir: stageDir})
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), art.Bytes)

	staged := filepath.Join(art.Path, "config.toml")
	got, err := os.ReadFile(staged)
	require.NoError(t, err)
	assert.Equal(t, content, got, "single file must be staged as fs/<basename>")

	restoreDir := t.TempDir()
	require.NoError(t, cap.Restore(ctx, sources.RestoreSpec{
		Locator:      srcFile,
		ArtifactPath: art.Path,
		Target:       restoreDir,
	}))
	restored, err := os.ReadFile(filepath.Join(restoreDir, "config.toml"))
	require.NoError(t, err)
	assert.Equal(t, content, restored)
}

// TestFilesystemSource_PreservesSymlinks asserts a symlink in the source tree —
// including one pointing at a directory and one pointing OUTSIDE the captured
// tree — is preserved as a symlink, not followed. Following an out-of-tree link
// (the ~/.vrooli/state → ~/.codex/auth.json pattern) would both break the copy
// and dereference deliberately-excluded files.
func TestFilesystemSource_PreservesSymlinks(t *testing.T) {
	t.Parallel()

	// An external directory the in-tree symlink will point at.
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("nope"), 0o600))

	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "real.txt"), []byte("real\n"), 0o640))
	require.NoError(t, os.Symlink(outside, filepath.Join(srcDir, "link-to-dir")))
	require.NoError(t, os.Symlink("real.txt", filepath.Join(srcDir, "link-to-file")))

	ctx := context.Background()
	cap, err := sources.NewProductionRegistry(sources.ExecRunner{}).Capturer(sources.KindFilesystem)
	require.NoError(t, err)

	stageDir := t.TempDir()
	art, err := cap.Capture(ctx, sources.CaptureSpec{Locator: srcDir, StageDir: stageDir})
	require.NoError(t, err, "capturing a tree with symlinks must not error")

	// Restore exercises copyTree (the path that still copies on the in-place
	// model): the restored tree must preserve symlinks as links and must NOT
	// have dereferenced the out-of-tree link into the artifact.
	restoreDir := t.TempDir()
	require.NoError(t, cap.Restore(ctx, sources.RestoreSpec{
		Locator:      srcDir,
		ArtifactPath: art.Path,
		Target:       restoreDir,
	}))

	info, err := os.Lstat(filepath.Join(restoreDir, "link-to-dir"))
	require.NoError(t, err)
	assert.NotZero(t, info.Mode()&fs.ModeSymlink, "link-to-dir must be preserved as a symlink")
	link, err := os.Readlink(filepath.Join(restoreDir, "link-to-dir"))
	require.NoError(t, err)
	assert.Equal(t, outside, link, "symlink target must be copied verbatim, not dereferenced")

	// The excluded out-of-tree target must NOT have been pulled in as a real file.
	assert.NoFileExists(t, filepath.Join(restoreDir, "secret.txt"))
}

// checksumTree walks root and returns map[relative-path]sha256hex for every file.
func checksumTree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		f, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		defer f.Close()
		h := sha256.New()
		if _, copyErr := io.Copy(h, f); copyErr != nil {
			return copyErr
		}
		out[rel] = fmt.Sprintf("%x", h.Sum(nil))
		return nil
	})
	require.NoError(t, err)
	return out
}
