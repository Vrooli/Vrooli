package runtimesupervisor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/envkit-go"
	"github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/accel"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	serviceParameterD = 20
)

const (
	DefaultRenewInterval        = tuning.ControlPlaneClientTimeout
	DefaultLeaseTTL             = scenarioruntime.DefaultSupervisedLeaseTTL
	DefaultHealthInterval       = tuning.SupervisorHealthInterval
	DefaultMaxHealthConcurrency = 16
	DefaultBatchSize            = 250
	DefaultRecoveryQuietPeriod  = tuning.ExtendedOperationTimeout
	DefaultRecoveryCooldown     = tuning.LongOperationTimeout
	DefaultRecoveryConcurrency  = 1
	DefaultPressureSomeAvg10    = 10.0
	// DefaultPressureCPUSomeAvg10 is deliberately much higher than the memory
	// threshold: CPU contention is normal on a build host, and only a run queue
	// deep enough to stall most work should gate recovery.
	DefaultPressureCPUSomeAvg10 = 50.0

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
	HomeDir              string
	DBPath               string
	SupervisorID         string
	Version              string
	RenewInterval        time.Duration
	LeaseTTL             time.Duration
	HealthInterval       time.Duration
	MaxHealthConcurrency int
	BatchSize            int
	Clock                scenarioruntime.Clock
	StoreFactory         StoreFactory
	HostProvider         hostsession.Provider
	PIDRunning           PIDRunningFunc
	PortListener         PortListenerFunc
	HealthProber         HealthProberFunc
	PressureProvider     PressureProvider
	RecoveryLaunch       RecoveryLaunchFunc
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
}

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
	Recovery         RecoveryReport `json:"recovery"`
	// Accelerator records what this tick decided about host accelerator
	// readiness. Zero-valued when no accelerator probe is configured.
	Accelerator AcceleratorReprobeReport `json:"accelerator"`
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
	svc := New(cfg)
	defer svc.Close()
	if err := svc.ensureStarted(ctx); err != nil {
		return err
	}
	svc.logf("supervisor started supervisor_id=%s pid=%d lease_ttl=%s renew_interval=%s",
		svc.session.SupervisorID, os.Getpid(), normalizeLeaseTTL(cfg.LeaseTTL), normalizeRenewInterval(cfg.RenewInterval))
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
	w := s.cfg.Stderr
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, "%s runtime-supervisor: %s\n", s.now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

