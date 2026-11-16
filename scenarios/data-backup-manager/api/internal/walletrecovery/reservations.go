package walletrecovery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrReserved                      = errors.New("target×destination is reserved for wallet recovery attestation")
	ErrReadReserved                  = errors.New("target×destination is reserved for active wallet recovery")
	ErrCommitAcknowledgementRequired = errors.New("commit-pending wallet recovery reservation requires a verifiable Treasury commit acknowledgement before release")
)

type Reservation struct {
	RequestID, TargetID, DestinationID, SnapshotID string
	TombstoneSHA256                                string
	SnapshotGeneration                             int64
	ExpiresAt                                      time.Time
}

// ReservationStore is the durable CAS boundary. Reserve is idempotent for the
// same request and snapshot; another live request cannot replace it.
type ReservationStore interface {
	Reserve(context.Context, Reservation, time.Time) error
	Release(context.Context, string, string) error
	Commit(context.Context, string, string) error
}

// Generation returns the durable source-mutation generation used to bind
// evidence collection to the snapshot reservation CAS.
func (s SQLiteReservations) Generation(ctx context.Context, targetID, destinationID string) (int64, error) {
	if err := s.EnsureSchema(ctx); err != nil {
		return 0, err
	}
	var generation int64
	err := s.DB.QueryRowContext(ctx, `SELECT generation FROM dbm_wallet_snapshot_generation WHERE target_id=? AND destination_id=?`, targetID, destinationID).Scan(&generation)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return generation, err
}

// MutationGate is shared with the runs owner. Every snapshot producing path
// acquires this before reading/capturing source data and holds it through the
// Kopia commit. That makes Reserve reject already-running work as well as block
// work started after the reservation.
type MutationGate interface {
	BeginMutation(context.Context, string, string) (release func(), err error)
}

type SQLiteReservations struct {
	DB    *sql.DB
	Lease time.Duration
}

