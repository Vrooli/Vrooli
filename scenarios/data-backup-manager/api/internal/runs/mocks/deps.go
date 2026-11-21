// Package mocks holds co-located test doubles for the runs domain seams.
package mocks

import (
	"context"

	"data-backup-manager/internal/runs"
)

// FakePlanLookup returns canned plan specs keyed by plan id.
type FakePlanLookup struct {
	Plans map[string]runs.PlanForRun
	Err   error
}

func (f *FakePlanLookup) PlanForRun(_ context.Context, planID string) (runs.PlanForRun, error) {
	if f.Err != nil {
		return runs.PlanForRun{}, f.Err
	}
	p, ok := f.Plans[planID]
	if !ok {
		return runs.PlanForRun{}, &notFound{"plan", planID}
	}
	return p, nil
}

// FakeTargetLookup returns canned target specs keyed by target id.
type FakeTargetLookup struct {
	Targets map[string]runs.TargetForRun
	Err     error
}

func (f *FakeTargetLookup) TargetForRun(_ context.Context, targetID string) (runs.TargetForRun, error) {
	if f.Err != nil {
		return runs.TargetForRun{}, f.Err
	}
	t, ok := f.Targets[targetID]
	if !ok {
		return runs.TargetForRun{}, &notFound{"target", targetID}
	}
	// Real targets always carry a stable owner/name; tests that only care about
	// capture behavior should not accidentally exercise an identity failure.
	if t.Owner == "" {
		t.Owner = t.ID
	}
	if t.Name == "" {
		t.Name = "data"
	}
	return t, nil
}

// FakeActiveTargetLookup returns the current live-catalog target ids.
type FakeActiveTargetLookup struct {
	TargetIDs []string
	Err       error
}

func (f *FakeActiveTargetLookup) ActiveTargetIDs(_ context.Context) ([]string, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	return append([]string(nil), f.TargetIDs...), nil
}

// FakeDestinationLookup returns canned destinations and a programmable cap
// decision. BlockFn, when set, decides WouldBlock per destination+bytes.
type FakeDestinationLookup struct {
	Destinations   map[string]runs.DestinationForRun
	Err            error
	BlockFn        func(destID string, pendingBytes int64) (bool, string, error)
	DestinationFn  func(destID string) (runs.DestinationForRun, error)
	WithResolvedFn func(context.Context, string, func(runs.DestinationForRun) error) error
}

func (f *FakeDestinationLookup) DestinationForRun(_ context.Context, destID string) (runs.DestinationForRun, error) {
	if f.DestinationFn != nil {
		return f.DestinationFn(destID)
	}
	if f.Err != nil {
		return runs.DestinationForRun{}, f.Err
	}
	d, ok := f.Destinations[destID]
	if !ok {
		return runs.DestinationForRun{}, &notFound{"destination", destID}
	}
	return d, nil
}

func (f *FakeDestinationLookup) WithResolvedDestination(ctx context.Context, destID string, use func(runs.DestinationForRun) error) error {
	if f.WithResolvedFn != nil {
		return f.WithResolvedFn(ctx, destID, use)
	}
	d, err := f.DestinationForRun(ctx, destID)
	if err != nil {
		return err
	}
	if use == nil {
		return nil
	}
	return use(d)
}

// WithResolvedDestinationForRead shares the write fake's resolution: tests care
// about the callback contract, not the filesystem state.
func (f *FakeDestinationLookup) WithResolvedDestinationForRead(ctx context.Context, destID string, use func(runs.DestinationForRun) error) error {
	return f.WithResolvedDestination(ctx, destID, use)
}

func (f *FakeDestinationLookup) WithResolvedDestinationForWrite(ctx context.Context, destID string, use func(runs.DestinationForRun) error) error {
	return f.WithResolvedDestination(ctx, destID, use)
}

func (f *FakeDestinationLookup) WouldBlock(_ context.Context, destID string, pendingBytes int64) (bool, string, error) {
	if f.BlockFn != nil {
		return f.BlockFn(destID, pendingBytes)
	}
	return false, "", nil
}

// FakeEventSink records emitted backup-outcome events.
type FakeEventSink struct {
	Events []runs.RunOutcomeEvent
}

func (f *FakeEventSink) BackupOutcome(_ context.Context, ev runs.RunOutcomeEvent) {
	f.Events = append(f.Events, ev)
}

// Compile-time guarantees.
var (
	_ runs.PlanLookup         = (*FakePlanLookup)(nil)
	_ runs.TargetLookup       = (*FakeTargetLookup)(nil)
	_ runs.ActiveTargetLookup = (*FakeActiveTargetLookup)(nil)
	_ runs.DestinationLookup  = (*FakeDestinationLookup)(nil)
	_ runs.EventSink          = (*FakeEventSink)(nil)
)

type notFound struct {
	kind string
	id   string
}

func (e *notFound) Error() string { return e.kind + " " + e.id + " not found" }