//nolint:gocyclo // each tick reconciles independent lease, health, process, and recovery states.
func (s *Service) Tick(ctx context.Context) (TickReport, error) {
	if err := s.ensureStarted(ctx); err != nil {
		return TickReport{}, err
	}
	session, err := s.store.HeartbeatSupervisorSession(ctx, s.session.SupervisorID, normalizeLeaseTTL(s.cfg.LeaseTTL))
	if err != nil {
		return TickReport{}, err
	}
	s.session = session

	instances, err := s.store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return TickReport{}, fmt.Errorf("list supervised runtime candidates: %w", err)
	}
	claimsByInstance := make(map[string][]scenarioruntime.PortClaim, len(instances))
	refsByInstance := make(map[string][]scenarioruntime.ProcessRef, len(instances))
	processes := map[string]scenarioruntime.ProcessEvidence{}
	listeners := map[int]scenarioruntime.ListenerEvidence{}
	healthByInstance := map[string]scenarioruntime.HealthSnapshot{}
	instancesByID := make(map[string]scenarioruntime.Instance, len(instances))
	for i := range instances {
		instance := instances[i]
		instancesByID[instance.InstanceID] = instance
		claims, err := s.store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
			InstanceID: instance.InstanceID,
			Statuses:   scenarioruntime.ActivePortClaimStatuses(),
		})
		if err != nil {
			return TickReport{}, fmt.Errorf("list claims for %s: %w", instance.InstanceID, err)
		}
		refs, err := s.store.ListProcessRefs(ctx, instance.InstanceID)
		if err != nil {
			return TickReport{}, fmt.Errorf("list process refs for %s: %w", instance.InstanceID, err)
		}
		claimsByInstance[instance.InstanceID] = claims
		refsByInstance[instance.InstanceID] = refs
		health, err := s.store.GetHealthSnapshot(ctx, instance.InstanceID)
		if err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
			return TickReport{}, fmt.Errorf("get health snapshot for %s: %w", instance.InstanceID, err)
		}
		if err == nil {
			healthByInstance[instance.InstanceID] = health
		}
	}
	// One listener snapshot per tick, captured AFTER the claim reads so the
	// evidence is at least as fresh as the claim set; every claim of every
	// instance is answered from it (previously: one lsof + N ps PER CLAIM,
	// every tick, forever).
	portListener := s.tickPortListener()
	for _, instance := range instances {
		claims := claimsByInstance[instance.InstanceID]
		for key, evidence := range scenarioruntime.ProcessEvidenceFromRefs(refsByInstance[instance.InstanceID], s.pidRunning) {
			processes[key] = evidence
		}
		for port, evidence := range listenerEvidenceFromActiveClaims(claims, portListener) {
			listeners[port] = evidence
		}
		for _, claim := range claims {
			if !scenarioruntime.IsActivePortClaimStatus(claim.Status) || claim.Port <= 0 {
				continue
			}
			evidence, ok := listeners[claim.Port]
			if !ok {
				continue
			}
			if _, err := s.store.UpdatePortClaimListenerEvidence(ctx, claim.ClaimID, scenarioruntime.ListenerObservationFromEvidence(s.now(), evidence)); err != nil {
				// Evidence is an observation, not authority. A claim released
				// concurrently is the common cause and is not a tick failure.
				s.logf("could not record listener evidence for claim %s: %v", claim.ClaimID, err)
			}
		}
	}
	reconciled := s.reconcileStartingInstances(ctx, instances, claimsByInstance, refsByInstance, healthByInstance, listeners)
	if len(reconciled) > 0 {
		for _, instance := range reconciled {
			instancesByID[instance.InstanceID] = instance
			for i := range instances {
				if instances[i].InstanceID == instance.InstanceID {
					instances[i] = instance
					break
				}
			}
			claims, err := s.store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
				InstanceID: instance.InstanceID,
				Statuses:   scenarioruntime.ActivePortClaimStatuses(),
			})
			if err != nil {
				return TickReport{}, fmt.Errorf("reload claims for reconciled %s: %w", instance.InstanceID, err)
			}
			claimsByInstance[instance.InstanceID] = claims
		}
	}

	plan := scenarioruntime.PlanSupervision(scenarioruntime.SupervisionPlanInput{
		Now:              s.now(),
		CurrentBootID:    s.host.BootID,
		SupervisorID:     s.session.SupervisorID,
		Instances:        instances,
		ClaimsByInstance: claimsByInstance,
		RefsByInstance:   refsByInstance,
		Processes:        processes,
		Listeners:        listeners,
		HealthByInstance: healthByInstance,
		HealthInterval:   normalizeHealthInterval(s.cfg.HealthInterval),
		BatchSize:        normalizeBatchSize(s.cfg.BatchSize),
	})
	// From here on the tick is best-effort per instance. Renewal is the
	// supervisor's core duty and it must not be abandoned wholesale because one
	// instance was stopped, restarted, or claimed elsewhere mid-tick; every
	// scenario that CAN be renewed gets renewed, and what could not is counted
	// and logged.
	report := TickReport{SupervisorID: s.session.SupervisorID, Unverified: len(plan.Unverified), Expired: len(plan.Expire), HealthProbeCount: len(plan.HealthProbes)}
	for _, batch := range plan.RenewalBatches {
		claimed := s.claimRenewalBatch(ctx, batch, instancesByID)
		renewed, err := s.store.HeartbeatSupervisedLeaseBatch(ctx, claimed, normalizeLeaseTTL(s.cfg.LeaseTTL))
		if err != nil {
			// A whole-batch failure is a store problem, not an instance
			// problem: surface it so the consecutive-failure counter can act.
			return TickReport{}, fmt.Errorf("heartbeat supervised lease batch: %w", err)
		}
		report.Renewed += len(renewed)
		// Counted against the PLANNED batch, not the claimable subset, so an
		// instance dropped at either step is visible.
		if skipped := len(batch) - len(renewed); skipped > 0 {
			report.Skipped += skipped
		}
		for _, instance := range renewed {
			_, err := s.store.UpdateInstanceReconciliation(ctx, instance.InstanceID, instance.Generation, string(scenarioruntime.ReconcileVerifiedRunning), "supervisor renewed verified running lease")
			if err != nil {
				s.logf("could not record reconciliation for %s: %v", instance.InstanceID, err)
			}
		}
	}
	for _, item := range plan.Unverified {
		_, err := s.store.UpdateInstanceReconciliation(ctx, item.Instance.InstanceID, item.Instance.Generation, string(item.Classification), item.Reason)
		if err != nil {
			s.logf("could not record %s reconciliation for %s: %v", item.Classification, item.Instance.InstanceID, err)
		}
	}
	for _, item := range plan.Expire {
		if _, err := s.store.ExpireInstance(ctx, item.Instance.InstanceID, item.Reason); err != nil {
			s.logf("could not expire stale runtime %s (%s): %v", item.Instance.Scenario, item.Instance.InstanceID, err)
			continue
		}
		s.logf("expired stale runtime %s instance=%s reason=%q", item.Instance.Scenario, item.Instance.InstanceID, item.Reason)
	}
	// Retire dead peers every tick, not only at startup: a predecessor killed
	// mid-restart still has an unexpired deadline when its successor boots, so a
	// startup-only sweep leaves it claiming to run until something else looks.
	s.reapDeadSupervisorSessions(ctx)
	// Health probing and recovery are secondary to renewal, which has already
	// happened by this point. Neither may retroactively fail the tick and cost
	// the fleet its leases.
	probed, err := s.executeHealthProbes(ctx, plan.HealthProbes, instancesByID, claimsByInstance)
	if err != nil {
		s.logf("health probing degraded this tick: %v", err)
	}
	report.HealthProbeCount = probed
	recovery, err := s.reconcileRecovery(ctx)
	if err != nil {
		s.logf("recovery reconciliation degraded this tick: %v", err)
	}
	report.Recovery = recovery
	report.Accelerator = s.reprobeAccelerators(ctx)
	s.lastReport = report
	return report, nil
}

