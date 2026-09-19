package signals

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// routedBlobStore resolves the BlobStore root per request so test-mode image
// capture never writes into the operator's primary signal archive.
type routedBlobStore struct{ roots *filerouting.RoutedRoots }

func newRoutedBlobStore(roots *filerouting.RoutedRoots) (blobstore.BlobStore, error) {
	if roots == nil {
		return nil, fmt.Errorf("routed blob store requires file roots")
	}
	return &routedBlobStore{roots: roots}, nil
}

func (s *routedBlobStore) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	store, err := s.store(ctx)
	if err != nil {
		return err
	}
	if err := store.Put(ctx, key, r, mime); err != nil {
		return err
	}
	s.roots.RecordWrite(ctx)
	return nil
}

func (s *routedBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	store, err := s.store(ctx)
	if err != nil {
		return nil, "", err
	}
	return store.Get(ctx, key)
}

func (s *routedBlobStore) Delete(ctx context.Context, key string) error {
	store, err := s.store(ctx)
	if err != nil {
		return err
	}
	if err := store.Delete(ctx, key); err != nil {
		return err
	}
	s.roots.RecordWrite(ctx)
	return nil
}

func (s *routedBlobStore) store(ctx context.Context) (blobstore.BlobStore, error) {
	root, err := s.roots.Pick(ctx, storage.ClassData)
	if err != nil {
		return nil, fmt.Errorf("resolve routed signal blob root: %w", err)
	}
	return blobstore.NewFilesystemBlobStore(filepath.Join(root, "signals")), nil
}

var _ blobstore.BlobStore = (*routedBlobStore)(nil)
