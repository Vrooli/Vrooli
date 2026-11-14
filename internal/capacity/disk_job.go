package capacity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// DiskJobPath is one write-capable path and its peak additional bytes. Paths
// resolving to the same canonical directory are deduplicated (maximum peak),
// while distinct paths on one filesystem are summed.
type DiskJobPath struct {
	Path              string `json:"path"`
	Role              string `json:"role"`
	PeakBytes         int64  `json:"peak_bytes"`
	DockerContainer   string `json:"docker_container,omitempty"`
	DockerDestination string `json:"docker_destination,omitempty"`
	DockerUpperDir    bool   `json:"docker_upper_dir,omitempty"`
	DockerRootDir     bool   `json:"docker_root_dir,omitempty"`
}

// DiskJobRequest atomically reserves every filesystem touched by one job.
type DiskJobRequest struct {
	OwnerID      string        `json:"owner_id"`
	JobID        string        `json:"job_id"`
	ProcessID    int           `json:"process_id,omitempty"`
	ProcessStart string        `json:"process_start,omitempty"`
	Paths        []DiskJobPath `json:"paths"`
	FloorBytes   int64         `json:"floor_bytes"`
}

type DiskJobFilesystem struct {
	FilesystemID       string `json:"filesystem_id"`
	PeakBytes          int64  `json:"peak_bytes"`
	FloorBytes         int64  `json:"floor_bytes"`
	AvailableBytes     int64  `json:"available_bytes"`
	OtherReservedBytes int64  `json:"other_reserved_bytes"`
	ActualDeltaBytes   *int64 `json:"actual_delta_bytes,omitempty"`
	ActualDeltaBasis   string `json:"actual_delta_basis"`
}

const diskJobActualDeltaBasis = "filesystem_available_bytes_decrease"

type DiskJobPathReceipt struct {
	Path         string              `json:"path"`
	Role         string              `json:"role"`
	FilesystemID string              `json:"filesystem_id"`
	PeakBytes    int64               `json:"peak_bytes"`
	Samples      []DiskJobPathSample `json:"samples,omitempty"`
}

// DiskJobPathSample records owner-supplied byte usage for one declared path.
// The first sample is the baseline; later samples must be monotonic.
type DiskJobPathSample struct {
	Path              string    `json:"path"`
	FilesystemID      string    `json:"filesystem_id"`
	Bytes             int64     `json:"bytes"`
	BaselineBytes     int64     `json:"baseline_bytes"`
	DeltaBytes        int64     `json:"delta_bytes"`
	ObservedAt        time.Time `json:"observed_at"`
	ExceedsPeak       bool      `json:"exceeds_peak"`
	ReservedPeakBytes int64     `json:"reserved_peak_bytes"`
}

type DiskJobHeadroomFilesystem struct {
	FilesystemID            string `json:"filesystem_id"`
	CurrentFreeBytes        int64  `json:"current_free_bytes"`
	OtherReservedBytes      int64  `json:"other_reserved_bytes"`
	CurrentJobPeakBytes     int64  `json:"current_job_peak_bytes"`
	FloorBytes              int64  `json:"floor_bytes"`
	SafeAdditionalPeakBytes int64  `json:"safe_additional_peak_bytes"`
}

type DiskJobHeadroom struct {
	LeaseID     string                      `json:"lease_id"`
	Status      string                      `json:"status"`
	Filesystems []DiskJobHeadroomFilesystem `json:"filesystems"`
}

// DiskHeadroomSnapshot is a read-only refresh of free space against every
// filesystem that currently has a live disk reservation or job lease. It
// includes all active peaks, including the lease that the owner would observe
// separately through DiskJobHeadroom.
type DiskHeadroomSnapshot struct {
	ObservedAt         time.Time                `json:"observed_at"`
	ActiveJobs         []DiskJobReceipt         `json:"active_jobs"`
	ActiveLegacyLeases []DiskReservationRecord  `json:"active_legacy_leases"`
	Filesystems        []DiskFilesystemHeadroom `json:"filesystems"`
}

type DiskFilesystemHeadroom struct {
	FilesystemID            string `json:"filesystem_id"`
	ProbePath               string `json:"probe_path"`
	CurrentFreeBytes        int64  `json:"current_free_bytes"`
	ActiveReservedBytes     int64  `json:"active_reserved_bytes"`
	FloorBytes              int64  `json:"floor_bytes"`
	SafeAdditionalPeakBytes int64  `json:"safe_additional_peak_bytes"`
}

