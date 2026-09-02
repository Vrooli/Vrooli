package credentialgrant

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type SQLiteRepository struct {
	db  SQLExecutor
	now func() time.Time
}

func NewSQLiteRepository(db SQLExecutor, now func() time.Time) *SQLiteRepository {
	if now == nil {
		now = time.Now
	}
	return &SQLiteRepository{db: db, now: now}
}

func (r *SQLiteRepository) Create(ctx context.Context, grant Grant) (Grant, error) {
	if grant.ID == "" {
		grant.ID = uuid.NewString()
	}
	if grant.GrantedAt.IsZero() {
		grant.GrantedAt = r.now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO credential_grants (id,node_id,logical_id,field,class,retention,generation,granted_at,revoked_at,acked_generation,receipt_at,receipt_accepted,receipt_reason) VALUES (?,?,?,?,?,?,?,?,'',?,'',?,?)`, grant.ID, grant.NodeID, grant.LogicalID, grant.Field, grant.Class, grant.Retention, grant.Generation, grant.GrantedAt.UTC().Format(time.RFC3339Nano), grant.AckedGeneration, grant.ReceiptAccepted, grant.ReceiptReason)
	if err != nil {
		return Grant{}, fmt.Errorf("create credential grant: %w", err)
	}
	return grant, nil
}

func (r *SQLiteRepository) List(ctx context.Context, nodeID string) ([]Grant, error) {
	query := `SELECT id,node_id,logical_id,field,class,retention,generation,granted_at,revoked_at,acked_generation,receipt_at,receipt_accepted,receipt_reason FROM credential_grants WHERE revoked_at=''`
	args := []any{}
	if nodeID != "" {
		query += ` AND node_id=?`
		args = append(args, nodeID)
	}
	query += ` ORDER BY granted_at,id`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list credential grants: %w", err)
	}
	defer rows.Close()
	var grants []Grant
	for rows.Next() {
		var grant Grant
		var class, retention, grantedAt, revokedAt, receiptAt string
		if err := rows.Scan(&grant.ID, &grant.NodeID, &grant.LogicalID, &grant.Field, &class, &retention, &grant.Generation, &grantedAt, &revokedAt, &grant.AckedGeneration, &receiptAt, &grant.ReceiptAccepted, &grant.ReceiptReason); err != nil {
			return nil, err
		}
		grant.Class, grant.Retention = Class(class), Retention(retention)
		grant.GrantedAt, err = time.Parse(time.RFC3339Nano, grantedAt)
		if err != nil {
			return nil, err
		}
		if revokedAt != "" {
			grant.RevokedAt, err = time.Parse(time.RFC3339Nano, revokedAt)
			if err != nil {
				return nil, err
			}
		}
		if receiptAt != "" {
			grant.ReceiptAt, err = time.Parse(time.RFC3339Nano, receiptAt)
			if err != nil {
				return nil, err
			}
		}
		grants = append(grants, grant)
	}
	return grants, rows.Err()
}

func (r *SQLiteRepository) RecordReceipt(ctx context.Context, id string, generation int64, accepted bool, reason string, at time.Time) error {
	result, err := r.db.ExecContext(ctx, `UPDATE credential_grants SET receipt_at=?, receipt_accepted=?, receipt_reason=? WHERE id=? AND revoked_at=''`, at.UTC().Format(time.RFC3339Nano), accepted, reason, id)
	if err != nil {
		return fmt.Errorf("record credential receipt: %w", err)
	}
	if accepted {
		_, err = r.db.ExecContext(ctx, `UPDATE credential_grants SET acked_generation=CASE WHEN acked_generation<? THEN ? ELSE acked_generation END WHERE id=? AND revoked_at=''`, generation, generation, id)
	}
	if err != nil {
		return fmt.Errorf("ack credential receipt: %w", err)
	}
	_ = result
	return nil
}

func (r *SQLiteRepository) Revoke(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `UPDATE credential_grants SET revoked_at=? WHERE id=? AND revoked_at=''`, r.now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SQLiteRepository) Ack(ctx context.Context, id string, generation int64) error {
	result, err := r.db.ExecContext(ctx, `UPDATE credential_grants SET acked_generation=? WHERE id=? AND revoked_at='' AND acked_generation<?`, generation, id, generation)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	_ = count
	return nil
}

func (r *SQLiteRepository) BumpGeneration(ctx context.Context, logicalID, field string) (int64, error) {
	var current int64
	if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(generation),0) FROM credential_grants WHERE logical_id=? AND field=? AND revoked_at=''`, logicalID, field).Scan(&current); err != nil {
		return 0, fmt.Errorf("read credential grant generation: %w", err)
	}
	initial := current + 1
	if initial < 1 {
		initial = 1
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO credential_generations (logical_id,field,generation,updated_at) VALUES (?,?,?,?) ON CONFLICT(logical_id,field) DO UPDATE SET generation=credential_generations.generation+1, updated_at=excluded.updated_at`, logicalID, field, initial, r.now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, fmt.Errorf("bump credential generation: %w", err)
	}
	var generation int64
	if err := r.db.QueryRowContext(ctx, `SELECT generation FROM credential_generations WHERE logical_id=? AND field=?`, logicalID, field).Scan(&generation); err != nil {
		return 0, fmt.Errorf("read credential generation: %w", err)
	}
	return generation, nil
}

func (r *SQLiteRepository) SetGrantGeneration(ctx context.Context, id, nodeID string, generation int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE credential_grants SET generation=?, acked_generation=0 WHERE id=? AND node_id=? AND revoked_at=''`, generation, id, nodeID)
	if err != nil {
		return fmt.Errorf("set grant generation: %w", err)
	}
	return nil
}
