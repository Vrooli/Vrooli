package capacity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
)

// DiskJobPath declares one owner-controlled write path and its peak additional
// bytes. The capacity owner canonicalizes it and deduplicates aliases.
type DiskJobPath struct {
	Path              string `json:"path"`
	Role              string `json:"role"`
	PeakBytes         int64  `json:"peakBytes"`
	DockerContainer   string `json:"dockerContainer,omitempty"`
	DockerDestination string `json:"dockerDestination,omitempty"`
	DockerUpperDir    bool   `json:"dockerUpperDir,omitempty"`
	DockerRootDir     bool   `json:"dockerRootDir,omitempty"`
}

type DiskJobRequest struct {
	OwnerID      string        `json:"ownerId"`
	JobID        string        `json:"jobId"`
	ProcessID    int           `json:"processId,omitempty"`
	ProcessStart string        `json:"processStart,omitempty"`
	Paths        []DiskJobPath `json:"paths"`
	FloorBytes   int64         `json:"floorBytes"`
}

// DiskJobCredential is the private capability plus producer identity serialized
// by `vrooli capacity disk-reserve` into a 0600 lease file.
type DiskJobCredential struct {
	ID           string `json:"id"`
	Capability   string `json:"capability"`
	ProcessID    int    `json:"process_id"`
	ProcessStart string `json:"process_start"`
}

type DiskJobFilesystem struct {
	FilesystemID       string `json:"filesystemId"`
	PeakBytes          int64  `json:"peakBytes"`
	FloorBytes         int64  `json:"floorBytes"`
	AvailableBytes     int64  `json:"availableBytes"`
	OtherReservedBytes int64  `json:"otherReservedBytes"`
	ActualDeltaBytes   *int64 `json:"actualDeltaBytes,omitempty"`
	ActualDeltaBasis   string `json:"actualDeltaBasis"`
}

type DiskJobPathReceipt struct {
	Path         string              `json:"path"`
	Role         string              `json:"role"`
	FilesystemID string              `json:"filesystemId"`
	PeakBytes    int64               `json:"peakBytes"`
	Samples      []DiskJobPathSample `json:"samples,omitempty"`
}

type DiskJobPathSample struct {
	Path              string    `json:"path"`
	FilesystemID      string    `json:"filesystemId"`
	Bytes             int64     `json:"bytes"`
	BaselineBytes     int64     `json:"baselineBytes"`
	DeltaBytes        int64     `json:"deltaBytes"`
	ReservedPeakBytes int64     `json:"reservedPeakBytes"`
	ObservedAt        time.Time `json:"observedAt"`
	ExceedsPeak       bool      `json:"exceedsPeak"`
}

type DiskJobReceipt struct {
	ID                string               `json:"id"`
	OwnerID           string               `json:"ownerId"`
	JobID             string               `json:"jobId"`
	ProcessID         int                  `json:"processId"`
	ProcessStart      string               `json:"processStart"`
	Status            string               `json:"status"`
	Paths             []DiskJobPathReceipt `json:"paths"`
	Filesystems       []DiskJobFilesystem  `json:"filesystems"`
	CreatedAt         time.Time            `json:"createdAt"`
	UpdatedAt         time.Time            `json:"updatedAt"`
	LastHeartbeatAt   time.Time            `json:"lastHeartbeatAt"`
	HeartbeatDeadline time.Time            `json:"heartbeatDeadlineAt"`
	CleanupVerified   bool                 `json:"cleanupVerified"`
}

type DiskJobHeadroomFilesystem struct {
	FilesystemID            string `json:"filesystemId"`
	CurrentFreeBytes        int64  `json:"currentFreeBytes"`
	OtherReservedBytes      int64  `json:"otherReservedBytes"`
	CurrentJobPeakBytes     int64  `json:"currentJobPeakBytes"`
	FloorBytes              int64  `json:"floorBytes"`
	SafeAdditionalPeakBytes int64  `json:"safeAdditionalPeakBytes"`
}

type DiskJobHeadroom struct {
	LeaseID     string                      `json:"leaseId"`
	Status      string                      `json:"status"`
	Filesystems []DiskJobHeadroomFilesystem `json:"filesystems"`
}