// DiskJobReceipt is both the current lease view and its terminal growth
// record. Heartbeat deadlines signal staleness but never release capacity.
type DiskJobReceipt struct {
	ID                string               `json:"id"`
	OwnerID           string               `json:"owner_id"`
	JobID             string               `json:"job_id"`
	ProcessID         int                  `json:"process_id"`
	ProcessStart      string               `json:"process_start"`
	Status            string               `json:"status"`
	Paths             []DiskJobPathReceipt `json:"paths"`
	Filesystems       []DiskJobFilesystem  `json:"filesystems"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
	LastHeartbeatAt   time.Time            `json:"last_heartbeat_at"`
	HeartbeatDeadline time.Time            `json:"heartbeat_deadline_at"`
	CleanupVerified   bool                 `json:"cleanup_verified"`
}

type diskJobFS struct {
	id, path   string
	peak, free int64
}

type diskJobPath struct {
	path, role, fsid string
	probe            string
	peak             int64
}

// ReserveDiskJob atomically measures and reserves all declared paths. The
// returned capability must remain private to the owning job process.
func (s *SQLiteStore) ReserveDiskJob(ctx context.Context, req DiskJobRequest) (DiskJobReceipt, string, error) {
	if s == nil || s.db == nil || strings.TrimSpace(req.OwnerID) == "" || strings.TrimSpace(req.JobID) == "" || len(req.Paths) == 0 || req.FloorBytes < 0 {
		return DiskJobReceipt{}, "", fmt.Errorf("%w: owner, job, paths, and nonnegative floor are required", ErrInvalidClaim)
	}
	req.FloorBytes = maxInt64(req.FloorBytes, s.diskFloorMinimum)
	unlock, err := s.LockAdmission(ctx)
	if err != nil {
		return DiskJobReceipt{}, "", fmt.Errorf("lock disk job admission: %w", err)
	}
	defer unlock()

	paths, filesystems, err := resolveDiskJobPaths(ctx, req.Paths)
	if err != nil {
		return DiskJobReceipt{}, "", err
	}
	processID := req.ProcessID
	processStart := req.ProcessStart
	if processID == 0 && processStart == "" {
		processID = os.Getpid()
		processStart, err = processStartIdentity(processID)
	} else if processID <= 0 || processStart == "" {
		return DiskJobReceipt{}, "", fmt.Errorf("process id and process start identity must be supplied together")
	}
	if err != nil {
		return DiskJobReceipt{}, "", fmt.Errorf("identify disk job process: %w", err)
	}
	if !processIdentityMatches(processID, processStart) {
		return DiskJobReceipt{}, "", fmt.Errorf("disk job owner process %d is not live with the supplied start identity", processID)
	}
	id, err := newDiskReservationID()
	if err != nil {
		return DiskJobReceipt{}, "", err
	}
	capability, err := newDiskJobCapability()
	if err != nil {
		return DiskJobReceipt{}, "", err
	}
	capHash := sha256.Sum256([]byte(capability))
	now := s.now()
	deadline := now.Add(diskReservationTTL)
	var receipts []DiskJobFilesystem
	err = s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		for _, fs := range filesystems {
			var oldReserved, jobReserved, globalFloor int64
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(peak_bytes),0), COALESCE(MAX(floor_bytes),0) FROM capacity_disk_reservations WHERE filesystem_id=? AND status='active'`, fs.id).Scan(&oldReserved, &globalFloor); err != nil {
				return fmt.Errorf("sum legacy disk reservations: %w", err)
			}
			var jobsFloor int64
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(f.peak_bytes),0), COALESCE(MAX(j.floor_bytes),0) FROM capacity_disk_job_filesystems f JOIN capacity_disk_jobs j USING(job_lease_id) WHERE f.filesystem_id=? AND j.status='active'`, fs.id).Scan(&jobReserved, &jobsFloor); err != nil {
				return fmt.Errorf("sum disk job reservations: %w", err)
			}
			if oldReserved > math.MaxInt64-jobReserved {
				return fmt.Errorf("filesystem %s reservation overflow", fs.id)
			}
			reserved := oldReserved + jobReserved
			floor := maxInt64(req.FloorBytes, maxInt64(globalFloor, jobsFloor))
			if reserved > math.MaxInt64-fs.peak || reserved+fs.peak > math.MaxInt64-floor || fs.free < reserved+fs.peak+floor {
				return fmt.Errorf("%w: filesystem %s has %d bytes free, %d bytes reserved, request peak %d, required floor %d", ErrDiskAdmissionDenied, fs.id, fs.free, reserved, fs.peak, floor)
			}
			receipts = append(receipts, DiskJobFilesystem{FilesystemID: fs.id, PeakBytes: fs.peak, FloorBytes: floor, AvailableBytes: fs.free, OtherReservedBytes: reserved})
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO capacity_disk_jobs(job_lease_id,owner_id,job_id,process_id,process_start,capability_hash,status,floor_bytes,created_at,updated_at,last_heartbeat_at,heartbeat_deadline_at) VALUES(?,?,?,?,?,?,'active',?,?,?,?,?)`, id, req.OwnerID, req.JobID, processID, processStart, hex.EncodeToString(capHash[:]), req.FloorBytes, formatTime(now), formatTime(now), formatTime(now), formatTime(deadline)); err != nil {
			return fmt.Errorf("create disk job lease: %w", err)
		}
		for _, fs := range filesystems {
			var other int64
			for _, r := range receipts {
				if r.FilesystemID == fs.id {
					other = r.OtherReservedBytes
					break
				}
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO capacity_disk_job_filesystems(job_lease_id,filesystem_id,filesystem_path,peak_bytes,baseline_free_bytes,available_at_admission_bytes,other_reserved_bytes) VALUES(?,?,?,?,?,?,?)`, id, fs.id, fs.path, fs.peak, fs.free, fs.free, other); err != nil {
				return fmt.Errorf("record disk job filesystem: %w", err)
			}
		}
		for _, p := range paths {
			if _, err := tx.ExecContext(ctx, `INSERT INTO capacity_disk_job_paths(job_lease_id,filesystem_id,canonical_path,role,peak_bytes) VALUES(?,?,?,?,?)`, id, p.fsid, p.path, p.role, p.peak); err != nil {
				return fmt.Errorf("record disk job path: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return DiskJobReceipt{}, "", err
	}
	r := DiskJobReceipt{ID: id, OwnerID: req.OwnerID, JobID: req.JobID, ProcessID: processID, ProcessStart: processStart, Status: "active", Filesystems: receipts, CreatedAt: now, UpdatedAt: now, LastHeartbeatAt: now, HeartbeatDeadline: deadline}
	for _, p := range paths {
		r.Paths = append(r.Paths, DiskJobPathReceipt{Path: p.path, Role: p.role, FilesystemID: p.fsid, PeakBytes: p.peak})
	}
	return r, capability, nil
}

func (s *SQLiteStore) HeartbeatDiskJob(ctx context.Context, id, capability string, processID int, processStart string) error {
	return s.mutateDiskJob(ctx, id, capability, processID, processStart, func(tx *sql.Tx, now time.Time) error {
		res, err := tx.ExecContext(ctx, `UPDATE capacity_disk_jobs SET updated_at=?,last_heartbeat_at=?,heartbeat_deadline_at=? WHERE job_lease_id=? AND status='active'`, formatTime(now), formatTime(now), formatTime(now.Add(diskReservationTTL)), id)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("disk job %s is not active", id)
		}
		return nil
	})
}

// SampleDiskJobPath stores an owner-measured byte count for an already
// admitted canonical write path. Passing a largerPeakBytes atomically raises
// that path's reservation only if current filesystem capacity still preserves
// every active reservation and the strongest floor.
func (s *SQLiteStore) SampleDiskJobPath(ctx context.Context, id, capability string, processID int, processStart, path, dockerContainer, dockerDestination string, dockerUpperDir, dockerRootDir bool, bytes, largerPeakBytes int64) (DiskJobReceipt, error) {
	if bytes < 0 || largerPeakBytes < 0 || strings.TrimSpace(path) == "" {
		return DiskJobReceipt{}, fmt.Errorf("disk job path sample requires a path and nonnegative byte counts")
	}
	if err := s.verifyDiskJobOwner(ctx, id, capability, processID, processStart); err != nil {
		return DiskJobReceipt{}, err
	}
	canonical, fsid, _, probePath, err := resolveDiskJobPath(ctx, DiskJobPath{Path: path, DockerContainer: dockerContainer, DockerDestination: dockerDestination, DockerUpperDir: dockerUpperDir, DockerRootDir: dockerRootDir, PeakBytes: 1})
	if err != nil {
		return DiskJobReceipt{}, err
	}
	err = s.mutateDiskJob(ctx, id, capability, processID, processStart, func(tx *sql.Tx, now time.Time) error {
		currentFS, currentProbe, err := diskJobFilesystemIdentity(canonical)
		if err != nil {
			if _, _, _, probeErr := diskFilesystem(canonical); probeErr != nil {
				return err
			}
			currentFS = fsid
		}
		if currentFS != fsid {
			return fmt.Errorf("sampled path filesystem identity changed: initial=%s current=%s", fsid, currentFS)
		}
		var declaredFS string
		var oldPeak int64
		if err := tx.QueryRowContext(ctx, `SELECT filesystem_id,peak_bytes FROM capacity_disk_job_paths WHERE job_lease_id=? AND canonical_path=?`, id, canonical).Scan(&declaredFS, &oldPeak); err != nil {
			return fmt.Errorf("sample path is not declared by this lease: %w", err)
		}
		if declaredFS != fsid {
			return fmt.Errorf("sampled path filesystem identity changed: lease=%s current=%s", declaredFS, fsid)
		}
		var baseline, previous int64
		lookupErr := tx.QueryRowContext(ctx, `SELECT baseline_bytes,bytes FROM capacity_disk_job_path_samples WHERE job_lease_id=? AND canonical_path=? ORDER BY sample_id DESC LIMIT 1`, id, canonical).Scan(&baseline, &previous)
		if errors.Is(lookupErr, sql.ErrNoRows) {
			baseline, previous = bytes, bytes
		} else if lookupErr != nil {
			return lookupErr
		} else if bytes < previous {
			return fmt.Errorf("disk path byte sample decreased from %d to %d; preserve monotonic evidence", previous, bytes)
		}
		delta := bytes - baseline
		if largerPeakBytes > 0 {
			if largerPeakBytes < oldPeak || largerPeakBytes < delta {
				return fmt.Errorf("requested peak %d must cover prior peak %d and measured delta %d", largerPeakBytes, oldPeak, delta)
			}
			if largerPeakBytes > oldPeak {
				_, currentFS, free, err := diskFilesystem(currentProbe)
				if err != nil {
					_, currentFS, free, err = diskFilesystem(probePath)
					if err != nil {
						return err
					}
				}
				if currentFS != fsid {
					return fmt.Errorf("filesystem identity changed for %s", canonical)
				}
				var currentPathPeak, floor, otherLegacy, otherJobs, globalFloor, jobsFloor int64
				if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(peak_bytes),0) FROM capacity_disk_job_paths WHERE job_lease_id=? AND filesystem_id=?`, id, fsid).Scan(&currentPathPeak); err != nil {
					return err
				}
				if currentPathPeak < oldPeak || currentPathPeak-oldPeak > math.MaxInt64-(largerPeakBytes-oldPeak) {
					return fmt.Errorf("filesystem peak overflow on %s", fsid)
				}
				currentPathPeak = currentPathPeak - oldPeak + largerPeakBytes
				if err := tx.QueryRowContext(ctx, `SELECT floor_bytes FROM capacity_disk_jobs WHERE job_lease_id=? AND status='active'`, id).Scan(&floor); err != nil {
					return err
				}
				if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(peak_bytes),0),COALESCE(MAX(floor_bytes),0) FROM capacity_disk_reservations WHERE filesystem_id=? AND status='active'`, fsid).Scan(&otherLegacy, &globalFloor); err != nil {
					return err
				}
				if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(f.peak_bytes),0),COALESCE(MAX(j.floor_bytes),0) FROM capacity_disk_job_filesystems f JOIN capacity_disk_jobs j USING(job_lease_id) WHERE f.filesystem_id=? AND j.status='active' AND j.job_lease_id<>?`, fsid, id).Scan(&otherJobs, &jobsFloor); err != nil {
					return err
				}
				floor = maxInt64(floor, maxInt64(globalFloor, jobsFloor))
				if otherLegacy > math.MaxInt64-otherJobs || otherLegacy+otherJobs > math.MaxInt64-currentPathPeak || otherLegacy+otherJobs+currentPathPeak > math.MaxInt64-floor || free < otherLegacy+otherJobs+currentPathPeak+floor {
					return fmt.Errorf("%w: measured path extension on %s would violate reserved peaks or %d-byte floor", ErrDiskAdmissionDenied, fsid, floor)
				}
				if _, err := tx.ExecContext(ctx, `UPDATE capacity_disk_job_paths SET peak_bytes=? WHERE job_lease_id=? AND canonical_path=?`, largerPeakBytes, id, canonical); err != nil {
					return err
				}
				if _, err := tx.ExecContext(ctx, `UPDATE capacity_disk_job_filesystems SET peak_bytes=? WHERE job_lease_id=? AND filesystem_id=?`, currentPathPeak, id, fsid); err != nil {
					return err
				}
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO capacity_disk_job_path_samples(job_lease_id,canonical_path,filesystem_id,bytes,baseline_bytes,delta_bytes,reserved_peak_bytes,exceeded_reserved_peak,observed_at) VALUES(?,?,?,?,?,?,?,?,?)`, id, canonical, fsid, bytes, baseline, delta, oldPeak, boolToInt(delta > oldPeak), formatTime(now)); err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `UPDATE capacity_disk_jobs SET updated_at=?,last_heartbeat_at=?,heartbeat_deadline_at=? WHERE job_lease_id=?`, formatTime(now), formatTime(now), formatTime(now.Add(diskReservationTTL)), id)
		return err
	})
	if err != nil {
		return DiskJobReceipt{}, err
	}
	return s.GetDiskJob(ctx, id)
}

func (s *SQLiteStore) verifyDiskJobOwner(ctx context.Context, id, capability string, processID int, processStart string) error {
	if s == nil || s.db == nil || strings.TrimSpace(id) == "" || capability == "" {
		return fmt.Errorf("disk job lease id and capability are required")
	}
	var storedHash, storedStart, status string
	var storedPID int
	if err := s.db.QueryRowContext(ctx, `SELECT capability_hash,process_id,process_start,status FROM capacity_disk_jobs WHERE job_lease_id=?`, id).Scan(&storedHash, &storedPID, &storedStart, &status); err != nil {
		return err
	}
	h := sha256.Sum256([]byte(capability))
	if storedHash != hex.EncodeToString(h[:]) || storedPID != processID || storedStart != processStart {
		return fmt.Errorf("disk job lease owner identity does not match")
	}
	if status != "active" {
		return fmt.Errorf("disk job %s is not active", id)
	}
	if !processIdentityMatches(processID, processStart) {
		return fmt.Errorf("disk job owner process %d is stopped or its identity changed", processID)
	}
	return nil
}

func (s *SQLiteStore) CompleteDiskJob(ctx context.Context, id, capability string, processID int, processStart string) (DiskJobReceipt, error) {
	err := s.mutateDiskJob(ctx, id, capability, processID, processStart, func(tx *sql.Tx, now time.Time) error {
		if err := recordDiskJobGrowth(ctx, tx, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE capacity_disk_jobs SET status='completed',cleanup_verified=1,updated_at=? WHERE job_lease_id=? AND status='active'`, formatTime(now), id)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("disk job %s is not active", id)
		}
		return nil
	})
	if err != nil {
		return DiskJobReceipt{}, err
	}
	return s.GetDiskJob(ctx, id)
}

