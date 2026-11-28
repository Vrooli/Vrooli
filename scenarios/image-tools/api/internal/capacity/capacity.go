// Package capacity owns image-tools' share of the host resource ledger: GPU,
// RAM, CPU and disk claims are acquired, renewed and released here, independent
// of what is being executed. The inference engine, the model installer and the
// shared CPU job scheduler all admit through this one boundary, so a claim made
// by one cannot be invisible to the others.
package capacity

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/capabilities"
	"image-tools/internal/jobs"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

// ClaimTTL bounds an op claim's liveness without a heartbeat. A
// generation is bounded (seconds–minutes) and run() releases the claim in a
// defer; the TTL is the crash-safety net so an abandoned claim is swept by the
// broker rather than leaking a VRAM reservation forever.
const ClaimTTL = 10 * time.Minute

// GPUFor binds accounting and execution to the same freshly observed
// physical device. Backends without device selection retain their qualified
// single-GPU path with UUID accounting; unidentified hardware requires explicit
// qualification rather than silently claiming a different device.
func GPUFor(host capabilities.Host, bytes int64, provider backends.Provider) (capabilities.GPU, error) {
	binder, canBind := provider.(interface{ SupportsGPUUUID() bool })
	canBind = canBind && binder.SupportsGPUUUID()
	var best capabilities.GPU
	var bestFree uint64
	found := false
	for _, g := range host.GPUs {
		free, known := g.VRAMFreeBytes()
		if !known || g.UUID == "" || g.VRAMBytes < uint64(bytes) {
			continue
		}
		if !canBind && (len(host.GPUs) != 1 || g.Index != 0) {
			continue
		}
		if !found || free > bestFree || (free == bestFree && g.Index < best.Index) {
			best, bestFree, found = g, free, true
		}
	}
	if !found {
		return capabilities.GPU{}, fmt.Errorf("capacity: no GPU with known capacity and a supported physical-device binding; use a CPU-capable backend or qualify device selection")
	}
	return best, nil
}

// Lease retains the shared owner's exact lease along with diagnostics.
// Only an explicit valid denial permits a CPU fallback; protocol/cleanup failures
// are errors and must never start unadmitted GPU work.
type Lease struct {
	ClaimID      string
	Generation   int64
	DegradeToCPU bool
	Warnings     []string
	ReclaimBytes int64
	Enforce      string
	native       *vroolicli.GPUCapacityLease
}

type ClaimOptions struct {
	Priority     string
	AllowReclaim bool
	GPUUUID      string
}

// Broker is the engine seam; the shared typed client owns CLI policy,
// fresh UUID/index validation, claim renewal and cancellation-safe cleanup.
type Broker interface {
	Claim(context.Context, string, int64, ClaimOptions) (Lease, error)
	Release(context.Context, Lease) error
	Heartbeat(context.Context, Lease) (Lease, error)
}

// memoryLease lets a warm worker retain a GPU claim using the same
// cancellation-safe lease-group lifecycle as its RAM and model-file claims.
type memoryLease struct {
	mu     sync.Mutex
	broker Broker
	lease  Lease
}

func (l *memoryLease) Heartbeat(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	updated, err := l.broker.Heartbeat(ctx, l.lease)
	if err == nil {
		l.lease = updated
	}
	return err
}

func (l *memoryLease) Release(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.broker.Release(ctx, l.lease)
}

// ActivityBroker is the optional Broker capability for reporting a claim's
// idleness. The broker never infers it: a claim nobody reports on reads idle
// while its job runs, which makes it reclaimable by higher-priority work.
type ActivityBroker interface {
	Activity(ctx context.Context, lease Lease, active bool) error
}

// ActivityLease is the renewal-surface counterpart: a lease whose owner can say
// whether it is serving work.
type ActivityLease = sharedcapacity.ActivityReporter

// ReportActivity reports a GPU claim active or idle through broker, which must
// support it. A broker that does not (a test double, an advisory broker) is not
// an error: there is nothing to report to.
func ReportActivity(ctx context.Context, broker Broker, lease Lease, active bool) error {
	reporter, ok := broker.(ActivityBroker)
	if !ok || lease.ClaimID == "" {
		return nil
	}
	return reporter.Activity(ctx, lease, active)
}

