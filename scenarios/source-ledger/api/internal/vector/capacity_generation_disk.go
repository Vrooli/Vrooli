package vector

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

// GenerationDiskJobRequest is the small owner contract translated to
// packages/capacity.DiskJobRequest by the governed composition adapter. Keeping
// this package free of that import lets focused Source Ledger tests run before
// SDA admits the dependency.
type GenerationDiskJobRequest struct {
	OwnerID    string
	JobID      string
	Path       string
	Role       string
	PeakBytes  int64
	FloorBytes int64
}

// GenerationDiskJobLease mirrors only the shared lease operations that this
// producer needs. A capacity.DiskJobLease can be exposed through a tiny
// closure-backed bridge once the governed dependency is available.
type GenerationDiskJobLease interface {
	ReceiptID() string
	Complete(context.Context) error
}

type GenerationDiskJobReserve func(context.Context, GenerationDiskJobRequest) (GenerationDiskJobLease, error)

// GenerationDiskJobReceipt is the owner-visible identity needed to reattach
// one persisted reservation after the Source Ledger producer has restarted.
// The capacity owner remains authoritative for checking the producer exit and
// rotating the private lease capability.
type GenerationDiskJobReceipt struct {
	ID     string
	Owner  string
	JobID  string
	Status string
	Path   string
	Role   string
	Peak   int64
}

type (
	GenerationDiskJobFind     func(context.Context, string, string) (GenerationDiskJobReceipt, bool, error)
	GenerationDiskJobReattach func(context.Context, string, string, string) (GenerationDiskJobLease, error)
)

// generationDiskJobKey is what an operation ID must mean on every retry.
type generationDiskJobKey struct {
	targetID string
	recipeID string
	peak     int64
}

type generationDiskJob struct {
	key     generationDiskJobKey
	receipt string
	lease   GenerationDiskJobLease
	done    bool
}

// CapacityGenerationDiskAdmission translates the generation backend's stable
// operation identity into a shared capacity disk-job reservation. It holds an
// uncertain lease in memory and releases it only after the backend reports
// successful physical completion. The capacity receipt itself is durable in
// Source Ledger's operation record.
//
// On process restart, Find locates the owner's durable job receipt and
// Reattach asks the shared capacity owner to verify old-producer exit and
// rotate the private capability. Missing or mismatched callback pairs fail
// closed; a receipt alone never proves that the reservation can be released.
type CapacityGenerationDiskAdmission struct {
	OwnerID    string
	Path       string
	FloorBytes int64
	Reserve    GenerationDiskJobReserve
	Find       GenerationDiskJobFind
	Reattach   GenerationDiskJobReattach

	mu   sync.Mutex
	jobs map[string]*generationDiskJob
}

