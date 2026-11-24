package runtimesupervisor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/vrooli/internal/agentsession"
	"github.com/vrooli/vrooli/internal/process"
	runtimestorage "github.com/vrooli/vrooli/internal/resources/runtime/storage"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/tuning"
)

// reapOrphanedAgentSessions stops agent sessions whose owner is confirmed
// gone, at most once per configured interval. It is off by default: the
// interval's compiled default is zero, and the operator path `vrooli agent
// reap --orphaned [--dry-run]` exists so the policy can be watched before it
// is trusted unattended. A reap failure is logged and never fails the tick.
func (s *Service) reapOrphanedAgentSessions(ctx context.Context) AgentReapReport {
	interval := s.cfg.AgentOrphanReapInterval
	if interval <= 0 {
		interval = tuning.AgentOrphanReapInterval()
	}
	if interval <= 0 {
		return AgentReapReport{}
	}
	now := s.now()
	if !s.lastAgentReap.IsZero() && now.Sub(s.lastAgentReap) < interval {
		return AgentReapReport{}
	}
	s.lastAgentReap = now
	reap := s.cfg.AgentOrphanReap
	if reap == nil {
		reap = agentsession.ReapOrphaned
	}
	results, err := reap(ctx, agentsession.ReapRequest{
		Home: s.cfg.HomeDir, DBPath: s.cfg.DBPath, Initiator: "runtime-supervisor",
		Policy: agentsession.ReapPolicy{OlderThan: agentsession.DefaultReapOlderThan},
	})
	report := AgentReapReport{Ran: true}
	if err != nil {
		s.logf("orphaned agent session reap degraded: %v", err)
		return report
	}
	for _, result := range results {
		switch result.Outcome {
		case agentsession.OutcomeStopped:
			report.Stopped++
		case agentsession.OutcomeKept, agentsession.OutcomeOwnerReturns:
		default:
			report.Failed++
			s.logf("orphaned agent session %s not stopped: %s %s", result.Candidate.Session.ScopeUnit(), result.Outcome, result.Error)
		}
	}
	if report.Stopped > 0 {
		s.logf("orphaned agent session reap stopped %d session(s)", report.Stopped)
	}
	return report
}

// boundLogs keeps scenario and resource logs within process.DefaultLogFileLimit.
// Detached processes write directly to their log files, so the supervisor owns
// retention for both the runtime-home and platform resource-storage roots.
func (s *Service) boundLogs() LogBoundReport {
	now := s.now()
	if !s.lastLogBound.IsZero() && now.Sub(s.lastLogBound) < logBoundInterval {
		return LogBoundReport{}
	}
	s.lastLogBound = now
	root, err := process.LogsRoot(s.cfg.HomeDir)
	if err != nil {
		s.logf("log bound skipped: %v", err)
		return LogBoundReport{}
	}
	roots := []string{root}
	resolver, err := runtimestorage.NewResolver(runtimestorage.ResolverConfig{
		AppID: "vrooli",
		UserHomeDir: func() (string, error) {
			if strings.TrimSpace(s.cfg.HomeDir) != "" {
				return s.cfg.HomeDir, nil
			}
			return os.UserHomeDir()
		},
	})
	if err != nil {
		s.logf("resource log root resolution degraded: %v", err)
	} else if resourceRoot, rootErr := resolver.ResourcesRoot(runtimestorage.ClassLogs, ""); rootErr != nil {
		s.logf("resource log root resolution degraded: %v", rootErr)
	} else {
		roots = append(roots, resourceRoot)
	}
	seen := make(map[string]struct{}, len(roots))
	var report LogBoundReport
	for _, candidate := range roots {
		candidate = filepath.Clean(candidate)
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		result, boundErr := process.BoundLogs(candidate, process.DefaultLogFileLimit, process.DefaultLogTailKeep)
		if boundErr != nil {
			s.logf("log bound degraded for %s: %v", candidate, boundErr)
			continue
		}
		report.Ran = true
		report.Trimmed += len(result.Trimmed)
		report.ReclaimedBytes += result.ReclaimedBytes
		for _, problem := range result.Errors {
			s.logf("log bound could not trim %s", problem)
		}
		if len(result.Trimmed) > 0 {
			s.logf("log bound trimmed %d file(s) under %s, reclaimed %d bytes: %s", len(result.Trimmed), candidate, result.ReclaimedBytes, strings.Join(result.Trimmed, ", "))
		}
	}
	return report
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
	return scenarioruntime.HealthProbeDue(snapshot, now, interval)
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
			BuildIdentity:                 scenarioruntime.HealthStatusUnknown,
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
		BuildIdentity:                 supervisorSessionBuildIdentity(session),
		RecoveryPolicies:              policies,
		PressureEpochs:                epochs,
		LastTick:                      s.lastReport,
	}, nil
}

