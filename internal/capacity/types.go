package capacity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"
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
var DefaultHeartbeatTTL = tuning.CapacityHeartbeatTTL()

// DefaultIdleGrace is how long a claim must report activity_state=="idle"
// before it becomes reclaim-eligible. Age alone never triggers reclaim; this is
// only the dwell time AFTER the work-owner has reported idle.
var DefaultIdleGrace = tuning.ReachabilityTimeout()

// DefaultSweepInterval is the target cadence for the opportunistic resident-claim
// sweep (§8.6). The always-on maintenance pass drives Sweep on lifecycle
// activity; the opportunistic sweeps on admission/list/reconcile are debounced to
// this interval so rapid reads do not re-collect the GPU snapshot on every call.
var DefaultSweepInterval = tuning.CredentialServiceTimeout()

// DefaultDegradeDebounce is the minimum dwell between two degrade actuations of
// the same target, preventing a flapping requester from thrashing a resident
// server's VRAM (§8.8 anti-thrash).
var DefaultDegradeDebounce = tuning.CapacityHeartbeatTTL()

// DefaultUpshiftHeadroom is the free VRAM (bytes) that must be available before
// the actuator may upshift a degraded, idle claim back toward its preferred step
// (§8.8 hysteresis).
const DefaultUpshiftHeadroom = 2 * 1024 * 1024 * 1024 // 2 GiB

// DefaultObservedPeakHalflife is the decay half-life of the per-claim observed
// high-water mark (§Phase 2 sampling): a recorded peak loses half its value each
// half-life so a stale spike does not pin a reservation forever, yet a single
// idle reading never erases a real working-set peak.
var DefaultObservedPeakHalflife = tuning.CapacityObservedPeakHalflife()

// DefaultRecommendHeadroomPct is the safety margin (percent) added above a
// claim's observed peak when right-sizing recommends a smaller reservation
// (§Phase 4). A recommendation is never below observed_peak * (1 + pct/100).
const DefaultRecommendHeadroomPct = 20

// DefaultSwapPressureThreshold is the conservative swap-occupancy fallback
// percent used only when fresh memory PSI is unavailable. Occupancy alone is
// not proof of current paging; production adapters apply MemoryPressurePolicy
// before pure Decide. The legacy threshold remains unchanged.
const DefaultSwapPressureThreshold = 50

// DefaultTerminalRetention is how long a terminal (released/expired/preempted)
// claim survives in the ledger before terminal-claim GC prunes it. Terminal
// claims hold no capacity; they are kept briefly so a `capacity list` right after
// a release/expire still shows recent history, then pruned so the ledger does not
// accumulate dead rows.
var DefaultTerminalRetention = tuning.DailyRetentionWindow()

// GCResult reports what terminal-claim GC pruned: the number of rows deleted and
// the sum of their last-recorded amount_bytes (informational — terminal claims
// hold no live capacity).
type GCResult struct {
	Count int   `json:"count"`
	Bytes int64 `json:"bytes"`
}

const (
	FootprintSourceManifestDefault = "manifest_default"
	FootprintSourceMeasured        = "measured"
	ResidencyBackendCPU            = "cpu"
	ResidencyBackendCUDA           = "cuda"
	ResidencyBackendMetal          = "metal"
	ResidencyBackendROCm           = "rocm"
	ResidencyBackendVulkan         = "vulkan"
	ResidencyBackendUnloaded       = "unloaded"
)

// Keep release evidence no older than the default host GPU inventory TTL.
// This cap is intentionally fixed and conservative even when inventory caching
// is configured with a longer lifetime.
const maxResidencyFootprintAge = 2 * time.Minute

// Footprint is the durable, monotonic high-water mark for one effective
// resource configuration. It is deliberately separate from claim history so
// terminal-claim GC cannot erase measured operating knowledge.
type Footprint struct {
	Resource    string    `json:"resource"`
	Rung        string    `json:"rung"`
	TunablesKey string    `json:"tunables_key"`
	GPUIndex    int       `json:"gpu_index"`
	PeakBytes   int64     `json:"peak_bytes"`
	Samples     int64     `json:"samples"`
	FirstSeenAt time.Time `json:"first_seen_at"`
	LastSeenAt  time.Time `json:"last_seen_at"`
	Source      string    `json:"source"`
}