func listenerEvidenceFromActiveClaims(claims []scenarioruntime.PortClaim, inspect PortListenerFunc) map[int]scenarioruntime.ListenerEvidence {
	if inspect == nil {
		return nil
	}
	out := make(map[int]scenarioruntime.ListenerEvidence)
	for _, claim := range claims {
		if !scenarioruntime.IsActivePortClaimStatus(claim.Status) || claim.Port <= 0 {
			continue
		}
		out[claim.Port] = inspect(claim.Port)
	}
	return out
}

// reconcileStartingInstances promotes starting instances whose processes and
// listeners prove they are up. Every step is per-instance: an instance that
// cannot be promoted right now is left alone for the next tick rather than
// aborting promotion for the instances behind it in the list.
func (s *Service) reconcileStartingInstances(ctx context.Context, instances []scenarioruntime.Instance, claimsByInstance map[string][]scenarioruntime.PortClaim, refsByInstance map[string][]scenarioruntime.ProcessRef, healthByInstance map[string]scenarioruntime.HealthSnapshot, listeners map[int]scenarioruntime.ListenerEvidence) []scenarioruntime.Instance {
	var reconciled []scenarioruntime.Instance
	for _, instance := range instances {
		if instance.Status != scenarioruntime.StatusStarting {
			continue
		}
		claims := claimsByInstance[instance.InstanceID]
		if !startingInstanceHasLiveProcess(refsByInstance[instance.InstanceID], s.pidRunning) || !allActiveClaimsListening(claims, listeners) {
			continue
		}
		health := healthByInstance[instance.InstanceID]
		if health.InstanceID == "" || shouldProbeStartupHealth(health, s.now(), normalizeHealthInterval(s.cfg.HealthInterval)) {
			health = s.probeHealth(ctx, instance, claims)
			if _, err := s.store.UpsertHealthSnapshot(ctx, health); err != nil {
				s.logf("could not record startup health for %s: %v", instance.InstanceID, err)
			}
			healthByInstance[instance.InstanceID] = health
		}
		if !runtimeHealthAllowsStartupReconciliation(health) {
			continue
		}
		bindFailed := false
		for _, claim := range claims {
			if claim.Status != scenarioruntime.ClaimStatusReserved {
				continue
			}
			if _, err := s.store.BindPortClaim(ctx, claim.ClaimID); err != nil {
				s.logf("could not bind startup claim %s for %s: %v", claim.ClaimID, instance.Scenario, err)
				bindFailed = true
				break
			}
		}
		if bindFailed {
			// Promoting an instance whose ports are not all bound would
			// advertise a runtime the registry cannot fully account for.
			continue
		}
		updated, err := s.store.UpdateInstanceStatus(ctx, instance.InstanceID, instance.Generation, scenarioruntime.StatusRunning, instance.Phase)
		if err != nil {
			s.logf("could not promote starting instance %s (%s): %v", instance.Scenario, instance.InstanceID, err)
			continue
		}
		if _, err := s.store.UpdateInstanceReconciliation(ctx, updated.InstanceID, updated.Generation, string(scenarioruntime.ReconcileVerifiedRunning), "supervisor reconciled live startup runtime"); err != nil {
			s.logf("could not record startup reconciliation for %s: %v", updated.InstanceID, err)
		}
		reconciled = append(reconciled, updated)
	}
	return reconciled
}

