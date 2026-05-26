// Package restores is the domain-scoped home for restore/verify operations:
// RestoreTarget restores a snapshot to a caller-chosen location; VerifyTarget
// test-restores to a scratch directory, checksums the result, records
// last-verified-at, and cleans up — it is the gate that must pass before any
// committed runtime data is removed from git. A false "verified" is the worst
// possible failure (OT-P0-006): the service invariants and lifecycle model
// actively prevent it.
//
// The orchestration depends on narrow reader/effect seams this package declares
// (deps.go): TargetLookup, DestinationLookup, the KopiaEngine, the sources
// Registry, and the Repository. None of the sibling domains import restores, so
// there is no import cycle; main.go wires thin adapters from the concrete
// services to these seams.
package restores

import (
	"fmt"
	"time"
)

// RestoreMode distinguishes a real restore from a verify test-restore.
type RestoreMode string

const (
	ModeRestore RestoreMode = "restore"
	ModeVerify  RestoreMode = "verify"
)

// RestoreStatus is the lifecycle state of a restore/verify operation. The legal
// transitions and invariants live in lifecycle.go (the Level-2 workflow model).
type RestoreStatus string

const (
	RestoreRequested RestoreStatus = "requested"
	RestoreRestoring RestoreStatus = "restoring"
	RestoreVerifying RestoreStatus = "verifying"
	RestoreVerified  RestoreStatus = "verified"
	RestoreRestored  RestoreStatus = "restored"
	RestoreFailed    RestoreStatus = "failed"
)

// Restore is the internal domain shape for one restore/verify operation.
type Restore struct {
	ID             string
	TargetID       string
	DestinationID  string
	SnapshotID     string
	Mode           RestoreMode
	Status         RestoreStatus
	Location       string    // restore destination path (restore mode); empty for verify
	Checksum       string    // sha256 of restored artifact (verify mode evidence)
	LastVerifiedAt time.Time // set when a verify succeeds; zero otherwise
	RequestedAt    time.Time
	FinishedAt     time.Time
	Error          string
}

// VerifiedStatus is the latest successful verify for one target, derived from
// restore history. It is the "proven-restorable" half of a target's posture
// (the runs rollup owns the "backed up" half). Powers the verified-restore-age
// metric and the unverified chip without a per-target restore-history fan-out.
type VerifiedStatus struct {
	TargetID       string
	LastVerifiedAt time.Time
	SnapshotID     string // snapshot that was proven restorable
}

// ErrRestoreNotFound is the typed sentinel for an unknown restore id.
type ErrRestoreNotFound struct{ ID string }

func (e ErrRestoreNotFound) Error() string { return fmt.Sprintf("restore %q not found", e.ID) }

// ErrInvalidRestore is the typed validation sentinel.
type ErrInvalidRestore struct {
	Field  string
	Reason string
}

func (e ErrInvalidRestore) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }
