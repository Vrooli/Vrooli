package maintenance

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type trackedProcessRef struct {
	PID  int
	PGID int
}

type runtimeMaintenanceStore interface {
	scenarioruntime.QueryRepository
	scenarioruntime.CleanupRepository
	scenarioruntime.ProcessRefRepository
	scenarioruntime.EventRepository
	ListSupervisorSessions(ctx context.Context, filter scenarioruntime.SupervisorSessionFilter) ([]scenarioruntime.SupervisorSession, error)
	ListExpiredActivePortClaims(ctx context.Context, at time.Time) ([]scenarioruntime.PortClaim, error)
	ReleaseActivePortClaimsForInstance(ctx context.Context, instanceID string) ([]scenarioruntime.PortClaim, error)
	StopLease(ctx context.Context, instanceID string, generation int64, reason string) (scenarioruntime.Instance, error)
	GetInstances(ctx context.Context, instanceIDs []string) (map[string]scenarioruntime.Instance, error)
	ListProcessRefsForInstances(ctx context.Context, instanceIDs []string) (map[string][]scenarioruntime.ProcessRef, error)
	GetHealthSnapshots(ctx context.Context, instanceIDs []string) (map[string]scenarioruntime.HealthSnapshot, error)
}

func openRuntimeRegistryIfPresent(home string) (runtimeMaintenanceStore, func(), error) {
	path, err := scenarioruntime.DefaultDBPath(home)
	if err != nil {
		return nil, nil, err
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, func() {}, nil
		}
		return nil, nil, err
	}
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home, DBPath: path})
	if err != nil {
		return nil, nil, err
	}
	return store, func() { _ = store.Close() }, nil
}

