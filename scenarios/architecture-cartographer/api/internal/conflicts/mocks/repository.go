// Package mocks holds in-memory fakes for the conflicts domain.
package mocks

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"architecture-cartographer/internal/conflicts"

	"github.com/google/uuid"
)

// FakeRepository satisfies conflicts.Repository.
type FakeRepository struct {
	mu sync.Mutex

	Conflicts []conflicts.Conflict

	UpsertErr error
	GetErr    error
	ListErr   error

	UpsertCalls atomic.Int64
	GetCalls    atomic.Int64
	ListCalls   atomic.Int64
}

func (f *FakeRepository) UpsertConflict(_ context.Context, c conflicts.Conflict) (conflicts.Conflict, error) {
	f.UpsertCalls.Add(1)
	if f.UpsertErr != nil {
		return conflicts.Conflict{}, f.UpsertErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if c.ID == "" {
		c.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	if c.DetectedAt.IsZero() {
		c.DetectedAt = now
	}
	c.UpdatedAt = now
	for i, existing := range f.Conflicts {
		if existing.ID == c.ID {
			f.Conflicts[i] = c
			return c, nil
		}
	}
	f.Conflicts = append(f.Conflicts, c)
	return c, nil
}

func (f *FakeRepository) GetConflict(_ context.Context, id string) (conflicts.Conflict, error) {
	f.GetCalls.Add(1)
	if f.GetErr != nil {
		return conflicts.Conflict{}, f.GetErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.Conflicts {
		if c.ID == id {
			return c, nil
		}
	}
	return conflicts.Conflict{}, conflicts.ErrConflictNotFound{ID: id}
}

func (f *FakeRepository) ListConflicts(_ context.Context, filter conflicts.ListConflictsFilter) (conflicts.ConflictPage, error) {
	f.ListCalls.Add(1)
	if f.ListErr != nil {
		return conflicts.ConflictPage{}, f.ListErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []conflicts.Conflict
	for _, c := range f.Conflicts {
		if filter.Scenario != "" && c.Scenario != filter.Scenario {
			continue
		}
		if len(filter.Types) > 0 {
			matched := false
			for _, ty := range filter.Types {
				if ty == c.Type {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, c)
	}
	return conflicts.ConflictPage{Conflicts: out}, nil
}

var _ conflicts.Repository = (*FakeRepository)(nil)

// FakeDetector emits a canned conflict list. Used by service + registry tests.
type FakeDetector struct {
	NameValue        string
	DescriptionValue string
	TypesEmitted     []string
	Conflicts        []conflicts.Conflict
	DetectErr        error
	DetectCalls      atomic.Int64
}

func (f *FakeDetector) Name() string         { return f.NameValue }
func (f *FakeDetector) Description() string  { return f.DescriptionValue }
func (f *FakeDetector) EmitsTypes() []string { return f.TypesEmitted }
func (f *FakeDetector) Class() conflicts.FindingClass {
	if len(f.Conflicts) > 0 && f.Conflicts[0].FindingClass != conflicts.FindingClassUnspecified {
		return f.Conflicts[0].FindingClass
	}
	return conflicts.FindingClassDeterministic
}

func (f *FakeDetector) Detect(_ context.Context, _ conflicts.DetectInput) ([]conflicts.Conflict, error) {
	f.DetectCalls.Add(1)
	if f.DetectErr != nil {
		return nil, f.DetectErr
	}
	out := make([]conflicts.Conflict, len(f.Conflicts))
	copy(out, f.Conflicts)
	return out, nil
}

var _ conflicts.Detector = (*FakeDetector)(nil)

// FakeResolver implements conflicts.Resolver for service tests.
type FakeResolver struct {
	NameValue        string
	DescriptionValue string
	Kinds            []conflicts.FixKind
	NeedsApply       bool
	ResolveErr       error
	ResolveCalls     atomic.Int64
}

func (f *FakeResolver) Name() string                      { return f.NameValue }
func (f *FakeResolver) Description() string               { return f.DescriptionValue }
func (f *FakeResolver) HandlesKinds() []conflicts.FixKind { return f.Kinds }
func (f *FakeResolver) RequiresApply() bool               { return f.NeedsApply }
func (f *FakeResolver) Resolve(_ context.Context, _ conflicts.Conflict, _ conflicts.Fix) error {
	f.ResolveCalls.Add(1)
	return f.ResolveErr
}

var _ conflicts.Resolver = (*FakeResolver)(nil)
