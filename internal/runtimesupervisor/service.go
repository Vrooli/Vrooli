package runtimesupervisor

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/clock"
	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/accel"
	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	serviceParameterD = 20
)

var (
	DefaultRenewInterval       = tuning.ControlPlaneClientTimeout()
	DefaultLeaseTTL            = scenarioruntime.DefaultSupervisedLeaseTTL
	DefaultHealthInterval      = tuning.SupervisorHealthInterval()
	DefaultRecoveryQuietPeriod = tuning.SupervisorRecoveryQuietPeriod()
	DefaultRecoveryCooldown    = tuning.SupervisorRecoveryCooldown()
)

const (
	DefaultMaxHealthConcurrency = 16
	DefaultBatchSize            = 250
	DefaultRecoveryConcurrency  = 1

	ModeEnv  = "VROOLI_RUNTIME_SUPERVISOR"
	ModeOff  = "off"
	ModeOn   = "on"
	ModeAuto = "auto"

	StatusStale = "stale"
	StatusDead  = "dead"
)

type Store interface {
	scenarioruntime.LifecycleRepository
	scenarioruntime.SupervisorRepository
	scenarioruntime.QueryRepository
	scenarioruntime.PortClaimRepository
	scenarioruntime.ProcessRefRepository
	scenarioruntime.CleanupRepository
	scenarioruntime.HealthRepository
	scenarioruntime.RecoveryRepository
	ExpireStaleEditorLeases(ctx context.Context, at time.Time, guard scenarioruntime.StartingLeaseGuard) ([]scenarioruntime.EditorLease, error)
	Close() error
}

type (
	StoreFactory     func(ctx context.Context, cfg scenarioruntime.Config) (Store, error)
	PIDRunningFunc   func(pid int) bool
	PortListenerFunc func(port int) scenarioruntime.ListenerEvidence
	HealthProberFunc func(ctx context.Context, instance scenarioruntime.Instance, claims []scenarioruntime.PortClaim) scenarioruntime.HealthSnapshot
	// PressureProvider is the boundary between the recovery controller and a
	// host/monitor implementation. Unknown evidence is intentionally distinct
	// from clear pressure so recovery can fail closed on collector degradation.
	PressureProvider interface {
		Snapshot(ctx context.Context) PressureState
	}
	// RecoveryLaunchFunc is the only launch seam recovery is allowed to use.
	// Its production adapter will invoke lifecycle start/restart rather than a
	// binary or shell process directly.
	RecoveryLaunchFunc func(ctx context.Context, request RecoveryLaunchRequest) error
	StopPIDFunc        func(pid int) error
)

type PressureState struct {
	Known         bool
	UnderPressure bool
	ObservedAt    time.Time
	Source        string
	Reason        string
}

type RecoveryLaunchRequest struct {
	Scenario string
	Variant  string
	EpochID  string
	Attempt  int
	DryRun   bool
}

type Config struct {
	// RootDir is the source repository root used by capacity maintenance to
	// discover resource-declared claim specs during recurring resident scans.
	RootDir              string
	HomeDir              string
	DBPath               string
	SupervisorID         string
	Version              string
	BuildIdentity        string
	Takeover             bool
	RenewInterval        time.Duration
	LeaseTTL             time.Duration
	HealthInterval       time.Duration
	MaxHealthConcurrency int
	BatchSize            int
	Clock                clock.Clock
	StoreFactory         StoreFactory
	HostProvider         hostsession.Provider
	PIDRunning           PIDRunningFunc
	PortListener         PortListenerFunc
	HealthProber         HealthProberFunc
	PressureProvider     PressureProvider
	RecoveryLaunch       RecoveryLaunchFunc
	StopPIDFunc          func(pid int) error
	RecoveryQuietPeriod  time.Duration
	RecoveryCooldown     time.Duration
	RecoveryConcurrency  int
	PressureSomeAvg10    float64
	Stdout               io.Writer
	Stderr               io.Writer
	Executable           string
	// AcceleratorProbe drives the accelerator re-probe pass. nil disables it,
	// which is the correct behaviour anywhere the control plane's resource
	// layer is not reachable.
	AcceleratorProbe AcceleratorProbe
	// AgentOrphanReapInterval is how often the supervisor stops agent
	// sessions whose owner is confirmed gone. Zero takes
	// tuning.AgentOrphanReapInterval(), whose compiled default is zero: the
	// periodic reap is OFF unless the operator sets
	// VROOLI_TUNING_AGENT_ORPHAN_REAP_INTERVAL for the supervisor.
	AgentOrphanReapInterval time.Duration
	// AgentOrphanReap performs one reap; nil uses agentsession.ReapOrphaned.
	AgentOrphanReap func(ctx context.Context, request agentsession.ReapRequest) ([]agentsession.Result, error)
	// ManagedUpdateReconcile resumes only durable, already-admitted operations.
	// It runs in one separate bounded worker, never on the fleet renewal path.
	ManagedUpdateReconcile func(context.Context) error
	// LightingReconcile converges explicitly enabled host lighting independently
	// of fleet renewal and managed-update work. Nil leaves the worker disabled.
	LightingReconcile func(context.Context) error
}

