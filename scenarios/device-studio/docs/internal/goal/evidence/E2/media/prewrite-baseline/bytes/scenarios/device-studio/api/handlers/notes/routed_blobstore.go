package notes

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

// routedFilesystemBlobStore selects the attachment root from each request
// context. It retains the roots router, never a path selected for a request.
type routedFilesystemBlobStore struct {
	roots *filerouting.RoutedRoots
}

func newRoutedFilesystemBlobStore(roots *filerouting.RoutedRoots) (blobstore.BlobStore, error) {
	if roots == nil {
		return nil, fmt.Errorf("notes attachment file roots are required")
	}
	return &routedFilesystemBlobStore{roots: roots}, nil
}

func (s *routedFilesystemBlobStore) store(ctx context.Context) (blobstore.BlobStore, error) {
	root, err := s.roots.PickRequired(ctx, storage.ClassData)
	if err != nil {
		return nil, fmt.Errorf("resolve notes attachment data root: %w", err)
	}
	return blobstore.NewFilesystemBlobStore(filepath.Join(root, "attachments")), nil
}

func (s *routedFilesystemBlobStore) Put(ctx context.Context, key string, r io.Reader, mime string) error {
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

func (s *routedFilesystemBlobStore) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	store, err := s.store(ctx)
	if err != nil {
		return nil, "", err
	}
	return store.Get(ctx, key)
}

func (s *routedFilesystemBlobStore) Delete(ctx context.Context, key string) error {
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