// DiskJobRecoveryEvidence is the affirmative owner report required before
// asking the capacity service to reconcile an abandoned job. The capacity
// store independently verifies that the bound producer identity is gone.
type DiskJobRecoveryEvidence struct {
	OwnerID                string `json:"ownerId"`
	ProcessStoppedVerified bool   `json:"processStoppedVerified"`
	CleanupVerified        bool   `json:"cleanupVerified"`
}

// DiskJobLease holds its capability in memory so mutation calls require both
// the matching process identity and unguessable lease secret.
type DiskJobLease interface {
	Heartbeat(context.Context) error
	Observe(context.Context) (DiskJobHeadroom, error)
	Extend(context.Context, []DiskJobPath) (DiskJobReceipt, error)
	SamplePath(context.Context, string, int64, int64) (DiskJobReceipt, error)
	Complete(context.Context) (DiskJobReceipt, error)
	Release(context.Context) error
	Reservation() DiskJobReceipt
}

// DiskDockerPathSampler is an optional extension for leases that need to
// sample paths resolved through Docker's authoritative daemon metadata. It is
// separate from DiskJobLease so existing scenario adapters do not need to
// implement Docker-specific sampling when they use only accessible paths.
type DiskDockerPathSampler interface {
	SampleDockerPath(context.Context, string, string, string, bool, bool, int64, int64) (DiskJobReceipt, error)
}

// DiskJobAdmitter is the owner boundary used by install and render producers.
type DiskJobAdmitter interface {
	ReserveDiskJob(context.Context, DiskJobRequest) (DiskJobLease, error)
}

// ReserveDiskJob atomically admits one complete job across its write paths.
func (b *Broker) ReserveDiskJob(ctx context.Context, request DiskJobRequest) (DiskJobLease, error) {
	if b == nil || b.store == nil {
		return nil, errors.New("capacity broker is unavailable")
	}
	in := internalcapacity.DiskJobRequest{OwnerID: request.OwnerID, JobID: request.JobID, ProcessID: request.ProcessID, ProcessStart: request.ProcessStart, FloorBytes: request.FloorBytes}
	for _, p := range request.Paths {
		in.Paths = append(in.Paths, internalcapacity.DiskJobPath{Path: p.Path, Role: p.Role, PeakBytes: p.PeakBytes, DockerContainer: p.DockerContainer, DockerDestination: p.DockerDestination, DockerUpperDir: p.DockerUpperDir, DockerRootDir: p.DockerRootDir})
	}
	r, capability, err := b.store.ReserveDiskJob(ctx, in)
	if err != nil {
		return nil, err
	}
	return &diskJobLease{store: b.store, capability: capability, processID: r.ProcessID, processStart: r.ProcessStart, reservation: fromInternalDiskJobReceipt(r)}, nil
}

// ListDiskJobs returns durable receipts from the shared capacity owner. A
// non-empty ownerID scopes the result to one scenario owner.
func (b *Broker) ListDiskJobs(ctx context.Context, ownerID string, activeOnly bool) ([]DiskJobReceipt, error) {
	if b == nil || b.store == nil {
		return nil, errors.New("capacity broker is unavailable")
	}
	ownerID = strings.TrimSpace(ownerID)
	if ownerID == "" {
		return nil, errors.New("capacity disk job list requires owner id")
	}
	rows, err := b.store.ListDiskJobsForOwner(ctx, ownerID, activeOnly)
	if err != nil {
		return nil, err
	}
	out := make([]DiskJobReceipt, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromInternalDiskJobReceipt(row))
	}
	return out, nil
}

// GetDiskJob returns one owner's persisted receipt without exposing the
// private lease capability.
func (b *Broker) GetDiskJob(ctx context.Context, ownerID, id string) (DiskJobReceipt, error) {
	if b == nil || b.store == nil {
		return DiskJobReceipt{}, errors.New("capacity broker is unavailable")
	}
	ownerID, id = strings.TrimSpace(ownerID), strings.TrimSpace(id)
	if ownerID == "" || id == "" {
		return DiskJobReceipt{}, errors.New("capacity disk job read requires owner and lease id")
	}
	r, err := b.store.GetDiskJob(ctx, id)
	if err != nil {
		return DiskJobReceipt{}, err
	}
	if r.OwnerID != ownerID {
		return DiskJobReceipt{}, fmt.Errorf("capacity disk job %s is not visible to owner %q", id, ownerID)
	}
	return fromInternalDiskJobReceipt(r), nil
}

