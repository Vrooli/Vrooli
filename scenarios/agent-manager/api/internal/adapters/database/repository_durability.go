package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"agent-manager/internal/durability"
)

// durabilityBoundaryRepository stores the analysis epoch as durable state.
//
// The epoch answers "from when can this project grade its own work", and the
// answer must be the same for every process that asks. An environment-derived
// value would let two readers disagree about which work is in scope, and an
// operator could not tell a deliberate scope limit from a misconfiguration.
type durabilityBoundaryRepository struct{ db *DB }

var _ durability.BoundaryStore = (*durabilityBoundaryRepository)(nil)

type durabilityBoundaryRow struct {
	Epoch         string `db:"epoch"`
	Reason        string `db:"reason"`
	EstablishedAt string `db:"established_at"`
	EstablishedBy string `db:"established_by"`
}

// Boundary returns the stored epoch, seeding it on first read. Seeding records
// the moment attribution capture actually became available on this deployment
// rather than a date compiled into the binary, and once written it is never
// recomputed.
func (r *durabilityBoundaryRepository) Boundary(ctx context.Context, seed durability.Boundary) (durability.Boundary, error) {
	var row durabilityBoundaryRow
	err := r.db.GetContext(ctx, &row, `SELECT epoch, reason, established_at, established_by FROM durability_boundary WHERE id = 1`)
	switch {
	case err == nil:
		epoch, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(row.Epoch))
		if parseErr != nil {
			return durability.Boundary{}, wrapDBError("get", "DurabilityBoundary", "singleton", parseErr)
		}
		return durability.Boundary{Epoch: epoch.UTC(), Reason: row.Reason}, nil
	case errors.Is(err, sql.ErrNoRows):
		if _, execErr := r.db.ExecContext(ctx,
			`INSERT INTO durability_boundary (id, epoch, reason, established_by) VALUES (1, ?, ?, ?)
			 ON CONFLICT (id) DO NOTHING`,
			seed.Epoch.UTC().Format(time.RFC3339), seed.Reason, "agent-manager",
		); execErr != nil {
			return durability.Boundary{}, wrapDBError("create", "DurabilityBoundary", "singleton", execErr)
		}
		// Re-read rather than returning the seed: a concurrent writer may have
		// won the insert, and every reader must agree on one epoch.
		return r.Boundary(ctx, seed)
	default:
		return durability.Boundary{}, wrapDBError("get", "DurabilityBoundary", "singleton", err)
	}
}
