package experiment

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

// BlobBytes is the byte-level blob seam the experiment service uses for large
// reports and retained artifacts. SQLite stores only opaque refs.
type BlobBytes interface {
	Put(ctx context.Context, key string, data []byte, mime string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

type blobBytesAdapter struct {
	bs blobstore.BlobStore
}

// NewBlobBytes wraps an existing BlobStore for tests or alternate storage.
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

// NewFilesystemBlobBytes returns filesystem-backed experiment blobs rooted
// under <DataDir>/experiment-blobs for the variant-aware audio-tools namespace.
func NewFilesystemBlobBytes(namespace string) (BlobBytes, error) {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{
		AppID:   "vrooli",
		Profile: corestorage.ProfileAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("experiment: resolver: %w", err)
	}
	paths, err := resolver.Resolve(corestorage.Options{ScenarioID: namespace})
	if err != nil {
		return nil, fmt.Errorf("experiment: resolve namespace %q: %w", namespace, err)
	}
	root := filepath.Join(paths.DataDir, "experiment-blobs")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("experiment: create blob root: %w", err)
	}
	return NewBlobBytes(blobstore.NewFilesystemBlobStore(root)), nil
}
