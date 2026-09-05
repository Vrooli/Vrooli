// Package mocks provides in-memory fakes for the apply domain seams so service
// and handler unit tests run without touching the filesystem or the brands /
// assets / assignments domains. Mirrors internal/generation/mocks.
package mocks

import (
	"context"
	"sync"

	"brand-manager/internal/apply"
)

// FakeBrandStore satisfies apply.BrandStore. Known holds the brands that exist
// (keyed by id).
type FakeBrandStore struct {
	mu     sync.Mutex
	Known  map[string]apply.BrandView
	GetErr error
}

func (f *FakeBrandStore) Get(_ context.Context, brandID string) (apply.BrandView, error) {
	if f.GetErr != nil {
		return apply.BrandView{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.Known[brandID]
	if !ok {
		return apply.BrandView{}, apply.ErrBrandNotFound{ID: brandID}
	}
	return b, nil
}

// Seed registers a brand so Get returns it.
func (f *FakeBrandStore) Seed(b apply.BrandView) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Known == nil {
		f.Known = map[string]apply.BrandView{}
	}
	f.Known[b.ID] = b
}

var _ apply.BrandStore = (*FakeBrandStore)(nil)

// FakeAssetStore satisfies apply.AssetStore. Assets maps "<brandID>/<kind>" to
// the content returned; an absent key reports found=false.
type FakeAssetStore struct {
	mu      sync.Mutex
	Assets  map[string]apply.AssetContent
	ReadErr error
}

func (f *FakeAssetStore) Read(_ context.Context, brandID, kind string) (apply.AssetContent, bool, error) {
	if f.ReadErr != nil {
		return apply.AssetContent{}, false, f.ReadErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.Assets[brandID+"/"+kind]
	if !ok {
		return apply.AssetContent{}, false, nil
	}
	return c, true, nil
}

// Seed registers an asset for (brandID, kind).
func (f *FakeAssetStore) Seed(brandID, kind string, content apply.AssetContent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Assets == nil {
		f.Assets = map[string]apply.AssetContent{}
	}
	f.Assets[brandID+"/"+kind] = content
}

var _ apply.AssetStore = (*FakeAssetStore)(nil)

// RecordedAssignment captures a single AssignmentRecorder.Record call.
type RecordedAssignment struct {
	BrandID  string
	Scenario string
	Elements []string
}

// FakeAssignmentRecorder satisfies apply.AssignmentRecorder, recording each call.
type FakeAssignmentRecorder struct {
	mu        sync.Mutex
	Records   []RecordedAssignment
	RecordErr error
}

func (f *FakeAssignmentRecorder) Record(_ context.Context, brandID, scenario string, elements []string) error {
	if f.RecordErr != nil {
		return f.RecordErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Records = append(f.Records, RecordedAssignment{
		BrandID:  brandID,
		Scenario: scenario,
		Elements: append([]string(nil), elements...),
	})
	return nil
}

// Recorded returns a copy of the recorded assignments.
func (f *FakeAssignmentRecorder) Recorded() []RecordedAssignment {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]RecordedAssignment(nil), f.Records...)
}

var _ apply.AssignmentRecorder = (*FakeAssignmentRecorder)(nil)

// FakeWorkspace satisfies apply.Workspace with an in-memory filesystem. Scenarios
// holds the set of scenario names that "exist"; Files maps "<scenario>/<rel>" to
// its bytes and records every write.
type FakeWorkspace struct {
	mu        sync.Mutex
	Scenarios map[string]bool
	Files     map[string][]byte
	ExistsErr error
	WriteErr  error
}

func (f *FakeWorkspace) ScenarioExists(_ context.Context, scenario string) (bool, error) {
	if f.ExistsErr != nil {
		return false, f.ExistsErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Scenarios[scenario], nil
}

func (f *FakeWorkspace) ReadFile(_ context.Context, scenario, rel string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Files[scenario+"/"+rel], nil
}

func (f *FakeWorkspace) WriteFile(_ context.Context, scenario, rel string, data []byte) error {
	if f.WriteErr != nil {
		return f.WriteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Files == nil {
		f.Files = map[string][]byte{}
	}
	f.Files[scenario+"/"+rel] = append([]byte(nil), data...)
	return nil
}

// SeedScenario marks a scenario as existing.
func (f *FakeWorkspace) SeedScenario(scenario string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.Scenarios == nil {
		f.Scenarios = map[string]bool{}
	}
	f.Scenarios[scenario] = true
}

// Written returns a copy of the file at "<scenario>/<rel>", or nil if unwritten.
func (f *FakeWorkspace) Written(scenario, rel string) []byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Files[scenario+"/"+rel]
}

// WriteCount returns the number of distinct files written.
func (f *FakeWorkspace) WriteCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Files)
}

var _ apply.Workspace = (*FakeWorkspace)(nil)

// FakeService satisfies apply.Service for handler tests that drive the transport
// edge without the orchestration logic. The Func fields override per-method
// behaviour; nil fields return zero values.
type FakeService struct {
	PreviewFunc func(ctx context.Context, in apply.Request) (apply.Result, error)
	ApplyFunc   func(ctx context.Context, in apply.Request) (apply.Result, error)
}

func (f FakeService) Preview(ctx context.Context, in apply.Request) (apply.Result, error) {
	if f.PreviewFunc != nil {
		return f.PreviewFunc(ctx, in)
	}
	return apply.Result{}, nil
}

func (f FakeService) Apply(ctx context.Context, in apply.Request) (apply.Result, error) {
	if f.ApplyFunc != nil {
		return f.ApplyFunc(ctx, in)
	}
	return apply.Result{}, nil
}

var _ apply.Service = (*FakeService)(nil)