type Service struct {
	cfg        Config
	store      Store
	session    scenarioruntime.SupervisorSession
	host       hostsession.Snapshot
	ownsStore  bool
	lastReport TickReport
	// acceleratorWatcher holds the previous readiness observation so a device
	// that appears after boot is seen as a transition rather than as a steady
	// state.
	acceleratorWatcher accel.ReadinessWatcher
	// lastLogBound is when the scenario log size bound last ran; it runs at
	// most once per logBoundInterval, not every renewal tick.
	lastLogBound time.Time
	// lastAgentReap is when the orphaned agent session reap last ran.
	lastAgentReap time.Time
	logMu         sync.Mutex
}

// logBoundInterval spaces log bound sweeps. A runaway writer is caught within
// this window; a sweep over an ordinary logs tree costs a directory walk.
const logBoundInterval = 5 * time.Minute

type TickReport struct {
	SupervisorID string `json:"supervisor_id"`
	Renewed      int    `json:"renewed"`
	// Skipped counts leases in this tick's batch that no longer belonged to
	// this supervisor by the time the batch executed — stopped, regenerated,
	// or claimed by a peer. Normal in small numbers; a persistently high count
	// means two supervisors are fighting.
	Skipped          int            `json:"skipped"`
	Expired          int            `json:"expired"`
	Unverified       int            `json:"unverified"`
	HealthProbeCount int            `json:"health_probe_count"`
	DemandStopped    int            `json:"demand_stopped"`
	DemandFailures   int            `json:"demand_failures"`
	Recovery         RecoveryReport `json:"recovery"`
	// Accelerator records what this tick decided about host accelerator
	// readiness. Zero-valued when no accelerator probe is configured.
	Accelerator AcceleratorReprobeReport `json:"accelerator"`
	// LogBound reports the scenario log size bound when it ran this tick.
	LogBound LogBoundReport `json:"log_bound"`
	// AgentReap reports the orphaned agent session reap when it ran.
	AgentReap AgentReapReport `json:"agent_reap"`
}

// AgentReapReport is one periodic orphaned agent session reap.
type AgentReapReport struct {
	Ran     bool `json:"ran"`
	Stopped int  `json:"stopped"`
	Failed  int  `json:"failed"`
}

type LogBoundReport struct {
	Ran            bool  `json:"ran"`
	Trimmed        int   `json:"trimmed"`
	ReclaimedBytes int64 `json:"reclaimed_bytes"`
}

type RecoveryReport struct {
	EpochID  string `json:"epoch_id,omitempty"`
	Gated    int    `json:"gated"`
	Queued   int    `json:"queued"`
	Restored int    `json:"restored"`
	Skipped  int    `json:"skipped"`
	Failed   int    `json:"failed"`
}

