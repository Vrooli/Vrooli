package mocks

import (
	"context"
	"sync"

	"data-backup-manager/internal/destinations"
)

// FakeBundleWriter is an in-memory destinations.BundleWriter. It records the
// prepared repositories and written metadata so service tests can assert the
// bundle root / repository path / manifest fields without touching the real
// filesystem. Per-method error knobs inject failures.
type FakeBundleWriter struct {
	mu sync.Mutex

	Prepared []PreparedRepo
	Metadata []destinations.BundleMetadata

	PrepareErr  error
	MetadataErr error
}

// PreparedRepo captures one PrepareRepository call.
type PreparedRepo struct {
	BundleRoot     string
	RepositoryPath string
}

// NewFakeBundleWriter returns an empty recorder.
func NewFakeBundleWriter() *FakeBundleWriter { return &FakeBundleWriter{} }

// Compile-time guarantee.
var _ destinations.BundleWriter = (*FakeBundleWriter)(nil)

func (f *FakeBundleWriter) PrepareRepository(_ context.Context, bundleRoot, repositoryPath string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.PrepareErr != nil {
		return f.PrepareErr
	}
	f.Prepared = append(f.Prepared, PreparedRepo{BundleRoot: bundleRoot, RepositoryPath: repositoryPath})
	return nil
}

func (f *FakeBundleWriter) WriteMetadata(_ context.Context, meta destinations.BundleMetadata) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.MetadataErr != nil {
		return f.MetadataErr
	}
	f.Metadata = append(f.Metadata, meta)
	return nil
}
