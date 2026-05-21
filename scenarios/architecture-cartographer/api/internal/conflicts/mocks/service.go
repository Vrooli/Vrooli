package mocks

import (
	"context"
	"sync"
	"sync/atomic"

	"architecture-cartographer/internal/conflicts"
)

// FakeService satisfies conflicts.Service for handler tests.
type FakeService struct {
	mu sync.Mutex

	Conflicts []conflicts.Conflict
	Detectors []conflicts.DetectorDescriptor
	Resolvers []conflicts.ResolverDescriptor
	NextErr   error

	UpsertCalls   atomic.Int64
	DetectCalls   atomic.Int64
	ValidateCalls atomic.Int64
	GetCalls      atomic.Int64
	AssignCalls   atomic.Int64
	ResolveCalls  atomic.Int64
	ReopenCalls   atomic.Int64
	ListCalls     atomic.Int64
	ListDetCalls  atomic.Int64
	ListResCalls  atomic.Int64
}

func (f *FakeService) DetectConflicts(_ context.Context, in conflicts.DetectOrchestrationInput) ([]conflicts.Conflict, error) {
	f.DetectCalls.Add(1)
	if f.NextErr != nil {
		return nil, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]conflicts.Conflict, len(f.Conflicts))
	copy(out, f.Conflicts)
	for i := range out {
		if out[i].Scenario == "" {
			out[i].Scenario = in.Scenario
		}
	}
	return out, nil
}

func (f *FakeService) ValidateConflicts(_ context.Context, _ string) ([]conflicts.Conflict, bool, error) {
	f.ValidateCalls.Add(1)
	if f.NextErr != nil {
		return nil, false, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]conflicts.Conflict, 0, len(f.Conflicts))
	clean := true
	for _, c := range f.Conflicts {
		if c.Status == conflicts.ResolutionStatusResolved ||
			c.Status == conflicts.ResolutionStatusForceResolved ||
			c.Status == conflicts.ResolutionStatusCommitted {
			continue
		}
		out = append(out, c)
		if c.Severity == conflicts.SeverityError {
			clean = false
		}
	}
	return out, clean, nil
}

func (f *FakeService) UpsertConflicts(_ context.Context, scenario string, in []conflicts.Conflict) ([]conflicts.Conflict, error) {
	f.UpsertCalls.Add(1)
	if f.NextErr != nil {
		return nil, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range in {
		c.Scenario = scenario
		f.Conflicts = append(f.Conflicts, c)
	}
	out := make([]conflicts.Conflict, len(in))
	copy(out, in)
	return out, nil
}

func (f *FakeService) GetConflict(_ context.Context, id string) (conflicts.Conflict, error) {
	f.GetCalls.Add(1)
	if f.NextErr != nil {
		return conflicts.Conflict{}, f.NextErr
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

func (f *FakeService) AssignConflict(_ context.Context, id, domain, note string, dryRun bool) (conflicts.Conflict, bool, error) {
	f.AssignCalls.Add(1)
	if f.NextErr != nil {
		return conflicts.Conflict{}, dryRun, f.NextErr
	}
	return conflicts.Conflict{ID: id, AssignedDomain: domain, ResolutionNote: note}, dryRun, nil
}

func (f *FakeService) ResolveConflict(_ context.Context, id, note string, force, dryRun bool) (conflicts.Conflict, bool, bool, error) {
	f.ResolveCalls.Add(1)
	if f.NextErr != nil {
		return conflicts.Conflict{}, dryRun, false, f.NextErr
	}
	target := conflicts.ResolutionStatusResolved
	if force {
		target = conflicts.ResolutionStatusForceResolved
	}
	return conflicts.Conflict{ID: id, Status: target, ResolutionNote: note}, dryRun, false, nil
}

func (f *FakeService) ReopenConflict(_ context.Context, id, note string, dryRun bool) (conflicts.Conflict, bool, error) {
	f.ReopenCalls.Add(1)
	if f.NextErr != nil {
		return conflicts.Conflict{}, dryRun, f.NextErr
	}
	return conflicts.Conflict{ID: id, Status: conflicts.ResolutionStatusDetected, ResolutionNote: note}, dryRun, nil
}

func (f *FakeService) ListConflicts(_ context.Context, _ conflicts.ListConflictsFilter) (conflicts.ConflictPage, error) {
	f.ListCalls.Add(1)
	if f.NextErr != nil {
		return conflicts.ConflictPage{}, f.NextErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]conflicts.Conflict, len(f.Conflicts))
	copy(out, f.Conflicts)
	return conflicts.ConflictPage{Conflicts: out}, nil
}

func (f *FakeService) ListDetectors(_ context.Context) []conflicts.DetectorDescriptor {
	f.ListDetCalls.Add(1)
	return f.Detectors
}

func (f *FakeService) ListResolvers(_ context.Context) []conflicts.ResolverDescriptor {
	f.ListResCalls.Add(1)
	return f.Resolvers
}

var _ conflicts.Service = (*FakeService)(nil)
