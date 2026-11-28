package capacity

import (
	"context"
	"time"
)

// Claim constructs a claim at the recommended grant or, on a non-grant verdict,
// as a reserved preferred amount. A recorded claim
// is accounting, not proof that the recommendation permits execution.
func (r CapacityRequest) Claim(v Verdict) CapacityClaim {
	amount, status := r.PreferredBytes, StatusReserved
	if v.Granted() {
		amount, status = v.GrantedBytes, StatusGranted
		if v.Kind == VerdictDegrade {
			status = StatusDegraded
		}
	}
	return CapacityClaim{
		OwnerKind:      r.OwnerKind,
		OwnerID:        r.OwnerID,
		InstanceID:     r.InstanceID,
		ResourceKind:   r.ResourceKind,
		GPUIndex:       r.GPUIndex,
		AmountBytes:    amount,
		PreferredBytes: r.PreferredBytes,
		FloorBytes:     r.FloorBytes,
		Priority:       r.Priority,
		Protected:      r.Protected,
		YieldWhenIdle:  r.YieldWhenIdle,
		IdleUnloadTTL:  r.IdleUnloadTTL,
		IdleGrace:      r.IdleGrace,
		Status:         status,
		DegradeProfile: r.Profile,
	}
}

// Verdict is the output of Decide (plan §8.5). It carries no enforcement side
// effects; the caller (advisory or enforced) acts on it.
type Verdict struct {
	Kind          string   `json:"kind"` // grant | degrade | queue | deny
	GrantedBytes  int64    `json:"granted_bytes"`
	Step          string   `json:"step,omitempty"` // profile label of the granted step
	Reason        string   `json:"reason,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
	QueuePosition int      `json:"queue_position,omitempty"`
	// ReclaimTargets are claim IDs the caller would need to reclaim (degrade or
	// preempt, per the escalation ladder) to realize this grant. Empty when the
	// grant fits in immediately-free capacity. Decide only names them; the
	// actuator (§8.8) decides whether/how to act.
	ReclaimTargets []string `json:"reclaim_targets,omitempty"`
	// ReclaimBytes is the deficit (bytes) that must be freed from idle
	// lower-priority claims to realize this grant — i.e. how much the granted step
	// exceeds immediately-free capacity. Zero when the grant fits in free space.
	// The enforce-mode actuator plans an escalation to free exactly this much.
	ReclaimBytes int64 `json:"reclaim_bytes,omitempty"`
}

// Granted reports whether the verdict permits the caller to proceed on the GPU
// (grant or degrade). Queue/Deny do not.
func (v Verdict) Granted() bool {
	return v.Kind == VerdictGrant || v.Kind == VerdictDegrade
}

// Finding is one reconciliation result (plan §7 Phase 2 / doc.go).
type Finding struct {
	Class             string `json:"class"` // claimed | unclaimed | over_claim | declared_unclaimed
	OwnerID           string `json:"owner_id"`
	OwnerKind         string `json:"owner_kind,omitempty"`
	ResourceKind      string `json:"resource_kind"`
	GPUIndex          *int   `json:"gpu_index,omitempty"`
	PID               int    `json:"pid,omitempty"`
	PIDs              []int  `json:"pids,omitempty"`
	ProcessName       string `json:"process_name,omitempty"`
	ObservedBytes     int64  `json:"observed_bytes"`
	ObservedPeakBytes int64  `json:"observed_peak_bytes,omitempty"`
	ClaimedBytes      int64  `json:"claimed_bytes,omitempty"`
	ClaimID           string `json:"claim_id,omitempty"`
	Severity          string `json:"severity"` // info | warn
	Message           string `json:"message"`
}

// ExpiryExclusions preserves claims whose absence has not been established.
// Diagnostic only: exclusion does not authorize or block a ledger mutation.
type ExpiryExclusions map[string]string

func (e ExpiryExclusions) Contains(claimID string) bool { _, ok := e[claimID]; return ok }

// AbsenceProof binds a complete process probe to one captured claim generation.
// Repositories revalidate the CURRENT liveness boundary atomically at mutation.
type AbsenceProof struct {
	Generation int64
	ObservedAt time.Time
}
type AbsenceProofs map[string]AbsenceProof

// ClaimRepository is the per-concern repository for claim lifecycle (mirrors
// scenarioruntime's LifecycleRepository split).
type ClaimRepository interface {
	CreateClaim(ctx context.Context, claim CapacityClaim, ttl time.Duration) (CapacityClaim, error)
	HeartbeatClaim(ctx context.Context, claimID string, generation int64, ttl time.Duration) (CapacityClaim, error)
	ReportActivity(ctx context.Context, claimID string, generation int64, state string) (CapacityClaim, error)
	DegradeClaim(ctx context.Context, claimID string, generation int64, step string, amountBytes int64) (CapacityClaim, error)
	UpshiftClaim(ctx context.Context, claimID string, generation int64, step string, amountBytes int64) (CapacityClaim, error)
	// ResizeClaim changes an existing claim's granted amount in place and
	// leaves its declared preferred amount alone. It exists so an adopter whose real footprint moves does not have
	// to release and re-claim: that pattern is what produced 3,059 ledger rows
	// for one resource, and it loses the observed-usage history the
	// right-sizing advisory reads.
	ResizeClaim(ctx context.Context, claimID string, generation int64, amountBytes int64) (CapacityClaim, error)
	ReleaseClaim(ctx context.Context, claimID string) (CapacityClaim, error)
	PreemptClaim(ctx context.Context, claimID string, generation int64, reason string) (CapacityClaim, error)
	ExpireStaleClaims(ctx context.Context, at time.Time, proofs AbsenceProofs) ([]CapacityClaim, error)
	RecordObserved(ctx context.Context, claimID string, observed, peak int64, at time.Time, absence *AbsenceProof) (CapacityClaim, error)
	GetClaim(ctx context.Context, claimID string) (CapacityClaim, error)
	ListClaims(ctx context.Context, filter ClaimFilter) ([]CapacityClaim, error)
}

// FootprintRepository owns durable footprint observations independently from
// the short-lived claim ledger.
type FootprintRepository interface {
	RecordFootprint(ctx context.Context, observation FootprintObservation) (Footprint, error)
	ListFootprints(ctx context.Context, filter FootprintFilter) ([]Footprint, error)
	ResetFootprints(ctx context.Context, resource string) (int64, error)
}

// PolicyRepository is the per-concern repository for tunable levers.
type PolicyRepository interface {
	GetPolicy(ctx context.Context) (Policy, error)
	SetPolicyKey(ctx context.Context, key, value string) (Policy, error)
}
