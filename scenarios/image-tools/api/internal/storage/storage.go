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
	"github.com/vrooli/api-core/filerouting"
	corestorage "github.com/vrooli/api-core/storage"
)

// Store persists image blobs under api-core storage and routes outputs to
// either the managed blob store or a caller-supplied host path.
type Store struct {
	blobs     blobstore.BlobStore
	root      string
	cacheRoot string
	roots     *filerouting.RoutedRoots
}

// New resolves the blob root for the given storage namespace (use
// corestorage.ScenarioNamespace("image-tools"), which is shadow/variant aware)
// and returns a filesystem-backed Store rooted there. The root lives under the
// platform data class, outside the repo.
func New(namespace string) (*Store, error) {
	paths, err := ResolvePaths(namespace)
	if err != nil {
		return nil, err
	}
	return NewWithRoutedRoots(paths, nil)
}

// ResolvePaths resolves the scenario's primary storage roots. Model assets and
// model assets stay in data; regenerable outputs and runtime material use cache.
func ResolvePaths(namespace string) (corestorage.Paths, error) {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{
		AppID:   "vrooli",
		Profile: corestorage.ProfileAuto,
	})
	if err != nil {
		return corestorage.Paths{}, fmt.Errorf("storage: resolver: %w", err)
	}
	paths, err := resolver.Resolve(corestorage.Options{ScenarioID: namespace})
	if err != nil {
		return corestorage.Paths{}, fmt.Errorf("storage: resolve namespace %q: %w", namespace, err)
	}
	return paths, nil
}

// NewWithRoutedRoots keeps inputs and retained preparations in data, and
// derived out/ blobs in cache. Both classes honor active leased roots per call.
func NewWithRoutedRoots(paths corestorage.Paths, routes *filerouting.RoutedRoots) (*Store, error) {
	root := filepath.Join(paths.DataDir, "blobs")
	// Switching the writer must never strand existing opaque out/ references.
	// Placement migrations run while the owner is stopped; do not perform an
	// uncoordinated copy or introduce a permanently ambiguous two-root lookup.
	if _, err := os.Stat(filepath.Join(root, "out")); err == nil {
		return nil, fmt.Errorf("storage: legacy derived outputs remain at %s; stop image-tools and migrate that tree to %s through Storage Manager placement before starting", filepath.Join(root, "out"), filepath.Join(paths.CacheDir, "blobs", "out"))
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("storage: inspect legacy output placement: %w", err)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create blob root: %w", err)
	}
	return &Store{blobs: blobstore.NewFilesystemBlobStore(root), root: root, cacheRoot: filepath.Join(paths.CacheDir, "blobs"), roots: routes}, nil
}

// NewWithBlobStore wraps an existing BlobStore (the test seam — pass an
// in-memory store, or a filesystem store rooted at a temp dir).
func NewWithBlobStore(bs blobstore.BlobStore, root string) *Store {
	return &Store{blobs: bs, root: root}
}

// Root returns the absolute blob root (diagnostics only).
func (s *Store) Root() string { return s.root }

// PathForKey returns the filesystem path for a managed blob key. It is a
// read-only diagnostics seam for the owner cleanup provider; deletion still
// goes through Store.Delete so the blob-store abstraction remains authoritative.
func (s *Store) PathForKey(key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	return s.PathForKeyContext(context.Background(), key)
}

// PathForKeyContext resolves the managed path using the request's root routing.
func (s *Store) PathForKeyContext(ctx context.Context, key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	root, err := s.blobRoot(ctx, key)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, filepath.FromSlash(key)), nil
}

func (s *Store) blobRoot(ctx context.Context, key string) (string, error) {
	class := corestorage.ClassData
	root := s.root
	if strings.HasPrefix(key, "out/") && s.cacheRoot != "" {
		class, root = corestorage.ClassCache, s.cacheRoot
	}
	if s.roots == nil {
		return root, nil
	}
	dataRoot, err := s.roots.PickRequired(ctx, class)
	if err != nil {
		return "", fmt.Errorf("storage: select routed blob root: %w", err)
	}
	return filepath.Join(dataRoot, "blobs"), nil
}

func (s *Store) blobStore(ctx context.Context, key string) (blobstore.BlobStore, error) {
	if s.roots == nil && (s.cacheRoot == "" || !strings.HasPrefix(key, "out/")) {
		return s.blobs, nil
	}
	root, err := s.blobRoot(ctx, key)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("storage: create blob root: %w", err)
	}
	return blobstore.NewFilesystemBlobStore(root), nil
}

// Put stores bytes under key with the given MIME type.
func (s *Store) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	bs, err := s.blobStore(ctx, key)
	if err != nil {
		return err
	}
	if err := bs.Put(ctx, key, r, mime); err != nil {
		return fmt.Errorf("storage: put %q: %w", key, err)
	}
	if s.roots != nil {
		s.roots.RecordWrite(ctx)
	}
	return nil
}

// Get opens the blob stored under key, returning its reader and MIME type.
func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	if err := validateKey(key); err != nil {
		return nil, "", err
	}
	bs, err := s.blobStore(ctx, key)
	if err != nil {
		return nil, "", err
	}
	rc, mime, err := bs.Get(ctx, key)
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
	bs, err := s.blobStore(ctx, key)
	if err != nil {
		return err
	}
	if err := bs.Delete(ctx, key); err != nil {
		return fmt.Errorf("storage: delete %q: %w", key, err)
	}
	if s.roots != nil {
		s.roots.RecordWrite(ctx)
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