type FootprintFilter struct {
	Resource    string
	Rung        string
	TunablesKey *string
	GPUIndex    *int
}

type FootprintObservation struct {
	Resource    string
	Rung        string
	TunablesKey string
	GPUIndex    int
	Bytes       int64
	Source      string
	ObservedAt  time.Time
}

// ModelIdentity names the exact model artifact involved in a residency
// observation. Both fields are required before an observation can be joined
// to model-specific policy or evidence; a display name alone is not identity.
type ModelIdentity struct {
	Name   string `json:"name,omitempty"`
	Digest string `json:"digest,omitempty"`
}

func (m ModelIdentity) Complete() bool {
	return strings.TrimSpace(m.Name) != "" && strings.TrimSpace(m.Digest) != ""
}

// ResidencyPlacement describes a desired or observed location without
// collapsing unknown evidence into CPU or absence. Known is explicit because
// an empty backend can mean either a verified unload or a failed observation.
type ResidencyPlacement struct {
	Backend string `json:"backend,omitempty"`
	Known   bool   `json:"known"`
	Reason  string `json:"reason,omitempty"`
}

// ResidencyFootprint is one measured physical sample. Source and ObservedAt
// preserve provenance and freshness independently from the transition's claim
// that the sample has been verified.
type ResidencyFootprint struct {
	Bytes      int64     `json:"bytes"`
	Source     string    `json:"source"`
	ObservedAt time.Time `json:"observed_at"`
	Verified   bool      `json:"verified"`
}