func listRuntimeClaims(ctx context.Context, store runtimeMaintenanceStore, port int, scenarioName string, hostListenerInUse bool) ([]RuntimeClaimInfo, error) {
	claims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		Scenario: scenarioName,
		Statuses: append(scenarioruntime.ActivePortClaimStatuses(), scenarioruntime.ClaimStatusExpired),
	})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	host, _ := hostsession.DefaultProvider{}.Current(ctx, "")
	supervisors, err := store.ListSupervisorSessions(ctx, scenarioruntime.SupervisorSessionFilter{Statuses: []string{scenarioruntime.SupervisorStatusRunning}})
	if err != nil {
		return nil, err
	}
	supervisorsByID := supervisorSessionsByID(supervisors)
	pidRunning := newPIDLivenessMemo(processIsRunning)

	// Batch the per-claim instance/ref/health lookups: 3 queries instead of
	// 3×N sequential round trips.
	instanceIDs := make([]string, 0, len(claims))
	for _, claim := range claims {
		if port > 0 && claim.Port != port {
			continue
		}
		instanceIDs = append(instanceIDs, claim.InstanceID)
	}
	instancesByID, err := store.GetInstances(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}
	refsByInstance, err := store.ListProcessRefsForInstances(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}
	healthByInstance, err := store.GetHealthSnapshots(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}

	// Capture listener evidence ONCE, after the claim set is read: evidence
	// must be at least as fresh as the claims, or a port bound between
	// capture and query would read as not-listening.
	listenerSnapshot := captureListenerSnapshotFn()
	out := make([]RuntimeClaimInfo, 0, len(claims))
	for _, claim := range claims {
		if port > 0 && claim.Port != port {
			continue
		}
		item := RuntimeClaimInfo{
			ClaimID:                   claim.ClaimID,
			InstanceID:                claim.InstanceID,
			Scenario:                  claim.Scenario,
			PortName:                  claim.PortName,
			EnvVar:                    claim.EnvVar,
			Port:                      claim.Port,
			BindHost:                  claim.BindHost,
			URL:                       claim.URL,
			ClaimStatus:               claim.Status,
			CreatedAt:                 claim.CreatedAt,
			UpdatedAt:                 claim.UpdatedAt,
			ExpiresAt:                 claim.ExpiresAt,
			LastBoundAt:               claim.LastBoundAt,
			LastListenerCheckAt:       claim.LastListenerCheckAt,
			LastListenerSeenAt:        claim.LastListenerSeenAt,
			FirstUnboundAt:            claim.FirstUnboundAt,
			ConsecutiveListenerMisses: claim.ConsecutiveListenerMisses,
			ListenerStatus:            claim.ListenerStatus,
			ListenerPID:               claim.ListenerPID,
			ListenerProcessLabel:      claim.ListenerProcessLabel,
		}
		var runtimeInstance scenarioruntime.Instance
		var hasRuntimeInstance bool
		instance, hasInstance := instancesByID[claim.InstanceID]
		if hasInstance {
			runtimeInstance = instance
			hasRuntimeInstance = true
			item.Generation = instance.Generation
			item.InstanceStatus = instance.Status
			item.SupervisorID = instance.SupervisorID
			item.HeartbeatDeadline = instance.HeartbeatDeadlineAt
			if instance.HeartbeatDeadlineAt != nil {
				fresh := instance.HeartbeatDeadlineAt.After(now)
				item.LeaseFresh = &fresh
			}
			if instance.SupervisorID != "" {
				if supervisor, ok := supervisorsByID[instance.SupervisorID]; ok {
					item.SupervisorStatus = supervisor.Status
					item.SupervisorDeadline = &supervisor.HeartbeatDeadlineAt
					fresh := supervisor.HeartbeatDeadlineAt.After(now)
					item.SupervisorFresh = &fresh
				} else {
					fresh := false
					item.SupervisorFresh = &fresh
				}
			}
			refs := refsByInstance[claim.InstanceID]
			reconciled := scenarioruntime.ReconcileRuntime(scenarioruntime.ReconcileInput{
				Now:           now,
				CurrentBootID: host.BootID,
				Instance:      instance,
				Claims:        []scenarioruntime.PortClaim{claim},
				ProcessRefs:   refs,
				Processes:     scenarioruntime.ProcessEvidenceFromRefs(refs, pidRunning),
				Listeners:     runtimeListenerEvidence(listenerSnapshot, []scenarioruntime.PortClaim{claim}, refs),
			})
			item.Reconciliation = reconciled.Classification
			item.ReconcileReason = reconciled.Reason
			item.Authoritative = &reconciled.Authoritative
			if len(reconciled.Claims) > 0 {
				claimResult := reconciled.Claims[0]
				item.Reconciliation = claimResult.Classification
				item.ReconcileReason = claimResult.Reason
				item.Authoritative = &claimResult.Authoritative
			}
		}
		health, hasHealth := healthByInstance[claim.InstanceID]
		if hasHealth {
			item.HealthStatus = health.Status
			item.HealthReady = health.Readiness
		}
		if hasRuntimeInstance {
			authoritative := false
			hasAuthoritative := false
			if item.Authoritative != nil {
				authoritative = *item.Authoritative
				hasAuthoritative = true
			}
			recommendation := scenarioruntime.ClassifyPortEvidence(scenarioruntime.PortEvidenceInput{
				Claim:             claim,
				Instance:          runtimeInstance,
				Health:            health,
				Reconciliation:    item.Reconciliation,
				Authoritative:     authoritative,
				HasAuthoritative:  hasAuthoritative,
				HostListenerInUse: hostListenerInUse && port > 0 && claim.Port == port,
			})
			item.RecommendationCode = recommendation.Code
			item.RecommendationConfidence = recommendation.Confidence
			item.RecommendationRationale = recommendation.Rationale
		}
		out = append(out, item)
	}
	return out, nil
}

func supervisorSessionsByID(sessions []scenarioruntime.SupervisorSession) map[string]scenarioruntime.SupervisorSession {
	out := make(map[string]scenarioruntime.SupervisorSession, len(sessions))
	for _, session := range sessions {
		if session.SupervisorID == "" {
			continue
		}
		current, ok := out[session.SupervisorID]
		if !ok || session.StartedAt.After(current.StartedAt) {
			out[session.SupervisorID] = session
		}
	}
	return out
}

