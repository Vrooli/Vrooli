package localqemu

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/capacity"
)

const (
	capacityOwnerID  = "compute-manager:local-qemu"
	qemuDiskFloor    = int64(120) << 30
	qemuDiskMax      = int64(25) << 30
	qemuDiskOverhead = int64(64) << 20
)

// CapacityReceipt is persisted with the manager-owned guest metadata so a
// replacement Compute Manager process can reattach the exact host admission.
type CapacityReceipt struct {
	ResourceOwnerID string  `json:"resource_owner_id"`
	RAMBytes        int64   `json:"ram_bytes"`
	CPUMilli        int64   `json:"cpu_milli"`
	DiskOwnerID     string  `json:"disk_owner_id"`
	DiskJobID       string  `json:"disk_job_id"`
	DiskLeaseID     string  `json:"disk_lease_id"`
	DiskPath        string  `json:"disk_path"`
	DiskPeakBytes   int64   `json:"disk_peak_bytes"`
	DiskFloorBytes  int64   `json:"disk_floor_bytes"`
	Profile         Profile `json:"profile"`
}

type AdmissionRequest struct {
	InstanceID string
	Workdir    string
	Profile    Profile
	Memory     string
	CPUs       int
	DiskPeak   int64
}

type CapacityAdmission interface {
	Acquire(context.Context, AdmissionRequest) (*QEMULease, error)
	Reattach(context.Context, CapacityReceipt) (*QEMULease, error)
	FindOrphan(context.Context, string, string) (CapacityReceipt, bool, error)
	Reconcile(context.Context, CapacityReceipt) error
	ReleaseResources(context.Context, CapacityReceipt) error
}

type resourceCapacityBroker interface {
	AcquireNoReclaim(context.Context, string, int64, int64) (capacity.Lease, capacity.Verdict, error)
	ReattachResources(context.Context, string, int64, int64) (capacity.RenewableLease, bool, error)
	ReserveDiskJob(context.Context, capacity.DiskJobRequest) (capacity.DiskJobLease, error)
	ReattachDiskJob(context.Context, string, string, string) (capacity.DiskJobLease, error)
	GetDiskJob(context.Context, string, string) (capacity.DiskJobReceipt, error)
	ListDiskJobs(context.Context, string, bool) ([]capacity.DiskJobReceipt, error)
	ReconcileDiskJob(context.Context, string, capacity.DiskJobRecoveryEvidence) (capacity.DiskJobReceipt, error)
	ReleaseRun(context.Context, string, string) error
}

type BrokerAdmission struct {
	Broker resourceCapacityBroker
}

// NewBrokerAdmission binds local-QEMU jobs to the shared host capacity ledger.
func NewBrokerAdmission(broker resourceCapacityBroker) *BrokerAdmission {
	return &BrokerAdmission{Broker: broker}
}

type QEMULease struct {
	Resources capacity.RenewableLease
	Disk      capacity.DiskJobLease
	Receipt   CapacityReceipt
	admitter  *BrokerAdmission
}

func (l *QEMULease) Heartbeat(ctx context.Context) error {
	if l == nil || l.Resources == nil || l.Disk == nil {
		return errors.New("local-QEMU capacity lease is incomplete")
	}
	if err := l.Resources.Heartbeat(ctx); err != nil {
		return fmt.Errorf("renew QEMU RAM/CPU capacity: %w", err)
	}
	if err := l.Disk.Heartbeat(ctx); err != nil {
		return fmt.Errorf("renew QEMU disk capacity: %w", err)
	}
	return nil
}

// Release is called only after the QEMU process is confirmed stopped and its
// owned workdir has been cleaned. Disk reconciliation requires those facts.
func (l *QEMULease) Release(ctx context.Context) error {
	if l == nil {
		return errors.New("local-QEMU capacity lease is unavailable")
	}
	var errs []error
	if l.Disk != nil {
		if err := l.Disk.Release(ctx); err != nil {
			if l.admitter == nil || l.admitter.Broker == nil {
				errs = append(errs, err)
			} else if reconcileErr := l.admitter.Reconcile(ctx, l.Receipt); reconcileErr != nil {
				errs = append(errs, errors.Join(err, reconcileErr))
			}
		}
	}
	if l.Resources != nil {
		if err := l.Resources.Release(ctx); err != nil {
			errs = append(errs, fmt.Errorf("release QEMU RAM/CPU capacity: %w", err))
		}
	}
	if l.admitter != nil {
		if err := l.admitter.ReleaseResources(ctx, l.Receipt); err != nil {
			errs = append(errs, fmt.Errorf("reconcile QEMU RAM/CPU capacity: %w", err))
		}
	}
	return errors.Join(errs...)
}

