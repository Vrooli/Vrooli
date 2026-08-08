// Package capacity exposes the narrow public admission seam used by modules
// that cannot import Vrooli's root internal/capacity package.
package capacity

import (
	"context"
	"fmt"
	"sync"
	"time"

	internalcapacity "github.com/vrooli/vrooli/internal/capacity"
	"github.com/vrooli/vrooli/internal/hostinventory"
)

// Verdict describes whether a short-lived operation may consume its requested
// host capacity. Queue and deny both fall back to serial execution upstream.
type Verdict struct {
	Kind   string
	Reason string
}

// Lease releases all claims acquired for one operation.
type Lease interface {
	Release(context.Context) error
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

	// cachedSnapshot is the last host reading and when it was taken. It is
	// guarded by mu, which Acquire already holds for its whole body.
	cachedSnapshot hostinventory.Snapshot
	cachedAt       time.Time
	// now is a clock seam so the TTL is testable without sleeping.
	now func() time.Time
}

// NewBroker opens the shared capacity ledger. An empty dbPath uses the normal
// Vrooli runtime-home state path.
func NewBroker(ctx context.Context, dbPath string) (*Broker, error) {
	store, err := internalcapacity.NewSQLiteStore(ctx, internalcapacity.Config{DBPath: dbPath})
	if err != nil {
		return nil, err
	}
	return &Broker{store: store, source: internalcapacity.HostInventorySource{}}, nil
}

// NewBrokerWithSource is a test seam and is also useful to callers that have
// already collected a host snapshot through another lifecycle pass.
func NewBrokerWithSource(store *internalcapacity.SQLiteStore, source internalcapacity.CapacitySource) *Broker {
	if source == nil {
		source = internalcapacity.HostInventorySource{}
	}
	return &Broker{store: store, source: source}
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
	if b == nil || b.store == nil {
		return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: "capacity broker is unavailable"}, nil
	}
	if ownerID == "" || (ramBytes <= 0 && cpuMilli <= 0) {
		return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: "capacity request is missing owner or measured size"}, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	snapshot, err := b.snapshot(ctx)
	if err != nil {
		return nil, Verdict{Kind: internalcapacity.VerdictDeny, Reason: "capacity sensing unavailable: " + err.Error()}, nil
	}
	policy, err := b.store.GetPolicy(ctx)
	if err != nil {
		return nil, Verdict{}, err
	}
	now := time.Now().UTC()
	var claimIDs []string
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
			return nil, Verdict{}, err
		}
		req := internalcapacity.CapacityRequest{
			OwnerKind:      internalcapacity.OwnerKindOp,
			OwnerID:        ownerID,
			ResourceKind:   spec.kind,
			PreferredBytes: spec.want,
			FloorBytes:     spec.want,
			Priority:       internalcapacity.PriorityBatch,
			TTL:            15 * time.Minute,
		}
		verdict := internalcapacity.Decide(req, snapshot, ledger, policy, now)
		if !verdict.Granted() {
			for _, claimID := range claimIDs {
				_, _ = b.store.ReleaseClaim(ctx, claimID)
			}
			return nil, Verdict{Kind: verdict.Kind, Reason: verdict.Reason}, nil
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
			for _, claimID := range claimIDs {
				_, _ = b.store.ReleaseClaim(ctx, claimID)
			}
			return nil, Verdict{}, fmt.Errorf("create %s capacity claim: %w", spec.kind, err)
		}
		claimIDs = append(claimIDs, claim.ClaimID)
	}
	return &lease{broker: b, claimIDs: claimIDs}, Verdict{Kind: internalcapacity.VerdictGrant}, nil
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

type lease struct {
	broker   *Broker
	claimIDs []string
}

func (l *lease) Release(ctx context.Context) error {
	if l == nil || l.broker == nil {
		return nil
	}
	l.broker.mu.Lock()
	defer l.broker.mu.Unlock()
	var first error
	for _, claimID := range l.claimIDs {
		if _, err := l.broker.store.ReleaseClaim(ctx, claimID); err != nil && first == nil {
			first = err
		}
	}
	l.claimIDs = nil
	return first
}
