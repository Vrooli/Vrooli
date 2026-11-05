package audits

import "context"

// Repository persists snapshot audit records.
//
// seam: Repository persists audit history. Production wires SqliteRepository
// (sqlite.go); tests wire mocks.FakeRepository.
type Repository interface {
	// CreateAudit persists a new audit record and returns it with ID set.
	CreateAudit(ctx context.Context, a Audit) (Audit, error)

	// UpdateAuditStatus persists a (non-terminal) status transition and bumps
	// the record's heartbeat. Used by the worker for requested→running.
	UpdateAuditStatus(ctx context.Context, id string, status AuditStatus) error

	// FinishAudit persists the terminal record: status, restorable, the live and
	// snapshot inventories, the comparison, snapshot_time, finished_at, and any
	// error. It is the single terminal-write the worker and startup
	// reconciliation share (reconciliation passes a failed record with no
	// inventories).
	FinishAudit(ctx context.Context, a Audit) error

	// ListNonTerminalAudits returns records left in a non-terminal status
	// (requested/running) — the orphans startup reconciliation closes.
	ListNonTerminalAudits(ctx context.Context) ([]Audit, error)

	// GetAudit returns the audit record by id, or ErrAuditNotFound.
	GetAudit(ctx context.Context, id string) (Audit, error)

	// ListAudits returns audit records newest-first, optionally filtered by
	// target id. limit <= 0 returns no rows.
	ListAudits(ctx context.Context, targetID string, limit int) ([]Audit, error)
}
