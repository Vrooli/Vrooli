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
	CASInputs      []registry.UpdateCASInput
	RevokeIDs      []string
	RemoveIDs      []string
	GetIDs         []string

	RegisterOut     registry.Node
	RegisterErr     error
	ListOut         []registry.Node
	ListErr         error
	GetOut          registry.Node
	GetErr          error
	UpdateOut       registry.Node
	CASOut          registry.UpdateCASResult
	UpdateErr       error
	CASErr          error
	RevokeOut       registry.Node
	RevokeErr       error
	RemoveErr       error
	ArchitectureErr error
	ArchitectureIDs []string
	MachineArchs    []string
	BinaryArchs     []string
	GenerationIDs   []string
	Generations     []int64

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

func (f *FakeService) GetByPairingCorrelation(_ context.Context, correlationID string) (registry.Node, error) {
	return f.Get(context.Background(), correlationID)
}

func (f *FakeService) Update(_ context.Context, in registry.UpdateInput) (registry.Node, error) {
	f.UpdateInputs = append(f.UpdateInputs, in)
	return f.UpdateOut, f.UpdateErr
}

func (f *FakeService) UpdateCAS(_ context.Context, in registry.UpdateCASInput) (registry.UpdateCASResult, error) {
	f.CASInputs = append(f.CASInputs, in)
	return f.CASOut, f.CASErr
}

func (f *FakeService) Revoke(_ context.Context, id string) (registry.Node, error) {
	f.RevokeCalls.Add(1)
	f.RevokeIDs = append(f.RevokeIDs, id)
	return f.RevokeOut, f.RevokeErr
}

func (f *FakeService) Remove(_ context.Context, id string) error {
	f.RemoveIDs = append(f.RemoveIDs, id)
	return f.RemoveErr
}

func (f *FakeService) UpdateArchitecture(_ context.Context, id, machineArch, binaryArch string) error {
	if f.ArchitectureErr != nil {
		return f.ArchitectureErr
	}
	f.ArchitectureIDs = append(f.ArchitectureIDs, id)
	f.MachineArchs = append(f.MachineArchs, machineArch)
	f.BinaryArchs = append(f.BinaryArchs, binaryArch)
	return nil
}

func (f *FakeService) SetEnrollmentGeneration(_ context.Context, id string, generation int64) (registry.Node, error) {
	f.GenerationIDs = append(f.GenerationIDs, id)
	f.Generations = append(f.Generations, generation)
	out := f.GetOut
	out.ID = id
	out.EnrollmentGeneration = generation
	return out, f.GetErr
}
