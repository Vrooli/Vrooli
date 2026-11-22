package models

import (
	"context"
	"errors"
	"testing"

	"github.com/vrooli/vrooli/packages/capacity"
)

type testDiskAdmitter struct{}

func (testDiskAdmitter) ReserveDiskJob(_ context.Context, r capacity.DiskJobRequest) (capacity.DiskJobLease, error) {
	return &testDiskLease{receipt: capacity.DiskJobReceipt{ID: r.JobID, OwnerID: r.OwnerID, JobID: r.JobID, Status: "active"}}, nil
}

type testDiskLease struct{ receipt capacity.DiskJobReceipt }

func (l *testDiskLease) Heartbeat(context.Context) error { return nil }
func (l *testDiskLease) Observe(context.Context) (capacity.DiskJobHeadroom, error) {
	return capacity.DiskJobHeadroom{LeaseID: l.receipt.ID, Status: "active"}, nil
}
func (l *testDiskLease) Extend(context.Context, []capacity.DiskJobPath) (capacity.DiskJobReceipt, error) {
	return l.receipt, nil
}
func (l *testDiskLease) SamplePath(context.Context, string, int64, int64) (capacity.DiskJobReceipt, error) {
	return l.receipt, nil
}
func (l *testDiskLease) Complete(context.Context) (capacity.DiskJobReceipt, error) {
	l.receipt.Status = "completed"
	return l.receipt, nil
}
func (l *testDiskLease) Release(ctx context.Context) error    { _, err := l.Complete(ctx); return err }
func (l *testDiskLease) Reservation() capacity.DiskJobReceipt { return l.receipt }

// refusingDiskAdmitter fails the test if an install reserves disk.
type refusingDiskAdmitter struct{ t *testing.T }

func (a refusingDiskAdmitter) ReserveDiskJob(_ context.Context, r capacity.DiskJobRequest) (capacity.DiskJobLease, error) {
	a.t.Errorf("revalidation reserved a disk peak: %+v", r.Paths)
	return nil, errors.New("unexpected disk reservation")
}

func TestRevalidatingValidWeightsReservesNoDiskPeak(t *testing.T) {
	f := newInstallFixture(t)
	ctx := context.Background()
	if _, err := f.in.Install(ctx, installTestModelID, nil); err != nil {
		t.Fatal(err)
	}
	downloads := f.downloads
	// A second install of intact weights (the `models install` / auto
	// revalidation path for a pending smoke) writes nothing, so it must not
	// plan or reserve an acquisition peak.
	f.in.DiskAdmission = refusingDiskAdmitter{t: t}
	if _, err := f.in.InstallForJob(ctx, installTestModelID, "revalidate-job", nil); err != nil {
		t.Fatalf("revalidate: %v", err)
	}
	if f.downloads != downloads {
		t.Fatalf("revalidation downloaded again: %d -> %d", downloads, f.downloads)
	}
}
