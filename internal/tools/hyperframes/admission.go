package hyperframes

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/packages/capacity"
)

// InvocationAdmission describes this invocation, including exact-output reuse.
// The immutable render bundle remains the receipt of the original rendering.
type InvocationAdmission struct {
	OwnerID      string                    `json:"owner_id"`
	RAMBytes     int64                     `json:"ram_bytes"`
	CPUMilli     int64                     `json:"cpu_milli"`
	Verdict      string                    `json:"verdict"`
	Reason       string                    `json:"reason,omitempty"`
	Release      string                    `json:"release"`
	DiskJob      *capacity.DiskJobReceipt  `json:"disk_job,omitempty"`
	DiskHeadroom *capacity.DiskJobHeadroom `json:"disk_headroom,omitempty"`
}

type renderBroker interface {
	Acquire(context.Context, string, int64, int64) (capacity.Lease, capacity.Verdict, error)
	ReserveDiskJob(context.Context, capacity.DiskJobRequest) (capacity.DiskJobLease, error)
	AttachDiskJobLease(context.Context, capacity.DiskJobCredential) (capacity.DiskJobLease, error)
	Close() error
}

type renderBrokerFactory func(context.Context) (renderBroker, error)

func newRenderBroker(ctx context.Context) (renderBroker, error) {
	return capacity.NewBroker(ctx, "")
}

var errOwnedCleanup = errors.New("owned render cleanup is unverified")

func withRenderAdmission(ctx context.Context, memoryMiB int, factory renderBrokerFactory, interval time.Duration, work func(context.Context) (Receipt, error)) (result Receipt, resultErr error) {
	return withRenderAdmissionAndDisk(ctx, memoryMiB, factory, interval, nil, "", work)
}

func withRenderAdmissionAndDisk(ctx context.Context, memoryMiB int, factory renderBrokerFactory, interval time.Duration, diskPaths []capacity.DiskJobPath, diskLeaseFile string, work func(context.Context) (Receipt, error)) (result Receipt, resultErr error) {
	a := &InvocationAdmission{RAMBytes: int64(memoryMiB) << 20, CPUMilli: 2000, Verdict: "unavailable", Release: "not_acquired"}
	defer func() { result.Admission = a }()
	if memoryMiB < 1024 || memoryMiB > 16384 || interval <= 0 {
		return result, errors.New("invalid render admission budget")
	}
	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return result, err
	}
	a.OwnerID = "hyperframes:" + hex.EncodeToString(nonce[:])
	b, err := factory(ctx)
	if err != nil {
		return result, fmt.Errorf("render capacity unavailable: %w", err)
	}
	if b == nil {
		return result, errors.New("render capacity broker is nil")
	}
	defer func() { resultErr = errors.Join(resultErr, b.Close()) }()
	l, v, err := b.Acquire(ctx, a.OwnerID, a.RAMBytes, a.CPUMilli)
	a.Verdict, a.Reason = v.Kind, v.Reason
	if l != nil {
		a.Release = "pending"
		defer func() {
			if errors.Is(resultErr, errOwnedCleanup) {
				a.Release = "retained_until_expiry_cleanup_unverified"
				return
			}
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			if e := l.Release(cleanup); e != nil {
				a.Release = "failed"
				resultErr = errors.Join(resultErr, fmt.Errorf("release render capacity: %w", e))
			} else {
				a.Release = "released"
			}
		}()
	}
	if err != nil {
		return result, fmt.Errorf("render capacity acquisition: %w", err)
	}
	if v.Kind != "grant" || l == nil {
		return result, fmt.Errorf("render capacity %s: %s; retry when capacity is available", v.Kind, v.Reason)
	}
	var diskLease capacity.DiskJobLease
	ownsDiskLease := false
	if diskLeaseFile != "" {
		credential, credentialErr := readDiskJobCredential(ctx, diskLeaseFile)
		if credentialErr != nil {
			return result, fmt.Errorf("read parent disk lease: %w", credentialErr)
		}
		diskLease, err = b.AttachDiskJobLease(ctx, credential)
		if err != nil {
			return result, fmt.Errorf("attach parent disk lease: %w", err)
		}
	} else if len(diskPaths) > 0 {
		diskLease, err = b.ReserveDiskJob(ctx, capacity.DiskJobRequest{
			OwnerID: a.OwnerID, JobID: a.OwnerID + ":render", Paths: diskPaths,
		})
		if err != nil {
			return result, fmt.Errorf("render disk admission: %w", err)
		}
		ownsDiskLease = true
	}
	if (diskLeaseFile != "" || len(diskPaths) > 0) && diskLease == nil {
		return result, errors.New("capacity broker returned no disk job lease")
	}
	if diskLease != nil {
		reservation := diskLease.Reservation()
		a.DiskJob = &reservation
		if ownsDiskLease {
			defer func() {
				if errors.Is(resultErr, errOwnedCleanup) {
					return
				}
				completeCtx, completeCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
				defer completeCancel()
				completed, completeErr := diskLease.Complete(completeCtx)
				if completeErr != nil {
					resultErr = errors.Join(resultErr, fmt.Errorf("complete render disk admission: %w", completeErr))
					return
				}
				a.DiskJob = &completed
			}()
		}
		if len(diskPaths) > 0 {
			updated, extendErr := diskLease.Extend(ctx, diskPaths)
			if extendErr != nil {
				return result, fmt.Errorf("extend render disk admission before writes: %w", extendErr)
			}
			a.DiskJob = &updated
		}
		headroom, observeErr := diskLease.Observe(ctx)
		if observeErr != nil {
			return result, fmt.Errorf("observe render disk lease: %w", observeErr)
		}
		a.DiskHeadroom = &headroom
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	renewable, ok := l.(capacity.RenewableLease)
	if !ok {
		return result, errors.New("render capacity lease cannot renew")
	}
	workCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	stop := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		tick := time.NewTicker(interval)
		defer tick.Stop()
		for {
			select {
			case <-stop:
				done <- nil
				return
			case <-workCtx.Done():
				done <- nil
				return
			case <-tick.C:
				renewCtx, end := context.WithTimeout(workCtx, 5*time.Second)
				err := renewable.Heartbeat(renewCtx)
				if err == nil && diskLease != nil {
					err = diskLease.Heartbeat(renewCtx)
					reservation := diskLease.Reservation()
					a.DiskJob = &reservation
				}
				end()
				if err != nil {
					// Normal completion cancels any in-flight heartbeat.
					select {
					case <-stop:
						done <- nil
						return
					default:
					}
					cancel()
					done <- fmt.Errorf("render capacity renewal: %w", err)
					return
				}
			}
		}
	}()
	result, resultErr = work(workCtx)
	close(stop)
	cancel()
	resultErr = errors.Join(resultErr, ctx.Err(), <-done)
	return result, resultErr
}
