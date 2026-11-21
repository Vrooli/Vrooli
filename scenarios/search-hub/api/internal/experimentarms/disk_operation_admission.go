package experimentarms

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
	capacity "github.com/vrooli/vrooli/packages/capacity"
)

const qdrantFleetDiskOperationSchema = `CREATE TABLE IF NOT EXISTS qdrant_fleet_disk_operations (
  operation_id TEXT PRIMARY KEY,
  operation TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  arm_id TEXT NOT NULL,
  target_id TEXT NOT NULL,
  peak_bytes INTEGER NOT NULL,
  capacity_lease_id TEXT NOT NULL,
  state TEXT NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_qdrant_fleet_disk_operations_target
  ON qdrant_fleet_disk_operations(target_id, created_at);`

// FleetDiskJobAdmitter is the narrow capacity owner seam. The production
// implementation is packages/capacity.Broker; tests inject a disk-job fake.
type FleetDiskJobAdmitter interface {
	ReserveDiskJob(context.Context, capacity.DiskJobRequest) (capacity.DiskJobLease, error)
}

type FleetDiskJobRecovery interface {
	ReconcileDiskJob(context.Context, string, capacity.DiskJobRecoveryEvidence) (capacity.DiskJobReceipt, error)
}

// DiskJobFleetAdmissionGate binds every Qdrant mutation receipt to a real
// shared-capacity disk reservation. The durable operation row records the
// target and generation identity that the generic disk-job ledger cannot
// represent. Its private lease stays in this process and is released only
// after the Qdrant effect returns successfully.
type DiskJobFleetAdmissionGate struct {
	db          SQLExecutor
	capacity    FleetDiskJobAdmitter
	storagePath string
	ownerID     string
	now         func() time.Time
	mu          sync.Mutex
	leases      map[string]capacity.DiskJobLease
}

var _ aisearch.FleetAdmissionGate = (*DiskJobFleetAdmissionGate)(nil)
var _ aisearch.FleetAdmissionLifecycle = (*DiskJobFleetAdmissionGate)(nil)

func NewDiskJobFleetAdmissionGate(db SQLExecutor, admitter FleetDiskJobAdmitter, storagePath, ownerID string) (*DiskJobFleetAdmissionGate, error) {
	if db == nil || admitter == nil || strings.TrimSpace(storagePath) == "" || strings.TrimSpace(ownerID) == "" {
		return nil, errors.New("Qdrant fleet disk admission requires owner database, capacity broker, storage path and owner identity")
	}
	return &DiskJobFleetAdmissionGate{db: db, capacity: admitter, storagePath: strings.TrimSpace(storagePath), ownerID: strings.TrimSpace(ownerID), now: time.Now, leases: make(map[string]capacity.DiskJobLease)}, nil
}

func (g *DiskJobFleetAdmissionGate) Initialize(ctx context.Context) error {
	if g == nil || g.db == nil {
		return errors.New("Qdrant fleet disk admission owner is unavailable")
	}
	_, err := g.db.ExecContext(ctx, qdrantFleetDiskOperationSchema)
	return err
}