// ReconcileDiskJob releases an abandoned lease only after the scenario owner
// confirms process stop and cleanup; the capacity store independently checks
// that the recorded process identity is no longer running.
func (b *Broker) ReconcileDiskJob(ctx context.Context, id string, evidence DiskJobRecoveryEvidence) (DiskJobReceipt, error) {
	if b == nil || b.store == nil {
		return DiskJobReceipt{}, errors.New("capacity broker is unavailable")
	}
	evidence.OwnerID = strings.TrimSpace(evidence.OwnerID)
	if evidence.OwnerID == "" || !evidence.ProcessStoppedVerified || !evidence.CleanupVerified {
		return DiskJobReceipt{}, errors.New("disk job recovery requires owner, verified process stop, and verified cleanup")
	}
	if _, err := b.GetDiskJob(ctx, evidence.OwnerID, id); err != nil {
		return DiskJobReceipt{}, err
	}
	r, err := b.store.ReconcileDiskJob(ctx, id, true, true)
	if err != nil {
		return DiskJobReceipt{}, err
	}
	return fromInternalDiskJobReceipt(r), nil
}

// AttachDiskJobLease opens a delegated handle from a private capability file.
// The store verifies the secret and the live producer PID/start token for every
// mutation, so a short-lived child command can heartbeat/extend the same owner
// job without becoming its owner or completing it.
func (b *Broker) AttachDiskJobLease(ctx context.Context, c DiskJobCredential) (DiskJobLease, error) {
	if b == nil || b.store == nil {
		return nil, errors.New("capacity broker is unavailable")
	}
	if c.ID == "" || c.Capability == "" || c.ProcessID <= 0 || c.ProcessStart == "" {
		return nil, errors.New("disk job credential is incomplete")
	}
	if _, err := b.store.ObserveDiskJob(ctx, c.ID, c.Capability, c.ProcessID, c.ProcessStart); err != nil {
		return nil, err
	}
	r, err := b.store.GetDiskJob(ctx, c.ID)
	if err != nil {
		return nil, err
	}
	return &diskJobLease{store: b.store, capability: c.Capability, processID: c.ProcessID, processStart: c.ProcessStart, reservation: fromInternalDiskJobReceipt(r)}, nil
}

type diskJobLease struct {
	mu           sync.Mutex
	store        *internalcapacity.SQLiteStore
	capability   string
	processID    int
	processStart string
	reservation  DiskJobReceipt
}

func (l *diskJobLease) Reservation() DiskJobReceipt {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.reservation
}
func (l *diskJobLease) Heartbeat(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l == nil || l.store == nil {
		return fmt.Errorf("disk job lease is unavailable")
	}
	if err := l.store.HeartbeatDiskJob(ctx, l.reservation.ID, l.capability, l.processID, l.processStart); err != nil {
		return err
	}
	r, err := l.store.GetDiskJob(ctx, l.reservation.ID)
	if err == nil {
		l.reservation = fromInternalDiskJobReceipt(r)
	}
	return err
}
func (l *diskJobLease) Observe(ctx context.Context) (DiskJobHeadroom, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l == nil || l.store == nil {
		return DiskJobHeadroom{}, fmt.Errorf("disk job lease is unavailable")
	}
	r, err := l.store.ObserveDiskJob(ctx, l.reservation.ID, l.capability, l.processID, l.processStart)
	if err != nil {
		return DiskJobHeadroom{}, err
	}
	return fromInternalDiskJobHeadroom(r), nil
}
func (l *diskJobLease) Extend(ctx context.Context, paths []DiskJobPath) (DiskJobReceipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l == nil || l.store == nil {
		return DiskJobReceipt{}, fmt.Errorf("disk job lease is unavailable")
	}
	add := make([]internalcapacity.DiskJobPath, 0, len(paths))
	for _, p := range paths {
		add = append(add, internalcapacity.DiskJobPath{Path: p.Path, Role: p.Role, PeakBytes: p.PeakBytes, DockerContainer: p.DockerContainer, DockerDestination: p.DockerDestination, DockerUpperDir: p.DockerUpperDir, DockerRootDir: p.DockerRootDir})
	}
	r, err := l.store.ExtendDiskJob(ctx, l.reservation.ID, l.capability, l.processID, l.processStart, add)
	if err == nil {
		l.reservation = fromInternalDiskJobReceipt(r)
	}
	return l.reservation, err
}
func (l *diskJobLease) SamplePath(ctx context.Context, path string, bytes, largerPeakBytes int64) (DiskJobReceipt, error) {
	return l.SampleDockerPath(ctx, path, "", "", false, false, bytes, largerPeakBytes)
}

