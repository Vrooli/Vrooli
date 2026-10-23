package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"image-tools/internal/backends"
	"image-tools/internal/capabilities"

	vroolicli "github.com/vrooli/vrooli-cli-go"
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

// capacityClaimTTL bounds an op claim's liveness without a heartbeat. A
// generation is bounded (seconds–minutes) and run() releases the claim in a
// defer; the TTL is the crash-safety net so an abandoned claim is swept by the
// broker rather than leaking a VRAM reservation forever.
const capacityClaimTTL = 10 * time.Minute

// capacityGPU binds accounting and execution to the same freshly observed
// physical device. Backends without device selection retain their qualified
// single-GPU path with UUID accounting; unidentified hardware requires explicit
// qualification rather than silently claiming a different device.
func capacityGPU(host capabilities.Host, bytes int64, provider backends.Provider) (capabilities.GPU, error) {
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
		return capabilities.GPU{}, fmt.Errorf("ai: no GPU with known capacity and a supported physical-device binding; use a CPU-capable backend or qualify device selection")
	}
	return best, nil
}

// CapacityLease retains the shared owner's exact lease along with diagnostics.
// Only an explicit valid denial permits a CPU fallback; protocol/cleanup failures
// are errors and must never start unadmitted GPU work.
type CapacityLease struct {
	ClaimID      string
	Generation   int64
	DegradeToCPU bool
	Warnings     []string
	ReclaimBytes int64
	Enforce      string
	native       *vroolicli.GPUCapacityLease
}

type CapacityClaimOptions struct {
	Priority     string
	AllowReclaim bool
	GPUUUID      string
}

// CapacityBroker is the engine seam; the shared typed client owns CLI policy,
// fresh UUID/index validation, claim renewal and cancellation-safe cleanup.
type CapacityBroker interface {
	Claim(context.Context, string, int64, CapacityClaimOptions) (CapacityLease, error)
	Release(context.Context, CapacityLease) error
	Heartbeat(context.Context, CapacityLease) (CapacityLease, error)
}

type CLICapacityBroker struct{ Client *vroolicli.Client }

func (b *CLICapacityBroker) Claim(ctx context.Context, ownerID string, preferredBytes int64, opts CapacityClaimOptions) (CapacityLease, error) {
	client := b.Client
	if client == nil {
		client = vroolicli.New()
	}
	native, err := client.AcquireGPUCapacity(ctx, vroolicli.GPUCapacityRequest{
		OwnerID: ownerID, GPUUUID: opts.GPUUUID, VRAMBytes: preferredBytes,
		Priority: normalizeCapacityPriority(opts.Priority), AllowReclaim: opts.AllowReclaim, TTL: capacityClaimTTL,
	})
	if err != nil {
		var denied *vroolicli.GPUCapacityDenied
		if errors.As(err, &denied) {
			return CapacityLease{DegradeToCPU: true, Warnings: append(append([]string{}, denied.Warnings...), denied.Reason), ReclaimBytes: denied.ReclaimBytes, Enforce: denied.Enforce}, nil
		}
		return CapacityLease{}, err
	}
	return CapacityLease{ClaimID: native.ClaimID(), Generation: native.Generation(), Warnings: native.Warnings(), ReclaimBytes: native.ReclaimBytes(), Enforce: native.Enforce(), native: native}, nil
}

func (b *CLICapacityBroker) Heartbeat(ctx context.Context, lease CapacityLease) (CapacityLease, error) {
	if lease.native == nil {
		return lease, fmt.Errorf("capacity heartbeat: no owned GPU lease")
	}
	if err := lease.native.Heartbeat(ctx); err != nil {
		return lease, err
	}
	return lease, nil
}

func (b *CLICapacityBroker) Release(ctx context.Context, lease CapacityLease) error {
	if lease.native == nil {
		if lease.ClaimID == "" {
			return nil
		}
		return fmt.Errorf("capacity release %s: no owned GPU lease", lease.ClaimID)
	}
	return lease.native.Release(ctx)
}

// keepCapacityAlive shares the job cancellation path with renewal failure. Stop
// and join renewal before releasing the claim so no late heartbeat can race it.
func keepCapacityAlive(ctx context.Context, broker CapacityBroker, lease CapacityLease, interval time.Duration) (context.Context, func()) {
	return keepLeaseAlive(ctx, "GPU", func(beatCtx context.Context) error {
		updated, err := broker.Heartbeat(beatCtx, lease)
		if err == nil {
			lease = updated
		}
		return err
	}, interval)
}

func keepLeaseAlive(ctx context.Context, kind string, heartbeat func(context.Context) error, interval time.Duration) (context.Context, func()) {
	ctx, cancel := context.WithCancelCause(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				beatCtx, stop := context.WithTimeout(ctx, 5*time.Second)
				err := heartbeat(beatCtx)
				stop()
				if err != nil {
					cancel(fmt.Errorf("%s capacity renewal lost: %w", kind, err))
					return
				}
			}
		}
	}()
	return ctx, func() { cancel(context.Canceled); <-done }
}

// MemoryLease stays live until inference and any resident worker have stopped.
// Production uses the shared broker; tests inject admission without host writes.
type MemoryLease interface {
	Heartbeat(context.Context) error
	Release(context.Context) error
}

type MemoryAdmission func(context.Context, string, int64) (MemoryLease, error)

func keepMemoryAlive(ctx context.Context, lease MemoryLease) (context.Context, func()) {
	return keepLeaseAlive(ctx, "RAM", lease.Heartbeat, sharedcapacity.RecommendedLeaseHeartbeatInterval())
}

type localMemoryLease struct {
	sharedcapacity.RenewableLease
	broker *sharedcapacity.Broker
}

func (l *localMemoryLease) Release(ctx context.Context) error {
	return errors.Join(l.RenewableLease.Release(ctx), l.broker.Close())
}

// AcquireLocalRAM admits the declared host memory independently of GPU tier.
// It never reclaims another owner's resident models or invokes a paid fallback.
func AcquireLocalRAM(ctx context.Context, owner string, bytes int64) (MemoryLease, error) {
	broker, err := sharedcapacity.NewBroker(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("open RAM capacity broker: %w", err)
	}
	lease, verdict, err := broker.AcquireNoReclaim(ctx, owner, bytes, 0)
	if err != nil || verdict.Kind != "grant" || lease == nil {
		var releaseErr error
		if lease != nil {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			releaseErr = lease.Release(cleanup)
			cancel()
		}
		return nil, errors.Join(fmt.Errorf("RAM capacity admission %s: %s", verdict.Kind, verdict.Reason), err, releaseErr, broker.Close())
	}
	renewable, ok := lease.(sharedcapacity.RenewableLease)
	if !ok {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		return nil, errors.Join(errors.New("RAM lease cannot be renewed"), lease.Release(cleanup), broker.Close())
	}
	return &localMemoryLease{RenewableLease: renewable, broker: broker}, nil
}

type operationMemoryContextKey struct{}

func normalizeCapacityPriority(priority string) string {
	switch priority {
	case "batch", "service", "interactive":
		return priority
	default:
		return "service"
	}
}
