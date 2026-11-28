package runtimesupervisor

import (
	"context"
	"errors"
	"fmt"

	"github.com/vrooli/vrooli/internal/maintenance"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

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
	// Demand retention is a control-plane concern. Run it through maintenance
	// so process termination keeps the same exact-identity safeguards as the
	// operator cleanup path; a transient demand sweep failure must not cost the
	// supervisor its fleet heartbeat.
	demandStopped, demandFailures := 0, 0
	if demandReport, demandErr := (maintenance.NewControllerWithDBPath("", s.cfg.HomeDir, s.cfg.DBPath)).ReconcileDemandLeasesAt(ctx, s.now()); demandErr != nil {
		s.logf("demand lease reconciliation degraded this tick: %v", demandErr)
	} else if len(demandReport.Stopped) > 0 || len(demandReport.Failed) > 0 {
		for _, item := range demandReport.Stopped {
			if item.Name != "demand-leases" {
				demandStopped++
			}
		}
		demandFailures = len(demandReport.Failed)
		s.logf("demand lease reconciliation stopped=%d failed=%d", demandStopped, demandFailures)
	}
	// Capacity residency also needs an always-on owner. Keeping this beside
	// demand reconciliation makes declared idle-unload TTLs effective without
	// waiting for an unrelated cleanup/start operation. Failures are isolated
	// from the supervisor heartbeat just like demand-reconciliation failures.
	if capacityReport, capacityErr := (maintenance.NewController(s.cfg.RootDir, s.cfg.HomeDir)).ReconcileCapacity(ctx); capacityErr != nil {
		s.logf("capacity residency reconciliation degraded this tick: %v", capacityErr)
	} else if len(capacityReport.Stopped) > 0 || len(capacityReport.Failed) > 0 {
		s.logf("capacity residency stopped=%d failed=%d", len(capacityReport.Stopped), len(capacityReport.Failed))
	}
	// Editor-session expiry uses the same deadline-plus-proof-of-death rule as
	// operator cleanup. This retires only bookkeeping for owners proven gone;
	// it never signals or stops an editor process. Keep this in the supervisor
	// tick so stale rows do not depend on an operator invoking cleanup.
	expiredEditors, editorErr := s.store.ExpireStaleEditorLeases(ctx, s.now(), scenarioruntime.StartingLeaseGuard{
		CurrentBootID: s.host.BootID,
		PIDRunning:    s.pidRunning,
	})
	if editorErr != nil {
		s.logf("editor lease reconciliation degraded this tick: %v", editorErr)
	} else if len(expiredEditors) > 0 {
		s.logf("retired %d editor lease(s) whose process is gone", len(expiredEditors))
	}

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
	observations := map[string]scenarioruntime.ListenerObservation{}
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
			observations[claim.ClaimID] = scenarioruntime.ListenerObservationFromEvidence(s.now(), evidence)
		}
	}
	if len(observations) > 0 {
		// One write per tick. Evidence is an observation, not authority: a
		// claim released concurrently is the common cause of a miss and is not
		// a tick failure, so the batch reports once with the claim count.
		if _, err := s.store.UpdatePortClaimListenerEvidenceBatch(ctx, observations); err != nil {
			s.logf("could not record listener evidence for %d claims: %v", len(observations), err)
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
	report := TickReport{SupervisorID: s.session.SupervisorID, Unverified: len(plan.Unverified), Expired: len(plan.Expire), HealthProbeCount: len(plan.HealthProbes), DemandStopped: demandStopped, DemandFailures: demandFailures}
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
	report.LogBound = s.boundLogs()
	report.AgentReap = s.reapOrphanedAgentSessions(ctx)
	s.lastReport = report
	return report, nil
}