func profileBudget(profile Profile, memory string, cpus int) (int64, int64, error) {
	var memoryBytes int64
	var expectedMemory string
	var expectedCPUs int
	switch profile {
	case ProfileHeadlessLinux:
		memoryBytes, expectedMemory, expectedCPUs = 2<<30, "2048", 2
	case ProfileDesktopLinux:
		memoryBytes, expectedMemory, expectedCPUs = 4<<30, "4096", 4
	default:
		return 0, 0, fmt.Errorf("local-QEMU capacity budget is unknown for profile %q", profile)
	}
	if memory != expectedMemory || cpus != expectedCPUs {
		return 0, 0, fmt.Errorf("profile %s requires memory=%s MiB and cpus=%d", profile, expectedMemory, expectedCPUs)
	}
	return memoryBytes, int64(cpus) * 1000, nil
}

func receiptProfileBudget(profile Profile) (int64, int64, error) {
	switch profile {
	case ProfileHeadlessLinux:
		return 2 << 30, 2000, nil
	case ProfileDesktopLinux:
		return 4 << 30, 4000, nil
	default:
		return 0, 0, fmt.Errorf("local-QEMU capacity budget is unknown for profile %q", profile)
	}
}

func (a *BrokerAdmission) Acquire(ctx context.Context, request AdmissionRequest) (*QEMULease, error) {
	if a == nil || a.Broker == nil {
		return nil, errors.New("shared capacity broker is unavailable")
	}
	ram, cpu, err := profileBudget(request.Profile, request.Memory, request.CPUs)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.InstanceID) == "" || !filepath.IsAbs(request.Workdir) || request.DiskPeak <= 0 || request.DiskPeak > qemuDiskMax {
		return nil, fmt.Errorf("local-QEMU admission requires an instance, absolute workdir, and measured disk peak <= %d bytes", qemuDiskMax)
	}
	resourceOwner := capacity.OwnerIDFor("compute-manager", request.InstanceID, "local-qemu")
	if resourceOwner == "" {
		return nil, errors.New("local-QEMU instance id cannot be used as a stable capacity owner")
	}
	resourceLease, found, err := a.Broker.ReattachResources(ctx, resourceOwner, ram, cpu)
	if err != nil {
		return nil, err
	}
	if !found {
		var verdict capacity.Verdict
		resourceLeaseRaw, verdict, err := a.Broker.AcquireNoReclaim(ctx, resourceOwner, ram, cpu)
		if err != nil {
			return nil, fmt.Errorf("admit QEMU RAM/CPU (%s): %w", verdict.Reason, err)
		}
		if verdict.Kind != "grant" {
			return nil, fmt.Errorf("admit QEMU RAM/CPU denied: %s", verdict.Reason)
		}
		var ok bool
		resourceLease, ok = resourceLeaseRaw.(capacity.RenewableLease)
		if !ok {
			_ = resourceLeaseRaw.Release(context.Background())
			return nil, errors.New("capacity broker returned a non-renewable QEMU resource lease")
		}
	}
	activity, ok := resourceLease.(capacity.ActivityReporter)
	if !ok {
		_ = resourceLease.Release(context.Background())
		return nil, errors.New("capacity broker returned a QEMU lease without activity reporting")
	}
	if err := activity.ReportActivity(ctx, true); err != nil {
		_ = resourceLease.Release(context.Background())
		return nil, fmt.Errorf("activate QEMU RAM/CPU lease: %w", err)
	}
	jobs, err := a.Broker.ListDiskJobs(ctx, capacityOwnerID, true)
	if err != nil {
		_ = resourceLease.Release(context.Background())
		return nil, fmt.Errorf("inspect prior QEMU disk job: %w", err)
	}
	for _, job := range jobs {
		if job.JobID == request.InstanceID {
			_ = resourceLease.Release(context.Background())
			return nil, fmt.Errorf("QEMU disk job %s already has an active admission; reconcile it before retrying", request.InstanceID)
		}
	}
	diskLease, err := a.Broker.ReserveDiskJob(ctx, capacity.DiskJobRequest{
		OwnerID: capacityOwnerID, JobID: request.InstanceID, FloorBytes: qemuDiskFloor,
		Paths: []capacity.DiskJobPath{{Path: request.Workdir, Role: "local-qemu-guest-workdir", PeakBytes: request.DiskPeak}},
	})
	if err != nil {
		_ = resourceLease.Release(context.Background())
		return nil, fmt.Errorf("admit QEMU disk writes: %w", err)
	}
	receipt := CapacityReceipt{
		ResourceOwnerID: resourceOwner, RAMBytes: ram, CPUMilli: cpu,
		DiskOwnerID: capacityOwnerID, DiskJobID: request.InstanceID,
		DiskLeaseID: diskLease.Reservation().ID, DiskPath: filepath.Clean(request.Workdir), DiskPeakBytes: request.DiskPeak,
		DiskFloorBytes: qemuDiskFloor, Profile: request.Profile,
	}
	receiptBytes, err := json.Marshal(receipt)
	if err != nil {
		_ = cleanupOwnedWorkdir(request.Workdir)
		_ = diskLease.Release(context.Background())
		_ = resourceLease.Release(context.Background())
		return nil, fmt.Errorf("encode local-QEMU capacity receipt: %w", err)
	}
	if err := storage.WriteFileAtomic(filepath.Join(request.Workdir, ".capacity-admission.json"), receiptBytes, 0o600); err != nil {
		cleanupErr := cleanupOwnedWorkdir(request.Workdir)
		releaseErr := diskLease.Release(context.Background())
		resourceErr := resourceLease.Release(context.Background())
		return nil, errors.Join(fmt.Errorf("persist local-QEMU capacity receipt: %w", err), cleanupErr, releaseErr, resourceErr)
	}
	return &QEMULease{Resources: resourceLease, Disk: diskLease, Receipt: receipt, admitter: a}, nil
}

