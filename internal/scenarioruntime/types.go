package scenarioruntime

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	// SchemaVersion stamps PRAGMA user_version so an older or unknown database
	// fails loudly instead of being misread. schemaSQL is declarative — it
	// always describes the full current shape; there is no in-code migration
	// ladder (greenfield posture). A version bump means: edit schemaSQL, bump
	// this constant, and convert any existing local DB with a one-shot
	// operator-run script (see docs/plans/
	// project-internal-greenfield-migration-purge-plan.md).
	SchemaVersion = 8

	StatusStarting = "starting"
	StatusRunning  = "running"
	StatusStopping = "stopping"
	StatusStopped  = "stopped"
	StatusFailed   = "failed"
	StatusExpired  = "expired"

	SupervisorStatusRunning  = "running"
	SupervisorStatusStopping = "stopping"
	SupervisorStatusStopped  = "stopped"
	SupervisorStatusFailed   = "failed"

	OwnerKindLifecycle  = "lifecycle"
	OwnerKindSupervisor = "supervisor"
	OwnerKindManual     = "manual"

	ClaimStatusReserved = "reserved"
	ClaimStatusBound    = "bound"
	ClaimStatusReleased = "released"
	ClaimStatusExpired  = "expired"

	ListenerStatusUnknown               = "unknown"
	ListenerStatusListening             = "listening"
	ListenerStatusNotListening          = "not_listening"
	ListenerStatusInspectionUnavailable = "inspection_unavailable"
	ListenerStatusForeignListener       = "foreign_listener"

	HealthStatusHealthy       = "healthy"
	HealthStatusDegraded      = "degraded"
	HealthStatusUnhealthy     = "unhealthy"
	HealthStatusUnknown       = "unknown"
	HealthStatusNotConfigured = "not_configured"

	SupervisionPolicyManaged = "managed"
	SupervisionPolicyManual  = "manual"

	DefaultHeartbeatTTL = 30 * time.Second
	// DefaultSupervisedLeaseTTL is the deadline written when ownership sits
	// with a supervisor session rather than a lifecycle process. It is longer
	// than DefaultHeartbeatTTL because one supervisor renews the whole fleet on
	// a shared interval, so its window must absorb a slow tick and a restart.
	// Shared here so the handover and the renewer cannot drift apart.
	DefaultSupervisedLeaseTTL = 45 * time.Second
	// DefaultReservedClaimTTL bounds how long a reserved (not yet bound) port
	// claim survives without renewal before any allocator may expire it. The
	// lifecycle renews reserved claims alongside its instance heartbeats so a
	// slow start (e.g. a long setup build) keeps its ports.
	DefaultReservedClaimTTL       = 5 * time.Minute
	DefaultMaxHealthResponseBytes = 64 * 1024
)

var (
	activeInstanceStatuses        = []string{StatusStarting, StatusRunning}
	stopCandidateInstanceStatuses = []string{StatusStarting, StatusRunning, StatusFailed, StatusExpired}
	activePortClaimStatuses       = []string{ClaimStatusReserved, ClaimStatusBound}
)

var (
	ErrNotFound            = errors.New("scenario runtime record not found")
	ErrStaleGeneration     = errors.New("scenario runtime generation is stale")
	ErrActiveClaimConflict = errors.New("active scenario runtime port claim already exists")
	// ErrClaimNotReservable signals that BindPortClaim found the row but
	// its status was no longer 'reserved' — typically because another
	// process expired or released it between acquire and bind. Surfacing
	// this as a typed error lets the lifecycle layer treat "lost the
	// lease" cleanly instead of failing with a raw SQLite UNIQUE error.
	ErrClaimNotReservable = errors.New("scenario runtime port claim is no longer reservable")
)

func ActiveInstanceStatuses() []string {
	return append([]string(nil), activeInstanceStatuses...)
}

func IsActiveInstanceStatus(status string) bool {
	for _, active := range activeInstanceStatuses {
		if status == active {
			return true
		}
	}
	return false
}

func StopCandidateInstanceStatuses() []string {
	return append([]string(nil), stopCandidateInstanceStatuses...)
}

