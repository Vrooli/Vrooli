package capacity

import (
	"context"
	"errors"
	"time"
)

// SchemaVersion is the capacity ledger schema version, stamped into the SQLite
// database via PRAGMA user_version (mirroring internal/scenarioruntime).
// schemaSQL is declarative — it always describes the full current shape; there
// is no in-code migration ladder (greenfield posture). A version bump means:
// edit schemaSQL, bump this constant, and convert any existing local DB with a
// one-shot operator-run script (see docs/plans/
// project-internal-greenfield-migration-purge-plan.md). Older or unknown
// versions are rejected loudly rather than silently dropping live claims.
const SchemaVersion = 4

// Owner kinds — who holds a claim.
const (
	OwnerKindResource = "resource" // a long-lived resource (whisper, ollama, kyutai-stt)
	OwnerKindScenario = "scenario" // a scenario service
	OwnerKindOp       = "op"       // a short-lived operation (image-tools:job-<id>)
)

// Resource kinds. V1 enforces vram; the schema and decision logic are generic
// across ram/cpu for the V2 vision.
const (
	ResourceKindVRAM = "vram"
	ResourceKindRAM  = "ram"
	ResourceKindCPU  = "cpu"
)

// Claim statuses. Mirrors the reserved->granted->released/expired lifecycle of
// scenarioruntime port claims, with degraded/preempted added for the broker.
const (
	StatusReserved  = "reserved"
	StatusGranted   = "granted"
	StatusDegraded  = "degraded"
	StatusReleased  = "released"
	StatusExpired   = "expired"
	StatusPreempted = "preempted"
)

// Activity states. Reported by the work-owner, NEVER inferred from age/util.
const (
	ActivityActive = "active"
	ActivityIdle   = "idle"
)

// Priority tiers (higher wins). Stored as the integer rank on the claim so the
// schema stays engine-simple; the CLI accepts the tier name.
const (
	PriorityBatch       = 10 // background: image generation
	PriorityService     = 20 // resident model servers while idle
	PriorityInteractive = 30 // user-facing live: transcription, active coding agent
)

// Verdict kinds returned by Decide.
const (
	VerdictGrant   = "grant"
	VerdictDegrade = "degrade"
	VerdictQueue   = "queue"
	VerdictDeny    = "deny"
)

// Reconciliation finding classes.
const (
	FindingClaimed           = "claimed"
	FindingUnclaimed         = "unclaimed"
	FindingOverClaim         = "over_claim"
	FindingDeclaredUnclaimed = "declared_unclaimed"
)

// DefaultHeartbeatTTL bounds how long a claim survives without a heartbeat
// before the stale sweep expires it. Mirrors scenarioruntime's default.
const DefaultHeartbeatTTL = 30 * time.Second

// DefaultIdleGrace is how long a claim must report activity_state=="idle"
// before it becomes reclaim-eligible. Age alone never triggers reclaim; this is
// only the dwell time AFTER the work-owner has reported idle.
const DefaultIdleGrace = 60 * time.Second

// DefaultSweepInterval is the target cadence for the opportunistic resident-claim
// sweep (§8.6). The always-on maintenance pass drives Sweep on lifecycle
// activity; the opportunistic sweeps on admission/list/reconcile are debounced to
// this interval so rapid reads do not re-collect the GPU snapshot on every call.
const DefaultSweepInterval = 15 * time.Second

// DefaultDegradeDebounce is the minimum dwell between two degrade actuations of
// the same target, preventing a flapping requester from thrashing a resident
// server's VRAM (§8.8 anti-thrash).
const DefaultDegradeDebounce = 30 * time.Second

// DefaultUpshiftHeadroom is the free VRAM (bytes) that must be available before
// the actuator may upshift a degraded, idle claim back toward its preferred step
// (§8.8 hysteresis).
const DefaultUpshiftHeadroom = 2 * 1024 * 1024 * 1024 // 2 GiB

// DefaultObservedPeakHalflife is the decay half-life of the per-claim observed
// high-water mark (§Phase 2 sampling): a recorded peak loses half its value each
// half-life so a stale spike does not pin a reservation forever, yet a single
// idle reading never erases a real working-set peak.
const DefaultObservedPeakHalflife = 10 * time.Minute

// DefaultRecommendHeadroomPct is the safety margin (percent) added above a
// claim's observed peak when right-sizing recommends a smaller reservation
// (§Phase 4). A recommendation is never below observed_peak * (1 + pct/100).
const DefaultRecommendHeadroomPct = 20