func (a *BrokerAdmission) FindOrphan(ctx context.Context, instanceID, workdir string) (CapacityReceipt, bool, error) {
	if a == nil || a.Broker == nil || strings.TrimSpace(instanceID) == "" || !filepath.IsAbs(workdir) {
		return CapacityReceipt{}, false, errors.New("local-QEMU orphan lookup requires a broker, instance ID, and absolute workdir")
	}
	jobs, err := a.Broker.ListDiskJobs(ctx, capacityOwnerID, true)
	if err != nil {
		return CapacityReceipt{}, false, err
	}
	for _, job := range jobs {
		if job.JobID != instanceID {
			continue
		}
		if job.Status != "active" || len(job.Paths) != 1 || filepath.Clean(job.Paths[0].Path) != filepath.Clean(workdir) || job.Paths[0].PeakBytes <= 0 || job.Paths[0].PeakBytes > qemuDiskMax || len(job.Filesystems) != 1 || job.Filesystems[0].FloorBytes < qemuDiskFloor {
			return CapacityReceipt{}, true, errors.New("orphan local-QEMU disk lease does not match its manager-owned directory")
		}
		return CapacityReceipt{
			ResourceOwnerID: capacity.OwnerIDFor("compute-manager", instanceID, "local-qemu"),
			DiskOwnerID:     capacityOwnerID, DiskJobID: job.JobID, DiskLeaseID: job.ID,
			DiskPath: filepath.Clean(workdir), DiskPeakBytes: job.Paths[0].PeakBytes, DiskFloorBytes: qemuDiskFloor,
		}, true, nil
	}
	return CapacityReceipt{}, false, nil
}