func startingInstanceHasLiveProcess(refs []scenarioruntime.ProcessRef, pidRunning PIDRunningFunc) bool {
	if pidRunning == nil || len(refs) == 0 {
		return false
	}
	for _, ref := range refs {
		if ref.PID == nil || *ref.PID <= 0 {
			continue
		}
		if pidRunning(*ref.PID) {
			return true
		}
	}
	return false
}

func allActiveClaimsListening(claims []scenarioruntime.PortClaim, listeners map[int]scenarioruntime.ListenerEvidence) bool {
	active := 0
	for _, claim := range claims {
		if !scenarioruntime.IsActivePortClaimStatus(claim.Status) || claim.Port <= 0 {
			continue
		}
		active++
		evidence, ok := listeners[claim.Port]
		if !ok || !evidence.Known || !evidence.Listening {
			return false
		}
	}
	return active > 0
}

func shouldProbeStartupHealth(snapshot scenarioruntime.HealthSnapshot, now time.Time, interval time.Duration) bool {
	if interval <= 0 {
		return false
	}
	if snapshot.CheckedAt == nil {
		return true
	}
	return !snapshot.CheckedAt.Add(interval).After(now)
}

func runtimeHealthAllowsStartupReconciliation(health scenarioruntime.HealthSnapshot) bool {
	switch health.Status {
	case scenarioruntime.HealthStatusHealthy, scenarioruntime.HealthStatusDegraded, scenarioruntime.HealthStatusNotConfigured:
		return health.Readiness == nil || *health.Readiness
	default:
		return false
	}
}

// claimRenewalBatch takes ownership of any instance in the batch that is not
// already ours, and returns the claims that are safe to renew. An instance we
// could not claim — stopped, regenerated, or taken by a peer between the
// snapshot and now — is dropped from the batch rather than failing it: it is
// somebody else's instance now, which is a normal outcome and not this
// supervisor's problem to escalate.
func (s *Service) claimRenewalBatch(ctx context.Context, batch []scenarioruntime.SupervisionClaim, instances map[string]scenarioruntime.Instance) []scenarioruntime.SupervisionClaim {
	renewable := make([]scenarioruntime.SupervisionClaim, 0, len(batch))
	for _, claim := range batch {
		instance, ok := instances[claim.InstanceID]
		if !ok {
			continue
		}
		if instance.SupervisorID == claim.SupervisorID {
			renewable = append(renewable, claim)
			continue
		}
		claimed, err := s.store.ClaimSupervision(ctx, claim)
		if err != nil {
			s.logf("could not claim supervision of %s (%s): %v", instance.Scenario, claim.InstanceID, err)
			continue
		}
		instances[claim.InstanceID] = claimed
		renewable = append(renewable, claim)
	}
	return renewable
}

