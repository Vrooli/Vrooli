package mocks

import (
	"context"

	"data-backup-manager/internal/destinations"
)

// FakeService satisfies destinations.Service for handler tests that should not
// pull validation/repository/engine plumbing into scope. Records inputs and
// returns canned values gated on per-method error knobs.
type FakeService struct {
	// CreateDestination
	CreateInputs []destinations.CreateInput
	CreateOut    destinations.Destination
	CreateErr    error

	// GetDestination
	GetID  string
	GetOut destinations.Destination
	GetErr error

	// ListDestinations
	ListOut []destinations.Destination
	ListErr error

	// UpdateDestination
	UpdateInputs []destinations.UpdateInput
	UpdateOut    destinations.Destination
	UpdateErr    error

	// DeleteDestination
	DeleteIDs []string
	DeleteOut bool
	DeleteErr error

	// GetDestinationUsage
	UsageID  string
	UsageOut destinations.UsageReport
	UsageErr error

	// WouldBlock
	WouldBlockID      string
	WouldBlockPending int64
	WouldBlockOut     bool
	WouldBlockReason  string
	WouldBlockErr     error

	ReconcileErr error
}

// Compile-time guarantee.
var _ destinations.Service = (*FakeService)(nil)

func (f *FakeService) CreateDestination(_ context.Context, in destinations.CreateInput) (destinations.Destination, error) {
	f.CreateInputs = append(f.CreateInputs, in)
	if f.CreateErr != nil {
		return destinations.Destination{}, f.CreateErr
	}
	return f.CreateOut, nil
}

func (f *FakeService) GetDestination(_ context.Context, id string) (destinations.Destination, error) {
	f.GetID = id
	if f.GetErr != nil {
		return destinations.Destination{}, f.GetErr
	}
	return f.GetOut, nil
}

func (f *FakeService) ListDestinations(_ context.Context, _ int) ([]destinations.Destination, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	return f.ListOut, nil
}

func (f *FakeService) UpdateDestination(_ context.Context, in destinations.UpdateInput) (destinations.Destination, error) {
	f.UpdateInputs = append(f.UpdateInputs, in)
	if f.UpdateErr != nil {
		return destinations.Destination{}, f.UpdateErr
	}
	return f.UpdateOut, nil
}

func (f *FakeService) DeleteDestination(_ context.Context, id string, _ bool) (bool, error) {
	f.DeleteIDs = append(f.DeleteIDs, id)
	if f.DeleteErr != nil {
		return false, f.DeleteErr
	}
	return f.DeleteOut, nil
}

func (f *FakeService) GetDestinationUsage(_ context.Context, id string) (destinations.UsageReport, error) {
	f.UsageID = id
	if f.UsageErr != nil {
		return destinations.UsageReport{}, f.UsageErr
	}
	return f.UsageOut, nil
}

func (f *FakeService) WouldBlock(_ context.Context, destinationID string, pendingBytes int64) (bool, string, error) {
	f.WouldBlockID = destinationID
	f.WouldBlockPending = pendingBytes
	if f.WouldBlockErr != nil {
		return false, "", f.WouldBlockErr
	}
	return f.WouldBlockOut, f.WouldBlockReason, nil
}

func (f *FakeService) ReconcileCredentialReferences(_ context.Context) error {
	return f.ReconcileErr
}
