// Package capacity exposes the narrow public admission seam used by modules
// that cannot import Vrooli's root internal/capacity package.
package capacity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/artifactlock"
	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
	"github.com/vrooli/vrooli/internal/hostpressure"
)

const capacityHeartbeatTTL = 15 * time.Minute
const capacityCleanupTimeout = 5 * time.Second

// RecommendedLeaseHeartbeatInterval returns a renewal cadence derived from
// the broker's operation-claim TTL. Long-running owner work should call
// Heartbeat more frequently than this interval and still treat renewal errors
// as loss of admission.
func RecommendedLeaseHeartbeatInterval() time.Duration {
	return capacityHeartbeatTTL / 3
}

// Verdict describes whether a short-lived operation may consume its requested
// host capacity. Queue and deny both fall back to serial execution upstream.
type Verdict struct {
	Kind   string
	Reason string
}

// HostObservation is the read-only host state used by callers that need to
// explain an admission decision. It is sourced through the same short-lived
// snapshot cache as Acquire, so observing it does not perform a second probe.
type HostObservation struct {
	AvailableRAMBytes uint64  `json:"availableRamBytes"`
	Load1             float64 `json:"load1"`
	SwapUsedPercent   float64 `json:"swapUsedPercent"`
}

// Lease releases all claims acquired for one operation.
type Lease interface {
	Release(context.Context) error
}

// RenewableLease is implemented by leases that can extend their ledger claims.
// Lease remains unchanged for existing consumers; long-running operations can
// opt in to renewal when they hold the concrete broker lease.
type RenewableLease interface {
	Lease
	Heartbeat(context.Context) error
}

// GPUIdentity identifies the exact accelerator admitted for an operation.
// Callers should bind child processes to UUID (or the corresponding vendor
// selector) from this value rather than rediscovering a device by index.
type GPUIdentity struct {
	Index int
	UUID  string
	Name  string
}

// GPULease is a renewable operation lease bound to one observed GPU identity.
type GPULease interface {
	RenewableLease
	Identity() GPUIdentity
}

// snapshotTTL bounds how long one host-capacity reading is reused across
// admissions.
//
// It exists because Acquire is called once per admitted operation and
// hostinventory.Collect is not cheap: it shells out to nvidia-smi, docker,
// loginctl, systemctl, grdctl, and a Secret Service probe. On a healthy host
// that is ~175 ms; on 2026-08-08 a single wedged gnome-keyring-daemon pushed it
// to 8.2 s, and Test Genie — which admits every phase of every suite — spent
// 46.6% of its total wall-clock inside this call. Caching does not fix a wedged
// host, but it stops one slow probe from being multiplied by the admission
// count.
//
// Two seconds is chosen against what the reading is for: free RAM and CPU
// headroom, used to decide whether one more short-lived operation fits. Those
// move on a timescale of seconds under real load, and every claim this broker
// grants is already recorded in the ledger — which is read fresh on every
// Acquire — so concurrent grants inside one TTL window still see each other.
// The snapshot only supplies the host's total and free capacity, not the
// outstanding claims against it.
const snapshotTTL = 2 * time.Second

// Broker is a serialized adapter over the shared capacity ledger. It does not
// implement a second policy: Decide, claim persistence, and resource units all
// remain owned by internal/capacity.
type Broker struct {
	mu     sync.Mutex
	store  *internalcapacity.SQLiteStore
	source internalcapacity.CapacitySource
	// memoryPressure reads fresh Linux PSI for production admission. A nil
	// reader is intentionally unknown for injected snapshot brokers.
	memoryPressure func() hostpressure.Reading
	// admissionLockKey identifies this ledger across processes. Empty is reserved
	// for stores without a persistent SQLite path (for example :memory:).
	admissionLockKey string

	// cachedSnapshot is the last host reading and when it was taken. It is
	// guarded by mu, which Acquire already holds for its whole body.
	cachedSnapshot hostinventory.Snapshot
	cachedAt       time.Time
	// now is a clock seam so the TTL is testable without sleeping.
	now func() time.Time

	// afterClaimCreated is a narrow fault-injection seam for admission cleanup
	// tests. Production brokers leave it nil.
	afterClaimCreated func(internalcapacity.CapacityClaim)
}