// ReconcileDiskJob is the explicit crash/cancel recovery path. It releases an
// abandoned lease only after the caller verifies cleanup and the capacity
// owner independently confirms that the bound process identity is gone.
func (s *SQLiteStore) ReconcileDiskJob(ctx context.Context, id string, processStoppedVerified, cleanupVerified bool) (DiskJobReceipt, error) {
	if !processStoppedVerified || !cleanupVerified {
		return DiskJobReceipt{}, fmt.Errorf("disk job reconciliation requires verified stopped process and verified cleanup")
	}
	unlock, err := s.LockAdmission(ctx)
	if err != nil {
		return DiskJobReceipt{}, err
	}
	defer unlock()
	now := s.now()
	err = s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var pid int
		var start, status string
		if err := tx.QueryRowContext(ctx, `SELECT process_id,process_start,status FROM capacity_disk_jobs WHERE job_lease_id=?`, id).Scan(&pid, &start, &status); err != nil {
			return err
		}
		if status != "active" {
			return fmt.Errorf("disk job %s is not active", id)
		}
		if processIdentityMatches(pid, start) {
			return fmt.Errorf("disk job process %d is still running with the recorded identity", pid)
		}
		if err := recordDiskJobGrowth(ctx, tx, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx, `UPDATE capacity_disk_jobs SET status='reconciled',cleanup_verified=1,updated_at=? WHERE job_lease_id=? AND status='active'`, formatTime(now), id)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("disk job %s is not active", id)
		}
		return nil
	})
	if err != nil {
		return DiskJobReceipt{}, err
	}
	return s.GetDiskJob(ctx, id)
}