func (s *Service) Status(ctx context.Context) (StatusReport, error) {
	if err := s.ensureOpened(ctx); err != nil {
		return StatusReport{}, err
	}
	sessions, err := s.store.ListSupervisorSessions(ctx, scenarioruntime.SupervisorSessionFilter{Statuses: []string{scenarioruntime.SupervisorStatusRunning}})
	if err != nil {
		return StatusReport{}, err
	}
	var session scenarioruntime.SupervisorSession
	if len(sessions) > 0 {
		session = sessions[0]
	}
	policies, err := s.store.ListRecoveryPolicies(ctx, scenarioruntime.RecoveryPolicyFilter{})
	if err != nil {
		return StatusReport{}, fmt.Errorf("list runtime recovery policies: %w", err)
	}
	epochs, err := s.store.ListPressureEpochs(ctx, serviceParameterD)
	if err != nil {
		return StatusReport{}, fmt.Errorf("list runtime pressure epochs: %w", err)
	}
	if session.SupervisorID == "" {
		return StatusReport{
			Status:                        scenarioruntime.SupervisorStatusStopped,
			EffectiveRenewInterval:        normalizeRenewInterval(s.cfg.RenewInterval),
			EffectiveLeaseTTL:             normalizeLeaseTTL(s.cfg.LeaseTTL),
			EffectiveHealthInterval:       normalizeHealthInterval(s.cfg.HealthInterval),
			EffectiveMaxHealthConcurrency: normalizeMaxHealthConcurrency(s.cfg.MaxHealthConcurrency),
			EffectiveBatchSize:            normalizeBatchSize(s.cfg.BatchSize),
			EffectiveRecoveryQuietPeriod:  s.recoveryQuietPeriod(),
			EffectiveRecoveryCooldown:     s.recoveryCooldown(),
			EffectiveRecoveryConcurrency:  s.recoveryConcurrency(),
			RecoveryPolicies:              policies,
			PressureEpochs:                epochs,
			LastTick:                      s.lastReport,
		}, nil
	}
	status, reason := s.classifySupervisorSession(session)
	instances, err := s.store.ListInstances(ctx, scenarioruntime.InstanceFilter{SupervisorID: session.SupervisorID, Statuses: []string{scenarioruntime.StatusRunning}})
	if err != nil {
		return StatusReport{}, err
	}
	var unverified int
	for _, instance := range instances {
		if instance.ReconciliationStatus != "" && instance.ReconciliationStatus != string(scenarioruntime.ReconcileVerifiedRunning) {
			unverified++
		}
	}
	return StatusReport{
		SupervisorID:                  session.SupervisorID,
		Status:                        status,
		StatusReason:                  reason,
		HostBootID:                    session.HostBootID,
		HostSessionID:                 session.HostSessionID,
		PID:                           session.PID,
		LastHeartbeatAt:               session.LastHeartbeatAt,
		HeartbeatDeadlineAt:           session.HeartbeatDeadlineAt,
		SupervisedInstanceCount:       len(instances),
		UnverifiedInstanceCount:       unverified,
		EffectiveRenewInterval:        normalizeRenewInterval(s.cfg.RenewInterval),
		EffectiveLeaseTTL:             normalizeLeaseTTL(s.cfg.LeaseTTL),
		EffectiveHealthInterval:       normalizeHealthInterval(s.cfg.HealthInterval),
		EffectiveMaxHealthConcurrency: normalizeMaxHealthConcurrency(s.cfg.MaxHealthConcurrency),
		EffectiveBatchSize:            normalizeBatchSize(s.cfg.BatchSize),
		EffectiveRecoveryQuietPeriod:  s.recoveryQuietPeriod(),
		EffectiveRecoveryCooldown:     s.recoveryCooldown(),
		EffectiveRecoveryConcurrency:  s.recoveryConcurrency(),
		RecoveryPolicies:              policies,
		PressureEpochs:                epochs,
		LastTick:                      s.lastReport,
	}, nil
}

func (s *Service) classifySupervisorSession(session scenarioruntime.SupervisorSession) (string, string) {
	if session.Status != scenarioruntime.SupervisorStatusRunning {
		return session.Status, session.StopReason
	}
	now := s.now()
	if !session.HeartbeatDeadlineAt.IsZero() && !session.HeartbeatDeadlineAt.After(now) {
		return StatusStale, fmt.Sprintf("heartbeat deadline expired at %s", session.HeartbeatDeadlineAt.Format(time.RFC3339))
	}
	if session.PID != nil && *session.PID > 0 && !s.pidRunning(*session.PID) {
		return StatusDead, fmt.Sprintf("recorded PID %d is not running", *session.PID)
	}
	return scenarioruntime.SupervisorStatusRunning, ""
}

func (s *Service) Close() error {
	if s == nil || s.store == nil || !s.ownsStore {
		return nil
	}
	return s.store.Close()
}

func (s *Service) ensureStarted(ctx context.Context) error {
	if err := s.ensureOpened(ctx); err != nil {
		return err
	}
	if s.session.SupervisorID != "" {
		return nil
	}
	pid := os.Getpid()
	session, err := s.store.CreateSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:  s.cfg.SupervisorID,
		HostBootID:    s.host.BootID,
		HostSessionID: s.host.SessionID,
		PID:           &pid,
		Version:       s.cfg.Version,
	}, normalizeLeaseTTL(s.cfg.LeaseTTL))
	if err != nil {
		return err
	}
	s.session = session
	return nil
}

// reapDeadSupervisorSessions retires predecessor sessions this supervisor can
// prove are dead. A SIGKILLed supervisor never reached its graceful shutdown,
// so its row kept claiming status='running' indefinitely; taking over is the
// natural moment to clear them. Only provably dead sessions are touched, so a
// genuinely live peer is left alone. Failure is logged, never fatal: cleaning
// up bookkeeping must not prevent the fleet from being supervised.
func (s *Service) reapDeadSupervisorSessions(ctx context.Context) {
	expired, err := s.store.ExpireStaleSupervisorSessions(ctx, s.now(), scenarioruntime.StartingLeaseGuard{
		CurrentBootID: s.host.BootID,
		PIDRunning:    s.pidRunning,
	})
	if err != nil {
		s.logf("could not retire dead supervisor sessions: %v", err)
		return
	}
	if len(expired) > 0 {
		s.logf("retired %d dead supervisor session(s)", len(expired))
	}
}

