// Package mocks provides in-memory fakes for the discovery domain seams so
// service and handler unit tests run without touching the filesystem or the
// brands domain. Mirrors internal/apply/mocks.
package mocks

import (
	"context"
	"strconv"
	"sync"

	"brand-manager/internal/discovery"
)

// FakeScanner satisfies discovery.Scanner with an in-memory filesystem.
// Scenarios holds the names that "exist"; Files maps "<scenario>/<rel>" to its
// bytes; Dirs maps "<scenario>/<rel>" to its entry names.
type FakeScanner struct {
	mu        sync.Mutex
	Scenarios map[string]bool
	Files     map[string][]byte
	Dirs      map[string][]string
	ExistsErr error
	ReadErr   error
	ListErr   error
}

func (f *FakeScanner) ScenarioExists(_ context.Context, scenario string) (bool, error) {
	if f.ExistsErr != nil {
		return false, f.ExistsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Scenarios[scenario], nil
}

func (f *FakeScanner) ReadFile(_ context.Context, scenario, rel string) ([]byte, error) {
	if f.ReadErr != nil {
		return nil, f.ReadErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Files[scenario+"/"+rel], nil
}

func (f *FakeScanner) ListDir(_ context.Context, scenario, rel string) ([]string, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Dirs[scenario+"/"+rel], nil
}

// SeedScenario marks a scenario as existing.
func (f *FakeScanner) SeedScenario(scenario string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Scenarios == nil {
		f.Scenarios = map[string]bool{}
	}
	f.Scenarios[scenario] = true
}

// SeedFile registers a file's bytes at "<scenario>/<rel>".
func (f *FakeScanner) SeedFile(scenario, rel string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Files == nil {
		f.Files = map[string][]byte{}
	}
	f.Files[scenario+"/"+rel] = data
}

// SeedDir registers the entry names directly under "<scenario>/<rel>".
func (f *FakeScanner) SeedDir(scenario, rel string, entries []string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Dirs == nil {
		f.Dirs = map[string][]string{}
	}
	f.Dirs[scenario+"/"+rel] = entries
}

var _ discovery.Scanner = (*FakeScanner)(nil)

// FakeBrandStore satisfies discovery.BrandStore, recording each created draft and
// minting a sequential id.
type FakeBrandStore struct {
	mu        sync.Mutex
	Created   []discovery.DraftBrand
	CreateErr error
	nextID    int
}

func (f *FakeBrandStore) Create(_ context.Context, draft discovery.DraftBrand) (discovery.Created, error) {
	if f.CreateErr != nil {
		return discovery.Created{}, f.CreateErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextID++
	f.Created = append(f.Created, draft)
	return discovery.Created{
		ID:      "brand-" + strconv.Itoa(f.nextID),
		Name:    draft.Name,
		Version: 1,
	}, nil
}

// Recorded returns a copy of the drafts created.
func (f *FakeBrandStore) Recorded() []discovery.DraftBrand {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]discovery.DraftBrand(nil), f.Created...)
}

var _ discovery.BrandStore = (*FakeBrandStore)(nil)

// FakeService satisfies discovery.Service for handler tests that drive the
// transport edge without the orchestration logic. The Func fields override
// per-method behaviour; nil fields return zero values.
type FakeService struct {
	DiscoverFunc func(ctx context.Context, scenario string) (discovery.Result, error)
	ImportFunc   func(ctx context.Context, scenario string) (discovery.ImportResult, error)
}

func (f FakeService) Discover(ctx context.Context, scenario string) (discovery.Result, error) {
	if f.DiscoverFunc != nil {
		return f.DiscoverFunc(ctx, scenario)
	}
	return discovery.Result{}, nil
}

func (f FakeService) Import(ctx context.Context, scenario string) (discovery.ImportResult, error) {
	if f.ImportFunc != nil {
		return f.ImportFunc(ctx, scenario)
	}
	return discovery.ImportResult{}, nil
}

var _ discovery.Service = (*FakeService)(nil)
