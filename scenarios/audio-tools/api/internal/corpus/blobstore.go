package corpus

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vrooli/api-core/blobstore"
	corestorage "github.com/vrooli/api-core/storage"
)

// BlobBytes is the byte-level blob seam the corpus service uses to persist
// audio. Production wraps api-core's filesystem BlobStore (rooted under the
// git-ignored runtime data dir, namespace/variant aware); tests use an
// in-memory map. Audio bytes NEVER enter git or the SQLite metadata DB.
type BlobBytes interface {
	Put(ctx context.Context, key string, data []byte, mime string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// blobBytesAdapter adapts an io.Reader-based blobstore.BlobStore to the
// byte-oriented BlobBytes seam.
type blobBytesAdapter struct {
	bs blobstore.BlobStore
}

// NewBlobBytes wraps an existing BlobStore (the test seam — pass an
// in-memory store, or a filesystem store rooted at a temp dir).
func NewBlobBytes(bs blobstore.BlobStore) BlobBytes {
	return &blobBytesAdapter{bs: bs}
}

func (a *blobBytesAdapter) Put(ctx context.Context, key string, data []byte, mime string) error {
	return a.bs.Put(ctx, key, bytes.NewReader(data), mime)
}

func (a *blobBytesAdapter) Get(ctx context.Context, key string) ([]byte, error) {
	rc, _, err := a.bs.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (a *blobBytesAdapter) Delete(ctx context.Context, key string) error {
	return a.bs.Delete(ctx, key)
}

// NewFilesystemBlobBytes resolves the blob root for the given storage
// namespace (use corestorage.ScenarioNamespace("audio-tools"), which is
// shadow/variant aware — so a shadow instance's corpus blobs never collide
// with live's) and returns a filesystem-backed BlobBytes rooted under
// <DataDir>/corpus-blobs, outside the repo. Mirrors image-tools'
// internal/storage.New.
func NewFilesystemBlobBytes(namespace string) (BlobBytes, error) {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{
		AppID:   "vrooli",
		Profile: corestorage.ProfileAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("corpus: resolver: %w", err)
	}
	paths, err := resolver.Resolve(corestorage.Options{ScenarioID: namespace})
	if err != nil {
		return nil, fmt.Errorf("corpus: resolve namespace %q: %w", namespace, err)
	}
	root := filepath.Join(paths.DataDir, "corpus-blobs")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("corpus: create blob root: %w", err)
	}
	return NewBlobBytes(blobstore.NewFilesystemBlobStore(root)), nil
}