// DefaultSwapPressureThreshold is the fraction of swap in use at which the host
// is considered to be under memory pressure, expressed as a percent.
//
// Swap usage is the lagging signal the RAM figures miss: once pages are on
// disk, AvailableBytes can look healthy while the machine thrashes, and
// admitting more memory-hungry work then makes the situation worse rather than
// better. 50% is deliberately not aggressive — a host that has touched swap at
// all is not necessarily in trouble, but one running on half of it is.
const DefaultSwapPressureThreshold = 50

// DefaultTerminalRetention is how long a terminal (released/expired/preempted)
// claim survives in the ledger before terminal-claim GC prunes it. Terminal
// claims hold no capacity; they are kept briefly so a `capacity list` right after
// a release/expire still shows recent history, then pruned so the ledger does not
// accumulate dead rows.
const DefaultTerminalRetention = 24 * time.Hour

// GCResult reports what terminal-claim GC pruned: the number of rows deleted and
// the sum of their last-recorded amount_bytes (informational — terminal claims
// hold no live capacity).
type GCResult struct {
	Count int   `json:"count"`
	Bytes int64 `json:"bytes"`
}

var (
	// ErrNotFound is returned when a claim row does not exist.
	ErrNotFound = errors.New("capacity claim not found")
	// ErrStaleGeneration signals optimistic-concurrency loss: the claim was
	// mutated by another writer between read and write. Mirrors
	// scenarioruntime.ErrStaleGeneration.
	ErrStaleGeneration = errors.New("capacity claim generation is stale")
	// ErrInvalidClaim signals a malformed claim request (missing owner, bad
	// bytes, unknown resource kind).
	ErrInvalidClaim = errors.New("capacity claim is invalid")
)

// activeClaimStatuses are the statuses that occupy capacity (counted against
// the host) and are eligible for heartbeat/reconcile/reclaim.
var activeClaimStatuses = []string{StatusReserved, StatusGranted, StatusDegraded}

// ActiveClaimStatuses returns a copy of the statuses that occupy capacity.
func ActiveClaimStatuses() []string {
	return append([]string(nil), activeClaimStatuses...)
}

// IsActiveClaimStatus reports whether a claim in this status occupies capacity.
func IsActiveClaimStatus(status string) bool {
	for _, s := range activeClaimStatuses {
		if status == s {
			return true
		}
	}
	return false
}

// PriorityTierName maps a priority rank to its tier label (best-effort; unknown
// ranks render as the numeric value).
func PriorityTierName(priority int) string {
	switch {
	case priority >= PriorityInteractive:
		return "interactive"
	case priority >= PriorityService:
		return "service"
	default:
		return "batch"
	}
}

// ParsePriorityTier maps a tier name (or numeric string) to its integer rank.
// An empty or unknown name defaults to batch (the lowest, safest tier).
func ParsePriorityTier(name string) int {
	switch name {
	case "interactive":
		return PriorityInteractive
	case "service":
		return PriorityService
	case "batch", "":
		return PriorityBatch
	default:
		return PriorityBatch
	}
}

// Clock is the time seam for store operations. Production uses the real clock;
// tests provide fixed or manually advanced clocks. Mirrors scenarioruntime.
type Clock = TimeSource

type TimeSource interface {
	Now() time.Time
}

// DegradeStep is one rung of an adopter's degradation ladder.
type DegradeStep struct {
	Label       string `json:"label"`
	AmountBytes int64  `json:"amount_bytes"`
}

// DegradeApply declares how the broker asks the adopter to step (the adopter
// implements the resize). argv may contain a "{label}" placeholder.
type DegradeApply struct {
	Verb string   `json:"verb"`
	Argv []string `json:"argv"`
}

// DegradeProfile is the adopter-declared degradation ladder (plan §8.2). Steps
// are ordered top (preferred) to bottom (floor); the last step's amount equals
// the claim's floor_bytes. For image-tools the last step is "cpu" (amount 0).
type DegradeProfile struct {
	Steps   []DegradeStep `json:"steps"`
	Apply   DegradeApply  `json:"apply"`
	Upshift bool          `json:"upshift"`
}

