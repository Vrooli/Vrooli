package assets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BlobStore is the byte-persistence seam for asset files. Production wires the
// filesystem-backed implementation; service unit tests wire mocks.FakeBlobStore
// so they never touch disk. The store owns path safety: every method confirms
// the resolved path stays under the configured base directory.
type BlobStore interface {
	// Put writes data for (brandID, filename) and returns the stored path. The
	// path is stable per (brandID, filename), so a re-write overwrites in place.
	Put(brandID, filename string, data []byte) (string, error)

	// Get reads the bytes at path. Returns ErrAssetNotFound-free errors (the
	// service maps a missing file separately); a path escaping the base dir is
	// rejected.
	Get(path string) ([]byte, error)

	// Remove deletes the file at path. A missing file is not an error
	// (idempotent); a path escaping the base dir is rejected.
	Remove(path string) error
}

// fsBlobStore stores asset bytes under baseDir/<brand_id>/<filename>.
type fsBlobStore struct {
	baseDir string
}

// NewFSBlobStore constructs the production filesystem BlobStore rooted at
// baseDir (e.g. <scenario-data>/assets). The directory is created lazily on the
// first Put.
func NewFSBlobStore(baseDir string) BlobStore {
	return &fsBlobStore{baseDir: filepath.Clean(baseDir)}
}

var _ BlobStore = (*fsBlobStore)(nil)

func (s *fsBlobStore) Put(brandID, filename string, data []byte) (string, error) {
	// filename is already validated as a basename by the service; defend anyway.
	clean := filepath.Base(filename)
	dir := filepath.Join(s.baseDir, filepath.Base(brandID))
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("create asset dir: %w", err)
	}
	dest := filepath.Join(dir, clean)
	if !s.withinBase(dest) {
		return "", fmt.Errorf("asset path escapes base directory")
	}
	if err := writeFileAtomic(dest, data); err != nil {
		return "", fmt.Errorf("write asset file: %w", err)
	}
	return dest, nil
}

func (s *fsBlobStore) Get(path string) ([]byte, error) {
	clean := filepath.Clean(path)
	if !s.withinBase(clean) {
		return nil, fmt.Errorf("asset path escapes base directory")
	}
	data, err := os.ReadFile(clean)
	if err != nil {
		return nil, fmt.Errorf("read asset file: %w", err)
	}
	return data, nil
}

func (s *fsBlobStore) Remove(path string) error {
	clean := filepath.Clean(path)
	if !s.withinBase(clean) {
		return fmt.Errorf("asset path escapes base directory")
	}
	if err := os.Remove(clean); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove asset file: %w", err)
	}
	return nil
}

// withinBase reports whether clean is the base dir itself or a descendant of
// it, guarding against `..`-style traversal in a stored path.
func (s *fsBlobStore) withinBase(clean string) bool {
	if clean == s.baseDir {
		return true
	}
	return strings.HasPrefix(clean, s.baseDir+string(os.PathSeparator))
}

// writeFileAtomic writes data to a temp file in the destination directory and
// renames it into place, so a crash mid-write never leaves a truncated asset.
func writeFileAtomic(dest string, data []byte) error {
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".asset-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o640); err != nil {
		return err
	}
	return os.Rename(tmpName, dest)
}
