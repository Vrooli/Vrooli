package assets_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"brand-manager/internal/assets"

	"github.com/stretchr/testify/require"
)

func TestFSBlobStore_PutGetRemoveRoundTrip(t *testing.T) {
	base := t.TempDir()
	store := assets.NewFSBlobStore(base)

	path, err := store.Put("b1", "logo.png", []byte("bytes"))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(path, base), "stored path lives under the base dir")

	got, err := store.Get(path)
	require.NoError(t, err)
	require.Equal(t, "bytes", string(got))

	require.NoError(t, store.Remove(path))
	require.NoError(t, store.Remove(path), "removing a missing file is not an error")
	_, err = os.Stat(path)
	require.True(t, os.IsNotExist(err))
}

func TestFSBlobStore_PutIsStablePerBrandAndFilename(t *testing.T) {
	base := t.TempDir()
	store := assets.NewFSBlobStore(base)

	p1, err := store.Put("b1", "logo.png", []byte("v1"))
	require.NoError(t, err)
	p2, err := store.Put("b1", "logo.png", []byte("v2"))
	require.NoError(t, err)
	require.Equal(t, p1, p2, "the same (brand, filename) overwrites in place")

	got, err := store.Get(p2)
	require.NoError(t, err)
	require.Equal(t, "v2", string(got))
}

func TestFSBlobStore_RejectsPathEscape(t *testing.T) {
	base := t.TempDir()
	store := assets.NewFSBlobStore(base)

	// A path outside the base dir must be rejected by Get and Remove.
	outside := filepath.Join(filepath.Dir(base), "outside.png")
	require.NoError(t, os.WriteFile(outside, []byte("secret"), 0o600))

	_, err := store.Get(outside)
	require.Error(t, err, "reading outside the base dir is rejected")
	require.Error(t, store.Remove(outside), "removing outside the base dir is rejected")

	_, statErr := os.Stat(outside)
	require.NoError(t, statErr, "the rejected remove left the outside file untouched")
}

func TestFSBlobStore_PutSanitizesBrandAndFilename(t *testing.T) {
	base := t.TempDir()
	store := assets.NewFSBlobStore(base)

	// Even if a traversal-y brand/filename slips through, Put reduces them to
	// basenames so nothing escapes the base dir.
	path, err := store.Put("../evil", "../../escape.png", []byte("x"))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(path, base))
	require.Equal(t, "escape.png", filepath.Base(path))
}
