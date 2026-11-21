package experimentarms

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	aisearch "github.com/vrooli/ai-go/search"
	capacity "github.com/vrooli/vrooli/packages/capacity"
)

type fleetDiskJobAdmitterFake struct {
	request capacity.DiskJobRequest
	lease   *fleetDiskJobLeaseFake
	err     error
}

func (f *fleetDiskJobAdmitterFake) ReserveDiskJob(_ context.Context, request capacity.DiskJobRequest) (capacity.DiskJobLease, error) {
	f.request = request
	if f.err != nil {
		return nil, f.err
	}
	f.lease.receipt.OwnerID = request.OwnerID
	f.lease.receipt.JobID = request.JobID
	f.lease.receipt.ProcessID = os.Getpid()
	f.lease.receipt.ProcessStart = "test-process-start"
	f.lease.receipt.Status = "active"
	f.lease.receipt.CreatedAt = time.Now().UTC()
	f.lease.receipt.HeartbeatDeadline = time.Now().UTC().Add(15 * time.Minute)
	f.lease.receipt.Paths = []capacity.DiskJobPathReceipt{{Path: request.Paths[0].Path, PeakBytes: request.Paths[0].PeakBytes}}
	return f.lease, nil
}

type fleetDiskJobLeaseFake struct {
	receipt       capacity.DiskJobReceipt
	completeCalls int
	releaseCalls  int
}

func (f *fleetDiskJobLeaseFake) Heartbeat(context.Context) error { return nil }
func (f *fleetDiskJobLeaseFake) Observe(context.Context) (capacity.DiskJobHeadroom, error) {
	return capacity.DiskJobHeadroom{LeaseID: f.receipt.ID, Status: f.receipt.Status}, nil
}
func (f *fleetDiskJobLeaseFake) Extend(context.Context, []capacity.DiskJobPath) (capacity.DiskJobReceipt, error) {
	return f.receipt, nil
}
func (f *fleetDiskJobLeaseFake) SamplePath(context.Context, string, int64, int64) (capacity.DiskJobReceipt, error) {
	return f.receipt, nil
}
func (f *fleetDiskJobLeaseFake) Complete(context.Context) (capacity.DiskJobReceipt, error) {
	f.completeCalls++
	f.receipt.Status = "completed"
	return f.receipt, nil
}
func (f *fleetDiskJobLeaseFake) Release(context.Context) error {
	f.releaseCalls++
	_, err := f.Complete(context.Background())
	return err
}
func (f *fleetDiskJobLeaseFake) Reservation() capacity.DiskJobReceipt { return f.receipt }

func TestDiskJobFleetAdmissionBindsOwnerTargetPeakAndCompletesLease(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	admitter := &fleetDiskJobAdmitterFake{lease: &fleetDiskJobLeaseFake{receipt: capacity.DiskJobReceipt{
		ID: "disk-job-1", OwnerID: "qdrant-fleet.search-hub", JobID: "pending", Status: "active",
		CreatedAt: time.Now().UTC(), HeartbeatDeadline: time.Now().UTC().Add(15 * time.Minute),
		Paths: []capacity.DiskJobPathReceipt{{Path: "/qdrant/storage", PeakBytes: 4096}},
	}}}
	gate, err := NewDiskJobFleetAdmissionGate(db, admitter, "/qdrant/storage", "qdrant-fleet.search-hub")
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	gate.now = func() time.Time { return time.Now().UTC() }
	decision, err := gate.AdmitQdrantFleetOperation(ctx, aisearch.FleetStage, "arm-generation-2", "vf-candidate-2", 4096)
	if err != nil {
		t.Fatal(err)
	}
	if decision.OperationID == "" || decision.ReceiptID != "disk-job-1" || decision.TargetID != "vf-candidate-2" || decision.PeakBytes != 4096 || !decision.ExpiresAt.After(decision.ObservedAt) {
		t.Fatalf("incomplete admission receipt: %+v", decision)
	}
	if admitter.request.OwnerID != "qdrant-fleet.search-hub" || admitter.request.JobID != "qdrant-fleet-"+decision.OperationID || len(admitter.request.Paths) != 1 || admitter.request.Paths[0].Path != "/qdrant/storage" || admitter.request.Paths[0].PeakBytes != 4096 {
		t.Fatalf("disk capacity request lost operation binding: %+v", admitter.request)
	}
	var operation, owner, arm, target, state string
	var peak int64
	var leaseID string
	if err := db.QueryRowContext(ctx, `SELECT operation, owner_id, arm_id, target_id, peak_bytes, capacity_lease_id, state FROM qdrant_fleet_disk_operations WHERE operation_id=?`, decision.OperationID).Scan(&operation, &owner, &arm, &target, &peak, &leaseID, &state); err != nil {
		t.Fatal(err)
	}
	if operation != string(aisearch.FleetStage) || owner != "qdrant-fleet.search-hub" || arm != "arm-generation-2" || target != "vf-candidate-2" || peak != 4096 || leaseID != decision.ReceiptID || state != "active" {
		t.Fatalf("durable operation identity mismatch: op=%q owner=%q arm=%q target=%q peak=%d lease=%q state=%q", operation, owner, arm, target, peak, leaseID, state)
	}
	if err := gate.CompleteQdrantFleetOperation(ctx, decision, true); err != nil {
		t.Fatal(err)
	}
	if admitter.lease.completeCalls != 1 {
		t.Fatalf("successful provider job completed capacity lease %d times", admitter.lease.completeCalls)
	}
	if err := db.QueryRowContext(ctx, `SELECT state FROM qdrant_fleet_disk_operations WHERE operation_id=?`, decision.OperationID).Scan(&state); err != nil || state != "completed" {
		t.Fatalf("operation state=%q err=%v, want completed", state, err)
	}
}