func (l *memoryLease) ReportActivity(ctx context.Context, active bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return ReportActivity(ctx, l.broker, l.lease, active)
}

func (l *localMemoryLease) ReportActivity(ctx context.Context, active bool) error {
	if reporter, ok := l.RenewableLease.(ActivityLease); ok {
		return reporter.ReportActivity(ctx, active)
	}
	return nil
}

type CLIBroker struct{ Client *vroolicli.Client }

// Activity reports the exact claim active or idle through the shared client,
// which adopts the generation the broker advances on every report.
func (b *CLIBroker) Activity(ctx context.Context, lease Lease, active bool) error {
	if lease.native == nil {
		return fmt.Errorf("capacity activity %s: no owned GPU lease", lease.ClaimID)
	}
	return lease.native.ReportActivity(ctx, active)
}

func (b *CLIBroker) Claim(ctx context.Context, ownerID string, preferredBytes int64, opts ClaimOptions) (Lease, error) {
	client := b.Client
	if client == nil {
		client = vroolicli.New()
	}
	native, err := client.AcquireGPUCapacity(ctx, vroolicli.GPUCapacityRequest{
		OwnerID: ownerID, GPUUUID: opts.GPUUUID, VRAMBytes: preferredBytes,
		Priority: normalizeCapacityPriority(opts.Priority), AllowReclaim: opts.AllowReclaim, TTL: ClaimTTL,
	})
	if err != nil {
		var denied *vroolicli.GPUCapacityDenied
		if errors.As(err, &denied) {
			return Lease{DegradeToCPU: true, Warnings: append(append([]string{}, denied.Warnings...), denied.Reason), ReclaimBytes: denied.ReclaimBytes, Enforce: denied.Enforce}, nil
		}
		return Lease{}, err
	}
	return Lease{ClaimID: native.ClaimID(), Generation: native.Generation(), Warnings: native.Warnings(), ReclaimBytes: native.ReclaimBytes(), Enforce: native.Enforce(), native: native}, nil
}

func (b *CLIBroker) Heartbeat(ctx context.Context, lease Lease) (Lease, error) {
	if lease.native == nil {
		return lease, fmt.Errorf("capacity heartbeat: no owned GPU lease")
	}
	if err := lease.native.Heartbeat(ctx); err != nil {
		return lease, err
	}
	return lease, nil
}

func (b *CLIBroker) Release(ctx context.Context, lease Lease) error {
	if lease.native == nil {
		if lease.ClaimID == "" {
			return nil
		}
		return fmt.Errorf("capacity release %s: no owned GPU lease", lease.ClaimID)
	}
	return lease.native.Release(ctx)
}

// KeepAlive shares the job cancellation path with renewal failure. Stop
// and join renewal before releasing the claim so no late heartbeat can race it.
func KeepAlive(ctx context.Context, broker Broker, lease Lease, interval time.Duration) (context.Context, func()) {
	return keepLeaseAlive(ctx, "GPU", func(beatCtx context.Context) error {
		updated, err := broker.Heartbeat(beatCtx, lease)
		if err == nil {
			lease = updated
		}
		return err
	}, interval)
}

type heartbeatLease struct{ beat func(context.Context) error }

func (l heartbeatLease) Heartbeat(ctx context.Context) error { return l.beat(ctx) }
func (heartbeatLease) Release(context.Context) error         { return nil }

func keepLeaseAlive(ctx context.Context, kind string, heartbeat func(context.Context) error, interval time.Duration) (context.Context, func()) {
	scope := sharedcapacity.MaintainWithInterval(ctx, heartbeatLease{beat: func(ctx context.Context) error {
		if err := heartbeat(ctx); err != nil {
			return fmt.Errorf("%s capacity renewal: %w", kind, err)
		}
		return nil
	}}, interval)
	return scope.Context(), func() { _ = scope.Release(context.Background()) }
}

// MemoryLease stays live until inference and any resident worker have stopped.
// Production uses the shared broker; tests inject admission without host writes.
type MemoryLease interface {
	Heartbeat(context.Context) error
	Release(context.Context) error
}

