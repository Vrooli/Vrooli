// Package mocks holds co-located test doubles for the discovery domain seams.
// Lives in a mocks/ directory (no _test.go suffix) so sibling _test.go files
// can import it; never linked into production.
package mocks

import (
	"context"
	"sync"

	"data-backup-manager/internal/discovery"
)

// FakeVolumeScanner returns a fixed slice of volumes (or an error).
type FakeVolumeScanner struct {
	Volumes []discovery.Volume
	Err     error
}

func (f *FakeVolumeScanner) Scan(context.Context) ([]discovery.Volume, error) {
	return f.Volumes, f.Err
}

var _ discovery.VolumeScanner = (*FakeVolumeScanner)(nil)

// FakeTargetSourceScanner returns a fixed slice of candidates (or an error).
type FakeTargetSourceScanner struct {
	Candidates []discovery.TargetCandidate
	Err        error
}

func (f *FakeTargetSourceScanner) Scan(context.Context) ([]discovery.TargetCandidate, error) {
	return f.Candidates, f.Err
}

var _ discovery.TargetSourceScanner = (*FakeTargetSourceScanner)(nil)

// FakeTargetCatalog returns a fixed list of registered targets.
type FakeTargetCatalog struct {
	Targets []discovery.ExistingTarget
	Err     error
}

func (f *FakeTargetCatalog) ListAll(context.Context) ([]discovery.ExistingTarget, error) {
	return f.Targets, f.Err
}

var _ discovery.TargetCatalog = (*FakeTargetCatalog)(nil)

// FakeDestinationCatalog returns a fixed list of registered destinations.
type FakeDestinationCatalog struct {
	Destinations []discovery.ExistingDestination
	Err          error
}

func (f *FakeDestinationCatalog) ListAll(context.Context) ([]discovery.ExistingDestination, error) {
	return f.Destinations, f.Err
}

var _ discovery.DestinationCatalog = (*FakeDestinationCatalog)(nil)

// FakeProtectedPaths returns a fixed protected-path set.
type FakeProtectedPaths struct {
	Paths []string
	Err   error
}

func (f *FakeProtectedPaths) ProtectedPaths(context.Context) ([]string, error) {
	return f.Paths, f.Err
}

var _ discovery.ProtectedPaths = (*FakeProtectedPaths)(nil)

// FakeDismissalStore is an in-memory dismissal store. Concurrency-safe so it
// can be reused across parallel subtests.
type FakeDismissalStore struct {
	mu         sync.Mutex
	dismissed  map[string]string // id -> kind
	IsErr      error
	DismissErr error
}

// NewFakeDismissalStore returns an empty store, optionally pre-seeded with
// already-dismissed ids.
func NewFakeDismissalStore(seed ...string) *FakeDismissalStore {
	d := &FakeDismissalStore{dismissed: map[string]string{}}
	for _, id := range seed {
		d.dismissed[id] = "suggestion"
	}
	return d
}

func (f *FakeDismissalStore) IsDismissed(_ context.Context, id string) (bool, error) {
	if f.IsErr != nil {
		return false, f.IsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.dismissed[id]
	return ok, nil
}

func (f *FakeDismissalStore) Dismiss(_ context.Context, id, kind string) error {
	if f.DismissErr != nil {
		return f.DismissErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.dismissed[id] = kind
	return nil
}

var _ discovery.DismissalStore = (*FakeDismissalStore)(nil)
