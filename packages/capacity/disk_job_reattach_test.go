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

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
)

const diskJobReattachChild = "VROOLI_TEST_CAPACITY_REATTACH_CHILD"

func TestBrokerReattachesActiveDiskJobForExactOwnerAndJob(t *testing.T) {
	if os.Getenv(diskJobReattachChild) == "1" {
		capacityReattachChild(t)
		return
	}
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "capacity.db")
	zero := int64(0)
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: dbPath, MinimumDiskFloorBytes: &zero})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestBrokerReattachesActiveDiskJobForExactOwnerAndJob$")
	cmd.Env = append(os.Environ(), diskJobReattachChild+"=1", "VROOLI_TEST_CAPACITY_DB="+dbPath, "VROOLI_TEST_CAPACITY_PATH="+dir)
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
		t.Fatalf("capacity child did not report its lease: %v", scanner.Err())
	}
	leaseID := strings.TrimSpace(scanner.Text())
	store, err = internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: dbPath, MinimumDiskFloorBytes: &zero})
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	defer store.Close()
	broker := NewBrokerWithSource(store, nil)
	if _, err := broker.ReattachDiskJob(ctx, "scenario-owner", "stable-op-1", leaseID); err == nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal("broker reattached while the old producer was still running")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = stdin.Close()
	_ = cmd.Wait()
	if _, err := broker.ReattachDiskJob(ctx, "other-owner", "stable-op-1", leaseID); err == nil {
		t.Fatal("broker reattached a lease for another owner")
	}
	if _, err := broker.ReattachDiskJob(ctx, "scenario-owner", "other-op", leaseID); err == nil {
		t.Fatal("broker reattached a lease for another stable job")
	}
	lease, err := broker.ReattachDiskJob(ctx, "scenario-owner", "stable-op-1", leaseID)
	if err != nil {
		t.Fatal(err)
	}
	if got := lease.Reservation(); got.Status != "active" || got.OwnerID != "scenario-owner" || got.JobID != "stable-op-1" {
		t.Fatalf("reattached lease identity/status = %+v", got)
	}
	if _, err := broker.ReattachDiskJob(ctx, "scenario-owner", "stable-op-1", leaseID); err == nil {
		t.Fatal("broker rotated a lease away from its current live producer")
	}
	if _, err := lease.Complete(ctx); err != nil {
		t.Fatalf("complete reattached lease: %v", err)
	}
}

func capacityReattachChild(t *testing.T) {
	ctx := context.Background()
	zero := int64(0)
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: os.Getenv("VROOLI_TEST_CAPACITY_DB"), MinimumDiskFloorBytes: &zero})
	if err != nil {
		t.Fatal(err)
	}
	broker := NewBrokerWithSource(store, nil)
	lease, err := broker.ReserveDiskJob(ctx, DiskJobRequest{
		OwnerID: "scenario-owner", JobID: "stable-op-1",
		Paths: []DiskJobPath{{Path: os.Getenv("VROOLI_TEST_CAPACITY_PATH"), Role: "generation", PeakBytes: 1 << 20}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	_, _ = fmt.Fprintln(os.Stdout, lease.Reservation().ID)
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}
