package renderjobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"device-studio/internal/jobs"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

const minimumRenderDiskFreeBytes int64 = 120 << 30

// DiskJobAdmitter is the shared host-capacity owner boundary for one complete
// render attempt. Production supplies packages/capacity; tests use bounded
// fake filesystems and leases.
type DiskJobAdmitter = sharedcapacity.DiskJobAdmitter
type DiskJobLease = sharedcapacity.DiskJobLease

// NewSharedDiskAdmitter connects render jobs to the platform's durable
// cross-process disk owner. A broker handle remains open through completion so
// the producer can heartbeat while its detached native operation runs.
func NewSharedDiskAdmitter() DiskJobAdmitter { return sharedDiskAdmitter{} }

type sharedDiskAdmitter struct{}

func (sharedDiskAdmitter) ReserveDiskJob(ctx context.Context, request sharedcapacity.DiskJobRequest) (sharedcapacity.DiskJobLease, error) {
	broker, err := sharedcapacity.NewBroker(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("open platform disk capacity owner: %w", err)
	}
	lease, err := broker.ReserveDiskJob(ctx, request)
	if err != nil {
		return nil, errors.Join(err, broker.Close())
	}
	return &diskJobLeaseWithBroker{DiskJobLease: lease, broker: broker}, nil
}

type diskJobLeaseWithBroker struct {
	sharedcapacity.DiskJobLease
	broker *sharedcapacity.Broker
}

func (l *diskJobLeaseWithBroker) Complete(ctx context.Context) (sharedcapacity.DiskJobReceipt, error) {
	receipt, err := l.DiskJobLease.Complete(ctx)
	if err != nil {
		return receipt, err
	}
	return receipt, l.broker.Close()
}

// EstimateRenderPeakBytes conservatively bounds all owner-controlled writes
// from dispatch through output publication. All render artifacts and workspaces
// are below the routed data root, so one path covers their filesystem.
func EstimateRenderPeakBytes(job jobs.Job) (int64, error) {
	mode, err := operationModeForJob(job.Input)
	if err != nil {
		return 0, err
	}
	const MiB = int64(1 << 20)
	switch mode {
	case operationStillRender:
		return maxWorkerBytes + maxWorkerInputBytes + 2*maxOutputBytes + MiB, nil
	case operationSourceInspect:
		return maxWorkerBytes + maxInspectionInputBytes + maxInspectionSidecarBytes + MiB, nil
	case operationVideoRender:
		// Includes staged video inputs, the pinned worker, the capped MP4 in the
		// attempt workspace and its published copy, plus bounded receipts.
		return maxWorkerBytes + maxVideoWorkspaceInputBytes + 2*maxVideoOutputBytes + MiB, nil
	case operationPhysicalRender, operationPhysicalPreview, operationPhysicalMotion, operationPhysicalMotionV2, operationPhysicalRecording:
		// Physical motion can retain first/middle/last previews, an interrupted
		// preview, one checkpoint and one producer scratch frame concurrently.
		// Each is capped at 64 MiB by the worker; reserve all six plus the
		// workspace/published output pair and bounded input/worker bytes.
		return physicalWorkspaceInputMax + maxWorkerBytes + 2*maxOutputBytes + 6*(64<<20) + (8 << 20), nil
	case operationGeometryCreate:
		// Geometry creation stages at most four source images and publishes one
		// bounded response plus its retained evidence.
		return physicalWorkspaceInputMax + maxWorkerBytes + 2*maxOutputBytes + (8 << 20), nil
	default:
		return 0, fmt.Errorf("renderjobs: no disk peak forecast for operation %q", mode)
	}
}

type diskLeaseEntry struct {
	lease DiskJobLease
}

func (c *Coordinator) reserveDiskJob(ctx context.Context, route routePin, job jobs.Job) (DiskJobLease, bool, error) {
	if c.diskJobs == nil {
		return nil, false, errors.New("renderjobs: shared disk capacity owner is unavailable")
	}
	peak, err := EstimateRenderPeakBytes(job)
	if err != nil {
		return nil, false, err
	}
	lease, err := c.diskJobs.ReserveDiskJob(ctx, sharedcapacity.DiskJobRequest{
		OwnerID: OwnerID,
		JobID:   job.ID,
		Paths: []sharedcapacity.DiskJobPath{{
			Path: route.Root, Role: "device-studio-render-workspace-and-output", PeakBytes: peak,
		}},
		FloorBytes: minimumRenderDiskFreeBytes,
	})
	if err != nil {
		return nil, false, fmt.Errorf("renderjobs: reserve %d-byte render peak: %w", peak, err)
	}
	r := lease.Reservation()
	if r.ID == "" || r.OwnerID != OwnerID || r.JobID != job.ID || r.Status != "active" || len(r.Filesystems) != 1 || len(r.Paths) != 1 || r.Filesystems[0].PeakBytes < peak || r.Filesystems[0].FloorBytes < minimumRenderDiskFreeBytes {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		cleanupErr := lease.Release(cleanup)
		cancel()
		return nil, false, errors.Join(errors.New("renderjobs: shared disk owner returned an incomplete or undersized receipt"), cleanupErr)
	}
	c.diskLeaseMu.Lock()
	if c.diskLeases == nil {
		c.diskLeases = make(map[string]diskLeaseEntry)
	}
	if old, exists := c.diskLeases[job.ID]; exists {
		c.diskLeaseMu.Unlock()
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		cleanupErr := lease.Release(cleanup)
		cancel()
		if cleanupErr != nil {
			return nil, false, fmt.Errorf("renderjobs: duplicate disk lease cleanup failed: %w", cleanupErr)
		}
		if err := old.lease.Heartbeat(ctx); err != nil {
			return nil, false, fmt.Errorf("renderjobs: existing disk lease renewal failed: %w", err)
		}
		return old.lease, false, nil
	}
	c.diskLeases[job.ID] = diskLeaseEntry{lease: lease}
	c.diskLeaseMu.Unlock()
	return lease, true, nil
}

func (c *Coordinator) heartbeatDiskJob(ctx context.Context, jobID string) error {
	c.diskLeaseMu.Lock()
	entry, ok := c.diskLeases[jobID]
	c.diskLeaseMu.Unlock()
	if !ok {
		return nil // A restarted API cannot impersonate the old capacity producer.
	}
	if err := entry.lease.Heartbeat(ctx); err != nil {
		return fmt.Errorf("renderjobs: renew shared disk job %s: %w", jobID, err)
	}
	return nil
}

func (c *Coordinator) completeDiskJob(ctx context.Context, jobID string) error {
	c.diskLeaseMu.Lock()
	defer c.diskLeaseMu.Unlock()
	entry, ok := c.diskLeases[jobID]
	if !ok {
		return nil
	}
	receipt, err := entry.lease.Complete(ctx)
	if err != nil && receipt.Status != "completed" {
		return fmt.Errorf("renderjobs: complete shared disk job %s: %w", jobID, err)
	}
	delete(c.diskLeases, jobID)
	if err != nil {
		return fmt.Errorf("renderjobs: shared disk job %s completed but owner handle close failed: %w", jobID, err)
	}
	return nil
}

func (c *Coordinator) completeDiskJobDetached(ctx context.Context, jobID string) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	return c.completeDiskJob(cleanup, jobID)
}