func (g *DiskJobFleetAdmissionGate) AdmitQdrantFleetOperation(ctx context.Context, operation aisearch.FleetOperation, armID, targetID string, peakBytes int64) (aisearch.FleetAdmission, error) {
	if g == nil || g.capacity == nil || g.db == nil {
		return aisearch.FleetAdmission{}, aisearch.ErrFleetAdmissionRequired
	}
	armID, targetID = strings.TrimSpace(armID), strings.TrimSpace(targetID)
	if !validFleetDiskOperation(operation) || armID == "" || targetID == "" || peakBytes <= 0 {
		return aisearch.FleetAdmission{}, errors.New("Qdrant fleet disk admission requires a supported operation, arm, target and positive peak")
	}
	operationID, err := newFleetDiskOperationID()
	if err != nil {
		return aisearch.FleetAdmission{}, err
	}
	lease, err := g.capacity.ReserveDiskJob(ctx, capacity.DiskJobRequest{
		OwnerID: g.ownerID,
		JobID:   "qdrant-fleet-" + operationID,
		Paths:   []capacity.DiskJobPath{{Path: g.storagePath, Role: "qdrant-fleet-" + string(operation), PeakBytes: peakBytes}},
	})
	if err != nil {
		return aisearch.FleetAdmission{}, fmt.Errorf("reserve Qdrant storage for %s on %q: %w", operation, targetID, err)
	}
	reservation := lease.Reservation()
	now := g.now().UTC()
	if reservation.ID == "" || reservation.OwnerID != g.ownerID || reservation.JobID != "qdrant-fleet-"+operationID || reservation.ProcessID != os.Getpid() || reservation.ProcessStart == "" || reservation.Status != "active" || reservation.HeartbeatDeadline.IsZero() || !reservation.HeartbeatDeadline.After(now) || len(reservation.Paths) != 1 || filepath.Clean(reservation.Paths[0].Path) != filepath.Clean(g.storagePath) || reservation.Paths[0].PeakBytes < peakBytes {
		_ = lease.Release(context.WithoutCancel(ctx))
		return aisearch.FleetAdmission{}, errors.New("capacity owner returned an incomplete or stale Qdrant disk-job receipt")
	}
	_, err = g.db.ExecContext(ctx, `INSERT INTO qdrant_fleet_disk_operations(operation_id, operation, owner_id, arm_id, target_id, peak_bytes, capacity_lease_id, state, created_at, expires_at)
VALUES(?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)`, operationID, string(operation), g.ownerID, armID, targetID, peakBytes, reservation.ID, now.Format(time.RFC3339Nano), reservation.HeartbeatDeadline.UTC().Format(time.RFC3339Nano))
	if err != nil {
		_ = lease.Release(context.WithoutCancel(ctx))
		return aisearch.FleetAdmission{}, fmt.Errorf("persist Qdrant fleet disk operation identity: %w", err)
	}
	g.mu.Lock()
	g.leases[operationID] = lease
	g.mu.Unlock()
	return aisearch.FleetAdmission{OperationID: operationID, ReceiptID: reservation.ID, TargetID: targetID, PeakBytes: peakBytes, ObservedAt: now, ExpiresAt: reservation.HeartbeatDeadline}, nil
}

func (g *DiskJobFleetAdmissionGate) CompleteQdrantFleetOperation(ctx context.Context, admission aisearch.FleetAdmission, effectSucceeded bool) error {
	if g == nil || g.db == nil {
		return aisearch.ErrFleetAdmissionRequired
	}
	var storedTarget, storedLeaseID, state string
	var storedPeak int64
	if err := g.db.QueryRowContext(ctx, `SELECT target_id, capacity_lease_id, peak_bytes, state FROM qdrant_fleet_disk_operations WHERE operation_id = ? AND owner_id = ?`, admission.OperationID, g.ownerID).Scan(&storedTarget, &storedLeaseID, &storedPeak, &state); err != nil {
		return fmt.Errorf("read Qdrant fleet disk operation: %w", err)
	}
	if storedTarget != admission.TargetID || storedLeaseID != admission.ReceiptID || storedPeak != admission.PeakBytes {
		return errors.New("Qdrant fleet disk operation completion identity mismatch")
	}
	if state != "active" {
		return fmt.Errorf("Qdrant fleet disk operation %s is %q", admission.OperationID, state)
	}
	if !effectSucceeded {
		_, err := g.db.ExecContext(ctx, `UPDATE qdrant_fleet_disk_operations SET state='unresolved', error='provider effect failed or is uncertain' WHERE operation_id = ? AND state='active'`, admission.OperationID)
		return err
	}
	g.mu.Lock()
	lease := g.leases[admission.OperationID]
	g.mu.Unlock()
	if lease == nil {
		return errors.New("Qdrant fleet disk job lease is not attached to this process")
	}
	if _, err := lease.Complete(ctx); err != nil {
		_, _ = g.db.ExecContext(context.WithoutCancel(ctx), `UPDATE qdrant_fleet_disk_operations SET state='completion_failed', error=? WHERE operation_id = ? AND state='active'`, err.Error(), admission.OperationID)
		return err
	}
	completedAt := g.now().UTC().Format(time.RFC3339Nano)
	result, err := g.db.ExecContext(ctx, `UPDATE qdrant_fleet_disk_operations SET state='completed', completed_at=? WHERE operation_id=? AND state='active'`, completedAt, admission.OperationID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return errors.New("Qdrant fleet disk operation completion lost its state transition")
	}
	g.mu.Lock()
	delete(g.leases, admission.OperationID)
	g.mu.Unlock()
	return nil
}

