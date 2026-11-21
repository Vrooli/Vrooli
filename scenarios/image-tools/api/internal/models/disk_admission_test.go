package models

import (
	"context"

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