func recordDiskJobGrowth(ctx context.Context, tx *sql.Tx, id string) error {
	rows, err := tx.QueryContext(ctx, `SELECT filesystem_id,baseline_free_bytes,filesystem_path FROM capacity_disk_job_filesystems WHERE job_lease_id=?`, id)
	if err != nil {
		return err
	}
	type baseline struct {
		fsid string
		free int64
		path string
	}
	var baselines []baseline
	for rows.Next() {
		var b baseline
		if err := rows.Scan(&b.fsid, &b.free, &b.path); err != nil {
			rows.Close()
			return err
		}
		baselines = append(baselines, b)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, b := range baselines {
		_, fsid, free, err := diskFilesystem(b.path)
		if err != nil {
			return fmt.Errorf("measure completion filesystem %s: %w", b.fsid, err)
		}
		if fsid != b.fsid {
			return fmt.Errorf("filesystem identity changed for %s (was %s, now %s)", b.path, b.fsid, fsid)
		}
		delta := b.free - free
		if _, err := tx.ExecContext(ctx, `UPDATE capacity_disk_job_filesystems SET actual_delta_bytes=? WHERE job_lease_id=? AND filesystem_id=?`, delta, id, b.fsid); err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) ExtendDiskJob(ctx context.Context, id, capability string, processID int, processStart string, add []DiskJobPath) (DiskJobReceipt, error) {
	if len(add) == 0 {
		return DiskJobReceipt{}, fmt.Errorf("%w: at least one disk job path is required for extension", ErrInvalidClaim)
	}
	var receipt DiskJobReceipt
	err := s.mutateDiskJob(ctx, id, capability, processID, processStart, func(tx *sql.Tx, now time.Time) error {
		var jobFloor int64
		if err := tx.QueryRowContext(ctx, `SELECT floor_bytes FROM capacity_disk_jobs WHERE job_lease_id=? AND status='active'`, id).Scan(&jobFloor); err != nil {
			return err
		}
		newPaths, newFS, err := resolveDiskJobPaths(ctx, add)
		if err != nil {
			return err
		}
		for _, p := range newPaths {
			var prior int64
			err := tx.QueryRowContext(ctx, `SELECT peak_bytes FROM capacity_disk_job_paths WHERE job_lease_id=? AND canonical_path=?`, id, p.path).Scan(&prior)
			if err == nil {
				if p.peak < prior {
					p.peak = prior
				}
				if _, err := tx.ExecContext(ctx, `UPDATE capacity_disk_job_paths SET peak_bytes=?,role=? WHERE job_lease_id=? AND canonical_path=?`, p.peak, p.role, id, p.path); err != nil {
					return err
				}
			} else if !errors.Is(err, sql.ErrNoRows) {
				return err
			} else if _, err := tx.ExecContext(ctx, `INSERT INTO capacity_disk_job_paths(job_lease_id,filesystem_id,canonical_path,role,peak_bytes) VALUES(?,?,?,?,?)`, id, p.fsid, p.path, p.role, p.peak); err != nil {
				return err
			}
		}
		for _, fs := range newFS {
			var currentPeak, floor, oldPeak int64
			if err := tx.QueryRowContext(ctx, `SELECT peak_bytes,floor_bytes FROM capacity_disk_job_filesystems f JOIN capacity_disk_jobs j USING(job_lease_id) WHERE job_lease_id=? AND filesystem_id=?`, id, fs.id).Scan(&currentPeak, &floor); err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			floor = maxInt64(floor, jobFloor)
			var activeFloor int64
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(floor_bytes),0) FROM capacity_disk_reservations WHERE filesystem_id=? AND status='active'`, fs.id).Scan(&activeFloor); err != nil {
				return err
			}
			floor = maxInt64(floor, activeFloor)
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(j.floor_bytes),0) FROM capacity_disk_job_filesystems f JOIN capacity_disk_jobs j USING(job_lease_id) WHERE f.filesystem_id=? AND j.status='active' AND j.job_lease_id<>?`, fs.id, id).Scan(&activeFloor); err != nil {
				return err
			}
			floor = maxInt64(floor, activeFloor)
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(peak_bytes),0) FROM capacity_disk_job_paths WHERE job_lease_id=? AND filesystem_id=?`, id, fs.id).Scan(&oldPeak); err != nil {
				return err
			}
			var allOther int64
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(peak_bytes),0) FROM capacity_disk_reservations WHERE filesystem_id=? AND status='active'`, fs.id).Scan(&allOther); err != nil {
				return err
			}
			var otherJobs int64
			if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(f.peak_bytes),0) FROM capacity_disk_job_filesystems f JOIN capacity_disk_jobs j USING(job_lease_id) WHERE f.filesystem_id=? AND j.status='active' AND f.job_lease_id<>?`, fs.id, id).Scan(&otherJobs); err != nil {
				return err
			}
			if allOther > math.MaxInt64-otherJobs {
				return fmt.Errorf("reservation overflow on %s", fs.id)
			}
			if allOther > math.MaxInt64-otherJobs || allOther+otherJobs > math.MaxInt64-oldPeak || allOther+otherJobs+oldPeak > math.MaxInt64-floor || fs.free < allOther+otherJobs+oldPeak+floor {
				return fmt.Errorf("%w: extension on filesystem %s would leave less than %d bytes", ErrDiskAdmissionDenied, fs.id, floor)
			}
			if currentPeak == 0 {
				if _, err := tx.ExecContext(ctx, `INSERT INTO capacity_disk_job_filesystems(job_lease_id,filesystem_id,filesystem_path,peak_bytes,baseline_free_bytes,available_at_admission_bytes,other_reserved_bytes) VALUES(?,?,?,?,?,?,?)`, id, fs.id, fs.path, oldPeak, fs.free, fs.free, allOther+otherJobs); err != nil {
					return err
				}
			} else if _, err := tx.ExecContext(ctx, `UPDATE capacity_disk_job_filesystems SET peak_bytes=? WHERE job_lease_id=? AND filesystem_id=?`, oldPeak, id, fs.id); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE capacity_disk_jobs SET updated_at=?,last_heartbeat_at=?,heartbeat_deadline_at=? WHERE job_lease_id=? AND status='active'`, formatTime(now), formatTime(now), formatTime(now.Add(diskReservationTTL)), id)
		return err
	})
	if err != nil {
		return DiskJobReceipt{}, err
	}
	receipt, err = s.GetDiskJob(ctx, id)
	return receipt, err
}