func expireNonAuthoritativeRegistryState(ctx context.Context, store runtimeMaintenanceStore) ([]control.ResultItem, error) {
	host, err := hostsession.DefaultProvider{}.Current(ctx, "")
	if err != nil {
		return nil, err
	}
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return nil, err
	}
	stopped := make([]control.ResultItem, 0)
	pidRunning := newPIDLivenessMemo(processIsRunning)
	// Gather all store state FIRST, then capture listener evidence: this path
	// EXPIRES claims on known-absent listeners, so evidence captured before a
	// claim was read could wrongly expire a port bound in between. One
	// statuses-filtered claim query grouped in memory replaces the per-instance
	// N+1 (PortClaimFilter only takes a single InstanceID).
	activeClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		Statuses: scenarioruntime.ActivePortClaimStatuses(),
	})
	if err != nil {
		return nil, err
	}
	claimsByInstance := make(map[string][]scenarioruntime.PortClaim, len(instances))
	for _, claim := range activeClaims {
		claimsByInstance[claim.InstanceID] = append(claimsByInstance[claim.InstanceID], claim)
	}
	instanceIDs := make([]string, 0, len(instances))
	for _, instance := range instances {
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}
	refsByInstance, err := store.ListProcessRefsForInstances(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}
	listenerSnapshot := captureListenerSnapshotFn()
	for _, instance := range instances {
		claims := claimsByInstance[instance.InstanceID]
		refs := refsByInstance[instance.InstanceID]
		reconciled := scenarioruntime.ReconcileRuntime(scenarioruntime.ReconcileInput{
			Now:           time.Now().UTC(),
			CurrentBootID: host.BootID,
			Instance:      instance,
			Claims:        claims,
			ProcessRefs:   refs,
			Processes:     scenarioruntime.ProcessEvidenceFromRefs(refs, pidRunning),
			Listeners:     runtimeListenerEvidence(listenerSnapshot, claims, refs),
		})
		if reconciled.Classification == scenarioruntime.ReconcileUnverified {
			continue
		}
		if reconciled.Authoritative {
			for _, claim := range reconciled.Claims {
				if claim.Authoritative || claim.Claim.Status != scenarioruntime.ClaimStatusBound {
					continue
				}
				if _, err := store.ExpirePortClaim(ctx, claim.Claim.ClaimID); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
					return nil, err
				}
				stopped = append(stopped, control.Stopped(strconv.Itoa(claim.Claim.Port), "Expired non-authoritative registry claim "+claim.Claim.ClaimID))
			}
			continue
		}
		if _, err := store.ExpireInstance(ctx, instance.InstanceID, reconciled.Reason); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
			return nil, err
		}
		stopped = append(stopped, control.Stopped(instance.Scenario+"/"+instance.InstanceID, "Expired non-authoritative registry instance: "+reconciled.Reason))
		for _, claim := range claims {
			if _, err := store.ExpirePortClaim(ctx, claim.ClaimID); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
				return nil, err
			}
			stopped = append(stopped, control.Stopped(strconv.Itoa(claim.Port), "Expired non-authoritative registry claim "+claim.ClaimID))
		}
	}
	return stopped, nil
}

// finalizeStuckStoppingInstances finalizes runtime_instances rows that got
// stuck in status='stopping' because the lifecycle runner died between
// marking the instance stopping and finishing the stop. This is the reaper
// that enforces invariant I1: a lifecycle stop must never leave port claims,
// process_refs, or instance state unfinalized.
//
// An instance is considered stuck when any of the following hold:
//   - host_boot_id differs from the current boot (machine rebooted),
//   - owner_pid is unset or no longer points at a live process,
//   - heartbeat_deadline_at is in the past.
//
// For each stuck instance the reaper releases its active port claims, marks
// its process_refs exited, transitions the instance to stopped with
// stop_reason=reaper-finalize, and emits an instance_reaped event for
// forensics.
func finalizeStuckStoppingInstances(ctx context.Context, store runtimeMaintenanceStore) ([]control.ResultItem, error) {
	host, err := hostsession.DefaultProvider{}.Current(ctx, "")
	if err != nil {
		return nil, err
	}
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: []string{scenarioruntime.StatusStopping}})
	if err != nil {
		return nil, err
	}
	stopped := make([]control.ResultItem, 0)
	now := time.Now().UTC()
	pidRunning := newPIDLivenessMemo(processIsRunning)
	for _, instance := range instances {
		trigger, ok := stuckStoppingTrigger(instance, host.BootID, now, pidRunning)
		if !ok {
			continue
		}
		if err := scenarioruntime.FinalizeStuckInstance(ctx, store, instance, trigger, now); err != nil {
			return nil, err
		}
		stopped = append(stopped, control.Stopped(instance.Scenario+"/"+instance.InstanceID, "Finalized stuck-stopping runtime instance ("+trigger+")"))
	}
	return stopped, nil
}