func supervisorBuildIdentity(version string) string {
	// Version is the identity supplied by the build/runtime launcher. Keep a
	// stable non-empty value even for development binaries that do not embed a
	// source fingerprint yet.
	if strings.TrimSpace(version) != "" {
		return strings.TrimSpace(version)
	}
	return "unknown"
}

func (s *Service) buildIdentity() string {
	identity := strings.TrimSpace(s.cfg.BuildIdentity)
	if identity != "" && identity != scenarioruntime.HealthStatusUnknown {
		return identity
	}
	return supervisorBuildIdentity(s.cfg.Version)
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
	identity, err := supervisorSessionIdentity(session)
	if err != nil {
		return StatusStale, err.Error()
	}
	if identity.ProcessIncarnation != "" {
		if session.PID == nil || *session.PID <= 0 {
			return StatusStale, "owner incarnation has no process"
		}
		actual, err := process.Incarnation(*session.PID)
		if err != nil || actual != identity.ProcessIncarnation {
			return StatusStale, "supervisor owner process incarnation is no longer current"
		}
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
	s.reapDeadSupervisorSessions(ctx)
	sessions, err := s.store.ListSupervisorSessions(ctx, scenarioruntime.SupervisorSessionFilter{Statuses: []string{scenarioruntime.SupervisorStatusRunning}})
	if err != nil {
		return fmt.Errorf("inspect running supervisor sessions: %w", err)
	}
	for _, peer := range sessions {
		if peer.PID == nil || *peer.PID <= 0 || !s.pidRunning(*peer.PID) {
			// A running row with a dead owner blocks the claim for as long as
			// its lease lasts (up to 45 s), and ClaimSupervisorSession refuses
			// any running row regardless of liveness. The reaper above honours
			// the lease on purpose (a peer mid-heartbeat must not be reaped by
			// a stranger), but this process is not a stranger: it is the next
			// supervisor, and a dead PID is proof enough here.
			if _, err := s.store.StopSupervisorSession(ctx, peer.SupervisorID, scenarioruntime.SupervisorStatusFailed, "predecessor pid dead at successor start"); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
				return fmt.Errorf("retire dead predecessor supervisor %s: %w", peer.SupervisorID, err)
			}
			s.logf("retired predecessor supervisor %s: owner pid dead", peer.SupervisorID)
			continue
		}
		if !s.cfg.Takeover {
			return fmt.Errorf("%w: PID %d is live; stop it with `kill -TERM %d` or rerun with `vrooli runtime supervisor run --takeover`", scenarioruntime.ErrSupervisorAlreadyRunning, *peer.PID, *peer.PID)
		}
		if err := s.stopPID(*peer.PID); err != nil {
			return fmt.Errorf("take over runtime supervisor PID %d: %w", *peer.PID, err)
		}
		if _, err := s.store.StopSupervisorSession(ctx, peer.SupervisorID, scenarioruntime.SupervisorStatusStopping, "operator takeover"); err != nil {
			return fmt.Errorf("mark predecessor supervisor %s stopping: %w", peer.SupervisorID, err)
		}
	}
	pid := os.Getpid()
	session, err := s.store.ClaimSupervisorSession(ctx, scenarioruntime.SupervisorSession{
		SupervisorID:  s.cfg.SupervisorID,
		HostBootID:    s.host.BootID,
		HostSessionID: s.host.SessionID,
		PID:           &pid,
		Version:       s.cfg.Version,
		MetadataJSON:  s.supervisorOwnerMetadata(pid),
	}, normalizeLeaseTTL(s.cfg.LeaseTTL))
	if err != nil {
		return err
	}
	s.session = session
	return nil
}

func (s *Service) stopPID(pid int) error {
	if s.cfg.StopPIDFunc != nil {
		return s.cfg.StopPIDFunc(pid)
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return proc.Signal(syscall.SIGTERM)
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
