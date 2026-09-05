package mocks

import (
	"context"

	"data-backup-manager/internal/engine"
	"data-backup-manager/internal/preflight"
	"data-backup-manager/internal/runs"
)

// FakeService satisfies runs.Service for handler tests.
type FakeService struct {
	TriggerOut runs.Run
	TriggerErr error
	GetOut     runs.Run
	GetErr     error
	ListOut    []runs.Run
	ListErr    error
	StatusOut  []runs.TargetStatus
	StatusErr  error
	BrowseOut  []engine.SnapshotEntry
	BrowseErr  error
	StatsOut2  runs.RunStats
	StatsErr2  error

	TriggeredPlan string
}

var _ runs.Service = (*FakeService)(nil)

func (f *FakeService) TriggerRun(_ context.Context, planID string, _ runs.TriggerSource) (runs.Run, error) {
	f.TriggeredPlan = planID
	if f.TriggerErr != nil {
		return runs.Run{}, f.TriggerErr
	}
	return f.TriggerOut, nil
}

func (f *FakeService) Preflight(context.Context, string) (preflight.Result, error) {
	return preflight.Result{Ready: true}, nil
}

func (f *FakeService) GetRun(_ context.Context, _ string) (runs.Run, error) {
	if f.GetErr != nil {
		return runs.Run{}, f.GetErr
	}
	return f.GetOut, nil
}

func (f *FakeService) ListRuns(_ context.Context, _ string, _ int) ([]runs.Run, error) {
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	return f.ListOut, nil
}

func (f *FakeService) ListTargetStatus(_ context.Context, _ []string) ([]runs.TargetStatus, error) {
	if f.StatusErr != nil {
		return nil, f.StatusErr
	}
	return f.StatusOut, nil
}

func (f *FakeService) BrowseSnapshot(_ context.Context, _, _, _ string) ([]engine.SnapshotEntry, error) {
	if f.BrowseErr != nil {
		return nil, f.BrowseErr
	}
	return f.BrowseOut, nil
}

func (f *FakeService) GetRunStats(_ context.Context, _ string) (runs.RunStats, error) {
	if f.StatsErr2 != nil {
		return runs.RunStats{}, f.StatsErr2
	}
	return f.StatsOut2, nil
}

func (f *FakeService) Reconcile(context.Context) error { return nil }

func (f *FakeService) Shutdown(context.Context) error { return nil }
