// Package audits is the domain-scoped home for the generic snapshot audit: a
// scenario-agnostic proof that a backed-up snapshot contains the same generic
// filesystem/database shape as the live target. RunSnapshotAudit restores the
// snapshot to scratch (proving recoverability), captures the live target to
// scratch (read-only on live), walks both artifacts with one generic
// filesystem walker, and compares them by generic signals only — counts,
// bytes, path-list hash, tree content hash, and per-SQLite integrity/schema
// facts.
//
// DBM never learns any scenario's domain objects. The audit is evidence, not a
// destructive restore: it uses scratch directories only, always cleaned up,
// and never modifies live data. Records store relative paths, counts, and
// hashes — never file contents or secret values.
//
// The orchestration depends on narrow reader/effect seams declared in deps.go
// (TargetLookup, DestinationLookup), the engine.KopiaEngine, the sources
// Registry, and the Repository. None of the sibling domains import audits, so
// there is no import cycle; main.go wires thin adapters from the concrete
// services to these seams.
package audits

import (
	"fmt"
	"time"
)

// AuditStatus is the lifecycle state of a snapshot audit. RunSnapshotAudit is
// asynchronous (mirroring restores): a record is created in REQUESTED, the
// worker drives it through RUNNING to a terminal COMPLETED or FAILED.
type AuditStatus string

const (
	AuditRequested AuditStatus = "requested"
	AuditRunning   AuditStatus = "running"
	AuditCompleted AuditStatus = "completed"
	AuditFailed    AuditStatus = "failed"
)

// SqliteInventory is the generic integrity proof for one SQLite file. It holds
// no domain meaning: TableCount is a raw count of sqlite_master objects, never
// a per-domain-table interpretation, and no row contents are read.
type SqliteInventory struct {
	Path            string // relative path within the tree
	IntegrityStatus string // ok / failed / not_checked
	PageCount       int64
	PageSize        int64
	SchemaSHA256    string // sha256 of normalized sqlite_master schema
	TableCount      int64  // generic count of schema objects
}

// InventorySummary is the generic, domain-agnostic fingerprint of a directory
// tree (a restored snapshot artifact or a freshly captured live artifact).
type InventorySummary struct {
	Files           int64
	Directories     int64
	Symlinks        int64
	Other           int64 // devices/sockets/fifos: counted, never content-hashed
	RegularBytes    int64
	PathListSHA256  string
	TreeContentSHA  string // empty when content hashing was skipped
	SQLite          []SqliteInventory
	CapturedAt      time.Time
	UnreadablePaths []string
	// MaxModTime is the newest modification time seen in the tree. It is not
	// part of the wire shape; the service uses the live tree's MaxModTime to
	// decide whether a mismatch is explainable as drift (live changed after the
	// snapshot timestamp).
	MaxModTime time.Time
}

// AuditComparison is the generic verdict comparing live and snapshot
// inventories.
type AuditComparison struct {
	Matches               bool
	Mismatches            []string
	LiveNewerThanSnapshot bool
}

// Audit is the internal domain shape for one snapshot audit record.
type Audit struct {
	ID                 string
	TargetID           string
	DestinationID      string
	SnapshotID         string
	Status             AuditStatus
	IncludeContentHash bool
	IncludeSQLiteCheck bool
	Restorable         bool
	Live               *InventorySummary
	Snapshot           *InventorySummary
	Comparison         *AuditComparison
	SnapshotTime       time.Time
	RequestedAt        time.Time
	FinishedAt         time.Time
	// UpdatedAt is the heartbeat: the last time the record's status was
	// persisted, so an in-flight or wedged audit is observable and startup
	// reconciliation can age non-terminal audits (mirrors restores).
	UpdatedAt time.Time
	Error     string
}

// ErrAuditNotFound is the typed sentinel for an unknown audit id.
type ErrAuditNotFound struct{ ID string }

func (e ErrAuditNotFound) Error() string { return fmt.Sprintf("audit %q not found", e.ID) }

// ErrInvalidAudit is the typed validation sentinel.
type ErrInvalidAudit struct {
	Field  string
	Reason string
}

func (e ErrInvalidAudit) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }
