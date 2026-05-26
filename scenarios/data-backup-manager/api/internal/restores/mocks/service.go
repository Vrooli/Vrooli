package mocks

import (
	"context"

	"data-backup-manager/internal/restores"
)

// FakeService satisfies restores.Service for handler tests.
type FakeService struct {
	RestoreOut  restores.Restore
	RestoreErr  error
	VerifyOut   restores.Restore
	VerifyErr   error
	GetOut      restores.Restore
	GetErr      error
	ListOut     []restores.Restore
	ListErr     error
	VerifiedOut []restores.VerifiedStatus
	VerifiedErr error

	// Recorded calls for assertions.
	RestoreTargetCalls []restoreTargetCall
	VerifyTargetCalls  []verifyTargetCall
}

type restoreTargetCall struct {
	TargetID, DestinationID, SnapshotID, Location string
}

type verifyTargetCall struct {
	TargetID, DestinationID, SnapshotID string
}

var _ restores.Service = (*FakeService)(nil)

func (f *FakeService) RestoreTarget(_ context.Context, targetID, destinationID, snapshotID, location string) (restores.Restore, error) {
	f.RestoreTargetCalls = append(f.RestoreTargetCalls, restoreTargetCall{targetID, destinationID, snapshotID, location})
	if f.RestoreErr != nil {
		return restores.Restore{}, f.RestoreErr
	}
	return f.RestoreOut, nil
}

func (f *FakeService) VerifyTarget(_ context.Context, targetID, destinationID, snapshotID string) (restores.Restore, error) {
	f.VerifyTargetCalls = append(f.VerifyTargetCalls, verifyTargetCall{targetID, destinationID, snapshotID})
	if f.VerifyErr != nil {
		return restores.Restore{}, f.VerifyErr
	}
	return f.VerifyOut, nil
}

func (f *FakeService) GetRestore(_ context.Context, _ string) (restores.Restore, error) {
	if f.GetErr != nil {
		return restores.Restore{}, f.GetErr
	}
	return f.GetOut, nil
}

func (f *FakeService) ListRestores(_ context.Context, _ string, _ int) ([]restores.Restore, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	return f.ListOut, nil
}

func (f *FakeService) LastVerifiedByTarget(_ context.Context, _ []string) ([]restores.VerifiedStatus, error) {
	if f.VerifiedErr != nil {
		return nil, f.VerifiedErr
	}
	return f.VerifiedOut, nil
}