// OwnerIDFor formats an operation claim identity as scenario:run:phase. Colons
// delimit components; callers must provide components without colons.
func OwnerIDFor(scenario, runID, phase string) string {
	if strings.Contains(scenario, ":") || strings.Contains(runID, ":") || strings.Contains(phase, ":") {
		return ""
	}
	return scenario + ":" + runID + ":" + phase
}

// RunOwnerPrefix returns the prefix selecting all phase claims for one run.
// The returned format is scenario:run: and components must not contain colons.
func RunOwnerPrefix(scenario, runID string) string {
	if strings.Contains(scenario, ":") || strings.Contains(runID, ":") {
		return ""
	}
	return scenario + ":" + runID + ":"
}

// ReleaseRun releases every operation claim belonging to one run.
func (b *Broker) ReleaseRun(ctx context.Context, scenario, runID string) error {
	if b == nil || b.store == nil {
		return nil
	}
	prefix := RunOwnerPrefix(scenario, runID)
	if prefix == "" {
		return fmt.Errorf("invalid run owner identity")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	claims, err := b.store.ListClaims(ctx, internalcapacity.ClaimFilter{OwnerKind: internalcapacity.OwnerKindOp, OwnerIDPrefix: prefix, Statuses: internalcapacity.ActiveClaimStatuses()})
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if _, err := b.store.ReleaseClaim(ctx, claim.ClaimID); err != nil {
			return err
		}
	}
	return nil
}

// NewBroker opens the shared capacity ledger. An empty dbPath uses the normal
// Vrooli runtime-home state path outside tests; tests must pass an explicit path.
func NewBroker(ctx context.Context, dbPath string) (*Broker, error) {
	dbPath, err := internalcapacity.ResolveConfiguredDBPath(internalcapacity.Config{DBPath: dbPath})
	if err != nil {
		return nil, err
	}
	// Schema verification reapplies idempotent DDL even for an existing
	// ledger. Serialize construction with admissions before opening SQLite so
	// two first-use brokers cannot race in ensureSchema before Acquire's lock.
	lockKey := capacityAdmissionLockKey(dbPath)
	unlock := func() {}
	if lockKey != "" {
		var err error
		unlock, err = artifactlock.AcquireContext(ctx, lockKey)
		if err != nil {
			return nil, fmt.Errorf("capacity broker initialization lock unavailable: %w", err)
		}
	}
	defer unlock()
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: dbPath})
	if err != nil {
		return nil, err
	}
	broker := NewBrokerWithSource(store, internalcapacity.HostInventorySource{})
	broker.memoryPressure = internalcapacity.ReadHostMemoryPressure
	return broker, nil
}

func capacityAdmissionLockKey(dbPath string) string {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" || dbPath == ":memory:" {
		return ""
	}
	if absPath, err := filepath.Abs(filepath.Clean(dbPath)); err == nil {
		dbPath = absPath
	}
	// Resolve symlink aliases for existing ledgers, and canonicalize the parent
	// for a new ledger before its leaf exists, so initialization and admission
	// use the same identity through alternate runtime-home paths.
	if realPath, err := filepath.EvalSymlinks(dbPath); err == nil {
		dbPath = realPath
	} else if os.IsNotExist(err) {
		// Before the ledger file exists, canonicalize its parent so a symlinked
		// runtime-home directory still maps to the same initialization lock.
		if parent, parentErr := filepath.EvalSymlinks(filepath.Dir(dbPath)); parentErr == nil {
			dbPath = filepath.Join(parent, filepath.Base(dbPath))
		}
	}
	digest := sha256.Sum256([]byte(filepath.Clean(dbPath)))
	return "capacity-admission-" + hex.EncodeToString(digest[:])
}

// NewBrokerWithSource is a test seam and is also useful to callers that have
// already collected a host snapshot through another lifecycle pass.
func NewBrokerWithSource(store *internalcapacity.SQLiteStore, source internalcapacity.CapacitySource) *Broker {
	if source == nil {
		source = internalcapacity.HostInventorySource{}
	}
	var lockKey string
	if store != nil {
		lockKey = capacityAdmissionLockKey(store.DBPath())
	}
	return &Broker{store: store, source: source, admissionLockKey: lockKey}
}

