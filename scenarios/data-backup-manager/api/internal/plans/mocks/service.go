package mocks

import (
	"context"

	"data-backup-manager/internal/plans"
)

// FakeService satisfies plans.Service for handler tests that should not pull
// validation/repository plumbing into scope. Records inputs and returns canned
// values gated on per-method error knobs.
type FakeService struct {
	CreateInputs []plans.CreateInput
	CreateOut    plans.Plan
	CreateErr    error

	GetID  string
	GetOut plans.Plan
	GetErr error

	ListOut []plans.Plan
	ListErr error

	UpdateInputs []plans.UpdateInput
	UpdateOut    plans.Plan
	UpdateErr    error

	DeleteID  string
	DeleteOut bool
	DeleteErr error

	SchedulablePlansOut []plans.SchedulablePlan
	SchedulablePlansErr error
}

// Compile-time guarantee.
var _ plans.Service = (*FakeService)(nil)

func (f *FakeService) Create(_ context.Context, in plans.CreateInput) (plans.Plan, error) {
	f.CreateInputs = append(f.CreateInputs, in)
	if f.CreateErr != nil {
		return plans.Plan{}, f.CreateErr
	}
	return f.CreateOut, nil
}

func (f *FakeService) Get(_ context.Context, id string) (plans.Plan, error) {
	f.GetID = id
	if f.GetErr != nil {
		return plans.Plan{}, f.GetErr
	}
	return f.GetOut, nil
}

func (f *FakeService) List(_ context.Context, _ int) ([]plans.Plan, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	return f.ListOut, nil
}

func (f *FakeService) Update(_ context.Context, in plans.UpdateInput) (plans.Plan, error) {
	f.UpdateInputs = append(f.UpdateInputs, in)
	if f.UpdateErr != nil {
		return plans.Plan{}, f.UpdateErr
	}
	return f.UpdateOut, nil
}

func (f *FakeService) Delete(_ context.Context, id string) (bool, error) {
	f.DeleteID = id
	if f.DeleteErr != nil {
		return false, f.DeleteErr
	}
	return f.DeleteOut, nil
}

func (f *FakeService) SchedulablePlans(_ context.Context) ([]plans.SchedulablePlan, error) {
	if f.SchedulablePlansErr != nil {
		return nil, f.SchedulablePlansErr
	}
	return f.SchedulablePlansOut, nil
}