func (a *BrokerAdmission) Reattach(ctx context.Context, receipt CapacityReceipt) (*QEMULease, error) {
	if a == nil || a.Broker == nil || receipt.ResourceOwnerID == "" || receipt.RAMBytes <= 0 || receipt.CPUMilli <= 0 || receipt.DiskOwnerID != capacityOwnerID || receipt.DiskJobID == "" || receipt.DiskLeaseID == "" || !filepath.IsAbs(receipt.DiskPath) || receipt.DiskPeakBytes <= 0 || receipt.DiskPeakBytes > qemuDiskMax || receipt.DiskFloorBytes != qemuDiskFloor {
		return nil, errors.New("persisted local-QEMU capacity receipt is incomplete or not owned")
	}
	wantRAM, wantCPU, err := receiptProfileBudget(receipt.Profile)
	if err != nil || receipt.RAMBytes != wantRAM || receipt.CPUMilli != wantCPU {
		return nil, errors.Join(errors.New("persisted local-QEMU resource budget does not match its profile"), err)
	}
	job, err := a.Broker.GetDiskJob(ctx, receipt.DiskOwnerID, receipt.DiskLeaseID)
	if err != nil {
		return nil, fmt.Errorf("read persisted QEMU disk job: %w", err)
	}
	if job.JobID != receipt.DiskJobID || job.Status != "active" || len(job.Paths) != 1 || filepath.Clean(job.Paths[0].Path) != filepath.Clean(receipt.DiskPath) || job.Paths[0].PeakBytes != receipt.DiskPeakBytes || len(job.Filesystems) != 1 || job.Filesystems[0].FloorBytes < qemuDiskFloor {
		return nil, errors.New("persisted QEMU disk job does not match its manager-owned receipt")
	}
	resources, found, err := a.Broker.ReattachResources(ctx, receipt.ResourceOwnerID, receipt.RAMBytes, receipt.CPUMilli)
	if err != nil {
		return nil, err
	}
	if !found {
		var verdict capacity.Verdict
		resourcesRaw, verdict, err := a.Broker.AcquireNoReclaim(ctx, receipt.ResourceOwnerID, receipt.RAMBytes, receipt.CPUMilli)
		if err != nil || verdict.Kind != "grant" {
			return nil, errors.Join(fmt.Errorf("re-admit running QEMU RAM/CPU: %s", verdict.Reason), err)
		}
		resources, found = resourcesRaw.(capacity.RenewableLease)
		if !found {
			_ = resourcesRaw.Release(context.Background())
			return nil, errors.New("capacity broker returned a non-renewable QEMU resource lease")
		}
	}
	disk, err := a.Broker.ReattachDiskJob(ctx, receipt.DiskOwnerID, receipt.DiskJobID, receipt.DiskLeaseID)
	lease := &QEMULease{Resources: resources, Receipt: receipt, admitter: a}
	if err != nil {
		return lease, fmt.Errorf("reattach QEMU disk capacity: %w", err)
	}
	lease.Disk = disk
	activity, ok := lease.Resources.(capacity.ActivityReporter)
	if !ok {
		return lease, errors.New("capacity broker returned a reattached QEMU lease without activity reporting")
	}
	if err := activity.ReportActivity(ctx, true); err != nil {
		return lease, fmt.Errorf("activate reattached QEMU RAM/CPU lease: %w", err)
	}
	return lease, nil
}

func (a *BrokerAdmission) Reconcile(ctx context.Context, receipt CapacityReceipt) error {
	if a == nil || a.Broker == nil || receipt.DiskLeaseID == "" || receipt.DiskOwnerID != capacityOwnerID {
		return errors.New("cannot reconcile incomplete QEMU capacity receipt")
	}
	job, err := a.Broker.GetDiskJob(ctx, capacityOwnerID, receipt.DiskLeaseID)
	if err != nil {
		return errors.Join(err, a.ReleaseResources(ctx, receipt))
	}
	if job.Status != "completed" || !job.CleanupVerified {
		_, err = a.Broker.ReconcileDiskJob(ctx, receipt.DiskLeaseID, capacity.DiskJobRecoveryEvidence{
			OwnerID: capacityOwnerID, ProcessStoppedVerified: true, CleanupVerified: true,
		})
	}
	return errors.Join(err, a.ReleaseResources(ctx, receipt))
}

func (a *BrokerAdmission) ReleaseResources(ctx context.Context, receipt CapacityReceipt) error {
	if a == nil || a.Broker == nil || receipt.DiskJobID == "" || receipt.ResourceOwnerID != capacity.OwnerIDFor("compute-manager", receipt.DiskJobID, "local-qemu") {
		return errors.New("cannot release capacity claims without the exact local-QEMU owner identity")
	}
	return a.Broker.ReleaseRun(ctx, "compute-manager", receipt.DiskJobID)
}