// Close closes the underlying ledger.
func (b *Broker) Close() error {
	if b == nil || b.store == nil {
		return nil
	}
	return b.store.Close()
}

// Acquire admits one operation against RAM and CPU claims. A zero estimate is
// intentionally rejected by callers as unknown; this method only handles
// measured estimates.
func (b *Broker) Acquire(ctx context.Context, ownerID string, ramBytes, cpuMilli int64) (Lease, Verdict, error) {
	return b.acquire(ctx, ownerID, ramBytes, cpuMilli, false)
}

// AcquireNoReclaim admits an operation only from capacity that is already
// available. It applies the same fresh host sensing, pressure gates, and
// ledger-fit checks as Acquire, but will never schedule another owner's idle
// claim for reclamation.
func (b *Broker) AcquireNoReclaim(ctx context.Context, ownerID string, ramBytes, cpuMilli int64) (Lease, Verdict, error) {
	return b.acquire(ctx, ownerID, ramBytes, cpuMilli, true)
}

func (b *Broker) acquire(ctx context.Context, ownerID string, ramBytes, cpuMilli int64, noReclaim bool) (Lease, Verdict, error) {
	if b == nil || b.store == nil {
		return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: "capacity broker is unavailable"}, nil
	}
	if ownerID == "" || (ramBytes <= 0 && cpuMilli <= 0) {
		return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: "capacity request is missing owner or measured size"}, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.admissionLockKey != "" {
		unlock, err := artifactlock.AcquireContext(ctx, b.admissionLockKey)
		if err != nil {
			return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: "capacity admission lock unavailable: " + err.Error()}, nil
		}
		defer unlock()
	}
	// Admission is also an opportunistic liveness sweep. Operation claims are
	// short-lived and heartbeat-backed; if their owner crashed or the process
	// restarted, leaving them in the active ledger would make a dead operation
	// consume capacity until a separate maintenance pass happens. Reclaiming
	// elapsed claims here keeps admission authoritative at the point where it
	// makes a scheduling decision.
	at := time.Now().UTC()
	if b.now != nil {
		at = b.now().UTC()
	}
	// This path has no GPU absence proof. The owner defaults resident expiry to
	// unknown; operation TTL and non-GPU stale leases still expire normally.
	if _, err := b.store.ExpireStaleClaims(ctx, at, nil); err != nil {
		return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: "capacity liveness sweep failed: " + err.Error()}, nil
	}
	snapshot, err := b.snapshot(ctx)
	if err != nil {
		return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: "capacity sensing unavailable: " + err.Error()}, nil
	}
	policy, err := b.store.GetPolicy(ctx)
	if err != nil {
		return nil, Verdict{}, err
	}
	if ramBytes > 0 {
		var pressureErr error
		policy, pressureErr = internalcapacity.MemoryPressurePolicy(policy, b.memoryPressure)
		if pressureErr != nil {
			return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: pressureErr.Error()}, nil
		}
	}

	now := time.Now().UTC()
	var claims []leaseClaim
	rollback := func(cause error) error {
		if len(claims) == 0 {
			return cause
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), capacityCleanupTimeout)
		defer cancel()
		errs := make([]error, 0, len(claims)+1)
		if cause != nil {
			errs = append(errs, cause)
		}
		for _, claim := range claims {
			if _, releaseErr := b.store.ReleaseClaim(cleanupCtx, claim.id); releaseErr != nil {
				errs = append(errs, fmt.Errorf("rollback created capacity claim %s: %w", claim.id, releaseErr))
			}
		}
		return errors.Join(errs...)
	}
	for _, spec := range []struct {
		kind string
		want int64
	}{
		{kind: internalcapacity.ResourceKindRAM, want: ramBytes},
		{kind: internalcapacity.ResourceKindCPU, want: cpuMilli},
	} {
		if spec.want <= 0 {
			continue
		}
		ledger, err := b.store.ListClaims(ctx, internalcapacity.ClaimFilter{ResourceKind: spec.kind, Statuses: internalcapacity.ActiveClaimStatuses()})
		if err != nil {
			return nil, Verdict{}, rollback(err)
		}
		req := internalcapacity.CapacityRequest{
			OwnerKind:      internalcapacity.OwnerKindOp,
			OwnerID:        ownerID,
			ResourceKind:   spec.kind,
			PreferredBytes: spec.want,
			FloorBytes:     spec.want,
			Priority:       internalcapacity.PriorityBatch,
			NoReclaim:      noReclaim,
			TTL:            capacityHeartbeatTTL,
		}
		verdict := internalcapacity.Decide(req, snapshot, ledger, policy, now)
		if !verdict.Granted() {
			return nil, Verdict{Kind: verdict.Kind, Reason: verdict.Reason}, rollback(nil)
		}
		claim, err := b.store.CreateClaim(ctx, internalcapacity.CapacityClaim{
			OwnerKind:      req.OwnerKind,
			OwnerID:        req.OwnerID,
			ResourceKind:   req.ResourceKind,
			AmountBytes:    verdict.GrantedBytes,
			PreferredBytes: req.PreferredBytes,
			FloorBytes:     req.FloorBytes,
			Priority:       req.Priority,
			Status:         internalcapacity.StatusGranted,
		}, req.TTL)
		if err != nil {
			return nil, Verdict{}, rollback(fmt.Errorf("create %s capacity claim: %w", spec.kind, err))
		}
		claims = append(claims, leaseClaim{id: claim.ClaimID, generation: claim.Generation})
		if b.afterClaimCreated != nil {
			b.afterClaimCreated(claim)
		}
	}
	return &lease{broker: b, claims: claims}, Verdict{Kind: internalcapacity.VerdictGrant}, nil
}

