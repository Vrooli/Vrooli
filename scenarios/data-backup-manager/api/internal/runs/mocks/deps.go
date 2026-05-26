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
	return t, nil
}

// FakeDestinationLookup returns canned destinations and a programmable cap
// decision. BlockFn, when set, decides WouldBlock per destination+bytes.
type FakeDestinationLookup struct {
	Destinations map[string]runs.DestinationForRun
	Err          error
	BlockFn      func(destID string, pendingBytes int64) (bool, string, error)
}

func (f *FakeDestinationLookup) DestinationForRun(_ context.Context, destID string) (runs.DestinationForRun, error) {
	if f.Err != nil {
		return runs.DestinationForRun{}, f.Err
	}
	d, ok := f.Destinations[destID]
	if !ok {
		return runs.DestinationForRun{}, &notFound{"destination", destID}
	}
	return d, nil
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
	_ runs.PlanLookup        = (*FakePlanLookup)(nil)
	_ runs.TargetLookup      = (*FakeTargetLookup)(nil)
	_ runs.DestinationLookup = (*FakeDestinationLookup)(nil)
	_ runs.EventSink         = (*FakeEventSink)(nil)
)

type notFound struct {
	kind string
	id   string
}

func (e *notFound) Error() string { return e.kind + " " + e.id + " not found" }