func (s *Service) ensureOpened(ctx context.Context) error {
	if s.store != nil {
		return nil
	}
	hostProvider := s.cfg.HostProvider
	if hostProvider == nil {
		hostProvider = hostsession.DefaultProvider{}
	}
	host, err := hostProvider.Current(ctx, s.cfg.HomeDir)
	if err != nil {
		return fmt.Errorf("resolve host session: %w", err)
	}
	s.host = host
	factory := s.cfg.StoreFactory
	if factory == nil {
		factory = func(ctx context.Context, cfg scenarioruntime.Config) (Store, error) {
			return scenarioruntime.NewSQLiteStore(ctx, cfg)
		}
		s.ownsStore = true
	}
	store, err := factory(ctx, scenarioruntime.Config{HomeDir: s.cfg.HomeDir, DBPath: s.cfg.DBPath, Clock: s.cfg.Clock})
	if err != nil {
		return fmt.Errorf("open runtime registry: %w", err)
	}
	s.store = store
	return nil
}

func (s *Service) stopSession(ctx context.Context, status string, reason string) (scenarioruntime.SupervisorSession, error) {
	if s.store == nil || s.session.SupervisorID == "" {
		return scenarioruntime.SupervisorSession{}, nil
	}
	return s.store.StopSupervisorSession(ctx, s.session.SupervisorID, status, reason)
}

func (s *Service) now() time.Time {
	if s.cfg.Clock == nil {
		return time.Now().UTC()
	}
	return s.cfg.Clock.Now().UTC()
}

func (s *Service) pidRunning(pid int) bool {
	if s.cfg.PIDRunning != nil {
		return s.cfg.PIDRunning(pid)
	}
	return process.IsPIDRunning(pid)
}

// captureListenerSnapshotFn seams the per-tick snapshot capture so tests can
// pin the real tickPortListener branch (evidence conversion, capture-after-
// store-reads ordering) without reading the live host's TCP table.
var captureListenerSnapshotFn = network.CaptureTCPListenerSnapshot

// tickPortListener returns the per-tick listener evidence source: the
// configured override when present (tests), otherwise a closure over ONE
// freshly captured TCPListenerSnapshot shared by every port queried this
// tick.
func (s *Service) tickPortListener() PortListenerFunc {
	if s.cfg.PortListener != nil {
		return s.cfg.PortListener
	}
	snapshot := captureListenerSnapshotFn()
	return func(port int) scenarioruntime.ListenerEvidence {
		state := snapshot.Listening(port)
		if !state.Known {
			return scenarioruntime.ListenerEvidence{Known: false}
		}
		evidence := scenarioruntime.ListenerEvidence{Known: true, Listening: state.Listening}
		// Label provenance: listener_process_label previously carried the
		// per-PID `ps` command string; it now comes from /proc/<pid>/cmdline
		// (linux), the lsof comm name (darwin), or the process image path
		// (windows). Same information, new source; downstream only displays
		// it.
		if len(state.Listeners) > 0 {
			pid := state.Listeners[0].PID
			evidence.PID = &pid
			evidence.ProcessLabel = state.Listeners[0].Label
		}
		return evidence
	}
}

func (s *Service) executeHealthProbes(ctx context.Context, instanceIDs []string, instances map[string]scenarioruntime.Instance, claims map[string][]scenarioruntime.PortClaim) (int, error) {
	if len(instanceIDs) == 0 {
		return 0, nil
	}
	workers := normalizeMaxHealthConcurrency(s.cfg.MaxHealthConcurrency)
	if workers > len(instanceIDs) {
		workers = len(instanceIDs)
	}
	jobs := make(chan string)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error
	var completed int
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for instanceID := range jobs {
				if ctx.Err() != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = ctx.Err()
					}
					mu.Unlock()
					continue
				}
				instance, ok := instances[instanceID]
				if !ok {
					continue
				}
				snapshot := s.probeHealth(ctx, instance, claims[instanceID])
				if _, err := s.store.UpsertHealthSnapshot(ctx, snapshot); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("upsert health snapshot for %s: %w", instanceID, err)
					}
					mu.Unlock()
					continue
				}
				mu.Lock()
				completed++
				mu.Unlock()
			}
		}()
	}
	for _, instanceID := range instanceIDs {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return completed, ctx.Err()
		case jobs <- instanceID:
		}
	}
	close(jobs)
	wg.Wait()
	if firstErr != nil {
		return completed, firstErr
	}
	return completed, nil
}

