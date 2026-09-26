package mocks

import (
	"context"

	"data-backup-manager/internal/targets"
)

// FakeService satisfies targets.Service for handler tests that should not pull
// validation/repository plumbing into scope. Records inputs and returns canned
// values gated on per-method error knobs.
type FakeService struct {
	RegisterInputs []targets.RegisterInput
	RegisterOut    targets.Target
	RegisterErr    error

	DeregisterCalls [][2]string // {owner, name}
	DeregisterOut   bool
	DeregisterErr   error

	GetByID string
	GetOut  targets.Target
	GetErr  error

	ListOwner string
	ListOut   []targets.Target
	ListErr   error
}

// Compile-time guarantee.
var _ targets.Service = (*FakeService)(nil)

func (f *FakeService) Register(_ context.Context, in targets.RegisterInput) (targets.Target, error) {
	f.RegisterInputs = append(f.RegisterInputs, in)
	if f.RegisterErr != nil {
		return targets.Target{}, f.RegisterErr
	}
	return f.RegisterOut, nil
}

func (f *FakeService) Deregister(_ context.Context, owner, name string) (bool, error) {
	f.DeregisterCalls = append(f.DeregisterCalls, [2]string{owner, name})
	if f.DeregisterErr != nil {
		return false, f.DeregisterErr
	}
	return f.DeregisterOut, nil
}

func (f *FakeService) Get(_ context.Context, id string) (targets.Target, error) {
	f.GetByID = id
	if f.GetErr != nil {
		return targets.Target{}, f.GetErr
	}
	return f.GetOut, nil
}

func (f *FakeService) List(_ context.Context, owner string, _ int) ([]targets.Target, error) {
	f.ListOwner = owner
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	return f.ListOut, nil
}
