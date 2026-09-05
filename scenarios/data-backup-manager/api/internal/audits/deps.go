package audits

import (
	"context"

	"data-backup-manager/internal/sources"
)

// This file declares the narrow seams the audit orchestration depends on. Each
// is owned here (the consumer) and satisfied by a thin adapter over a sibling
// service, wired in main.go. Keeping them as audits-local interfaces means the
// audit logic is unit-testable against fakes without standing up the whole
// domain tree, and avoids importing the sibling domain packages (no import
// cycles).

// TargetForAudit is the slice of a target the auditor needs: the source kind
// and locator so it can capture the live target generically.
type TargetForAudit struct {
	ID      string
	Kind    sources.SourceKind
	Locator string
}

// TargetLookup resolves a target id to its audit spec.
//
// seam: implemented by an adapter over targets.Service in main.go.
type TargetLookup interface {
	TargetForAudit(ctx context.Context, targetID string) (TargetForAudit, error)
}

// DestinationForAudit is the slice of a destination the auditor needs. Name is
// the kopia repository name.
type DestinationForAudit struct {
	ID   string
	Name string
}

// DestinationLookup resolves a destination id to its audit spec.
//
// seam: implemented by an adapter over destinations.Service in main.go.
type DestinationLookup interface {
	DestinationForAudit(ctx context.Context, destinationID string) (DestinationForAudit, error)
}