func (s *Service) probeHealth(ctx context.Context, instance scenarioruntime.Instance, claims []scenarioruntime.PortClaim) scenarioruntime.HealthSnapshot {
	if s.cfg.HealthProber != nil {
		return s.cfg.HealthProber(ctx, instance, claims)
	}
	now := s.now()
	root := strings.TrimSpace(instance.ScopePath)
	if root == "" {
		return scenarioruntime.HealthSnapshot{
			InstanceID: instance.InstanceID,
			Scenario:   instance.Scenario,
			Status:     scenarioruntime.HealthStatusUnknown,
			CheckedAt:  &now,
			Error:      "runtime instance has no scope path for manifest health probing",
		}
	}
	item, err := scenario.Load(root, instance.Scenario, scenario.SandboxEnvFromEnv())
	if err != nil {
		return scenarioruntime.HealthSnapshot{
			InstanceID: instance.InstanceID,
			Scenario:   instance.Scenario,
			Status:     scenarioruntime.HealthStatusUnknown,
			CheckedAt:  &now,
			Error:      fmt.Sprintf("load scenario manifest: %v", err),
		}
	}
	return scenarioruntime.HealthProbe{Clock: s.cfg.Clock}.Probe(ctx, scenarioruntime.HealthProbeInput{
		InstanceID:   instance.InstanceID,
		Scenario:     instance.Scenario,
		HealthConfig: item.Manifest.HealthConfig(),
		Ports:        healthPortsFromClaims(item.Manifest, claims),
	})
}

func healthPortsFromClaims(manifest scenario.ServiceManifest, claims []scenarioruntime.PortClaim) map[string]int {
	ports := make(map[string]int)
	for _, claim := range claims {
		if claim.Port <= 0 {
			continue
		}
		envVar := strings.TrimSpace(claim.EnvVar)
		if envVar == "" {
			envVar = manifest.PortEnvVar(claim.PortName)
		}
		if envVar != "" {
			ports[envVar] = claim.Port
		}
	}
	return ports
}

func normalizeRenewInterval(v time.Duration) time.Duration {
	if v <= 0 {
		return DefaultRenewInterval
	}
	return v
}

func normalizeLeaseTTL(v time.Duration) time.Duration {
	if v <= 0 {
		return DefaultLeaseTTL
	}
	return v
}

func normalizeHealthInterval(v time.Duration) time.Duration {
	if v <= 0 {
		return DefaultHealthInterval
	}
	return v
}

func normalizeMaxHealthConcurrency(v int) int {
	if v <= 0 {
		return DefaultMaxHealthConcurrency
	}
	return v
}

func normalizeBatchSize(v int) int {
	if v <= 0 {
		return DefaultBatchSize
	}
	return v
}

func EnvConfig() Config {
	pressureThreshold := floatEnv("VROOLI_RUNTIME_PRESSURE_SOME_AVG10", DefaultPressureSomeAvg10)
	cpuPressureThreshold := floatEnv("VROOLI_RUNTIME_PRESSURE_CPU_SOME_AVG10", DefaultPressureCPUSomeAvg10)
	return Config{
		RenewInterval:        durationEnv("VROOLI_RUNTIME_SUPERVISOR_RENEW_INTERVAL", DefaultRenewInterval),
		LeaseTTL:             durationEnv("VROOLI_RUNTIME_SUPERVISOR_LEASE_TTL", DefaultLeaseTTL),
		HealthInterval:       durationEnv("VROOLI_RUNTIME_SUPERVISOR_HEALTH_INTERVAL", DefaultHealthInterval),
		MaxHealthConcurrency: intEnv("VROOLI_RUNTIME_SUPERVISOR_MAX_HEALTH_CONCURRENCY", DefaultMaxHealthConcurrency),
		BatchSize:            intEnv("VROOLI_RUNTIME_SUPERVISOR_BATCH_SIZE", DefaultBatchSize),
		RecoveryQuietPeriod:  durationEnv("VROOLI_RUNTIME_RECOVERY_QUIET_PERIOD", DefaultRecoveryQuietPeriod),
		RecoveryCooldown:     durationEnv("VROOLI_RUNTIME_RECOVERY_COOLDOWN", DefaultRecoveryCooldown),
		RecoveryConcurrency:  intEnv("VROOLI_RUNTIME_RECOVERY_CONCURRENCY", DefaultRecoveryConcurrency),
		PressureSomeAvg10:    pressureThreshold,
		PressureProvider:     NewHostPressureProviderWithCPU(pressureThreshold, cpuPressureThreshold),
	}
}

