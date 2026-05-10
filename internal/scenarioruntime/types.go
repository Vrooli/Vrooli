package scenarioruntime

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const (
	SchemaVersion = 4

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

	DefaultHeartbeatTTL           = 30 * time.Second
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
type Clock interface {
	Now() time.Time
}

type Instance struct {
	InstanceID           string
	Scenario             string
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
	ClaimID                   string
	InstanceID                string
	Scenario                  string
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

type InstanceFilter struct {
	Scenario     string
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
}

type SupervisorRepository interface {
	CreateSupervisorSession(ctx context.Context, session SupervisorSession, ttl time.Duration) (SupervisorSession, error)
	HeartbeatSupervisorSession(ctx context.Context, supervisorID string, ttl time.Duration) (SupervisorSession, error)
	StopSupervisorSession(ctx context.Context, supervisorID string, status string, reason string) (SupervisorSession, error)
	ListSupervisorSessions(ctx context.Context, filter SupervisorSessionFilter) ([]SupervisorSession, error)
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
}

type HealthRepository interface {
	UpsertHealthSnapshot(ctx context.Context, snapshot HealthSnapshot) (HealthSnapshot, error)
	GetHealthSnapshot(ctx context.Context, instanceID string) (HealthSnapshot, error)
}

type CleanupRepository interface {
	ExpireStaleStartingLeases(ctx context.Context, at time.Time) ([]Instance, error)
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