func ActivePortClaimStatuses() []string {
	return append([]string(nil), activePortClaimStatuses...)
}

func IsActivePortClaimStatus(status string) bool {
	for _, active := range activePortClaimStatuses {
		if status == active {
			return true
		}
	}
	return false
}

func IsDiscoverablePortClaimStatus(status string) bool {
	return status == ClaimStatusBound
}

// Clock is the time seam for repository operations. Production uses the real
// clock; tests provide fixed or manually advanced clocks.
type Clock = TimeSource

type TimeSource interface {
	Now() time.Time
}

type Instance struct {
	InstanceID string
	Scenario   string
	// Variant names which instance of the scenario this is ("live" for the
	// canonical primary, "shadow" etc. for alternates). Empty is normalized to
	// DefaultVariant on create, so pre-variant callers address the live instance.
	Variant              string
	Generation           int64
	ScopePath            string
	SandboxID            string
	Status               string
	Phase                string
	StartedAt            time.Time
	UpdatedAt            time.Time
	LastHeartbeatAt      *time.Time
	HeartbeatDeadlineAt  *time.Time
	StoppedAt            *time.Time
	StopReason           string
	OwnerKind            string
	OwnerPID             *int
	WorkingDir           string
	HostBootID           string
	HostSessionID        string
	SupervisorID         string
	SupervisedAt         *time.Time
	LastReconciledAt     *time.Time
	ReconciliationStatus string
	ReconciliationReason string
	SupervisionPolicy    string
	SchemaVersion        int
}

type SupervisorSession struct {
	SupervisorID        string
	HostBootID          string
	HostSessionID       string
	PID                 *int
	Status              string
	StartedAt           time.Time
	LastHeartbeatAt     time.Time
	HeartbeatDeadlineAt time.Time
	StoppedAt           *time.Time
	StopReason          string
	Version             string
	MetadataJSON        string
}

type PortClaim struct {
	ClaimID    string
	InstanceID string
	Scenario   string
	// Variant denormalizes the owning instance's variant onto the claim so
	// port-conflict queries can be scoped to a variant without a join. Empty is
	// normalized to DefaultVariant on acquire.
	Variant                   string
	PortName                  string
	EnvVar                    string
	Port                      int
	BindHost                  string
	URL                       string
	Status                    string
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
	ExpiresAt                 *time.Time
	LastBoundAt               *time.Time
	LastListenerCheckAt       *time.Time
	LastListenerSeenAt        *time.Time
	FirstUnboundAt            *time.Time
	ConsecutiveListenerMisses int
	ListenerStatus            string
	ListenerPID               *int
	ListenerProcessLabel      string
}

type HealthSnapshot struct {
	InstanceID    string
	Scenario      string
	Status        string
	Readiness     *bool
	CheckedAt     *time.Time
	LatencyMillis *int64
	Error         string
	ResponseJSON  string
	SchemaValid   *bool
}

type ProcessRef struct {
	RefID      string
	InstanceID string
	PID        *int
	PGID       *int
	ProcessID  string
	Step       string
	Command    string
	LogFile    string
	Status     string
	StartedAt  time.Time
	EndedAt    *time.Time
	HostBootID string
}

type Event struct {
	EventID     string
	InstanceID  string
	Scenario    string
	EventType   string
	CreatedAt   time.Time
	DetailsJSON string
}

// RecoveryPolicy is the durable declaration that permits a workload to be
// restored after a pressure incident. It is intentionally independent of an
// observed lease: a dead instance is never recoverable merely because it was
// once running. Recovery is disabled unless Enabled and Critical are both
// true, and OptOut always wins over every other field.
type RecoveryPolicy struct {
	Scenario       string
	Variant        string
	Critical       bool
	DependencyTier int
	Enabled        bool
	RetryBudget    int
	OptOut         bool
	UpdatedAt      time.Time
}

// RecoveryPolicyFilter narrows policy inspection. Empty fields are wildcards.
type RecoveryPolicyFilter struct {
	Scenario string
	Variant  string
	Enabled  *bool
}