func ModeFromEnv() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(ModeEnv))) {
	case ModeOff:
		return ModeOff
	case ModeOn:
		return ModeOn
	case ModeAuto:
		return ModeAuto
	default:
		return ModeAuto
	}
}

func EnsureRunning(ctx context.Context, cfg Config) error {
	mode := ModeFromEnv()
	if mode == ModeOff {
		return nil
	}
	svc := New(cfg)
	report, err := svc.Status(ctx)
	_ = svc.Close()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	if cfg.Clock != nil {
		now = cfg.Clock.Now().UTC()
	}
	if report.Status == scenarioruntime.SupervisorStatusRunning && report.HeartbeatDeadlineAt.After(now) {
		return nil
	}

	if err := ctx.Err(); err != nil {
		return err
	}
	// Prefer the installed service. Launching the supervisor ourselves makes it
	// a child of whatever ran this command, and on Linux it then lives in that
	// caller's cgroup: a supervisor started from a scenario's own service (or a
	// terminal multiplexer pane) dies whenever that unrelated thing restarts.
	// The service manager places it correctly and keeps restarting it.
	if started, err := platform.StartInstalledService(platform.ServiceInstallOptions{HomeDir: cfg.HomeDir, User: true}); err != nil {
		// Fall through to the direct launch: an unsupervised supervisor is
		// still better than none, and the failure is reported where the caller
		// can log it.
		fmt.Fprintf(orDiscard(cfg.Stderr), "warning: could not start the installed runtime supervisor service (%v); launching it directly\n", err)
	} else if started {
		return nil
	}

	exe := strings.TrimSpace(cfg.Executable)
	if exe == "" {
		var exeErr error
		exe, exeErr = os.Executable()
		if exeErr != nil {
			return fmt.Errorf("resolve vrooli executable for runtime supervisor: %w", exeErr)
		}
	}
	cmd := exec.Command(exe, "--no-stale-check", "runtime", "supervisor", "run")
	cmd.Env = supervisorCommandEnv(os.Environ(), cfg.HomeDir)
	// The spawned supervisor outlives this process, so inheriting the caller's
	// streams is worse than useless: writes land on the stderr of a CLI that
	// has already exited, and io.Discard loses the reason it died. Give it the
	// same log file the installed service writes to, opened as a real *os.File
	// so the child inherits the descriptor instead of depending on a copying
	// goroutine that dies with us.
	logFile, logErr := openSupervisorLog(cfg.HomeDir)
	if logErr != nil && cfg.Stderr != nil {
		fmt.Fprintf(cfg.Stderr, "warning: runtime supervisor log unavailable (%v); its output will be discarded\n", logErr)
	}
	if logFile != nil {
		defer logFile.Close()
	}
	cmd.Stdout = firstNonNilWriter(cfg.Stdout, logFile)
	cmd.Stderr = firstNonNilWriter(cfg.Stderr, logFile)
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start runtime supervisor: %w", err)
	}
	return cmd.Process.Release()
}

// orDiscard keeps optional diagnostics from panicking on an unset writer.
func orDiscard(w io.Writer) io.Writer {
	if w == nil {
		return io.Discard
	}
	return w
}

// openSupervisorLog appends to the supervisor's log file, creating the log
// directory when needed. A nil file (with the error) means the caller falls
// back to discarding output rather than failing the start.
func openSupervisorLog(homeDir string) (*os.File, error) {
	path, err := LogPath(homeDir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), tuning.PermDir); err != nil {
		return nil, fmt.Errorf("create runtime supervisor log dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, tuning.PermFile)
	if err != nil {
		return nil, fmt.Errorf("open runtime supervisor log: %w", err)
	}
	return file, nil
}

// firstNonNilWriter prefers an explicitly configured writer, falls back to the
// log file, and discards only when neither exists. The typed-nil check matters:
// a nil *os.File in an io.Writer is not a nil interface.
func firstNonNilWriter(configured io.Writer, logFile *os.File) io.Writer {
	if configured != nil {
		return configured
	}
	if logFile != nil {
		return logFile
	}
	return io.Discard
}

func supervisorCommandEnv(env []string, home string) []string {
	overlay := envkit.Env{ModeEnv + "=" + ModeOn}
	if strings.TrimSpace(home) != "" {
		overlay = append(overlay, "HOME="+home)
	}
	return envkit.WithOverlay(envkit.Env(env), envkit.SameScenario, overlay)
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func intEnv(name string, fallback int) int {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func floatEnv(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
