// Package mocks holds adoptions-domain test fakes co-located with the
// domain they double for. Deleting the domain folder takes its mocks
// with it; package graph reflects ownership (mocks imports adoptions;
// adoptions does not import mocks).
package mocks

import (
	"context"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"react-component-library/internal/adoptions"
)

// FakeRepository satisfies adoptions.Repository for service and
// handler tests that don't want the sqlite round-trip. In-memory map
// keyed by ID.
type FakeRepository struct {
	mu           sync.Mutex
	items        map[string]adoptions.Adoption
	CreateErr    error
	GetErr       error
	ListErr      error
	DeleteErr    error
	RefreshErr   error
	CreateCalls  atomic.Int64
	GetCalls     atomic.Int64
	ListCalls    atomic.Int64
	DeleteCalls  atomic.Int64
	RefreshCalls atomic.Int64
	NowFn        func() time.Time
}

func NewFakeRepository() *FakeRepository {
	return &FakeRepository{
		items: map[string]adoptions.Adoption{},
		NowFn: func() time.Time { return time.Now().UTC() },
	}
}

var _ adoptions.Repository = (*FakeRepository)(nil)

func (f *FakeRepository) Create(ctx context.Context, in adoptions.CreateInput) (adoptions.Adoption, error) {
	f.CreateCalls.Add(1)
	if f.CreateErr != nil {
		return adoptions.Adoption{}, f.CreateErr
	}
	if strings.TrimSpace(in.ComponentID) == "" {
		return adoptions.Adoption{}, adoptions.ErrInvalidAdoption{Field: "component_id", Reason: "required"}
	}
	if strings.TrimSpace(in.Scenario) == "" {
		return adoptions.Adoption{}, adoptions.ErrInvalidAdoption{Field: "scenario", Reason: "required"}
	}
	if strings.TrimSpace(in.AdoptedPath) == "" {
		return adoptions.Adoption{}, adoptions.ErrInvalidAdoption{Field: "adopted_path", Reason: "required"}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	id := strings.TrimSpace(in.ID)
	if id == "" {
		id = uuid.NewString()
	}
	a := adoptions.Adoption{
		ID:                    id,
		ComponentID:           in.ComponentID,
		LibraryID:             in.LibraryID,
		Scenario:              in.Scenario,
		AdoptedPath:           in.AdoptedPath,
		AdoptedVersion:        in.AdoptedVersion,
		SourceSHA256:          in.SourceSHA256,
		AdoptedSnapshotSHA256: in.AdoptedSnapshotSHA256,
		LibraryVersionStatus:  adoptions.LibraryVersionStatusCurrent,
		LocalStatus:           adoptions.LocalStatusClean,
		CreatedAt:             f.NowFn(),
		AppliedAt:             f.NowFn(),
		IncludeSuggestions:    append([]string(nil), in.IncludeSuggestions...),
		Files:                 append([]adoptions.AdoptionFile(nil), in.Files...),
		Mode:                  in.Mode,
	}
	if a.Mode == "" {
		a.Mode = adoptions.AdoptionModeCopied
	}
	f.items[a.ID] = a
	return a, nil
}

func (f *FakeRepository) UpdateMode(_ context.Context, id string, mode adoptions.AdoptionMode, reason string) (adoptions.Adoption, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.items[id]
	if !ok {
		return adoptions.Adoption{}, adoptions.ErrAdoptionNotFound{ID: id}
	}
	a.Mode = mode
	a.ForkReason = reason
	if reason != "" {
		a.ForkStatus = adoptions.ForkStatusDeclared
	}
	f.items[id] = a
	return a, nil
}

func (f *FakeRepository) UpdateLinked(_ context.Context, id, adoptedPath, adoptedVersion, sourceSHA256 string) (adoptions.Adoption, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.items[id]
	if !ok {
		return adoptions.Adoption{}, adoptions.ErrAdoptionNotFound{ID: id}
	}
	a.AdoptedPath = adoptedPath
	a.AdoptedVersion = adoptedVersion
	a.SourceSHA256 = sourceSHA256
	a.Mode = adoptions.AdoptionModeLinked
	a.LocalStatus = adoptions.LocalStatusClean
	a.LibraryVersionStatus = adoptions.LibraryVersionStatusCurrent
	a.ForkStatus = adoptions.ForkStatusNone
	a.ForkReason = ""
	a.Files = nil
	f.items[id] = a
	return a, nil
}

func (f *FakeRepository) CreateBatch(ctx context.Context, inputs []adoptions.CreateInput) ([]adoptions.Adoption, error) {
	created := make([]adoptions.Adoption, 0, len(inputs))
	for _, input := range inputs {
		adoption, err := f.Create(ctx, input)
		if err != nil {
			for _, prior := range created {
				_ = f.Delete(ctx, prior.ID)
			}
			return nil, err
		}
		created = append(created, adoption)
	}
	return created, nil
}

func (f *FakeRepository) UpdateAppliedSnapshot(_ context.Context, in adoptions.AppliedSnapshotUpdate) (adoptions.Adoption, error) {
	if strings.TrimSpace(in.ID) == "" {
		return adoptions.Adoption{}, adoptions.ErrInvalidAdoption{Field: "id", Reason: "required"}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.items[in.ID]
	if !ok {
		return adoptions.Adoption{}, adoptions.ErrAdoptionNotFound{ID: in.ID}
	}
	appliedAt := in.AppliedAt
	if appliedAt.IsZero() {
		appliedAt = f.NowFn()
	}
	a.AdoptedVersion = in.AdoptedVersion
	a.SourceSHA256 = in.SourceSHA256
	a.AdoptedSnapshotSHA256 = in.AdoptedSnapshotSHA256
	a.LibraryVersionStatus = adoptions.LibraryVersionStatusCurrent
	a.LocalStatus = adoptions.LocalStatusClean
	a.StatusDetail = ""
	a.RefreshedAt = appliedAt
	a.AppliedAt = appliedAt
	a.DriftBacklogRef = ""
	f.items[in.ID] = a
	return a, nil
}

func (f *FakeRepository) UpdateAppliedUnit(ctx context.Context, in adoptions.AppliedUnitUpdate) (adoptions.Adoption, error) {
	a, err := f.UpdateAppliedSnapshot(ctx, in.AppliedSnapshotUpdate)
	if err != nil {
		return adoptions.Adoption{}, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	a = f.items[in.ID]
	a.Files = append([]adoptions.AdoptionFile(nil), in.Files...)
	f.items[in.ID] = a
	return a, nil
}

func (f *FakeRepository) Rebaseline(_ context.Context, in adoptions.RebaselineInput) (adoptions.Adoption, error) {
	if strings.TrimSpace(in.ID) == "" {
		return adoptions.Adoption{}, adoptions.ErrInvalidAdoption{Field: "id", Reason: "required"}
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.items[in.ID]
	if !ok {
		return adoptions.Adoption{}, adoptions.ErrAdoptionNotFound{ID: in.ID}
	}
	a.AdoptedSnapshotSHA256 = in.AdoptedSnapshotSHA256
	if in.Files != nil {
		a.Files = append([]adoptions.AdoptionFile(nil), in.Files...)
	}
	f.items[in.ID] = a
	return a, nil
}

func (f *FakeRepository) Get(ctx context.Context, id string) (adoptions.Adoption, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return adoptions.Adoption{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	a, ok := f.items[id]
	if !ok {
		return adoptions.Adoption{}, adoptions.ErrAdoptionNotFound{ID: id}
	}
	return a, nil
}

func (f *FakeRepository) List(ctx context.Context, q adoptions.ListQuery) ([]adoptions.Adoption, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	if q.Limit <= 0 {
		return nil, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []adoptions.Adoption
	for _, a := range f.items {
		if q.ComponentID != "" && a.ComponentID != q.ComponentID {
			continue
		}
		if q.Scenario != "" && a.Scenario != q.Scenario {
			continue
		}
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

func (f *FakeRepository) ListEffective(_ context.Context, componentID string, limit int) ([]adoptions.EffectiveAdoption, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []adoptions.EffectiveAdoption
	for _, adoption := range f.items {
		for _, file := range adoption.Files {
			if file.SourceAssetID != componentID {
				continue
			}
			out = append(out, adoptions.EffectiveAdoption{SourceAssetID: file.SourceAssetID, SourceLibraryID: file.SourceLibraryID, SourceVersion: file.SourceVersion, Mediated: adoption.ComponentID != file.SourceAssetID, ParentAdoption: adoption})
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *FakeRepository) Delete(ctx context.Context, id string) error {
	f.DeleteCalls.Add(1)
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.items[id]; !ok {
		return adoptions.ErrAdoptionNotFound{ID: id}
	}
	delete(f.items, id)
	return nil
}

func (f *FakeRepository) ApplyRefresh(ctx context.Context, updates []adoptions.RefreshUpdate) (int, error) {
	f.RefreshCalls.Add(1)
	if f.RefreshErr != nil {
		return 0, f.RefreshErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	touched := 0
	for _, u := range updates {
		a, ok := f.items[u.ID]
		if !ok {
			continue
		}
		a.LibraryVersionStatus = u.LibraryVersionStatus
		a.LocalStatus = u.LocalStatus
		a.StatusDetail = u.StatusDetail
		a.RefreshedAt = u.RefreshedAt
		switch {
		case u.ClearDriftBacklogRef:
			a.DriftBacklogRef = ""
		case u.DriftBacklogRef != "":
			a.DriftBacklogRef = u.DriftBacklogRef
		}
		f.items[u.ID] = a
		touched++
	}
	return touched, nil
}

// Seed inserts a row verbatim — used by tests that need to drive
// Refresh without going through Create's required-field gate.
func (f *FakeRepository) Seed(a adoptions.Adoption) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	f.items[a.ID] = a
}
