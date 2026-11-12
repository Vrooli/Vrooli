//go:build linux

package capacity

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const diskJobChildEnv = "VROOLI_TEST_DISK_JOB_CHILD"

// This isolated temporary-ledger integration proves a killed producer keeps
// its reservation until the capacity owner verifies both stop and cleanup.
func TestDiskJobCrashedOwnerRetainsLeaseUntilVerifiedReconciliation(t *testing.T) {
	if os.Getenv(diskJobChildEnv) == "1" {
		diskJobChild(t)
		return
	}
	ctx := context.Background()
	root := t.TempDir()
	dbPath := filepath.Join(root, "capacity.db")
	zeroFloor := int64(0)
	initial, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	if err := initial.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestDiskJobCrashedOwnerRetainsLeaseUntilVerifiedReconciliation$")
	cmd.Env = append(os.Environ(), diskJobChildEnv+"=1", "VROOLI_TEST_DISK_DB="+dbPath, "VROOLI_TEST_DISK_PATH="+root)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("child did not report a lease: %v", scanner.Err())
	}
	leaseID := strings.TrimSpace(scanner.Text())
	if leaseID == "" {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("child reported an empty lease id")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = stdin.Close()
	_ = cmd.Wait()

	store, err := NewSQLiteStore(ctx, Config{DBPath: dbPath, MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	active, err := store.ListDiskJobs(ctx, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(active) != 1 || active[0].ID != leaseID || active[0].Status != "active" {
		t.Fatalf("crashed producer reservation was not retained: %+v", active)
	}
	if _, err := store.ReconcileDiskJob(ctx, leaseID, false, true); err == nil {
		t.Fatal("reconciled without stopped-process verification")
	}
	if _, err := store.ReconcileDiskJob(ctx, leaseID, true, false); err == nil {
		t.Fatal("reconciled without cleanup verification")
	}
	reconciled, err := store.ReconcileDiskJob(ctx, leaseID, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Status != "reconciled" || !reconciled.CleanupVerified {
		t.Fatalf("reconciliation receipt=%+v", reconciled)
	}
}

func diskJobChild(t *testing.T) {
	ctx := context.Background()
	path := os.Getenv("VROOLI_TEST_DISK_PATH")
	zeroFloor := int64(0)
	store, err := NewSQLiteStore(ctx, Config{DBPath: os.Getenv("VROOLI_TEST_DISK_DB"), MinimumDiskFloorBytes: &zeroFloor})
	if err != nil {
		t.Fatal(err)
	}
	r, _, err := store.ReserveDiskJob(ctx, DiskJobRequest{OwnerID: "crash-test", JobID: "producer-child", Paths: []DiskJobPath{{Path: path, Role: "temporary-producer-output", PeakBytes: 1 << 20}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	_, _ = fmt.Fprintln(os.Stdout, r.ID)
	_, _ = io.Copy(io.Discard, os.Stdin) // parent kills this producer after admission
	os.Exit(0)
}