// PressureEpoch is a correlated period of host pressure. The controller owns
// its state transitions; signal collectors only create or update evidence.
type PressureEpoch struct {
	EpochID     string
	Status      string
	Source      string
	DetectedAt  time.Time
	ClearedAt   *time.Time
	UpdatedAt   time.Time
	DetailsJSON string
}

const (
	PressureEpochDetected  = "detected"
	PressureEpochGated     = "gated"
	PressureEpochCleared   = "cleared"
	PressureEpochRegressed = "regressed"
)

// RecoveryDecision is the immutable audit record for one controller outcome.
// A stable IdempotencyKey deduplicates replay after a controller restart.
type RecoveryDecision struct {
	DecisionID     string
	EpochID        string
	Scenario       string
	Variant        string
	State          string
	Reason         string
	Attempt        int
	CooldownUntil  *time.Time
	IdempotencyKey string
	CreatedAt      time.Time
	DetailsJSON    string
}

const (
	RecoveryDecisionDetected = "detected"
	RecoveryDecisionGated    = "gated"
	RecoveryDecisionQueued   = "queued"
	RecoveryDecisionRestored = "restored"
	RecoveryDecisionSkipped  = "skipped"
	RecoveryDecisionFailed   = "failed"
)

type RecoveryDecisionFilter struct {
	EpochID  string
	Scenario string
	Variant  string
	Limit    int
}

type InstanceFilter struct {
	Scenario string
	// Variant, when non-empty, restricts the query to one variant so live and
	// shadow each resolve their own authoritative instance. Empty matches all
	// variants (the pre-variant behavior).
	Variant      string
	Statuses     []string
	SupervisorID string
}

type SupervisorSessionFilter struct {
	Statuses []string
}

type SupervisionClaim struct {
	InstanceID   string
	Generation   int64
	SupervisorID string
}

type PortClaimFilter struct {
	Scenario   string
	Variant    string
	InstanceID string
	Statuses   []string
}

type LifecycleRepository interface {
	CreateInstance(ctx context.Context, in Instance) (Instance, error)
	CreateLease(ctx context.Context, in Instance, ttl time.Duration) (Instance, error)
	HeartbeatLease(ctx context.Context, instanceID string, generation int64, ttl time.Duration) (Instance, error)
	ExpireStaleLeases(ctx context.Context, at time.Time) ([]Instance, error)
	StopLease(ctx context.Context, instanceID string, generation int64, reason string) (Instance, error)
	UpdateInstanceStatus(ctx context.Context, instanceID string, generation int64, status string, phase string) (Instance, error)
	GetInstance(ctx context.Context, instanceID string) (Instance, error)
	ListInstances(ctx context.Context, filter InstanceFilter) ([]Instance, error)
	// AttachLiveSupervision hands the instance to the live supervisor session
	// before the starting process exits, so lifecycle ownership never outlives
	// the command that created it. Reports false when no live session exists.
	AttachLiveSupervision(ctx context.Context, instanceID string, generation int64, ttl time.Duration) (Instance, bool, error)
}

type SupervisorRepository interface {
	CreateSupervisorSession(ctx context.Context, session SupervisorSession, ttl time.Duration) (SupervisorSession, error)
	HeartbeatSupervisorSession(ctx context.Context, supervisorID string, ttl time.Duration) (SupervisorSession, error)
	StopSupervisorSession(ctx context.Context, supervisorID string, status string, reason string) (SupervisorSession, error)
	ListSupervisorSessions(ctx context.Context, filter SupervisorSessionFilter) ([]SupervisorSession, error)
	// ExpireStaleSupervisorSessions marks provably dead sessions failed.
	// Without it a SIGKILLed supervisor leaves a row claiming status='running'
	// forever, and readers cannot tell the live supervisor from its corpses.
	ExpireStaleSupervisorSessions(ctx context.Context, at time.Time, guard StartingLeaseGuard) ([]SupervisorSession, error)
	ClaimSupervision(ctx context.Context, claim SupervisionClaim) (Instance, error)
	ReleaseSupervision(ctx context.Context, instanceID string, generation int64, supervisorID string) (Instance, error)
	UpdateInstanceReconciliation(ctx context.Context, instanceID string, generation int64, status string, reason string) (Instance, error)
	HeartbeatSupervisedLeaseBatch(ctx context.Context, claims []SupervisionClaim, ttl time.Duration) ([]Instance, error)
}