func TestDiskJobFleetAdmissionKeepsUncertainProviderEffectsReserved(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	admitter := &fleetDiskJobAdmitterFake{lease: &fleetDiskJobLeaseFake{receipt: capacity.DiskJobReceipt{
		ID: "disk-job-2", OwnerID: "qdrant-fleet.search-hub", JobID: "pending", Status: "active",
		CreatedAt: time.Now().UTC(), HeartbeatDeadline: time.Now().UTC().Add(15 * time.Minute),
		Paths: []capacity.DiskJobPathReceipt{{Path: "/qdrant/storage", PeakBytes: 8192}},
	}}}
	gate, err := NewDiskJobFleetAdmissionGate(db, admitter, "/qdrant/storage", "qdrant-fleet.search-hub")
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	decision, err := gate.AdmitQdrantFleetOperation(ctx, aisearch.FleetPromote, "arm-a", "vf-live", 8192)
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.CompleteQdrantFleetOperation(ctx, decision, false); err != nil {
		t.Fatal(err)
	}
	if admitter.lease.completeCalls != 0 {
		t.Fatal("uncertain alias effect released its disk reservation")
	}
	var state string
	if err := db.QueryRowContext(ctx, `SELECT state FROM qdrant_fleet_disk_operations WHERE operation_id=?`, decision.OperationID).Scan(&state); err != nil || state != "unresolved" {
		t.Fatalf("operation state=%q err=%v, want unresolved", state, err)
	}
	changed := decision
	changed.TargetID = "vf-other"
	if err := gate.CompleteQdrantFleetOperation(ctx, changed, true); err == nil {
		t.Fatal("completion accepted a different target identity")
	}
	if err := gate.ResolveUnresolvedAfterCleanup(ctx, decision, false); err == nil {
		t.Fatal("unverified cleanup released an unresolved operation reservation")
	}
	if err := gate.ResolveUnresolvedAfterCleanup(ctx, decision, true); err != nil {
		t.Fatal(err)
	}
	if admitter.lease.releaseCalls != 1 || admitter.lease.completeCalls != 1 {
		t.Fatalf("verified cleanup did not release exactly once: release=%d complete=%d", admitter.lease.releaseCalls, admitter.lease.completeCalls)
	}
}

func TestDiskJobFleetAdmissionRejectsInvalidOrUncoveredRequests(t *testing.T) {
	ctx := context.Background()
	db, _ := openRegistryDB(t)
	lease := &fleetDiskJobLeaseFake{receipt: capacity.DiskJobReceipt{
		ID: "disk-job-3", OwnerID: "qdrant-fleet.search-hub", JobID: "pending", Status: "active",
		CreatedAt: time.Now().UTC(), HeartbeatDeadline: time.Now().UTC().Add(15 * time.Minute),
		Paths: []capacity.DiskJobPathReceipt{{Path: "/qdrant/storage", PeakBytes: 1}},
	}}
	admitter := &fleetDiskJobAdmitterFake{lease: lease}
	gate, err := NewDiskJobFleetAdmissionGate(db, admitter, "/qdrant/storage", "qdrant-fleet.search-hub")
	if err != nil {
		t.Fatal(err)
	}
	if err := gate.Initialize(ctx); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		operation aisearch.FleetOperation
		arm       string
		target    string
		peak      int64
	}{{"semantic_query", "arm", "collection", 1}, {aisearch.FleetStage, "", "collection", 1}, {aisearch.FleetStage, "arm", "collection", 0}} {
		if _, err := gate.AdmitQdrantFleetOperation(ctx, tc.operation, tc.arm, tc.target, tc.peak); err == nil {
			t.Fatalf("invalid admission accepted: %+v", tc)
		}
	}
	if admitter.request.OwnerID != "" {
		t.Fatal("invalid requests reached the capacity owner")
	}
	admitter.err = errors.New("capacity denied")
	if _, err := gate.AdmitQdrantFleetOperation(ctx, aisearch.FleetStage, "arm", "collection", 1024); err == nil {
		t.Fatal("capacity denial was ignored")
	}
}
