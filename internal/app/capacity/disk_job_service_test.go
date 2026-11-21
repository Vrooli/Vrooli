package capacity

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	engine "github.com/vrooli/vrooli/internal/capacity"
)

func TestDiskJobServicePersistsLifecycleAndGrowthReceiptAcrossCalls(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(root, "ledger", "capacity.db")
	if err := os.MkdirAll(filepath.Dir(ledgerPath), 0o700); err != nil {
		t.Fatal(err)
	}
	zeroFloor := int64(0)
	svc := Service{OpenStore: func(ctx context.Context) (Store, error) {
		return engine.NewSQLiteStore(ctx, engine.Config{DBPath: ledgerPath, MinimumDiskFloorBytes: &zeroFloor})
	}}

	request := engine.DiskJobRequest{
		OwnerID: "capacity-service-test", JobID: "disk-receipt-e2e",
		Paths: []engine.DiskJobPath{{Path: target, Role: "output-and-temp", PeakBytes: 2 << 20}},
	}
	reserved, credential, err := svc.DiskJobReserve(ctx, request)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if reserved.Status != "active" || credential.ID != reserved.ID || credential.Capability == "" {
		t.Fatalf("reserve receipt/credential mismatch: receipt=%+v credential=%+v", reserved, credential)
	}

	listed, err := svc.DiskJobList(ctx, true)
	if err != nil || len(listed) != 1 || listed[0].ID != reserved.ID {
		t.Fatalf("active list = %+v, %v", listed, err)
	}
	if _, err := svc.DiskJobHeartbeat(ctx, credential); err != nil {
		t.Fatalf("heartbeat: %v", err)
	}
	observed, err := svc.DiskJobObserve(ctx, credential)
	if err != nil || observed.LeaseID != reserved.ID || len(observed.Filesystems) != 1 {
		t.Fatalf("observe = %+v, %v", observed, err)
	}

	extended, err := svc.DiskJobExtend(ctx, credential, []engine.DiskJobPath{{Path: target, Role: "output-and-temp", PeakBytes: 4 << 20}})
	if err != nil || extended.Filesystems[0].PeakBytes != 4<<20 {
		t.Fatalf("extend = %+v, %v", extended, err)
	}
	if err := os.WriteFile(filepath.Join(target, "showcase.bin"), bytes.Repeat([]byte{0x5a}, 1<<20), 0o600); err != nil {
		t.Fatal(err)
	}

	completed, err := svc.DiskJobComplete(ctx, credential)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if completed.Status != "completed" || !completed.CleanupVerified || len(completed.Filesystems) != 1 || completed.Filesystems[0].ActualDeltaBytes == nil {
		t.Fatalf("terminal receipt incomplete: %+v", completed)
	}
	// Filesystem free-space delta may be negative if unrelated same-host work
	// releases cache/data during the job; presence and signed value are the
	// contract, not a presumed positive attribution to this producer.
	outputInfo, err := os.Stat(filepath.Join(target, "showcase.bin"))
	if err != nil || outputInfo.Size() != 1<<20 {
		t.Fatalf("written output stat=%v error=%v", outputInfo, err)
	}
	active, err := svc.DiskJobList(ctx, true)
	if err != nil || len(active) != 0 {
		t.Fatalf("active list after complete = %+v, %v", active, err)
	}
	all, err := svc.DiskJobList(ctx, false)
	if err != nil || len(all) != 1 || all[0].ID != reserved.ID || all[0].Filesystems[0].ActualDeltaBytes == nil {
		t.Fatalf("durable completed receipt = %+v, %v", all, err)
	}
}