func (a *CapacityGenerationDiskAdmission) Admit(ctx context.Context, operationID, targetID, recipeID string, peakBytes int64) (string, error) {
	operationID = strings.TrimSpace(operationID)
	request, err := a.jobRequest(operationID, peakBytes)
	if err != nil {
		return "", err
	}
	if anyBlank(operationID, targetID, recipeID) || peakBytes <= 0 {
		return "", errors.New("Source Ledger capacity admission requires operation, target, recipe, and positive peak bytes")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	key := generationDiskJobKey{targetID: targetID, recipeID: recipeID, peak: peakBytes}
	if prior := a.jobs[operationID]; prior != nil {
		if prior.key != key {
			return "", ErrGenerationOperationConflict
		}
		return prior.receipt, nil
	}
	lease, err := a.acquire(ctx, request)
	if err != nil {
		return "", err
	}
	if a.jobs == nil {
		a.jobs = make(map[string]*generationDiskJob)
	}
	a.jobs[operationID] = &generationDiskJob{key: key, receipt: lease.ReceiptID(), lease: lease}
	return lease.ReceiptID(), nil
}

// jobRequest reserves the database directory so SQLite's WAL and journal
// sidecars stay inside the admitted write boundary.
func (a *CapacityGenerationDiskAdmission) jobRequest(operationID string, peakBytes int64) (GenerationDiskJobRequest, error) {
	if a == nil || a.Reserve == nil {
		return GenerationDiskJobRequest{}, ErrGenerationDiskAdmissionUnavailable
	}
	if anyBlank(a.OwnerID, a.Path) || a.FloorBytes < 0 {
		return GenerationDiskJobRequest{}, errors.New("Source Ledger capacity admission requires owner, database path, and non-negative floor")
	}
	databasePath, err := filepath.Abs(a.Path)
	if err != nil {
		return GenerationDiskJobRequest{}, fmt.Errorf("resolve Source Ledger capacity path: %w", err)
	}
	return GenerationDiskJobRequest{
		OwnerID: a.OwnerID, JobID: "source-ledger-vector-generation:" + operationID,
		Path: filepath.Dir(databasePath), Role: "sqlite-vector-generation", PeakBytes: peakBytes,
		FloorBytes: a.FloorBytes,
	}, nil
}

// acquire reattaches a reservation that survived a restart, or reserves a new
// one. Recovery needs both callbacks; either alone fails closed.
func (a *CapacityGenerationDiskAdmission) acquire(ctx context.Context, request GenerationDiskJobRequest) (GenerationDiskJobLease, error) {
	if (a.Find == nil) != (a.Reattach == nil) {
		return nil, errors.New("Source Ledger capacity recovery requires both job lookup and reattachment")
	}
	if a.Find != nil {
		prior, found, err := a.Find(ctx, a.OwnerID, request.JobID)
		if err != nil {
			return nil, fmt.Errorf("find Source Ledger generation disk job: %w", err)
		}
		if found {
			return a.reattach(ctx, request, prior)
		}
	}
	lease, err := a.Reserve(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("admit Source Ledger generation disk job: %w", err)
	}
	if lease == nil || anyBlank(lease.ReceiptID()) {
		return nil, errors.New("shared capacity returned an empty Source Ledger generation receipt")
	}
	return lease, nil
}

// reattach accepts only the active job this request would have reserved.
func (a *CapacityGenerationDiskAdmission) reattach(ctx context.Context, request GenerationDiskJobRequest, prior GenerationDiskJobReceipt) (GenerationDiskJobLease, error) {
	want := GenerationDiskJobReceipt{
		ID: prior.ID, Owner: request.OwnerID, JobID: request.JobID, Status: "active",
		Path: request.Path, Role: request.Role, Peak: request.PeakBytes,
	}
	if anyBlank(prior.ID) || prior != want {
		return nil, fmt.Errorf("Source Ledger generation disk job does not match the requested active reservation")
	}
	lease, err := a.Reattach(ctx, a.OwnerID, request.JobID, prior.ID)
	if err != nil {
		return nil, fmt.Errorf("reattach Source Ledger generation disk job: %w", err)
	}
	if lease == nil || lease.ReceiptID() != prior.ID {
		return nil, errors.New("reattached Source Ledger disk job returned a different receipt identity")
	}
	return lease, nil
}

// Complete(false) deliberately leaves the lease active. The generation may
// have durable partial effects, so releasing it would allow competing work to
// consume the reserved headroom before recovery determines the real footprint.
func (a *CapacityGenerationDiskAdmission) Complete(ctx context.Context, operationID, receiptID string, success bool) error {
	if a == nil {
		return ErrGenerationDiskAdmissionUnavailable
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	job := a.jobs[operationID]
	if job == nil || job.receipt != receiptID || receiptID == "" {
		return errors.New("Source Ledger generation capacity receipt identity mismatch")
	}
	if !success || job.done {
		return nil
	}
	if job.lease == nil {
		return ErrGenerationDiskAdmissionUnavailable
	}
	if err := job.lease.Complete(ctx); err != nil {
		return fmt.Errorf("complete Source Ledger generation disk job: %w", err)
	}
	job.done = true
	job.lease = nil
	return nil
}