func (s *SQLiteStore) mutateDiskJob(ctx context.Context, id, capability string, processID int, processStart string, fn func(*sql.Tx, time.Time) error) error {
	if s == nil || s.db == nil || strings.TrimSpace(id) == "" || capability == "" {
		return fmt.Errorf("disk job lease id and capability are required")
	}
	unlock, err := s.LockAdmission(ctx)
	if err != nil {
		return err
	}
	defer unlock()
	now := s.now()
	return s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		var storedHash, storedStart string
		var storedPID int
		if err := tx.QueryRowContext(ctx, `SELECT capability_hash,process_id,process_start FROM capacity_disk_jobs WHERE job_lease_id=?`, id).Scan(&storedHash, &storedPID, &storedStart); err != nil {
			return err
		}
		h := sha256.Sum256([]byte(capability))
		if storedHash != hex.EncodeToString(h[:]) || storedPID != processID || storedStart != processStart {
			return fmt.Errorf("disk job lease owner identity does not match")
		}
		if !processIdentityMatches(processID, processStart) {
			return fmt.Errorf("disk job owner process %d is stopped or its identity changed", processID)
		}
		return fn(tx, now)
	})
}

func (s *SQLiteStore) GetDiskJob(ctx context.Context, id string) (DiskJobReceipt, error) {
	if s == nil || s.db == nil {
		return DiskJobReceipt{}, errors.New("capacity store is unavailable")
	}
	var r DiskJobReceipt
	var created, updated, heartbeat, deadline string
	var clean int
	err := s.db.QueryRowContext(ctx, `SELECT job_lease_id,owner_id,job_id,process_id,process_start,status,created_at,updated_at,last_heartbeat_at,heartbeat_deadline_at,cleanup_verified FROM capacity_disk_jobs WHERE job_lease_id=?`, id).Scan(&r.ID, &r.OwnerID, &r.JobID, &r.ProcessID, &r.ProcessStart, &r.Status, &created, &updated, &heartbeat, &deadline, &clean)
	if err != nil {
		return r, err
	}
	r.CreatedAt, err = parseRequiredTime(created)
	if err != nil {
		return r, err
	}
	r.UpdatedAt, err = parseRequiredTime(updated)
	if err != nil {
		return r, err
	}
	r.LastHeartbeatAt, err = parseRequiredTime(heartbeat)
	if err != nil {
		return r, err
	}
	r.HeartbeatDeadline, err = parseRequiredTime(deadline)
	if err != nil {
		return r, err
	}
	r.CleanupVerified = clean != 0
	rows, err := s.db.QueryContext(ctx, `SELECT p.canonical_path,p.role,p.filesystem_id,p.peak_bytes FROM capacity_disk_job_paths p WHERE p.job_lease_id=? ORDER BY p.filesystem_id,p.canonical_path`, id)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var p DiskJobPathReceipt
		if err := rows.Scan(&p.Path, &p.Role, &p.FilesystemID, &p.PeakBytes); err != nil {
			rows.Close()
			return r, err
		}
		r.Paths = append(r.Paths, p)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return r, err
	}
	rows.Close()
	for i := range r.Paths {
		p := &r.Paths[i]
		samples, err := s.db.QueryContext(ctx, `SELECT bytes,baseline_bytes,delta_bytes,reserved_peak_bytes,exceeded_reserved_peak,observed_at FROM capacity_disk_job_path_samples WHERE job_lease_id=? AND canonical_path=? ORDER BY sample_id`, id, p.Path)
		if err != nil {
			return r, err
		}
		for samples.Next() {
			var sample DiskJobPathSample
			var observed string
			var exceeded int
			if err := samples.Scan(&sample.Bytes, &sample.BaselineBytes, &sample.DeltaBytes, &sample.ReservedPeakBytes, &exceeded, &observed); err != nil {
				samples.Close()
				return r, err
			}
			sample.Path, sample.FilesystemID = p.Path, p.FilesystemID
			sample.ObservedAt, err = parseRequiredTime(observed)
			if err != nil {
				samples.Close()
				return r, err
			}
			sample.ExceedsPeak = exceeded != 0
			p.Samples = append(p.Samples, sample)
		}
		if err := samples.Err(); err != nil {
			samples.Close()
			return r, err
		}
		samples.Close()
	}
	rows, err = s.db.QueryContext(ctx, `SELECT filesystem_id,peak_bytes,j.floor_bytes,available_at_admission_bytes,other_reserved_bytes,actual_delta_bytes FROM capacity_disk_job_filesystems f JOIN capacity_disk_jobs j USING(job_lease_id) WHERE f.job_lease_id=? ORDER BY filesystem_id`, id)
	if err != nil {
		return r, err
	}
	for rows.Next() {
		var f DiskJobFilesystem
		var delta sql.NullInt64
		f.ActualDeltaBasis = diskJobActualDeltaBasis
		if err := rows.Scan(&f.FilesystemID, &f.PeakBytes, &f.FloorBytes, &f.AvailableBytes, &f.OtherReservedBytes, &delta); err != nil {
			rows.Close()
			return r, err
		}
		if delta.Valid {
			f.ActualDeltaBytes = &delta.Int64
		}
		r.Filesystems = append(r.Filesystems, f)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return r, err
	}
	rows.Close()
	return r, nil
}