// ResolveUnresolvedAfterCleanup releases a failed operation's reservation only
// after the fleet owner has verified that its partial physical output was
// cleaned up. After a Search Hub restart, the capacity owner independently
// verifies that the old producer process is gone before it reconciles the
// disk job.
func (g *DiskJobFleetAdmissionGate) ResolveUnresolvedAfterCleanup(ctx context.Context, admission aisearch.FleetAdmission, cleanupVerified bool) error {
	if g == nil || g.db == nil || !cleanupVerified {
		return errors.New("Qdrant fleet cleanup recovery requires verified cleanup")
	}
	var targetID, leaseID, state string
	var peak int64
	if err := g.db.QueryRowContext(ctx, `SELECT target_id, capacity_lease_id, peak_bytes, state FROM qdrant_fleet_disk_operations WHERE operation_id = ? AND owner_id = ?`, admission.OperationID, g.ownerID).Scan(&targetID, &leaseID, &peak, &state); err != nil {
		return fmt.Errorf("read unresolved Qdrant fleet operation: %w", err)
	}
	if targetID != admission.TargetID || leaseID != admission.ReceiptID || peak != admission.PeakBytes || state != "unresolved" {
		return errors.New("Qdrant fleet cleanup recovery identity or state mismatch")
	}
	g.mu.Lock()
	lease := g.leases[admission.OperationID]
	g.mu.Unlock()
	if lease != nil {
		if err := lease.Release(ctx); err != nil {
			return fmt.Errorf("release cleaned Qdrant fleet disk job: %w", err)
		}
	} else {
		recovery, ok := g.capacity.(FleetDiskJobRecovery)
		if !ok {
			return errors.New("capacity owner cannot reconcile the abandoned Qdrant fleet disk job")
		}
		if _, err := recovery.ReconcileDiskJob(ctx, leaseID, capacity.DiskJobRecoveryEvidence{OwnerID: g.ownerID, ProcessStoppedVerified: true, CleanupVerified: true}); err != nil {
			return fmt.Errorf("reconcile cleaned Qdrant fleet disk job: %w", err)
		}
	}
	result, err := g.db.ExecContext(ctx, `UPDATE qdrant_fleet_disk_operations SET state='recovered_cleaned', completed_at=? WHERE operation_id=? AND state='unresolved'`, g.now().UTC().Format(time.RFC3339Nano), admission.OperationID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 1 {
		return errors.New("Qdrant fleet cleanup recovery lost its state transition")
	}
	g.mu.Lock()
	delete(g.leases, admission.OperationID)
	g.mu.Unlock()
	return nil
}

func validFleetDiskOperation(operation aisearch.FleetOperation) bool {
	switch operation {
	case aisearch.FleetStage, aisearch.FleetCatchUp, aisearch.FleetPromote, aisearch.FleetRollback, aisearch.FleetCollect:
		return true
	default:
		return false
	}
}

func newFleetDiskOperationID() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate Qdrant fleet operation identity: %w", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw[:])
	return encoded[0:8] + "-" + encoded[8:12] + "-" + encoded[12:16] + "-" + encoded[16:20] + "-" + encoded[20:32], nil
}