// stuckStoppingTrigger returns a short label for the condition that makes a
// stopping instance eligible for the reaper, or ("", false) if the instance
// still looks alive.
func stuckStoppingTrigger(instance scenarioruntime.Instance, currentBootID string, now time.Time, pidRunning func(int) bool) (string, bool) {
	if instance.Status != scenarioruntime.StatusStopping {
		return "", false
	}
	if currentBootID != "" && instance.HostBootID != "" && instance.HostBootID != currentBootID {
		return "boot_id_mismatch", true
	}
	if instance.OwnerPID == nil || *instance.OwnerPID <= 0 {
		return "owner_pid_missing", true
	}
	if !pidRunning(*instance.OwnerPID) {
		return "owner_pid_dead", true
	}
	if instance.HeartbeatDeadlineAt != nil && !instance.HeartbeatDeadlineAt.After(now) {
		return "heartbeat_expired", true
	}
	return "", false
}

func runtimeListenerEvidence(snapshot network.TCPListenerSnapshot, claims []scenarioruntime.PortClaim, refs []scenarioruntime.ProcessRef) map[int]scenarioruntime.ListenerEvidence {
	return scenarioruntime.ListenerEvidenceFromClaims(claims, refs, func(port int) scenarioruntime.ListenerEvidence {
		state := snapshot.Listening(port)
		if !state.Known {
			return scenarioruntime.ListenerEvidence{Known: false}
		}
		return scenarioruntime.ListenerEvidence{Known: true, Listening: state.Listening}
	})
}

func listRuntimeProcessRefs(ctx context.Context, store runtimeMaintenanceStore, claims []RuntimeClaimInfo) ([]RuntimeProcessRefInfo, error) {
	seen := make(map[string]struct{})
	pidRunning := newPIDLivenessMemo(processIsRunning)
	instanceIDs := make([]string, 0, len(claims))
	for _, claim := range claims {
		instanceIDs = append(instanceIDs, claim.InstanceID)
	}
	refsByInstance, err := store.ListProcessRefsForInstances(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}
	out := make([]RuntimeProcessRefInfo, 0)
	for _, claim := range claims {
		if _, ok := seen[claim.InstanceID]; ok {
			continue
		}
		seen[claim.InstanceID] = struct{}{}
		for _, ref := range refsByInstance[claim.InstanceID] {
			item := RuntimeProcessRefInfo{
				RefID:          ref.RefID,
				InstanceID:     ref.InstanceID,
				Scenario:       claim.Scenario,
				InstanceStatus: claim.InstanceStatus,
				PID:            ref.PID,
				PGID:           ref.PGID,
				ProcessID:      ref.ProcessID,
				Step:           ref.Step,
				Command:        ref.Command,
				Status:         ref.Status,
			}
			if ref.PID != nil {
				running := pidRunning(*ref.PID)
				item.PIDRunning = &running
			}
			out = append(out, item)
		}
	}
	return out, nil
}

func runtimeTrackedProcessRefs(home string) ([]trackedProcessRef, error) {
	store, closeStore, err := openRuntimeRegistryFn(home)
	if err != nil || store == nil {
		return nil, err
	}
	defer closeStore()

	instances, err := store.ListInstances(context.Background(), scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return nil, err
	}
	out := make([]trackedProcessRef, 0)
	for _, instance := range instances {
		refs, err := store.ListProcessRefs(context.Background(), instance.InstanceID)
		if err != nil {
			return nil, err
		}
		for _, ref := range refs {
			item := trackedProcessRef{}
			if ref.PID != nil {
				item.PID = *ref.PID
			}
			if ref.PGID != nil {
				item.PGID = *ref.PGID
			}
			if item.PID > 0 || item.PGID > 0 {
				out = append(out, item)
			}
		}
	}
	return out, nil
}

func processIsRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	if os.Getpid() == pid {
		return true
	}
	return pidIsRunningFn(pid)
}

// newPIDLivenessMemo wraps a liveness function with a per-run cache so each
// distinct PID is probed at most once per registry scan, regardless of how
// many claims or process refs reference it.
func newPIDLivenessMemo(isRunning func(int) bool) func(int) bool {
	cache := make(map[int]bool)
	return func(pid int) bool {
		if alive, ok := cache[pid]; ok {
			return alive
		}
		alive := isRunning(pid)
		cache[pid] = alive
		return alive
	}
}