func (s *SQLiteStore) ListDiskJobs(ctx context.Context, activeOnly bool) ([]DiskJobReceipt, error) {
	return s.listDiskJobs(ctx, "", activeOnly)
}

// ListDiskJobsForOwner returns only receipts for one scenario owner, avoiding
// a read of other owners' paths through shared adapters.
func (s *SQLiteStore) ListDiskJobsForOwner(ctx context.Context, ownerID string, activeOnly bool) ([]DiskJobReceipt, error) {
	if strings.TrimSpace(ownerID) == "" {
		return nil, fmt.Errorf("disk job owner id is required")
	}
	return s.listDiskJobs(ctx, strings.TrimSpace(ownerID), activeOnly)
}

func (s *SQLiteStore) listDiskJobs(ctx context.Context, ownerID string, activeOnly bool) ([]DiskJobReceipt, error) {
	query := `SELECT job_lease_id FROM capacity_disk_jobs`
	conditions := make([]string, 0, 2)
	args := make([]any, 0, 2)
	if ownerID != "" {
		conditions = append(conditions, `owner_id=?`)
		args = append(args, ownerID)
	}
	if activeOnly {
		conditions = append(conditions, `status='active'`)
	}
	if len(conditions) > 0 {
		query += ` WHERE ` + strings.Join(conditions, ` AND `)
	}
	query += ` ORDER BY created_at,job_lease_id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	out := make([]DiskJobReceipt, 0, len(ids))
	for _, id := range ids {
		r, e := s.GetDiskJob(ctx, id)
		if e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, nil
}

// ListDiskHeadroom refreshes free bytes for all filesystems that have active
// reservations and subtracts every active peak plus the strongest enforced
// floor. This is observational only; it must never be used as a replacement
// for atomic ReserveDiskJob admission.
func (s *SQLiteStore) ListDiskHeadroom(ctx context.Context) (DiskHeadroomSnapshot, error) {
	out := DiskHeadroomSnapshot{
		ActiveJobs: []DiskJobReceipt{}, ActiveLegacyLeases: []DiskReservationRecord{},
		Filesystems: []DiskFilesystemHeadroom{},
	}
	unlock, err := s.LockAdmission(ctx)
	if err != nil {
		return out, fmt.Errorf("lock disk headroom observation: %w", err)
	}
	defer unlock()
	out.ObservedAt = s.now()
	out.ActiveJobs, err = s.ListDiskJobs(ctx, true)
	if err != nil {
		return out, err
	}
	out.ActiveLegacyLeases, err = s.listDiskReservationsUnlocked(ctx, DiskReservationFilter{ActiveOnly: true})
	if err != nil {
		return out, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT filesystem_id, filesystem_path FROM capacity_disk_job_filesystems f JOIN capacity_disk_jobs j USING(job_lease_id) WHERE j.status='active' UNION ALL SELECT filesystem_id, path FROM capacity_disk_reservations WHERE status='active' ORDER BY filesystem_id, filesystem_path`)
	if err != nil {
		return out, err
	}
	paths := map[string]string{}
	for rows.Next() {
		var fsid, path string
		if err := rows.Scan(&fsid, &path); err != nil {
			rows.Close()
			return out, err
		}
		if paths[fsid] == "" {
			paths[fsid] = path
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return out, err
	}
	if err := rows.Close(); err != nil {
		return out, err
	}
	fsids := make([]string, 0, len(paths))
	for fsid := range paths {
		fsids = append(fsids, fsid)
	}
	sort.Strings(fsids)
	for _, expectedID := range fsids {
		path := paths[expectedID]
		canonical, fsid, free, err := diskFilesystem(path)
		if err != nil {
			return out, err
		}
		if fsid != expectedID {
			return out, fmt.Errorf("filesystem identity changed for %s: recorded %s, observed %s", path, expectedID, fsid)
		}
		var legacy, jobs, legacyFloor, jobsFloor int64
		if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(peak_bytes),0),COALESCE(MAX(floor_bytes),0) FROM capacity_disk_reservations WHERE filesystem_id=? AND status='active'`, fsid).Scan(&legacy, &legacyFloor); err != nil {
			return out, err
		}
		if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(f.peak_bytes),0),COALESCE(MAX(j.floor_bytes),0) FROM capacity_disk_job_filesystems f JOIN capacity_disk_jobs j USING(job_lease_id) WHERE f.filesystem_id=? AND j.status='active'`, fsid).Scan(&jobs, &jobsFloor); err != nil {
			return out, err
		}
		if legacy > math.MaxInt64-jobs {
			return out, fmt.Errorf("active reservations overflow accounting range on filesystem %s", fsid)
		}
		reserved := legacy + jobs
		floor := maxInt64(s.diskFloorMinimum, maxInt64(legacyFloor, jobsFloor))
		if reserved > math.MaxInt64-floor {
			return out, fmt.Errorf("active reservations and floor overflow accounting range on filesystem %s", fsid)
		}
		out.Filesystems = append(out.Filesystems, DiskFilesystemHeadroom{
			FilesystemID: fsid, ProbePath: canonical, CurrentFreeBytes: free,
			ActiveReservedBytes: reserved, FloorBytes: floor,
			SafeAdditionalPeakBytes: free - reserved - floor,
		})
	}
	return out, nil
}

