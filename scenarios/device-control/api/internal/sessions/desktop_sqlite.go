package sessions

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/vrooli/api-core/database"
)

//go:embed desktop.sql
var desktopSchema string

func DesktopSchema() string { return desktopSchema }

// SQLiteDesktopRepository keeps destination-local admission and receipts
// durable. One row represents one OS desktop session, shared by manual control
// and future flow callers. Storage is hidden from the helper's controller.
type SQLiteDesktopRepository struct {
	db          *sql.DB
	destination string
}

func NewSQLiteDesktopRepository(ctx context.Context, db *sql.DB, destination string) (*SQLiteDesktopRepository, error) {
	if db == nil || destination == "" {
		return nil, ErrDesktopAdmission
	}
	if err := database.EnsureSchemas(ctx, db, database.SchemaProviderFunc(DesktopSchema)); err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO device_control_desktop_sessions(destination,state) VALUES (?, '{}')`, destination); err != nil {
		return nil, err
	}
	return &SQLiteDesktopRepository{db: db, destination: destination}, nil
}

func (r *SQLiteDesktopRepository) Update(ctx context.Context, change func(*DesktopState) error) error {
	tx, err := r.acquire(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var raw string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM device_control_desktop_sessions WHERE destination=?`, r.destination).Scan(&raw); err != nil {
		return err
	}
	var state DesktopState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return err
	}
	state.grantRevoked = func(id string) (bool, error) {
		var count int
		err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM device_control_desktop_revocations WHERE destination=? AND grant_id=?`, r.destination, id).Scan(&count)
		return count != 0, err
	}
	if err := change(&state); err != nil {
		return err
	}
	if err := r.persistCleanup(ctx, tx, &state); err != nil {
		return err
	}
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE device_control_desktop_sessions SET state=? WHERE destination=?`, string(data), r.destination); err != nil {
		return err
	}
	return tx.Commit()
}