func (l *diskJobLease) SampleDockerPath(ctx context.Context, path, dockerContainer, dockerDestination string, dockerUpperDir, dockerRootDir bool, bytes, largerPeakBytes int64) (DiskJobReceipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l == nil || l.store == nil {
		return DiskJobReceipt{}, fmt.Errorf("disk job lease is unavailable")
	}
	r, err := l.store.SampleDiskJobPath(ctx, l.reservation.ID, l.capability, l.processID, l.processStart, path, dockerContainer, dockerDestination, dockerUpperDir, dockerRootDir, bytes, largerPeakBytes)
	if err == nil {
		l.reservation = fromInternalDiskJobReceipt(r)
	}
	return l.reservation, err
}
func (l *diskJobLease) Complete(ctx context.Context) (DiskJobReceipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l == nil || l.store == nil {
		return DiskJobReceipt{}, fmt.Errorf("disk job lease is unavailable")
	}
	r, err := l.store.CompleteDiskJob(ctx, l.reservation.ID, l.capability, l.processID, l.processStart)
	if err == nil {
		l.reservation = fromInternalDiskJobReceipt(r)
	}
	return l.reservation, err
}

func (l *diskJobLease) Release(ctx context.Context) error { _, err := l.Complete(ctx); return err }

func fromInternalDiskJobReceipt(r internalcapacity.DiskJobReceipt) DiskJobReceipt {
	out := DiskJobReceipt{ID: r.ID, OwnerID: r.OwnerID, JobID: r.JobID, ProcessID: r.ProcessID, ProcessStart: r.ProcessStart, Status: r.Status, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, LastHeartbeatAt: r.LastHeartbeatAt, HeartbeatDeadline: r.HeartbeatDeadline, CleanupVerified: r.CleanupVerified}
	for _, p := range r.Paths {
		path := DiskJobPathReceipt{Path: p.Path, Role: p.Role, FilesystemID: p.FilesystemID, PeakBytes: p.PeakBytes}
		for _, sample := range p.Samples {
			path.Samples = append(path.Samples, DiskJobPathSample{Path: sample.Path, FilesystemID: sample.FilesystemID, Bytes: sample.Bytes, BaselineBytes: sample.BaselineBytes, DeltaBytes: sample.DeltaBytes, ReservedPeakBytes: sample.ReservedPeakBytes, ObservedAt: sample.ObservedAt, ExceedsPeak: sample.ExceedsPeak})
		}
		out.Paths = append(out.Paths, path)
	}
	for _, f := range r.Filesystems {
		out.Filesystems = append(out.Filesystems, DiskJobFilesystem{FilesystemID: f.FilesystemID, PeakBytes: f.PeakBytes, FloorBytes: f.FloorBytes, AvailableBytes: f.AvailableBytes, OtherReservedBytes: f.OtherReservedBytes, ActualDeltaBytes: f.ActualDeltaBytes, ActualDeltaBasis: f.ActualDeltaBasis})
	}
	return out
}

func fromInternalDiskJobHeadroom(r internalcapacity.DiskJobHeadroom) DiskJobHeadroom {
	out := DiskJobHeadroom{LeaseID: r.LeaseID, Status: r.Status}
	for _, f := range r.Filesystems {
		out.Filesystems = append(out.Filesystems, DiskJobHeadroomFilesystem{FilesystemID: f.FilesystemID, CurrentFreeBytes: f.CurrentFreeBytes, OtherReservedBytes: f.OtherReservedBytes, CurrentJobPeakBytes: f.CurrentJobPeakBytes, FloorBytes: f.FloorBytes, SafeAdditionalPeakBytes: f.SafeAdditionalPeakBytes})
	}
	return out
}