// ObserveDiskJob reports conservative additional lease capacity on each
// filesystem. The calculation keeps this job's full current peak reserved;
// owners must request an extension before increasing their write bound.
func (s *SQLiteStore) ObserveDiskJob(ctx context.Context, id, capability string, processID int, processStart string) (DiskJobHeadroom, error) {
	var out DiskJobHeadroom
	if strings.TrimSpace(id) == "" || capability == "" {
		return out, fmt.Errorf("disk job lease id and capability are required")
	}
	var storedHash, storedStart, status string
	var storedPID int
	if err := s.db.QueryRowContext(ctx, `SELECT capability_hash,process_id,process_start,status FROM capacity_disk_jobs WHERE job_lease_id=?`, id).Scan(&storedHash, &storedPID, &storedStart, &status); err != nil {
		return out, err
	}
	h := sha256.Sum256([]byte(capability))
	if storedHash != hex.EncodeToString(h[:]) || storedPID != processID || storedStart != processStart {
		return out, fmt.Errorf("disk job lease owner identity does not match")
	}
	if !processIdentityMatches(processID, processStart) {
		return out, fmt.Errorf("disk job owner process %d is stopped or its identity changed", processID)
	}
	if status != "active" {
		return out, fmt.Errorf("disk job %s is not active", id)
	}
	r, err := s.GetDiskJob(ctx, id)
	if err != nil {
		return out, err
	}
	out.LeaseID = r.ID
	out.Status = r.Status
	if r.Status != "active" {
		return out, nil
	}
	for _, f := range r.Filesystems {
		var probePath string
		var floor int64
		if err := s.db.QueryRowContext(ctx, `SELECT filesystem_path,j.floor_bytes FROM capacity_disk_job_filesystems x JOIN capacity_disk_jobs j USING(job_lease_id) WHERE x.job_lease_id=? AND x.filesystem_id=?`, id, f.FilesystemID).Scan(&probePath, &floor); err != nil {
			return out, err
		}
		_, fsid, free, err := diskFilesystem(probePath)
		if err != nil {
			return out, err
		}
		if fsid != f.FilesystemID {
			return out, fmt.Errorf("filesystem identity changed for %s", probePath)
		}
		var legacy, otherJobs, legacyFloor, jobsFloor int64
		if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(peak_bytes),0),COALESCE(MAX(floor_bytes),0) FROM capacity_disk_reservations WHERE filesystem_id=? AND status='active'`, fsid).Scan(&legacy, &legacyFloor); err != nil {
			return out, err
		}
		if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(x.peak_bytes),0),COALESCE(MAX(j.floor_bytes),0) FROM capacity_disk_job_filesystems x JOIN capacity_disk_jobs j USING(job_lease_id) WHERE x.filesystem_id=? AND j.status='active' AND x.job_lease_id<>?`, fsid, id).Scan(&otherJobs, &jobsFloor); err != nil {
			return out, err
		}
		if legacy > math.MaxInt64-otherJobs {
			return out, fmt.Errorf("reservation overflow on filesystem %s", fsid)
		}
		other := legacy + otherJobs
		floor = maxInt64(floor, maxInt64(legacyFloor, jobsFloor))
		additional := int64(0)
		if other <= math.MaxInt64-f.PeakBytes && other+f.PeakBytes <= math.MaxInt64-floor {
			remaining := free - (other + f.PeakBytes + floor)
			if remaining > 0 {
				additional = remaining
			}
		}
		out.Filesystems = append(out.Filesystems, DiskJobHeadroomFilesystem{FilesystemID: fsid, CurrentFreeBytes: free, OtherReservedBytes: other, CurrentJobPeakBytes: f.PeakBytes, FloorBytes: floor, SafeAdditionalPeakBytes: additional})
	}
	return out, nil
}

