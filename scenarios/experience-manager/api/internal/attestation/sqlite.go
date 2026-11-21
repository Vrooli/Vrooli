package attestation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SQLExecutor is the narrow database surface used by the ledger.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

type sqliteRepository struct {
	db SQLExecutor
}

// NewSQLiteRepository constructs the production attestation repository.
func NewSQLiteRepository(db SQLExecutor) Repository {
	return &sqliteRepository{db: db}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) AppendAttestation(ctx context.Context, a Attestation) error {
	if strings.TrimSpace(a.ID) == "" {
		return fmt.Errorf("attestation requires id")
	}
	if strings.TrimSpace(a.Scenario) == "" || strings.TrimSpace(a.PageID) == "" || strings.TrimSpace(a.ClaimID) == "" {
		return fmt.Errorf("attestation requires scenario, page, and claim")
	}
	if strings.TrimSpace(a.Author) == "" || strings.TrimSpace(a.Rationale) == "" || strings.TrimSpace(a.ExpiresAt) == "" {
		return fmt.Errorf("attestation requires author, rationale, and expires_at")
	}
	_, err := r.db.ExecContext(ctx, `
INSERT INTO manual_attestations (
  id, scenario, page_id, claim_id, author, rationale, expires_at, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.Scenario, a.PageID, a.ClaimID, a.Author, a.Rationale, a.ExpiresAt, a.CreatedAt)
	if err != nil {
		return fmt.Errorf("append attestation %q: %w", a.ID, err)
	}
	return nil
}

func (r *sqliteRepository) ListAttestations(ctx context.Context, filter Filter) ([]Attestation, error) {
	query := `SELECT id, scenario, page_id, claim_id, author, rationale, expires_at, created_at FROM manual_attestations`
	var clauses []string
	var args []any
	if filter.Scenario != "" {
		clauses = append(clauses, "scenario = ?")
		args = append(args, filter.Scenario)
	}
	if filter.PageID != "" {
		clauses = append(clauses, "page_id = ?")
		args = append(args, filter.PageID)
	}
	if filter.ClaimID != "" {
		clauses = append(clauses, "claim_id = ?")
		args = append(args, filter.ClaimID)
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY created_at DESC, id DESC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list attestations: %w", err)
	}
	defer rows.Close()

	var out []Attestation
	for rows.Next() {
		var a Attestation
		if err := rows.Scan(&a.ID, &a.Scenario, &a.PageID, &a.ClaimID, &a.Author, &a.Rationale, &a.ExpiresAt, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan attestation: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attestations: %w", err)
	}
	return out, nil
}