type StatusReport struct {
	SupervisorID                  string                           `json:"supervisor_id"`
	Status                        string                           `json:"status"`
	StatusReason                  string                           `json:"status_reason,omitempty"`
	HostBootID                    string                           `json:"host_boot_id"`
	HostSessionID                 string                           `json:"host_session_id"`
	PID                           *int                             `json:"pid,omitempty"`
	LastHeartbeatAt               time.Time                        `json:"last_heartbeat_at"`
	HeartbeatDeadlineAt           time.Time                        `json:"heartbeat_deadline_at"`
	SupervisedInstanceCount       int                              `json:"supervised_instance_count"`
	UnverifiedInstanceCount       int                              `json:"unverified_instance_count"`
	EffectiveRenewInterval        time.Duration                    `json:"effective_renew_interval"`
	EffectiveLeaseTTL             time.Duration                    `json:"effective_lease_ttl"`
	EffectiveHealthInterval       time.Duration                    `json:"effective_health_interval"`
	EffectiveMaxHealthConcurrency int                              `json:"effective_max_health_concurrency"`
	EffectiveBatchSize            int                              `json:"effective_batch_size"`
	EffectiveRecoveryQuietPeriod  time.Duration                    `json:"effective_recovery_quiet_period"`
	EffectiveRecoveryCooldown     time.Duration                    `json:"effective_recovery_cooldown"`
	EffectiveRecoveryConcurrency  int                              `json:"effective_recovery_concurrency"`
	BuildIdentity                 string                           `json:"build_identity"`
	RecoveryPolicies              []scenarioruntime.RecoveryPolicy `json:"recovery_policies"`
	PressureEpochs                []scenarioruntime.PressureEpoch  `json:"pressure_epochs"`
	LastTick                      TickReport                       `json:"last_tick"`
}

func New(cfg Config) *Service {
	return &Service{cfg: cfg}
}

// MaxConsecutiveTickFailures bounds how long the supervisor keeps trying
// before it gives up and lets its service manager restart it with fresh state.
// At the default 10s renew interval this is ~100s of continuous failure.
const MaxConsecutiveTickFailures = 10

func Run(ctx context.Context, cfg Config) error {
	if !cfg.Takeover && runningUnderNativeUnit() {
		// The native unit is the one instance the host wants. A peer that is
		// live when the unit starts is either a supervisor a CLI launched
		// while the unit was down or a predecessor still shutting down;
		// exiting 1 and letting systemd retry every 5 s until its lease lapses
		// is what produced 1,070 restarts on 2026-09-02.
		cfg.Takeover = true
	}
	svc := New(cfg)
	defer svc.Close()
	if err := svc.ensureStarted(ctx); err != nil {
		svc.logf("supervisor cannot start: %v", err)
		return err
	}
	svc.logf("supervisor started supervisor_id=%s pid=%d lease_ttl=%s renew_interval=%s",
		svc.session.SupervisorID, os.Getpid(), normalizeLeaseTTL(cfg.LeaseTTL), normalizeRenewInterval(cfg.RenewInterval))
	workersCtx, cancelWorkers := context.WithCancel(ctx)
	stopUpdates := svc.startReconcileWorker(workersCtx, "managed update recovery", cfg.ManagedUpdateReconcile)
	stopLighting := svc.startReconcileWorker(workersCtx, "headless lighting", cfg.LightingReconcile)
	defer func() {
		cancelWorkers()
		stopLighting()
		stopUpdates()
	}()
	ticker := time.NewTicker(normalizeRenewInterval(cfg.RenewInterval))
	defer ticker.Stop()
	// A tick failure is no longer fatal. Renewal for the whole fleet used to
	// run through a single error return: any transient store contention, or one
	// instance changing state mid-tick, propagated out of Run and exited the
	// process — so every scenario's lease expired because one scenario was
	// stopped. Transient failures are logged and retried; only sustained
	// failure exits, because at that point a fresh process is the better
	// recovery and the service manager provides one.
	failures := 0
	for {
		if _, err := svc.Tick(ctx); err != nil {
			failures++
			svc.logf("tick failed (%d/%d consecutive): %v", failures, MaxConsecutiveTickFailures, err)
			if failures >= MaxConsecutiveTickFailures {
				svc.logf("giving up after %d consecutive tick failures; exiting for restart", failures)
				return err
			}
		} else if failures > 0 {
			svc.logf("tick recovered after %d consecutive failures", failures)
			failures = 0
		}
		select {
		case <-ctx.Done():
			_, _ = svc.stopSession(context.Background(), scenarioruntime.SupervisorStatusStopped, "context canceled")
			svc.logf("supervisor stopped supervisor_id=%s reason=%q", svc.session.SupervisorID, "context canceled")
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// logf writes one operational line to the supervisor's configured stderr, which
// its service unit points at ~/.vrooli/logs/runtime-supervisor.log. Kept
// deliberately sparse — startup, shutdown, and anything that went wrong — so
// the file stays readable and a failure has somewhere to be recorded.
func (s *Service) logf(format string, args ...any) {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	w := s.cfg.Stderr
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, "%s runtime-supervisor: %s\n", s.now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}
