package physicalruntime

import (
	"context"
	"time"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

// testDiskJobAdmitter gives render coordinator tests a deterministic in-memory
// disk owner. Production paths still use the shared capacity broker.
type testDiskJobAdmitter struct{}

func (testDiskJobAdmitter) ReserveDiskJob(_ context.Context, request sharedcapacity.DiskJobRequest) (sharedcapacity.DiskJobLease, error) {
	paths := make([]sharedcapacity.DiskJobPathReceipt, 0, len(request.Paths))
	for _, path := range request.Paths {
		paths = append(paths, sharedcapacity.DiskJobPathReceipt{Path: path.Path, Role: path.Role, PeakBytes: path.PeakBytes})
	}
	filesystems := make([]sharedcapacity.DiskJobFilesystem, 0, len(request.Paths))
	for range request.Paths {
		filesystems = append(filesystems, sharedcapacity.DiskJobFilesystem{FilesystemID: "test", PeakBytes: request.Paths[0].PeakBytes, FloorBytes: request.FloorBytes})
	}
	return &testDiskJobLease{receipt: sharedcapacity.DiskJobReceipt{ID: "test-disk-job", OwnerID: request.OwnerID, JobID: request.JobID, Status: "active", Paths: paths, Filesystems: filesystems, HeartbeatDeadline: time.Now().Add(time.Minute)}}, nil
}

type testDiskJobLease struct{ receipt sharedcapacity.DiskJobReceipt }

func (l *testDiskJobLease) Heartbeat(context.Context) error { return nil }
func (l *testDiskJobLease) Observe(context.Context) (sharedcapacity.DiskJobHeadroom, error) {
	return sharedcapacity.DiskJobHeadroom{}, nil
}
func (l *testDiskJobLease) Extend(context.Context, []sharedcapacity.DiskJobPath) (sharedcapacity.DiskJobReceipt, error) {
	return l.receipt, nil
}
func (l *testDiskJobLease) SamplePath(context.Context, string, int64, int64) (sharedcapacity.DiskJobReceipt, error) {
	return l.receipt, nil
}
func (l *testDiskJobLease) Complete(context.Context) (sharedcapacity.DiskJobReceipt, error) {
	l.receipt.Status = "completed"
	return l.receipt, nil
}
func (l *testDiskJobLease) Release(context.Context) error              { l.receipt.Status = "released"; return nil }
func (l *testDiskJobLease) Reservation() sharedcapacity.DiskJobReceipt { return l.receipt }
