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