// ResidencyTransition records one owner's transition attempt. Desired state,
// physical observation, model identity, reason, and footprint are independent
// facts. A partial or uncertain effect never reports released capacity.
type ResidencyTransition struct {
	ID              string             `json:"id"`
	Model           ModelIdentity      `json:"model"`
	Desired         ResidencyPlacement `json:"desired"`
	Observed        ResidencyPlacement `json:"observed"`
	Reason          string             `json:"reason"`
	StartedAt       time.Time          `json:"started_at"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	PreFootprint    ResidencyFootprint `json:"pre_footprint"`
	Footprint       ResidencyFootprint `json:"footprint"`
	EffectsVerified bool               `json:"effects_verified"`
}

// VerifiedReleasedBytes derives the released amount from verified pre/post
// measurements after a completed unload. Missing baselines, declared intent,
// and partial effects never prove that capacity is physically free.
func (t ResidencyTransition) VerifiedReleasedBytes() int64 {
	return t.verifiedReleasedBytesAt(time.Now())
}

func (t ResidencyTransition) verifiedReleasedBytesAt(now time.Time) int64 {
	if !t.verifiedUnloadAt(now) {
		return 0
	}
	return t.PreFootprint.Bytes - t.Footprint.Bytes
}

// VerifiedUnload reports an owner-verified absent model even when it held no
// VRAM before unload (for example a CPU-resident model). Such a release frees
// zero device bytes but still safely closes the exact zero-byte VRAM claim.
func (t ResidencyTransition) VerifiedUnload() bool {
	return t.verifiedUnloadAt(time.Now())
}

func (t ResidencyTransition) verifiedUnloadAt(now time.Time) bool {
	if t.ID == "" || !t.Model.Complete() || t.Reason == "" || t.StartedAt.IsZero() || t.CompletedAt == nil || !t.CompletedAt.After(t.StartedAt) {
		return false
	}
	if !t.EffectsVerified || !t.PreFootprint.Verified || !t.Footprint.Verified {
		return false
	}
	if t.PreFootprint.Source != FootprintSourceMeasured || t.Footprint.Source != FootprintSourceMeasured {
		return false
	}
	if t.PreFootprint.ObservedAt.IsZero() || t.Footprint.ObservedAt.IsZero() {
		return false
	}
	if t.PreFootprint.ObservedAt.After(t.StartedAt) || !t.Footprint.ObservedAt.After(t.StartedAt) || t.Footprint.ObservedAt.After(*t.CompletedAt) {
		return false
	}
	if now.Before(*t.CompletedAt) || t.StartedAt.Sub(t.PreFootprint.ObservedAt) > maxResidencyFootprintAge || now.Sub(t.Footprint.ObservedAt) > maxResidencyFootprintAge {
		return false
	}
	if t.PreFootprint.Bytes < 0 || t.Footprint.Bytes != 0 {
		return false
	}
	if !t.Desired.Known || !validResidencyBackend(t.Desired.Backend) || t.Desired.Backend != ResidencyBackendUnloaded {
		return false
	}
	if !t.Observed.Known || !validResidencyBackend(t.Observed.Backend) || t.Observed.Backend != ResidencyBackendUnloaded {
		return false
	}
	return true
}

func (t ResidencyTransition) hasFreshVerifiedObservation(now time.Time) bool {
	if t.CompletedAt == nil || now.Before(*t.CompletedAt) || !t.EffectsVerified || !t.Observed.Known || !validResidencyBackend(t.Observed.Backend) {
		return false
	}
	if !t.Footprint.Verified || t.Footprint.Source != FootprintSourceMeasured || t.Footprint.ObservedAt.IsZero() || !t.Footprint.ObservedAt.After(t.StartedAt) || t.Footprint.ObservedAt.After(*t.CompletedAt) {
		return false
	}
	return !t.Footprint.ObservedAt.After(now) && now.Sub(t.Footprint.ObservedAt) <= maxResidencyFootprintAge
}

func validResidencyBackend(backend string) bool {
	switch backend {
	case ResidencyBackendCPU, ResidencyBackendCUDA, ResidencyBackendMetal, ResidencyBackendROCm, ResidencyBackendVulkan, ResidencyBackendUnloaded:
		return true
	default:
		return false
	}
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
		return policyInteractive
	case priority >= PriorityService:
		return policyService
	default:
		return policyBatch
	}
}

// ParsePriorityTier maps a tier name (or numeric string) to its integer rank.
// An empty or unknown name defaults to batch (the lowest, safest tier).
func ParsePriorityTier(name string) int {
	switch name {
	case policyInteractive:
		return PriorityInteractive
	case policyService:
		return PriorityService
	case policyBatch, "":
		return PriorityBatch
	default:
		return PriorityBatch
	}
}

// DegradeStep is one rung of an adopter's degradation ladder.
type DegradeStep struct {
	Label       string `json:"label"`
	AmountBytes int64  `json:"amount_bytes"`
}

// UnmarshalJSON accepts the manifest seed spelling while retaining the live
// broker profile's amount_bytes wire contract. Resource schema validation
// prevents the runtime spelling from leaking back into resource manifests.
func (s *DegradeStep) UnmarshalJSON(data []byte) error {
	var wire struct {
		Label              string `json:"label"`
		AmountBytes        *int64 `json:"amount_bytes"`
		DefaultAmountBytes *int64 `json:"default_amount_bytes"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if wire.AmountBytes != nil && wire.DefaultAmountBytes != nil {
		return fmt.Errorf("degrade step cannot declare both amount_bytes and default_amount_bytes")
	}
	s.Label = wire.Label
	if wire.DefaultAmountBytes != nil {
		s.AmountBytes = *wire.DefaultAmountBytes
	} else if wire.AmountBytes != nil {
		s.AmountBytes = *wire.AmountBytes
	}
	return nil
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
	StepsSource string `json:"steps_source,omitempty"`
	// Model binds residency transitions to the exact model artifact owned by
	// this claim. It is optional for ordinary capacity-only profiles, but a
	// residency transition is rejected unless both name and digest are present.
	Model   ModelIdentity `json:"model,omitempty"`
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
	LastActiveAt        *time.Time // Last known active instant, including active-to-idle transition.
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
	OwnerKind     string
	OwnerID       string
	OwnerIDPrefix string
	ResourceKind  string
	GPUIndex      *int
	Statuses      []string
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
	// NoReclaim restricts this admission to already-free capacity. It does not
	// change priority or the resulting claim's idle-yield policy.
	NoReclaim     bool
	Protected     bool
	YieldWhenIdle bool
	IdleUnloadTTL time.Duration
	IdleGrace     time.Duration
	Profile       *DegradeProfile
	TTL           time.Duration
}

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
	// ResizeClaim changes an existing claim's granted and preferred amount in
	// place. It exists so an adopter whose real footprint moves does not have
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
