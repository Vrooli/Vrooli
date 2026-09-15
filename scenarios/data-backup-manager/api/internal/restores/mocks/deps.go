// Package mocks holds co-located test doubles for the restores domain seams.
package mocks

import (
	"context"

	"data-backup-manager/internal/restores"
)

// FakeTargetLookup returns canned target specs keyed by target id.
type FakeTargetLookup struct {
	Targets map[string]restores.TargetForRestore
	Err     error
}

func (f *FakeTargetLookup) TargetForRestore(_ context.Context, targetID string) (restores.TargetForRestore, error) {
	if f.Err != nil {
		return restores.TargetForRestore{}, f.Err
	}
	t, ok := f.Targets[targetID]
	if !ok {
		return restores.TargetForRestore{}, &notFound{"target", targetID}
	}
	return t, nil
}

// FakeDestinationLookup returns canned destination specs keyed by destination id.
type FakeDestinationLookup struct {
	Destinations map[string]restores.DestinationForRestore
	Err          error
}

func (f *FakeDestinationLookup) DestinationForRestore(_ context.Context, destID string) (restores.DestinationForRestore, error) {
	if f.Err != nil {
		return restores.DestinationForRestore{}, f.Err
	}
	d, ok := f.Destinations[destID]
	if !ok {
		return restores.DestinationForRestore{}, &notFound{"destination", destID}
	}
	return d, nil
}

// Compile-time guarantees.
var (
	_ restores.TargetLookup      = (*FakeTargetLookup)(nil)
	_ restores.DestinationLookup = (*FakeDestinationLookup)(nil)
)

type notFound struct {
	kind string
	id   string
}

func (e *notFound) Error() string { return e.kind + " " + e.id + " not found" }