// AcquireGPU admits one operation against a specific physical GPU. The UUID
// must match a device in the host snapshot; claims are charged to its current
// index and admission never reclaims another owner's resident model.
func (b *Broker) AcquireGPU(ctx context.Context, ownerID, expectedUUID string, vramBytes int64) (GPULease, Verdict, error) {
	deny := func(reason string) (GPULease, Verdict, error) {
		return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: reason}, nil
	}
	if b == nil || b.store == nil {
		return deny("capacity broker is unavailable")
	}
	ownerID, expectedUUID = strings.TrimSpace(ownerID), strings.TrimSpace(expectedUUID)
	if ownerID == "" || expectedUUID == "" || vramBytes <= 0 {
		return deny("GPU capacity request requires owner, exact UUID, and measured VRAM")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.admissionLockKey != "" {
		unlock, err := artifactlock.AcquireContext(ctx, b.admissionLockKey)
		if err != nil {
			return deny("capacity admission lock unavailable: " + err.Error())
		}
		defer unlock()
	}
	snapshot, err := b.snapshot(ctx)
	if err != nil {
		return deny("GPU sensing unavailable: " + err.Error())
	}
	var identity GPUIdentity
	for _, gpu := range snapshot.GPUs {
		if gpu.UUID == expectedUUID {
			if identity.UUID != "" {
				return deny("host inventory contains duplicate GPU UUID")
			}
			identity = GPUIdentity{Index: gpu.Index, UUID: gpu.UUID, Name: gpu.Name}
		}
	}
	if identity.UUID == "" {
		return deny("requested GPU UUID is absent from host inventory")
	}
	idx := identity.Index
	ledger, err := b.store.ListClaims(ctx, internalcapacity.ClaimFilter{ResourceKind: internalcapacity.ResourceKindVRAM, GPUIndex: &idx, Statuses: internalcapacity.ActiveClaimStatuses()})
	if err != nil {
		return nil, Verdict{}, err
	}
	policy, err := b.store.GetPolicy(ctx)
	if err != nil {
		return nil, Verdict{}, err
	}
	req := internalcapacity.CapacityRequest{
		OwnerKind: internalcapacity.OwnerKindOp, OwnerID: ownerID,
		ResourceKind: internalcapacity.ResourceKindVRAM, GPUIndex: &idx,
		PreferredBytes: vramBytes, FloorBytes: vramBytes,
		Priority: internalcapacity.PriorityInteractive, NoReclaim: true,
	}
	verdict := internalcapacity.Decide(req, snapshot, ledger, policy, time.Now().UTC())
	if !verdict.Granted() {
		return nil, Verdict{Kind: verdict.Kind, Reason: verdict.Reason}, nil
	}
	claim, err := b.store.CreateClaim(ctx, internalcapacity.CapacityClaim{
		OwnerKind: req.OwnerKind, OwnerID: req.OwnerID, ResourceKind: req.ResourceKind,
		GPUIndex: &idx, AmountBytes: verdict.GrantedBytes, PreferredBytes: req.PreferredBytes,
		FloorBytes: req.FloorBytes, Priority: req.Priority, Status: internalcapacity.StatusGranted,
		ActivityState: internalcapacity.ActivityActive, Protected: true,
	}, capacityHeartbeatTTL)
	if err != nil {
		return nil, Verdict{}, fmt.Errorf("create GPU capacity claim: %w", err)
	}
	return &gpuLease{lease: &lease{broker: b, claims: []leaseClaim{{id: claim.ClaimID, generation: claim.Generation}}}, identity: identity}, Verdict{Kind: internalcapacity.VerdictGrant}, nil
}