func resolveDiskJobPaths(ctx context.Context, in []DiskJobPath) ([]diskJobPath, []diskJobFS, error) {
	byPath := map[string]diskJobPath{}
	for _, p := range in {
		if strings.TrimSpace(p.Path) == "" || p.PeakBytes <= 0 {
			return nil, nil, fmt.Errorf("each disk job path needs a path and positive peak bytes")
		}
		canonical, fsid, free, probePath, err := resolveDiskJobPath(ctx, p)
		if err != nil {
			return nil, nil, err
		}
		if old, ok := byPath[canonical]; ok {
			if old.fsid != fsid {
				return nil, nil, fmt.Errorf("canonical disk job path %s resolved to conflicting filesystem identities %s and %s", canonical, old.fsid, fsid)
			}
			if p.PeakBytes > old.peak {
				old.peak = p.PeakBytes
			}
			if p.Role != "" && old.role != p.Role {
				old.role = strings.Trim(old.role+","+p.Role, ",")
			}
			byPath[canonical] = old
		} else {
			byPath[canonical] = diskJobPath{path: canonical, role: p.Role, fsid: fsid, probe: probePath, peak: p.PeakBytes}
		}
		_ = free
		_ = probePath
	}
	paths := make([]diskJobPath, 0, len(byPath))
	for _, p := range byPath {
		paths = append(paths, p)
	}
	sort.Slice(paths, func(i, j int) bool { return paths[i].path < paths[j].path })
	fsm := map[string]diskJobFS{}
	for _, p := range paths {
		probePath, fsid, free, err := diskFilesystem(p.probe)
		if err != nil {
			return nil, nil, err
		}
		fs := fsm[fsid]
		fs.id = fsid
		if fs.path == "" {
			fs.path = probePath
		}
		fs.free = free
		if fs.peak > math.MaxInt64-p.peak {
			return nil, nil, fmt.Errorf("filesystem %s peak overflow", fsid)
		}
		fs.peak += p.peak
		fsm[fsid] = fs
	}
	filesystems := make([]diskJobFS, 0, len(fsm))
	for _, fs := range fsm {
		filesystems = append(filesystems, fs)
	}
	sort.Slice(filesystems, func(i, j int) bool { return filesystems[i].id < filesystems[j].id })
	return paths, filesystems, nil
}

func diskPathIdentity(path string) (string, string, int64, string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", "", 0, "", err
	}
	candidate := filepath.Clean(abs)
	suffix := []string{}
	for {
		if _, err := os.Lstat(candidate); err == nil {
			break
		} else if os.IsPermission(err) {
			return "", "", 0, "", fmt.Errorf("disk job path %s is not traversable by the capacity owner: %w", path, err)
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			return "", "", 0, "", fmt.Errorf("no existing parent for disk job path %s", path)
		}
		suffix = append([]string{filepath.Base(candidate)}, suffix...)
		candidate = parent
	}
	resolved, fsid, free, err := diskFilesystem(candidate)
	if err != nil {
		return "", "", 0, "", err
	}
	canonical := filepath.Join(append([]string{resolved}, suffix...)...)
	return canonical, fsid, free, resolved, nil
}

func resolveDiskJobPath(ctx context.Context, p DiskJobPath) (string, string, int64, string, error) {
	if p.DockerRootDir {
		if p.DockerContainer != "" || p.DockerDestination != "" || p.DockerUpperDir {
			return "", "", 0, "", fmt.Errorf("Docker root path selector cannot be combined with a container, mount destination, or upper-directory selector")
		}
		return resolveDockerRootDir(ctx, p)
	}
	if p.DockerContainer != "" || p.DockerDestination != "" || p.DockerUpperDir {
		if p.DockerContainer == "" || (p.DockerUpperDir == (p.DockerDestination != "")) {
			return "", "", 0, "", fmt.Errorf("Docker path identity requires a container and exactly one mount destination or upper-directory mode")
		}
		return resolveDockerDiskPath(ctx, p)
	}
	canonical, fsid, free, probe, err := diskPathIdentity(p.Path)
	if err == nil {
		return canonical, fsid, free, probe, nil
	}
	return "", "", 0, "", err
}

func firstDiskJobPath(ctx context.Context, tx *sql.Tx, id, fsid string) (string, error) {
	var path string
	err := tx.QueryRowContext(ctx, `SELECT canonical_path FROM capacity_disk_job_paths WHERE job_lease_id=? AND filesystem_id=? ORDER BY canonical_path LIMIT 1`, id, fsid).Scan(&path)
	return path, err
}

func newDiskJobCapability() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