type QueryRepository interface {
	GetInstance(ctx context.Context, instanceID string) (Instance, error)
	ListInstances(ctx context.Context, filter InstanceFilter) ([]Instance, error)
	ListPortClaims(ctx context.Context, filter PortClaimFilter) ([]PortClaim, error)
	GetHealthSnapshot(ctx context.Context, instanceID string) (HealthSnapshot, error)
}

type PortClaimRepository interface {
	AcquirePortClaim(ctx context.Context, claim PortClaim) (PortClaim, error)
	BindPortClaim(ctx context.Context, claimID string) (PortClaim, error)
	ReleasePortClaim(ctx context.Context, claimID string) (PortClaim, error)
	ReleaseActivePortClaimsForInstance(ctx context.Context, instanceID string) ([]PortClaim, error)
	ListPortClaims(ctx context.Context, filter PortClaimFilter) ([]PortClaim, error)
	ListExpiredActivePortClaims(ctx context.Context, at time.Time) ([]PortClaim, error)
	UpdatePortClaimListenerEvidence(ctx context.Context, claimID string, evidence ListenerObservation) (PortClaim, error)
	RenewReservedPortClaimsForInstance(ctx context.Context, instanceID string, expiresAt time.Time) (int, error)
}

type HealthRepository interface {
	UpsertHealthSnapshot(ctx context.Context, snapshot HealthSnapshot) (HealthSnapshot, error)
	GetHealthSnapshot(ctx context.Context, instanceID string) (HealthSnapshot, error)
}

type CleanupRepository interface {
	ExpireStaleStartingLeases(ctx context.Context, at time.Time, guard StartingLeaseGuard) ([]Instance, error)
	ExpireInstance(ctx context.Context, instanceID string, reason string) (Instance, error)
	ExpirePortClaim(ctx context.Context, claimID string) (PortClaim, error)
}

type ProcessRefRepository interface {
	AddProcessRef(ctx context.Context, ref ProcessRef) (ProcessRef, error)
	UpdateProcessRefStatus(ctx context.Context, refID string, status string, endedAt *time.Time) (ProcessRef, error)
	ListProcessRefs(ctx context.Context, instanceID string) ([]ProcessRef, error)
}

type EventRepository interface {
	RecordEvent(ctx context.Context, event Event) (Event, error)
}

// RecoveryRepository persists the explicit desired-state/recovery contract.
// It deliberately exposes no implicit default policy: callers must upsert an
// explicit declaration before a workload can be selected for restoration.
type RecoveryRepository interface {
	UpsertRecoveryPolicy(ctx context.Context, policy RecoveryPolicy) (RecoveryPolicy, error)
	GetRecoveryPolicy(ctx context.Context, scenario, variant string) (RecoveryPolicy, error)
	ListRecoveryPolicies(ctx context.Context, filter RecoveryPolicyFilter) ([]RecoveryPolicy, error)
	CreatePressureEpoch(ctx context.Context, epoch PressureEpoch) (PressureEpoch, error)
	UpdatePressureEpoch(ctx context.Context, epoch PressureEpoch) (PressureEpoch, error)
	GetPressureEpoch(ctx context.Context, epochID string) (PressureEpoch, error)
	ListPressureEpochs(ctx context.Context, limit int) ([]PressureEpoch, error)
	RecordRecoveryDecision(ctx context.Context, decision RecoveryDecision) (RecoveryDecision, error)
	ListRecoveryDecisions(ctx context.Context, filter RecoveryDecisionFilter) ([]RecoveryDecision, error)
}

// LocalPortURL returns the canonical loopback URL stored with runtime port
// claims when the port kind has an HTTP surface Vrooli can safely advertise.
func LocalPortURL(portName string, port int) string {
	if port <= 0 {
		return ""
	}
	switch portName {
	case "api":
		return fmt.Sprintf("http://127.0.0.1:%d", port)
	default:
		return ""
	}
}

type ListenerObservation struct {
	CheckedAt    time.Time
	Status       string
	PID          *int
	ProcessLabel string
}
