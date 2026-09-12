package restores

import (
	"context"

	"data-backup-manager/internal/sources"
)

// This file declares the narrow seams the restores orchestration depends on.
// Each is owned here (the consumer) and satisfied by a thin adapter over a
// sibling service, wired in main.go. Keeping them as restores-local interfaces
// means the restore/verify logic is unit-testable against fakes without
// standing up the whole domain tree, and avoids importing the sibling domain
// packages (no import cycles).

// TargetForRestore is the slice of a target the restorer needs.
type TargetForRestore struct {
	ID      string
	Kind    sources.SourceKind
	Locator string
}

// TargetLookup resolves a target id to its restore spec.
//
// seam: implemented by an adapter over targets.Service in main.go.
type TargetLookup interface {
	TargetForRestore(ctx context.Context, targetID string) (TargetForRestore, error)
}

// DestinationForRestore is the slice of a destination the restorer needs.
// Name is the kopia repository name.
type DestinationForRestore struct {
	ID   string
	Name string
}

// DestinationLookup resolves a destination id to its restore spec.
//
// seam: implemented by an adapter over destinations.Service in main.go.
type DestinationLookup interface {
	DestinationForRestore(ctx context.Context, destinationID string) (DestinationForRestore, error)
}