func (s SQLiteReservations) EnsureSchema(ctx context.Context) error {
	if s.DB == nil {
		return errors.New("wallet recovery reservation database is required")
	}
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS dbm_wallet_recovery_reservations (
	 target_id TEXT NOT NULL, destination_id TEXT NOT NULL, request_id TEXT NOT NULL,
	 snapshot_id TEXT NOT NULL, tombstone_sha256 TEXT NOT NULL DEFAULT '', snapshot_generation INTEGER NOT NULL DEFAULT 0, expires_at TEXT NOT NULL, state TEXT NOT NULL,
	 PRIMARY KEY(target_id,destination_id), UNIQUE(request_id))`)
	if err != nil {
		return fmt.Errorf("ensure wallet recovery reservation schema: %w", err)
	}
	if err := ensureReservationColumn(ctx, s.DB, "tombstone_sha256", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureReservationColumn(ctx, s.DB, "snapshot_generation", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS dbm_wallet_snapshot_mutations (
	 target_id TEXT NOT NULL, destination_id TEXT NOT NULL, mutation_id TEXT NOT NULL,
	 started_at TEXT NOT NULL, PRIMARY KEY(target_id,destination_id,mutation_id))`)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS dbm_wallet_snapshot_generation (
	 target_id TEXT NOT NULL, destination_id TEXT NOT NULL, generation INTEGER NOT NULL,
	 PRIMARY KEY(target_id,destination_id))`)
	return err
}

// ReserveRead pins an exact current DBM snapshot while Treasury consumes it.
// The caller first compares the requested snapshot with DBM's catalog; this
// transaction closes the race with snapshot producers by checking the same
// generation and active-mutation table used by BeginMutation.
func (s SQLiteReservations) ReserveRead(ctx context.Context, r Reservation, now time.Time) error {
	if strings.TrimSpace(r.RequestID) == "" || strings.TrimSpace(r.TargetID) == "" || strings.TrimSpace(r.DestinationID) == "" || strings.TrimSpace(r.SnapshotID) == "" || r.SnapshotGeneration < 0 || r.ExpiresAt.IsZero() {
		return errors.New("complete read reservation identity and expiry required")
	}
	if err := s.EnsureSchema(ctx); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var active int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM dbm_wallet_snapshot_mutations WHERE target_id=? AND destination_id=?`, r.TargetID, r.DestinationID).Scan(&active); err != nil {
		return err
	}
	if active != 0 {
		return fmt.Errorf("snapshot mutation already in flight: %w", ErrReadReserved)
	}
	var generation int64
	err = tx.QueryRowContext(ctx, `SELECT generation FROM dbm_wallet_snapshot_generation WHERE target_id=? AND destination_id=?`, r.TargetID, r.DestinationID).Scan(&generation)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if generation != r.SnapshotGeneration {
		return errors.New("newest recovery snapshot changed before read reservation")
	}
	var requestID, snapshotID, expires, state string
	var storedGeneration int64
	err = tx.QueryRowContext(ctx, `SELECT request_id,snapshot_id,snapshot_generation,expires_at,state FROM dbm_wallet_recovery_reservations WHERE target_id=? AND destination_id=?`, r.TargetID, r.DestinationID).Scan(&requestID, &snapshotID, &storedGeneration, &expires, &state)
	if err == nil {
		until, parseErr := time.Parse(time.RFC3339Nano, expires)
		if parseErr != nil {
			return fmt.Errorf("corrupt wallet recovery reservation expiry; failing closed: %w", parseErr)
		}
		if state != "reserved" && state != "commit_pending" && state != "read_reserved" && state != "released" {
			return fmt.Errorf("unknown reservation state %q; failing closed", state)
		}
		if state == "commit_pending" || state == "reserved" && until.After(now) {
			return ErrReserved
		}
		if state == "read_reserved" && until.After(now) {
			if requestID != r.RequestID || snapshotID != r.SnapshotID || storedGeneration != r.SnapshotGeneration {
				return ErrReadReserved
			}
			_, err = tx.ExecContext(ctx, `UPDATE dbm_wallet_recovery_reservations SET expires_at=? WHERE target_id=? AND destination_id=? AND request_id=? AND state='read_reserved'`, r.ExpiresAt.UTC().Format(time.RFC3339Nano), r.TargetID, r.DestinationID, r.RequestID)
			if err != nil {
				return err
			}
			return tx.Commit()
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO dbm_wallet_recovery_reservations(target_id,destination_id,request_id,snapshot_id,tombstone_sha256,snapshot_generation,expires_at,state)
	 VALUES(?,?,?,?, '', ?,?,'read_reserved') ON CONFLICT(target_id,destination_id) DO UPDATE SET request_id=excluded.request_id,snapshot_id=excluded.snapshot_id,tombstone_sha256='',snapshot_generation=excluded.snapshot_generation,expires_at=excluded.expires_at,state='read_reserved'`, r.TargetID, r.DestinationID, r.RequestID, r.SnapshotID, r.SnapshotGeneration, r.ExpiresAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

// RenewRead extends an active read lease but never resurrects an expired one.
func (s SQLiteReservations) RenewRead(ctx context.Context, requestID, targetID string, expiresAt, now time.Time) error {
	if strings.TrimSpace(requestID) == "" || strings.TrimSpace(targetID) == "" || !expiresAt.After(now) {
		return errors.New("active read lease identity and future expiry are required")
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var currentExpiry string
	var reservedGeneration int64
	err = tx.QueryRowContext(ctx, `SELECT expires_at,snapshot_generation FROM dbm_wallet_recovery_reservations WHERE request_id=? AND target_id=? AND state='read_reserved'`, requestID, targetID).Scan(&currentExpiry, &reservedGeneration)
	if err != nil {
		return errors.New("active read lease not found")
	}
	until, err := time.Parse(time.RFC3339Nano, currentExpiry)
	if err != nil || !until.After(now) {
		return errors.New("read lease expired; acquire a fresh lease")
	}
	var active int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM dbm_wallet_snapshot_mutations WHERE target_id=? AND destination_id=(SELECT destination_id FROM dbm_wallet_recovery_reservations WHERE request_id=? AND target_id=?)`, targetID, requestID, targetID).Scan(&active); err != nil {
		return err
	}
	if active != 0 {
		return errors.New("snapshot mutation started while read lease was expiring")
	}
	var destinationID string
	if err := tx.QueryRowContext(ctx, `SELECT destination_id FROM dbm_wallet_recovery_reservations WHERE request_id=? AND target_id=?`, requestID, targetID).Scan(&destinationID); err != nil {
		return err
	}
	var generation int64
	err = tx.QueryRowContext(ctx, `SELECT generation FROM dbm_wallet_snapshot_generation WHERE target_id=? AND destination_id=?`, targetID, destinationID).Scan(&generation)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if generation != reservedGeneration {
		return errors.New("snapshot generation changed while read lease was active")
	}
	if _, err := tx.ExecContext(ctx, `UPDATE dbm_wallet_recovery_reservations SET expires_at=? WHERE request_id=? AND target_id=? AND state='read_reserved'`, expiresAt.UTC().Format(time.RFC3339Nano), requestID, targetID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s SQLiteReservations) ReleaseRead(ctx context.Context, requestID, targetID string) error {
	if strings.TrimSpace(requestID) == "" || strings.TrimSpace(targetID) == "" {
		return errors.New("read lease identity is required")
	}
	var state string
	err := s.DB.QueryRowContext(ctx, `SELECT state FROM dbm_wallet_recovery_reservations WHERE request_id=? AND target_id=?`, requestID, targetID).Scan(&state)
	if err != nil {
		return errors.New("read lease not found")
	}
	if state == "released" {
		return nil
	}
	if state != "read_reserved" {
		return errors.New("reservation is not an active read lease")
	}
	_, err = s.DB.ExecContext(ctx, `UPDATE dbm_wallet_recovery_reservations SET state='released' WHERE request_id=? AND target_id=? AND state='read_reserved'`, requestID, targetID)
	return err
}

func (s SQLiteReservations) Reserve(ctx context.Context, r Reservation, now time.Time) error {
	if err := normalizeReservation(r); err != nil {
		return err
	}
	if err := s.EnsureSchema(ctx); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var active int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM dbm_wallet_snapshot_mutations WHERE target_id=? AND destination_id=?`, r.TargetID, r.DestinationID).Scan(&active); err != nil {
		return err
	}
	if active > 0 {
		return errors.New("snapshot mutation already in flight for target×destination")
	}
	var generation int64
	err = tx.QueryRowContext(ctx, `SELECT generation FROM dbm_wallet_snapshot_generation WHERE target_id=? AND destination_id=?`, r.TargetID, r.DestinationID).Scan(&generation)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if generation != r.SnapshotGeneration {
		return errors.New("snapshot mutation occurred after evidence collection; recollect evidence")
	}
	var req, snap, tombstone, expires, state string
	err = tx.QueryRowContext(ctx, `SELECT request_id,snapshot_id,tombstone_sha256,expires_at,state FROM dbm_wallet_recovery_reservations WHERE target_id=? AND destination_id=?`, r.TargetID, r.DestinationID).Scan(&req, &snap, &tombstone, &expires, &state)
	if err == nil {
		until, parseErr := time.Parse(time.RFC3339Nano, expires)
		if parseErr != nil {
			return fmt.Errorf("corrupt reservation expiry; failing closed: %w", parseErr)
		}
		if state != "reserved" && state != "commit_pending" && state != "read_reserved" && state != "released" {
			return fmt.Errorf("unknown reservation state %q; failing closed", state)
		}
		if req == r.RequestID && snap == r.SnapshotID && tombstone == r.TombstoneSHA256 && (state == "reserved" || state == "commit_pending") {
			if state == "reserved" && !until.After(now) {
				_, err = tx.ExecContext(ctx, `UPDATE dbm_wallet_recovery_reservations SET expires_at=? WHERE target_id=? AND destination_id=? AND request_id=?`, r.ExpiresAt.UTC().Format(time.RFC3339Nano), r.TargetID, r.DestinationID, r.RequestID)
				if err != nil {
					return err
				}
			} else if state == "commit_pending" && !until.After(now) {
				_, err = tx.ExecContext(ctx, `UPDATE dbm_wallet_recovery_reservations SET expires_at=? WHERE target_id=? AND destination_id=? AND request_id=?`, r.ExpiresAt.UTC().Format(time.RFC3339Nano), r.TargetID, r.DestinationID, r.RequestID)
				if err != nil {
					return err
				}
			}
			return tx.Commit()
		}
		if state == "commit_pending" {
			return ErrReserved
		}
		if (state == "reserved" || state == "read_reserved") && until.After(now) {
			return ErrReserved
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO dbm_wallet_recovery_reservations(target_id,destination_id,request_id,snapshot_id,tombstone_sha256,snapshot_generation,expires_at,state)
	 VALUES(?,?,?,?,?,?,?,'reserved') ON CONFLICT(target_id,destination_id) DO UPDATE SET request_id=excluded.request_id,snapshot_id=excluded.snapshot_id,tombstone_sha256=excluded.tombstone_sha256,snapshot_generation=excluded.snapshot_generation,expires_at=excluded.expires_at,state='reserved'`, r.TargetID, r.DestinationID, r.RequestID, r.SnapshotID, r.TombstoneSHA256, r.SnapshotGeneration, r.ExpiresAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s SQLiteReservations) Release(ctx context.Context, requestID, targetID string) error {
	return s.finish(ctx, requestID, targetID, "released")
}
func (s SQLiteReservations) Commit(ctx context.Context, requestID, targetID string) error {
	return s.finish(ctx, requestID, targetID, "commit_pending")
}
func (s SQLiteReservations) finish(ctx context.Context, requestID, targetID, state string) error {
	if s.DB == nil {
		return errors.New("wallet recovery reservation database is required")
	}
	if state == "commit_pending" {
		var current, expiry string
		err := s.DB.QueryRowContext(ctx, `SELECT state,expires_at FROM dbm_wallet_recovery_reservations WHERE request_id=? AND target_id=?`, requestID, targetID).Scan(&current, &expiry)
		if err != nil {
			return errors.New("reservation not found")
		}
		if current == "commit_pending" {
			return nil
		}
		if current != "reserved" {
			return errors.New("reservation is not active")
		}
		until, err := time.Parse(time.RFC3339Nano, expiry)
		if err != nil {
			return errors.New("reservation expiry is corrupt")
		}
		if !until.After(time.Now().UTC()) {
			return errors.New("reservation expired before commit pin")
		}
		_, err = s.DB.ExecContext(ctx, `UPDATE dbm_wallet_recovery_reservations SET state='commit_pending' WHERE request_id=? AND target_id=? AND state='reserved'`, requestID, targetID)
		return err
	}
	if state == "released" {
		var current string
		err := s.DB.QueryRowContext(ctx, `SELECT state FROM dbm_wallet_recovery_reservations WHERE request_id=? AND target_id=?`, requestID, targetID).Scan(&current)
		if err != nil {
			return errors.New("reservation not found")
		}
		if current == "released" {
			return nil
		}
		if current == "commit_pending" {
			return ErrCommitAcknowledgementRequired
		}
		if current != "reserved" && current != "commit_pending" {
			return errors.New("reservation cannot be released from its current state")
		}
		_, err = s.DB.ExecContext(ctx, `UPDATE dbm_wallet_recovery_reservations SET state='released' WHERE request_id=? AND target_id=? AND state=?`, requestID, targetID, current)
		return err
	}
	res, err := s.DB.ExecContext(ctx, `UPDATE dbm_wallet_recovery_reservations SET state=? WHERE request_id=? AND target_id=? AND state='reserved'`, state, requestID, targetID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("reservation not found, expired, or already finalized")
	}
	return nil
}

func (s SQLiteReservations) BeginMutation(ctx context.Context, targetID, destinationID string) (func(), error) {
	if strings.TrimSpace(targetID) == "" || strings.TrimSpace(destinationID) == "" {
		return nil, errors.New("target and destination are required")
	}
	if err := s.EnsureSchema(ctx); err != nil {
		return nil, err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var requestID, expires, state string
	err = tx.QueryRowContext(ctx, `SELECT request_id,expires_at,state FROM dbm_wallet_recovery_reservations WHERE target_id=? AND destination_id=?`, targetID, destinationID).Scan(&requestID, &expires, &state)
	if err == nil {
		until, pe := time.Parse(time.RFC3339Nano, expires)
		if pe != nil {
			return nil, fmt.Errorf("corrupt reservation expiry; failing closed: %w", pe)
		}
		if state != "reserved" && state != "commit_pending" && state != "read_reserved" && state != "released" {
			return nil, fmt.Errorf("unknown reservation state %q; failing closed", state)
		}
		if state == "commit_pending" || (state == "reserved" || state == "read_reserved") && until.After(time.Now().UTC()) {
			return nil, ErrReserved
		}
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	if _, err = tx.ExecContext(ctx, `INSERT INTO dbm_wallet_snapshot_generation(target_id,destination_id,generation) VALUES(?,?,1) ON CONFLICT(target_id,destination_id) DO UPDATE SET generation=generation+1`, targetID, destinationID); err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO dbm_wallet_snapshot_mutations(target_id,destination_id,mutation_id,started_at) VALUES(?,?,?,?)`, targetID, destinationID, id, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	var once bool
	return func() {
		if once {
			return
		}
		once = true
		_, _ = s.DB.ExecContext(context.Background(), `DELETE FROM dbm_wallet_snapshot_mutations WHERE target_id=? AND destination_id=? AND mutation_id=?`, targetID, destinationID, id)
	}, nil
}

func normalizeReservation(r Reservation) error {
	if strings.TrimSpace(r.RequestID) == "" || strings.TrimSpace(r.TargetID) == "" || strings.TrimSpace(r.DestinationID) == "" || strings.TrimSpace(r.SnapshotID) == "" || !isSHA256(r.TombstoneSHA256) || r.SnapshotGeneration < 0 || r.ExpiresAt.IsZero() {
		return errors.New("complete reservation identity and expiry required")
	}
	return nil
}

func ensureReservationColumn(ctx context.Context, db *sql.DB, name, definition string) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(dbm_wallet_recovery_reservations)")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var col, typ string
		var def any
		if err := rows.Scan(&cid, &col, &typ, &notnull, &def, &pk); err != nil {
			return err
		}
		if col == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE dbm_wallet_recovery_reservations ADD COLUMN "+name+" "+definition); err != nil {
		return fmt.Errorf("add wallet recovery reservation column %s: %w", name, err)
	}
	return nil
}
