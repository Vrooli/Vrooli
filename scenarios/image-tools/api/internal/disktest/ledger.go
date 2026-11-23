// Package disktest is an in-memory stand-in for the shared disk-job ledger,
// for tests of code that reserves disk. It enforces the ledger's contract that
// matters to producers: one immutable receipt per (owner, job key), so a
// producer that reuses a key fails exactly as it does against the real ledger.
package disktest

import (
	"context"
	"fmt"
	"sync"

	"github.com/vrooli/vrooli/packages/capacity"
)

// Ledger records every reservation it admits.
type Ledger struct {
	mu       sync.Mutex
	rows     []*row
	producer map[string]bool // lease id -> producer still running
}

type row struct {
	receipt capacity.DiskJobReceipt
}

// DuplicateKeyError is the ledger's refusal of a reused (owner, job) key. Its
// text matches the real constraint failure so tests read like the live one.
func DuplicateKeyError() error {
	return fmt.Errorf("create disk job lease: constraint failed: UNIQUE constraint failed: capacity_disk_jobs.owner_id, capacity_disk_jobs.job_id")
}

// Seed adds a row left behind by an earlier attempt: status is "active" or
// "completed", and producerAlive says whether the process that wrote it runs.
func (l *Ledger) Seed(owner, jobID, status string, producerAlive bool) capacity.DiskJobReceipt {
	l.mu.Lock()
	defer l.mu.Unlock()
	r := l.insertLocked(owner, jobID, status)
	if l.producer == nil {
		l.producer = map[string]bool{}
	}
	l.producer[r.ID] = producerAlive
	return r
}

func (l *Ledger) insertLocked(owner, jobID, status string) capacity.DiskJobReceipt {
	r := capacity.DiskJobReceipt{ID: fmt.Sprintf("lease-%d", len(l.rows)+1), OwnerID: owner, JobID: jobID, Status: status}
	l.rows = append(l.rows, &row{receipt: r})
	return r
}

// ReserveDiskJob admits the request unless its (owner, job) key is taken.
func (l *Ledger) ReserveDiskJob(_ context.Context, req capacity.DiskJobRequest) (capacity.DiskJobLease, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.rows {
		if r.receipt.OwnerID == req.OwnerID && r.receipt.JobID == req.JobID {
			return nil, DuplicateKeyError()
		}
	}
	r := l.insertLocked(req.OwnerID, req.JobID, "active")
	return &lease{ledger: l, id: r.ID}, nil
}

// ListDiskJobs mirrors the broker listing for one owner.
func (l *Ledger) ListDiskJobs(_ context.Context, owner string, activeOnly bool) ([]capacity.DiskJobReceipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []capacity.DiskJobReceipt
	for _, r := range l.rows {
		if r.receipt.OwnerID == owner && (!activeOnly || r.receipt.Status == "active") {
			out = append(out, r.receipt)
		}
	}
	return out, nil
}

// ReattachDiskJob mirrors the broker: only an active lease whose producer has
// exited can be taken over, for the exact owner and job key.
func (l *Ledger) ReattachDiskJob(_ context.Context, owner, jobID, leaseID string) (capacity.DiskJobLease, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range l.rows {
		if r.receipt.ID != leaseID {
			continue
		}
		if r.receipt.OwnerID != owner || r.receipt.JobID != jobID {
			return nil, fmt.Errorf("disk job lease does not match owner and stable job identity")
		}
		if r.receipt.Status != "active" {
			return nil, fmt.Errorf("disk job %s is not active", leaseID)
		}
		if l.producer[leaseID] {
			return nil, fmt.Errorf("disk job process is still running with the recorded identity")
		}
		return &lease{ledger: l, id: leaseID}, nil
	}
	return nil, fmt.Errorf("disk job %s not found", leaseID)
}

// SupersedeDiskJobs closes the owner's active leases under the prefix whose
// producer has exited and whose key ownsKey accepts, as the shared broker does.
func (l *Ledger) SupersedeDiskJobs(ctx context.Context, owner string, ownsKey func(string) bool) (int, error) {
	active, _ := l.ListDiskJobs(ctx, owner, true)
	closed := 0
	for _, r := range active {
		if !ownsKey(r.JobID) {
			continue
		}
		lease, err := l.ReattachDiskJob(ctx, owner, r.JobID, r.ID)
		if err != nil {
			continue
		}
		if _, err := lease.Complete(ctx); err == nil {
			closed++
		}
	}
	return closed, nil
}

// Rows returns a copy of every receipt.
func (l *Ledger) Rows() []capacity.DiskJobReceipt {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]capacity.DiskJobReceipt, 0, len(l.rows))
	for _, r := range l.rows {
		out = append(out, r.receipt)
	}
	return out
}

// Active counts receipts still holding a reservation.
func (l *Ledger) Active() int {
	n := 0
	for _, r := range l.Rows() {
		if r.Status == "active" {
			n++
		}
	}
	return n
}

type lease struct {
	ledger *Ledger
	id     string
}

func (x *lease) find() *row {
	for _, r := range x.ledger.rows {
		if r.receipt.ID == x.id {
			return r
		}
	}
	return nil
}

func (x *lease) Heartbeat(context.Context) error { return nil }
func (x *lease) Observe(context.Context) (capacity.DiskJobHeadroom, error) {
	return capacity.DiskJobHeadroom{LeaseID: x.id, Status: x.Reservation().Status}, nil
}
func (x *lease) Extend(context.Context, []capacity.DiskJobPath) (capacity.DiskJobReceipt, error) {
	return x.Reservation(), nil
}
func (x *lease) SamplePath(context.Context, string, int64, int64) (capacity.DiskJobReceipt, error) {
	return x.Reservation(), nil
}
func (x *lease) Complete(context.Context) (capacity.DiskJobReceipt, error) {
	x.ledger.mu.Lock()
	defer x.ledger.mu.Unlock()
	r := x.find()
	r.receipt.Status = "completed"
	return r.receipt, nil
}
func (x *lease) Release(ctx context.Context) error { _, err := x.Complete(ctx); return err }
func (x *lease) Reservation() capacity.DiskJobReceipt {
	x.ledger.mu.Lock()
	defer x.ledger.mu.Unlock()
	return x.find().receipt
}
