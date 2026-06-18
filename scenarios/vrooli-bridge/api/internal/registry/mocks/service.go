package mocks

import (
	"context"
	"sync/atomic"

	"vrooli-bridge/internal/registry"
)

// FakeService is a registry.Service double for handler tests — it records
// inputs and returns canned values without validation/repository plumbing.
type FakeService struct {
	RegisterInputs []registry.RegisterInput
	UpdateInputs   []registry.UpdateInput
	RevokeIDs      []string
	GetIDs         []string

	RegisterOut registry.Node
	RegisterErr error
	ListOut     []registry.Node
	ListErr     error
	GetOut      registry.Node
	GetErr      error
	UpdateOut   registry.Node
	UpdateErr   error
	RevokeOut   registry.Node
	RevokeErr   error

	RegisterCalls atomic.Int64
	RevokeCalls   atomic.Int64
}

var _ registry.Service = (*FakeService)(nil)

func (f *FakeService) Register(_ context.Context, in registry.RegisterInput) (registry.Node, error) {
	f.RegisterCalls.Add(1)
	f.RegisterInputs = append(f.RegisterInputs, in)
	return f.RegisterOut, f.RegisterErr
}

func (f *FakeService) List(_ context.Context) ([]registry.Node, error) {
	return f.ListOut, f.ListErr
}

func (f *FakeService) Get(_ context.Context, id string) (registry.Node, error) {
	f.GetIDs = append(f.GetIDs, id)
	return f.GetOut, f.GetErr
}

func (f *FakeService) Update(_ context.Context, in registry.UpdateInput) (registry.Node, error) {
	f.UpdateInputs = append(f.UpdateInputs, in)
	return f.UpdateOut, f.UpdateErr
}

func (f *FakeService) Revoke(_ context.Context, id string) (registry.Node, error) {
	f.RevokeCalls.Add(1)
	f.RevokeIDs = append(f.RevokeIDs, id)
	return f.RevokeOut, f.RevokeErr
}