type MemoryAdmission func(context.Context, string, int64) (MemoryLease, error)

// DiskAdmission reserves the full estimated peak for one destination path.
// Owners heartbeat throughout work and release every returned lease on all exits.
type DiskAdmission interface {
	ReserveDiskJob(context.Context, sharedcapacity.DiskJobRequest) (sharedcapacity.DiskJobLease, error)
}

func (b *CLIBroker) ReserveDiskJob(ctx context.Context, request sharedcapacity.DiskJobRequest) (sharedcapacity.DiskJobLease, error) {
	broker, err := sharedcapacity.NewBroker(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("open shared disk capacity broker: %w", err)
	}
	lease, err := broker.ReserveDiskJob(ctx, request)
	if err != nil {
		return nil, errors.Join(err, broker.Close())
	}
	return &diskJobLeaseWithBroker{DiskJobLease: lease, broker: broker}, nil
}

// SupersedeDiskJobs closes every active lease of the owner whose job key
// ownsKey accepts and whose producer has exited. Each is reattached to this
// process (the ledger refuses while the recorded producer still runs) and then
// completed, so the reservation moves to the new attempt instead of being
// counted twice until the heartbeat lapses. It returns how many it closed.
func (b *CLIBroker) SupersedeDiskJobs(ctx context.Context, ownerID string, ownsKey func(string) bool) (int, error) {
	if ownsKey == nil {
		return 0, errors.New("supersede disk jobs requires a job key matcher")
	}
	broker, err := sharedcapacity.NewBroker(ctx, "")
	if err != nil {
		return 0, fmt.Errorf("open shared disk capacity broker: %w", err)
	}
	closed, supersedeErr := supersedeDiskJobs(ctx, broker, ownerID, ownsKey)
	return closed, errors.Join(supersedeErr, broker.Close())
}

// diskJobLedger is the part of the shared broker that superseding needs.
type diskJobLedger interface {
	ListDiskJobs(ctx context.Context, ownerID string, activeOnly bool) ([]sharedcapacity.DiskJobReceipt, error)
	ReattachDiskJob(ctx context.Context, ownerID, jobID, leaseID string) (sharedcapacity.DiskJobLease, error)
}

