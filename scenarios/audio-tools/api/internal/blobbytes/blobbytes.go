// Package blobbytes provides the byte-oriented adapter shared by audio-tools
// domains that persist large artifacts outside SQLite.
package blobbytes

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

// Store is the byte-level persistence seam used by domain services. Tests can
// provide an in-memory implementation while production uses BlobStore.
type Store interface {
	Put(ctx context.Context, key string, data []byte, mime string) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

type adapter struct{ store blobstore.BlobStore }

// New adapts an io.Reader-based BlobStore to Store.
func New(store blobstore.BlobStore) Store { return &adapter{store: store} }

func (a *adapter) Put(ctx context.Context, key string, data []byte, mime string) error {
	return a.store.Put(ctx, key, bytes.NewReader(data), mime)
}

func (a *adapter) Get(ctx context.Context, key string) ([]byte, error) {
	rc, _, err := a.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}

func (a *adapter) Delete(ctx context.Context, key string) error { return a.store.Delete(ctx, key) }

// NewFilesystem resolves a variant-aware scenario namespace and returns a
// filesystem-backed store under directory. Owner is included in diagnostics so
// domain-level operational errors remain actionable.
func NewFilesystem(namespace, directory, owner string) (Store, error) {
	resolver, err := corestorage.NewResolver(corestorage.ResolverConfig{AppID: "vrooli", Profile: corestorage.ProfileAuto})
	if err != nil {
		return nil, fmt.Errorf("%s: resolver: %w", owner, err)
	}
	paths, err := resolver.Resolve(corestorage.Options{ScenarioID: namespace})
	if err != nil {
		return nil, fmt.Errorf("%s: resolve namespace %q: %w", owner, namespace, err)
	}
	root := filepath.Join(paths.DataDir, directory)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("%s: create blob root: %w", owner, err)
	}
	return New(blobstore.NewFilesystemBlobStore(root)), nil
}
