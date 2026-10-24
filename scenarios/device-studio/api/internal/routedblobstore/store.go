// Package routedblobstore creates filesystem blobstores from the storage root
// selected for each request. It keeps domain data on the per-user data class
// and honors an active Test Genie file-root lease.
package routedblobstore

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/blobstore"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
)

type Store struct {
	roots     *filerouting.RoutedRoots
	namespace string
}

func New(roots *filerouting.RoutedRoots, namespace string) (*Store, error) {
	namespace = strings.TrimSpace(namespace)
	if roots == nil {
		return nil, fmt.Errorf("routed blobstore roots are required")
	}
	if namespace == "" || namespace == "." || namespace == ".." || strings.ContainsAny(namespace, `/\\`) {
		return nil, fmt.Errorf("routed blobstore namespace must be one path segment")
	}
	return &Store{roots: roots, namespace: namespace}, nil
}

func (s *Store) forRequest(ctx context.Context) (blobstore.BlobStore, error) {
	root, err := s.roots.PickRequired(ctx, storage.ClassData)
	if err != nil {
		return nil, fmt.Errorf("resolve %s storage data root: %w", s.namespace, err)
	}
	return blobstore.NewFilesystemBlobStore(filepath.Join(root, s.namespace)), nil
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, mime string) error {
	store, err := s.forRequest(ctx)
	if err != nil {
		return err
	}
	if err := store.Put(ctx, key, r, mime); err != nil {
		return err
	}
	s.roots.RecordWrite(ctx)
	return nil
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, string, error) {
	store, err := s.forRequest(ctx)
	if err != nil {
		return nil, "", err
	}
	return store.Get(ctx, key)
}

func (s *Store) Delete(ctx context.Context, key string) error {
	store, err := s.forRequest(ctx)
	if err != nil {
		return err
	}
	if err := store.Delete(ctx, key); err != nil {
		return err
	}
	s.roots.RecordWrite(ctx)
	return nil
}