type gpuLease struct {
	*lease
	identity GPUIdentity
}

func (l *gpuLease) Identity() GPUIdentity {
	if l == nil {
		return GPUIdentity{}
	}
	return l.identity
}

// ObserveHostState returns the latest host reading without creating a claim.
// It is intentionally separate from Acquire so shadow instrumentation can
// record the context of a verdict without changing ledger state.
func (b *Broker) ObserveHostState(ctx context.Context) (HostObservation, error) {
	if b == nil || b.source == nil {
		return HostObservation{}, fmt.Errorf("capacity broker is unavailable")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	snapshot, err := b.snapshot(ctx)
	if err != nil {
		return HostObservation{}, err
	}
	used := uint64(0)
	if snapshot.Swap.TotalBytes > snapshot.Swap.FreeBytes {
		used = snapshot.Swap.TotalBytes - snapshot.Swap.FreeBytes
	}
	percent := 0.0
	if snapshot.Swap.TotalBytes > 0 {
		percent = float64(used) * 100 / float64(snapshot.Swap.TotalBytes)
	}
	return HostObservation{
		AvailableRAMBytes: snapshot.Memory.AvailableBytes,
		Load1:             snapshot.Load.Load1,
		SwapUsedPercent:   percent,
	}, nil
}

// snapshot returns a host reading no older than snapshotTTL, collecting a fresh
// one otherwise. The caller must hold b.mu.
//
// A failed collection does not evict a still-valid cached reading, and never
// serves an expired one: sensing that has genuinely stopped working must reach
// the caller as a deny, not be papered over by a stale number that would grant
// capacity the host may no longer have.
func (b *Broker) snapshot(ctx context.Context) (hostinventory.Snapshot, error) {
	now := time.Now
	if b.now != nil {
		now = b.now
	}
	at := now()
	if !b.cachedAt.IsZero() && at.Sub(b.cachedAt) < snapshotTTL {
		return b.cachedSnapshot, nil
	}
	fresh, err := b.source.Snapshot(ctx)
	if err != nil {
		return hostinventory.Snapshot{}, err
	}
	b.cachedSnapshot, b.cachedAt = fresh, at
	return fresh, nil
}

type leaseClaim struct {
	id         string
	generation int64
}

type lease struct {
	mu       sync.Mutex
	broker   *Broker
	claims   []leaseClaim
	released bool
}

// Heartbeat extends every claim under its captured generation. If a claim was
// expired, released, or changed by another owner, the store rejects renewal;
// this method never recreates or revives a claim.
func (l *lease) Heartbeat(ctx context.Context) error {
	if l == nil || l.broker == nil {
		return fmt.Errorf("capacity lease is unavailable")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return fmt.Errorf("capacity lease has been released")
	}
	for _, claim := range l.claims {
		if _, err := l.broker.store.HeartbeatClaim(ctx, claim.id, claim.generation, capacityHeartbeatTTL); err != nil {
			return fmt.Errorf("renew capacity claim %s: %w", claim.id, err)
		}
	}
	return nil
}

func (l *lease) Release(ctx context.Context) error {
	if l == nil || l.broker == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.released {
		return nil
	}
	l.broker.mu.Lock()
	defer l.broker.mu.Unlock()
	var first error
	for _, claim := range l.claims {
		if _, err := l.broker.store.ReleaseClaim(ctx, claim.id); err != nil && first == nil {
			first = err
		}
	}
	if first == nil {
		l.claims = nil
		l.released = true
	}
	return first
}