func supersedeDiskJobs(ctx context.Context, ledger diskJobLedger, ownerID string, ownsKey func(string) bool) (int, error) {
	active, err := ledger.ListDiskJobs(ctx, ownerID, true)
	if err != nil {
		return 0, fmt.Errorf("list active disk jobs: %w", err)
	}
	var errs []error
	closed := 0
	for _, receipt := range active {
		if !ownsKey(receipt.JobID) {
			continue
		}
		lease, err := ledger.ReattachDiskJob(ctx, ownerID, receipt.JobID, receipt.ID)
		if err != nil {
			// A producer that is still running owns its lease; leave it.
			errs = append(errs, fmt.Errorf("reattach disk job %s: %w", receipt.ID, err))
			continue
		}
		if _, err := lease.Complete(ctx); err != nil {
			errs = append(errs, fmt.Errorf("complete superseded disk job %s: %w", receipt.ID, err))
			continue
		}
		closed++
	}
	return closed, errors.Join(errs...)
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

func (l *diskJobLeaseWithBroker) Release(ctx context.Context) error {
	if _, err := l.Complete(ctx); err != nil {
		return err
	}
	return nil
}

type localMemoryLease struct {
	sharedcapacity.RenewableLease
	broker *sharedcapacity.Broker
}

func (l *localMemoryLease) Release(ctx context.Context) error {
	if err := l.RenewableLease.Release(ctx); err != nil {
		return err
	}
	return l.broker.Close()
}

// AcquireLocalRAM admits the declared host memory independently of GPU tier.
// It never reclaims another owner's resident models or invokes a paid fallback.
func AcquireLocalRAM(ctx context.Context, owner string, bytes int64) (MemoryLease, error) {
	return acquireLocalResources(ctx, owner, bytes, 0)
}

// AcquireLocalCPU reserves millicores using the same cross-process broker.
func AcquireLocalCPU(ctx context.Context, owner string, milli int64) (MemoryLease, error) {
	return acquireLocalResources(ctx, owner, 0, milli)
}

func acquireLocalResources(ctx context.Context, owner string, bytes, cpuMilli int64) (MemoryLease, error) {
	broker, err := sharedcapacity.NewBroker(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("open RAM capacity broker: %w", err)
	}
	lease, verdict, err := broker.AcquireResources(ctx, sharedcapacity.Request{OwnerID: owner, RAMBytes: bytes, CPUMilli: cpuMilli, Priority: capacityPriority(ctx), NoReclaim: true})
	if err != nil || verdict.Kind != "grant" || lease == nil {
		var releaseErr error
		if lease != nil {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			releaseErr = lease.Release(cleanup)
			cancel()
		}
		closeErr := broker.Close()
		if err == nil && releaseErr == nil && closeErr == nil && verdict.Kind != "grant" {
			return nil, fmt.Errorf("%w: %s capacity admission: %s", jobs.ErrCapacityBusy, verdict.Kind, verdict.Reason)
		}
		return nil, errors.Join(fmt.Errorf("local capacity admission %s: %s", verdict.Kind, verdict.Reason), err, releaseErr, closeErr)
	}
	renewable, ok := lease.(sharedcapacity.RenewableLease)
	if !ok {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		return nil, errors.Join(errors.New("RAM lease cannot be renewed"), lease.Release(cleanup), broker.Close())
	}
	return &localMemoryLease{RenewableLease: renewable, broker: broker}, nil
}

// NewMemoryLease adapts a broker lease into the renewal surface the engine's
// lease group holds. A warm worker uses it to retain a GPU claim across idle
// periods, so the claim's lifetime follows model residency rather than the job.
func NewMemoryLease(broker Broker, lease Lease) MemoryLease {
	return &memoryLease{broker: broker, lease: lease}
}

type priorityKey struct{}

// WithPriority carries the job's admission priority to the broker. It is
// exported because the inference engine and the CPU scheduler both set it
// before admitting.
func WithPriority(ctx context.Context, priority string) context.Context {
	return context.WithValue(ctx, priorityKey{}, priority)
}

func capacityPriority(ctx context.Context) string {
	priority, _ := ctx.Value(priorityKey{}).(string)
	return normalizeCapacityPriority(priority)
}

// AdmitCPUWork is the shared scheduler's floor for local support work. Each
// operation's RAM and CPU come from its profile in support_profiles.json (the
// default is one core and 256 MiB). Model operations override it with declared
// model RAM and CPU profiles in Admit.
func AdmitCPUWork(ctx context.Context, job jobs.Job) (jobs.Admission, error) {
	if job.Lane == jobs.LaneNetwork {
		return jobs.Admission{Context: ctx}, nil
	}
	ctx = WithPriority(ctx, string(job.Priority))
	profile := SupportProfileFor(job.Operation)
	lease, err := acquireLocalResources(ctx, "image-tools:"+job.ID+":support", profile.RAMBytes, profile.CPUMilli)
	if err != nil {
		return jobs.Admission{}, err
	}
	scope := sharedcapacity.Maintain(ctx, lease)
	meta := map[string]string{
		"ram_capacity_claim_bytes": strconv.FormatInt(profile.RAMBytes, 10),
		"cpu_capacity_milli":       strconv.FormatInt(profile.CPUMilli, 10),
		"capacity_activity":        "active",
	}
	// The scheduler starts the work next, so the claim is serving it. A failed
	// report leaves the claim reading idle and never stops the work.
	if reporter, ok := lease.(ActivityLease); ok {
		if err := reporter.ReportActivity(context.WithoutCancel(ctx), true); err != nil {
			meta["capacity_activity"] = "unreported: " + err.Error()
		}
	}
	return jobs.Admission{
		Context: scope.Context(),
		// Scope.Release is idempotent and retry-safe after a ledger failure.
		ReleaseRetryable: true,
		Release:          func(cleanup context.Context) error { return scope.Release(cleanup) },
		Meta:             meta,
	}, nil
}

func normalizeCapacityPriority(priority string) string {
	switch priority {
	case "batch", "service", "interactive":
		return priority
	default:
		return "service"
	}
}