// Retry only lock acquisition, before change can perform an external effect.
// A competing capture/lease tick is ordinary contention, not helper failure.
func (r *SQLiteDesktopRepository) acquire(ctx context.Context) (*sql.Tx, error) {
	deadline := time.Now().Add(5 * time.Second)
	for {
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		result, err := tx.ExecContext(ctx, `UPDATE device_control_desktop_sessions SET state=state WHERE destination=?`, r.destination)
		if err == nil {
			count, countErr := result.RowsAffected()
			if countErr == nil && count == 1 {
				return tx, nil
			}
			tx.Rollback()
			if countErr != nil {
				return nil, countErr
			}
			return nil, fmt.Errorf("desktop state missing")
		}
		tx.Rollback()
		var sqliteError interface{ Code() int }
		if !errors.As(err, &sqliteError) || (sqliteError.Code()&255 != 5 && sqliteError.Code()&255 != 6) || !time.Now().Before(deadline) {
			return nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

// ReadCleanup reads exact historical identity without native work or a state
// update. This storage method is not an authorization boundary; an RPC caller
// must authenticate the original actor independently of the revoked input grant.
func (r *SQLiteDesktopRepository) ReadCleanup(ctx context.Context, lease DesktopLease) (DesktopCleanupReceipt, error) {
	var raw string
	if err := r.db.QueryRowContext(ctx, `SELECT receipt FROM device_control_desktop_cleanup_receipts WHERE destination=? AND epoch=?`, r.destination, strconv.FormatUint(lease.Epoch, 10)).Scan(&raw); err != nil {
		return DesktopCleanupReceipt{}, err
	}
	var receipt DesktopCleanupReceipt
	if err := json.Unmarshal([]byte(raw), &receipt); err != nil {
		return DesktopCleanupReceipt{}, err
	}
	if !sameDesktopLease(receipt.Lease, lease) {
		return DesktopCleanupReceipt{}, ErrDesktopAdmission
	}
	return receipt, nil
}

// Receipt writes and live authority removal commit atomically. Live JSON retains
// only failed cleanups needed by destination lifecycle reconciliation.
func (r *SQLiteDesktopRepository) persistCleanup(ctx context.Context, tx *sql.Tx, state *DesktopState) error {
	for epoch, receipt := range state.CleanupReceipts {
		key := strconv.FormatUint(epoch, 10)
		var raw string
		err := tx.QueryRowContext(ctx, `SELECT receipt FROM device_control_desktop_cleanup_receipts WHERE destination=? AND epoch=?`, r.destination, key).Scan(&raw)
		if err == nil {
			var previous DesktopCleanupReceipt
			if err := json.Unmarshal([]byte(raw), &previous); err != nil {
				return err
			}
			if !sameDesktopLease(previous.Lease, receipt.Lease) {
				return ErrDesktopAdmission
			}
			if previous.Released {
				receipt = previous
			}
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		data, err := json.Marshal(receipt)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO device_control_desktop_cleanup_receipts(destination,epoch,receipt) VALUES(?,?,?) ON CONFLICT(destination,epoch) DO UPDATE SET receipt=excluded.receipt`, r.destination, key, string(data)); err != nil {
			return err
		}
		if receipt.Released {
			delete(state.CleanupReceipts, epoch)
		}
	}
	return nil
}

// ReadRevokedCleanup serializes revocation and absence proof with Open/Act.
// The caller must verify the exact owner-signed revocation before entering here.
// No native effect occurs; an existing lease or failed cleanup remains pending.
func (r *SQLiteDesktopRepository) ReadRevokedCleanup(ctx context.Context, lease DesktopLease, grantID string) (DesktopCleanupReceipt, error) {
	if grantID == "" || lease.Ref.Validate() != nil {
		return DesktopCleanupReceipt{}, ErrDesktopAdmission
	}
	tx, err := r.acquire(ctx)
	if err != nil {
		return DesktopCleanupReceipt{}, err
	}
	defer tx.Rollback()
	metadata, err := json.Marshal(lease)
	if err != nil {
		return DesktopCleanupReceipt{}, err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO device_control_desktop_revocations(destination,grant_id,lease_metadata) VALUES(?,?,?) ON CONFLICT(destination,grant_id) DO UPDATE SET lease_metadata=excluded.lease_metadata WHERE lease_metadata=excluded.lease_metadata`, r.destination, grantID, string(metadata))
	if err != nil {
		return DesktopCleanupReceipt{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return DesktopCleanupReceipt{}, err
	}
	if n != 1 {
		return DesktopCleanupReceipt{}, ErrDesktopAdmission
	}
	var raw string
	if err := tx.QueryRowContext(ctx, `SELECT state FROM device_control_desktop_sessions WHERE destination=?`, r.destination).Scan(&raw); err != nil {
		return DesktopCleanupReceipt{}, err
	}
	var state DesktopState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return DesktopCleanupReceipt{}, err
	}
	receipt := DesktopCleanupReceipt{Lease: lease, Released: true, ObservedAt: time.Now().UTC()}
	err = tx.QueryRowContext(ctx, `SELECT receipt FROM device_control_desktop_cleanup_receipts WHERE destination=? AND epoch=?`, r.destination, strconv.FormatUint(lease.Epoch, 10)).Scan(&raw)
	if err == nil {
		var recorded DesktopCleanupReceipt
		if err := json.Unmarshal([]byte(raw), &recorded); err != nil {
			return DesktopCleanupReceipt{}, err
		}
		if sameDesktopLease(recorded.Lease, lease) {
			receipt = recorded
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return DesktopCleanupReceipt{}, err
	}
	if state.Lease != nil && state.Lease.Ref == lease.Ref {
		receipt.Released = false
	}
	for _, pending := range state.CleanupReceipts {
		if pending.Lease.Ref == lease.Ref && !pending.Released {
			receipt.Released = false
		}
	}
	if err := tx.Commit(); err != nil {
		return DesktopCleanupReceipt{}, err
	}
	return receipt, nil
}
