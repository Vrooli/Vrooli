// Package mocks provides in-memory fakes for the assets domain seams so service
// unit tests run without the sqlite round-trip or real disk. Mirrors
// internal/brands/mocks.
package mocks

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"brand-manager/internal/assets"

	"github.com/google/uuid"
)

// FakeRepository satisfies assets.Repository with an in-memory map keyed by ID,
// plus a (brand_id, filename) index so Upsert mirrors the sqlite natural-key
// conflict behaviour. Per-method error knobs drive failure paths; atomic
// counters keep `go test -race` quiet under fan-out.
type FakeRepository struct {
	mu     sync.Mutex
	byID   map[string]assets.Asset
	byName map[string]string // "brandID\x00filename" -> id

	UpsertErr error
	GetErr    error
	ListErr   error
	DeleteErr error

	UpsertCalls atomic.Int64
	DeleteCalls atomic.Int64
}

func nameKey(brandID, filename string) string { return brandID + "\x00" + filename }

func (f *FakeRepository) Upsert(_ context.Context, a assets.Asset) (assets.Asset, error) {
	f.UpsertCalls.Add(1)
	if f.UpsertErr != nil {
		return assets.Asset{}, f.UpsertErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensure()
	key := nameKey(a.BrandID, a.Filename)
	if existingID, ok := f.byName[key]; ok {
		// Replace in place: preserve original id + created_at.
		existing := f.byID[existingID]
		existing.MimeType = a.MimeType
		existing.FilePath = a.FilePath
		existing.Size = a.Size
		f.byID[existingID] = existing
		return existing, nil
	}
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	f.byID[a.ID] = a
	f.byName[key] = a.ID
	return a, nil
}

func (f *FakeRepository) Get(_ context.Context, id string) (assets.Asset, error) {
	if f.GetErr != nil {
		return assets.Asset{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.byID[id]
	if !ok {
		return assets.Asset{}, assets.ErrAssetNotFound{ID: id}
	}
	return a, nil
}

func (f *FakeRepository) ListByBrand(_ context.Context, brandID string) ([]assets.Asset, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]assets.Asset, 0, len(f.byID))
	for _, a := range f.byID {
		if brandID == "" || a.BrandID == brandID {
			out = append(out, a)
		}
	}
	// Newest-uploaded first, id as a stable tiebreaker.
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

func (f *FakeRepository) Delete(_ context.Context, id string) error {
	f.DeleteCalls.Add(1)
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.byID[id]
	if !ok {
		return assets.ErrAssetNotFound{ID: id}
	}
	delete(f.byID, id)
	delete(f.byName, nameKey(a.BrandID, a.Filename))
	return nil
}

// Seed inserts a directly, bypassing Upsert, for arranging test state.
func (f *FakeRepository) Seed(a assets.Asset) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ensure()
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	f.byID[a.ID] = a
	f.byName[nameKey(a.BrandID, a.Filename)] = a.ID
}

func (f *FakeRepository) ensure() {
	if f.byID == nil {
		f.byID = map[string]assets.Asset{}
	}
	if f.byName == nil {
		f.byName = map[string]string{}
	}
}

var _ assets.Repository = (*FakeRepository)(nil)

// FakeBlobStore satisfies assets.BlobStore with an in-memory map keyed by the
// path it returns from Put. Per-method error knobs drive failure paths.
type FakeBlobStore struct {
	mu    sync.Mutex
	files map[string][]byte

	PutErr    error
	GetErr    error
	RemoveErr error

	RemoveCalls atomic.Int64
}

func (f *FakeBlobStore) Put(brandID, filename string, data []byte) (string, error) {
	if f.PutErr != nil {
		return "", f.PutErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.files == nil {
		f.files = map[string][]byte{}
	}
	path := "mem://" + brandID + "/" + filename
	stored := append([]byte(nil), data...)
	f.files[path] = stored
	return path, nil
}

func (f *FakeBlobStore) Get(path string) ([]byte, error) {
	if f.GetErr != nil {
		return nil, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	data, ok := f.files[path]
	if !ok {
		return nil, assets.ErrAssetNotFound{ID: path}
	}
	return append([]byte(nil), data...), nil
}

func (f *FakeBlobStore) Remove(path string) error {
	f.RemoveCalls.Add(1)
	if f.RemoveErr != nil {
		return f.RemoveErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.files, path)
	return nil
}

var _ assets.BlobStore = (*FakeBlobStore)(nil)

// FakeBrandResolver satisfies assets.BrandResolver. Known holds the brand ids
// that exist; Err drives the storage-failure path.
type FakeBrandResolver struct {
	Known map[string]bool
	Err   error
}

func (f FakeBrandResolver) BrandExists(_ context.Context, brandID string) (bool, error) {
	if f.Err != nil {
		return false, f.Err
	}
	return f.Known[brandID], nil
}

var _ assets.BrandResolver = (*FakeBrandResolver)(nil)
