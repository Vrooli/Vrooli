// Package storage is image-tools' persistence seam for image bytes. It keeps
// pixels OUT of SQLite and out of the repo: blobs live under the api-core
// storage substrate (namespace-aware, outside-repo by default), addressed by
// opaque keys. Outputs are user-owned — a request may redirect a result to an
// arbitrary host path instead of the managed blob store.
//
// The ingest boundary is guarded against decompression bombs and oversize
// uploads (see guard.go); nothing in the rest of the scenario should accept raw
// image bytes without passing through Guard.Inspect first.
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/blobstore"
	corestorage "github.com/vrooli/api-core/storage"
)

// Store persists image blobs under api-core storage and routes outputs to
// either the managed blob store or a caller-supplied host path.
type Store struct {
	blobs blobstore.BlobStore
	root  string
}

// New resolves the blob root for the given storage namespace (use
// corestorage.ScenarioNamespace("image-tools"), which is shadow/variant aware)
// and returns a filesystem-backed Store rooted there. The root lives under the
// platform data class, outside the repo.
func New(namespace string) (*Store, error) {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{
		AppID:   "vrooli",
		Profile: corestorage.ProfileAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("storage: resolver: %w", err)
	}
	paths, err := resolver.Resolve(corestorage.Options{ScenarioID: namespace})
	if err != nil {
		return nil, fmt.Errorf("storage: resolve namespace %q: %w", namespace, err)
	}
	root := filepath.Join(paths.DataDir, "blobs")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create blob root: %w", err)
	}
	return NewWithBlobStore(blobstore.NewFilesystemBlobStore(root), root), nil
}

// NewWithBlobStore wraps an existing BlobStore (the test seam — pass an
// in-memory store, or a filesystem store rooted at a temp dir).
func NewWithBlobStore(bs blobstore.BlobStore, root string) *Store {
	return &Store{blobs: bs, root: root}
}

// Root returns the absolute blob root (diagnostics only).
func (s *Store) Root() string { return s.root }

// Put stores bytes under key with the given MIME type.
func (s *Store) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := s.blobs.Put(ctx, key, r, mime); err != nil {
		return fmt.Errorf("storage: put %q: %w", key, err)
	}
	return nil
}

// Get opens the blob stored under key, returning its reader and MIME type.
func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := validateKey(key); err != nil {
		return nil, "", err
	}
	rc, mime, err := s.blobs.Get(ctx, key)
	if err != nil {
		return nil, "", fmt.Errorf("storage: get %q: %w", key, err)
	}
	return rc, mime, nil
}

// Delete removes the blob under key (no error if already absent).
func (s *Store) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := s.blobs.Delete(ctx, key); err != nil {
		return fmt.Errorf("storage: delete %q: %w", key, err)
	}
	return nil
}

// OutputTarget selects where an operation result is written. Exactly one field
// must be set. BlobKey stores the result in the managed blob store (the
// default, returns the key as the ref). LocalPath writes the result directly to
// a host path the user owns (the per-request override, returns the path).
type OutputTarget struct {
	BlobKey   string
	LocalPath string
}

// Validate ensures exactly one target is set.
func (t OutputTarget) Validate() error {
	switch {
	case t.BlobKey != "" && t.LocalPath != "":
		return fmt.Errorf("storage: output target has both blob key and local path")
	case t.BlobKey == "" && t.LocalPath == "":
		return fmt.Errorf("storage: output target is empty")
	case t.BlobKey != "":
		return validateKey(t.BlobKey)
	default:
		if !filepath.IsAbs(t.LocalPath) {
			return fmt.Errorf("storage: local output path must be absolute: %q", t.LocalPath)
		}
		return nil
	}
}

// Write routes a result to its target and returns a reference (the blob key, or
// the absolute local path). Local writes are atomic (temp file + rename) and
// create parent directories, so a partially written file is never observable.
func (s *Store) Write(ctx context.Context, target OutputTarget, r io.Reader, mime string) (ref string, err error) {
	if err := target.Validate(); err != nil {
		return "", err
	}
	if target.BlobKey != "" {
		if err := s.Put(ctx, target.BlobKey, r, mime); err != nil {
			return "", err
		}
		return target.BlobKey, nil
	}
	if err := writeFileAtomic(target.LocalPath, r); err != nil {
		return "", fmt.Errorf("storage: write local output: %w", err)
	}
	return target.LocalPath, nil
}

// validateKey rejects keys that escape the blob root or are empty. The
// filesystem store re-checks, but validating here gives a uniform error and
// guards in-memory stores too.
func validateKey(key string) error {
	k := strings.TrimSpace(filepath.ToSlash(key))
	if k == "" {
		return fmt.Errorf("storage: key is required")
	}
	if strings.HasPrefix(k, "/") || k == ".." || strings.HasPrefix(k, "../") || strings.Contains(k, "/../") {
		return fmt.Errorf("storage: invalid key %q", key)
	}
	return nil
}

func writeFileAtomic(path string, r io.Reader) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".imgtmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if _, err := io.Copy(tmp, r); err != nil {
		cleanup()
		return fmt.Errorf("copy: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