// CapacityClaim is the ledger row (plan §8.1).
type CapacityClaim struct {
	ClaimID        string
	OwnerKind      string
	OwnerID        string
	InstanceID     string // links to a scenarioruntime instance when applicable
	ResourceKind   string
	GPUIndex       *int
	AmountBytes    int64 // current granted amount (the active degradation step)
	PreferredBytes int64 // top of the profile
	FloorBytes     int64 // min-viable
	Priority       int
	Protected      bool
	// YieldWhenIdle opts this claim into the idle-yield rule (§8.3): when it has
	// dwelt idle beyond idle_grace its effective priority drops to the
	// idle_yield_floor for reclaim eligibility, so active work at or above the
	// floor may reclaim its capacity. Active claims are NEVER demoted. Claims
	// without the flag keep the strict-priority rule byte-for-byte.
	YieldWhenIdle       bool
	Status              string
	ActivityState       string
	Generation          int64
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastHeartbeatAt     *time.Time
	HeartbeatDeadlineAt *time.Time
	LastActiveAt        *time.Time
	DegradeProfile      *DegradeProfile
	// ObservedBytes is the latest sampled GPU usage attributed to this claim's
	// owner (telemetry; never fed into Decide — contract C1).
	ObservedBytes int64
	// ObservedPeakBytes is the decaying high-water mark of observed usage
	// (contract C2): VRAM is non-compressible so we track the peak, not an average.
	ObservedPeakBytes int64
	// ObservedAt is when ObservedBytes/ObservedPeakBytes were last sampled; nil
	// until the first sample. Used to age the decaying peak.
	ObservedAt *time.Time
	// IdleUnloadTTL is the autonomous idle-unload dwell (§Phase 3): once the claim
	// has been idle this long the broker proactively degrades it to floor/unloaded
	// (advisory logs, enforce actuates). 0 disables autonomous unload for this
	// claim. Distinct from idle_grace (which gates demand-driven reclaim).
	IdleUnloadTTL time.Duration
	// IdleGrace is the demand-driven reclaim dwell for this specific claim. When
	// zero, policy.IdleGrace remains the fallback so existing claims preserve the
	// historical global behavior.
	IdleGrace time.Duration
}

// ClaimFilter narrows ListClaims.
type ClaimFilter struct {
	OwnerKind    string
	OwnerID      string
	ResourceKind string
	GPUIndex     *int
	Statuses     []string
}

// CapacityRequest is the input to Decide and CreateClaim.
type CapacityRequest struct {
	OwnerKind      string
	OwnerID        string
	InstanceID     string
	ResourceKind   string
	GPUIndex       *int
	PreferredBytes int64
	FloorBytes     int64
	Priority       int
	Protected      bool
	YieldWhenIdle  bool
	IdleUnloadTTL  time.Duration
	IdleGrace      time.Duration
	Profile        *DegradeProfile
	TTL            time.Duration
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
	ProcessName       string `json:"process_name,omitempty"`
	ObservedBytes     int64  `json:"observed_bytes"`
	ObservedPeakBytes int64  `json:"observed_peak_bytes,omitempty"`
	ClaimedBytes      int64  `json:"claimed_bytes,omitempty"`
	ClaimID           string `json:"claim_id,omitempty"`
	Severity          string `json:"severity"` // info | warn
	Message           string `json:"message"`
}

// ClaimRepository is the per-concern repository for claim lifecycle (mirrors
// scenarioruntime's LifecycleRepository split).
type ClaimRepository interface {
	CreateClaim(ctx context.Context, claim CapacityClaim, ttl time.Duration) (CapacityClaim, error)
	HeartbeatClaim(ctx context.Context, claimID string, generation int64, ttl time.Duration) (CapacityClaim, error)
	ReportActivity(ctx context.Context, claimID string, generation int64, state string) (CapacityClaim, error)
	DegradeClaim(ctx context.Context, claimID string, generation int64, step string, amountBytes int64) (CapacityClaim, error)
	UpshiftClaim(ctx context.Context, claimID string, generation int64, step string, amountBytes int64) (CapacityClaim, error)
	ReleaseClaim(ctx context.Context, claimID string) (CapacityClaim, error)
	PreemptClaim(ctx context.Context, claimID string, reason string) (CapacityClaim, error)
	ExpireStaleClaims(ctx context.Context, at time.Time) ([]CapacityClaim, error)
	RecordObserved(ctx context.Context, claimID string, observed, peak int64, at time.Time) (CapacityClaim, error)
	GetClaim(ctx context.Context, claimID string) (CapacityClaim, error)
	ListClaims(ctx context.Context, filter ClaimFilter) ([]CapacityClaim, error)
}

// PolicyRepository is the per-concern repository for tunable levers.
type PolicyRepository interface {
	GetPolicy(ctx context.Context) (Policy, error)
	SetPolicyKey(ctx context.Context, key, value string) (Policy, error)
}
