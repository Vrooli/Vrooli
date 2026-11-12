package capacity

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const diskReservationTTL = 15 * time.Minute

var ErrDiskAdmissionDenied = errors.New("capacity: disk reservation denied")

// DiskReservation describes peak additional bytes for one complete job. Path
// is an existing directory on the target filesystem. FloorBytes is the free
// space that must remain after this and every live reservation are accounted.
type DiskReservation struct {
	ID           string    `json:"id"`
	OwnerID      string    `json:"owner_id"`
	FilesystemID string    `json:"filesystem_id"`
	Path         string    `json:"path"`
	PeakBytes    int64     `json:"peak_bytes"`
	FloorBytes   int64     `json:"floor_bytes"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Available    int64     `json:"available_bytes"`
	Reserved     int64     `json:"other_reserved_bytes"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// DiskReservationRecord is the persisted legacy lease receipt. It deliberately
// excludes Available and Reserved because those values are admission-time
// observations and cannot be reconstructed from the durable row alone.
type DiskReservationRecord struct {
	ID           string    `json:"id"`
	OwnerID      string    `json:"owner_id"`
	FilesystemID string    `json:"filesystem_id"`
	Path         string    `json:"path"`
	PeakBytes    int64     `json:"peak_bytes"`
	FloorBytes   int64     `json:"floor_bytes"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// DiskReservationFilter selects durable reservation receipts from the shared ledger.
type DiskReservationFilter struct {
	FilesystemID string
	OwnerID      string
	ActiveOnly   bool
}

// ReserveDisk atomically measures filesystem headroom against all active
// reservations in the shared capacity ledger. Callers must retain the lease
// for the whole acquisition/render lifetime, heartbeat it, and release it on
// every terminal path.
func (s *SQLiteStore) ReserveDisk(ctx context.Context, ownerID, dir string, peakBytes, floorBytes int64) (DiskReservation, error) {
	if s == nil || s.db == nil || strings.TrimSpace(ownerID) == "" || peakBytes <= 0 || floorBytes < 0 {
		return DiskReservation{}, fmt.Errorf("%w: owner, existing path, positive peak, and nonnegative floor are required", ErrInvalidClaim)
	}
	floorBytes = maxInt64(floorBytes, s.diskFloorMinimum)
	unlock, err := s.LockAdmission(ctx)
	if err != nil {
		return DiskReservation{}, fmt.Errorf("lock disk admission: %w", err)
	}
	defer unlock()
	canonical, fsid, free, err := diskFilesystem(dir)
	if err != nil {
		return DiskReservation{}, err
	}
	now := s.now()
	expires := now.Add(diskReservationTTL)
	id, err := newDiskReservationID()
	if err != nil {
		return DiskReservation{}, err
	}
	var reserved int64
	var globalFloor int64
	err = s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		// A missed heartbeat is evidence of a stale owner, not proof that the
		// producer stopped or cleaned its staging tree. Keep the lease reserved.
		var legacyReserved, jobReserved int64
		var legacyFloor, jobFloor int64
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(peak_bytes),0), COALESCE(MAX(floor_bytes),0) FROM capacity_disk_reservations WHERE filesystem_id=? AND status='active'`, fsid).Scan(&legacyReserved, &legacyFloor); err != nil {
			return fmt.Errorf("sum active disk reservations: %w", err)
		}
		if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(f.peak_bytes),0), COALESCE(MAX(j.floor_bytes),0) FROM capacity_disk_job_filesystems f JOIN capacity_disk_jobs j USING(job_lease_id) WHERE f.filesystem_id=? AND j.status='active'`, fsid).Scan(&jobReserved, &jobFloor); err != nil {
			return fmt.Errorf("sum active disk jobs: %w", err)
		}
		if legacyReserved > math.MaxInt64-jobReserved {
			return fmt.Errorf("%w: active disk reservations overflow accounting range", ErrDiskAdmissionDenied)
		}
		reserved = legacyReserved + jobReserved
		globalFloor = maxInt64(legacyFloor, jobFloor)
		floor := maxInt64(floorBytes, globalFloor)
		if reserved > math.MaxInt64-peakBytes || reserved+peakBytes > math.MaxInt64-floor || free < reserved+peakBytes+floor {
			return fmt.Errorf("%w: filesystem %s has %d bytes free, %d reserved, request peak %d, required floor %d", ErrDiskAdmissionDenied, fsid, free, reserved, peakBytes, floor)
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO capacity_disk_reservations(reservation_id,owner_id,filesystem_id,path,peak_bytes,floor_bytes,status,created_at,updated_at,heartbeat_deadline_at) VALUES(?,?,?,?,?,?,'active',?,?,?)`, id, ownerID, fsid, canonical, peakBytes, floorBytes, formatTime(now), formatTime(now), formatTime(expires))
		if err != nil {
			return fmt.Errorf("record disk reservation: %w", err)
		}
		return nil
	})
	if err != nil {
		return DiskReservation{}, err
	}
	return DiskReservation{ID: id, OwnerID: ownerID, FilesystemID: fsid, Path: canonical, PeakBytes: peakBytes, FloorBytes: floorBytes, Available: free, Reserved: reserved, ExpiresAt: expires}, nil
}

func (s *SQLiteStore) HeartbeatDisk(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: reservation ID is required", ErrInvalidClaim)
	}
	unlock, err := s.LockAdmission(ctx)
	if err != nil {
		return fmt.Errorf("lock disk heartbeat: %w", err)
	}
	defer unlock()
	now := s.now()
	res, err := s.db.ExecContext(ctx, `UPDATE capacity_disk_reservations SET updated_at=?, heartbeat_deadline_at=? WHERE reservation_id=? AND status='active'`, formatTime(now), formatTime(now.Add(diskReservationTTL)), id)
	if err != nil {
		return fmt.Errorf("heartbeat disk reservation: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%w: disk reservation %s is absent or expired", ErrNotFound, id)
	}
	return nil
}

func (s *SQLiteStore) ReleaseDisk(ctx context.Context, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%w: reservation ID is required", ErrInvalidClaim)
	}
	unlock, err := s.LockAdmission(ctx)
	if err != nil {
		return fmt.Errorf("lock disk release: %w", err)
	}
	defer unlock()
	_, err = s.db.ExecContext(ctx, `UPDATE capacity_disk_reservations SET status='released',updated_at=? WHERE reservation_id=? AND status='active'`, formatTime(s.now()), id)
	if err != nil {
		return fmt.Errorf("release disk reservation: %w", err)
	}
	return nil
}

// ListDiskReservations returns reservation receipts for reconciliation. An
// overdue heartbeat deadline does not release a legacy reservation: only its
// owner can verify process stop and cleanup and release it.
func (s *SQLiteStore) ListDiskReservations(ctx context.Context, filter DiskReservationFilter) ([]DiskReservationRecord, error) {
	unlock, err := s.LockAdmission(ctx)
	if err != nil {
		return nil, fmt.Errorf("lock disk reservation listing: %w", err)
	}
	defer unlock()
	return s.listDiskReservationsUnlocked(ctx, filter)
}

func (s *SQLiteStore) listDiskReservationsUnlocked(ctx context.Context, filter DiskReservationFilter) ([]DiskReservationRecord, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("capacity store is unavailable")
	}
	// Expired heartbeat deadlines remain active until an owner verifies process
	// stop and cleanup through reconciliation.
	query := `SELECT reservation_id,owner_id,filesystem_id,path,peak_bytes,floor_bytes,status,created_at,updated_at,heartbeat_deadline_at FROM capacity_disk_reservations WHERE 1=1`
	args := []any{}
	if filter.FilesystemID != "" {
		query += ` AND filesystem_id=?`
		args = append(args, filter.FilesystemID)
	}
	if filter.OwnerID != "" {
		query += ` AND owner_id=?`
		args = append(args, filter.OwnerID)
	}
	if filter.ActiveOnly {
		query += ` AND status='active'`
	}
	query += ` ORDER BY created_at,reservation_id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list disk reservations: %w", err)
	}
	defer rows.Close()
	var out []DiskReservationRecord
	for rows.Next() {
		var r DiskReservationRecord
		var created, updated, deadline string
		if err := rows.Scan(&r.ID, &r.OwnerID, &r.FilesystemID, &r.Path, &r.PeakBytes, &r.FloorBytes, &r.Status, &created, &updated, &deadline); err != nil {
			return nil, err
		}
		r.CreatedAt, err = parseRequiredTime(created)
		if err != nil {
			return nil, err
		}
		r.UpdatedAt, err = parseRequiredTime(updated)
		if err != nil {
			return nil, err
		}
		r.ExpiresAt, err = parseRequiredTime(deadline)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func newDiskReservationID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate disk reservation ID: %w", err)
	}
	return "dsk-" + hex.EncodeToString(b[:]), nil
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
