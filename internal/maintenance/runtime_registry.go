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
	ExpireStaleSupervisorSessions(ctx context.Context, at time.Time, guard scenarioruntime.StartingLeaseGuard) ([]scenarioruntime.SupervisorSession, error)
	ListExpiredActivePortClaims(ctx context.Context, at time.Time) ([]scenarioruntime.PortClaim, error)
	PruneTerminalPortClaims(ctx context.Context, before time.Time) (int, error)
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

//nolint:gocyclo // runtime claim listing reconciles store, listener, ownership, and stale-state branches.
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

// PortReclaimCandidate names a declared port still held by a live process after
// its owning instance was expired as non-authoritative. The caller decides
// whether to evict the holder — and only ever does so once it has confirmed the
// PID is a stale Vrooli orphan from this install, never a foreign service.
type PortReclaimCandidate struct {
	Scenario string
	Port     int
	PID      int
}

func expireNonAuthoritativeRegistryState(ctx context.Context, store runtimeMaintenanceStore) ([]control.ResultItem, []PortReclaimCandidate, error) {
	host, err := hostsession.DefaultProvider{}.Current(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return nil, nil, err
	}
	// Gather all store state FIRST, then capture listener evidence: this path
	// EXPIRES claims on known-absent listeners, so evidence captured before a
	// claim was read could wrongly expire a port bound in between. One
	// statuses-filtered claim query grouped in memory replaces the per-instance
	// N+1 (PortClaimFilter only takes a single InstanceID).
	activeClaims, err := store.ListPortClaims(ctx, scenarioruntime.PortClaimFilter{
		Statuses: scenarioruntime.ActivePortClaimStatuses(),
	})
	if err != nil {
		return nil, nil, err
	}
	cleanup := registryCleanup{ctx: ctx, store: store, pidRunning: newPIDLivenessMemo(processIsRunning)}
	cleanup.claimsByInstance = make(map[string][]scenarioruntime.PortClaim, len(instances))
	for _, claim := range activeClaims {
		cleanup.claimsByInstance[claim.InstanceID] = append(cleanup.claimsByInstance[claim.InstanceID], claim)
	}
	instanceIDs := make([]string, 0, len(instances))
	for _, instance := range instances {
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}
	cleanup.refsByInstance, err = store.ListProcessRefsForInstances(ctx, instanceIDs)
	if err != nil {
		return nil, nil, err
	}
	cleanup.listenerSnapshot = captureListenerSnapshotFn()
	for _, instance := range instances {
		if err := cleanup.expireInstance(host.BootID, instance); err != nil {
			return nil, nil, err
		}
	}

	// A claim can still be reserved/bound while its instance has already left
	// the active statuses (unclean stop, crash, reaped lease). Such claims are
	// invisible to the active-instance walk above, so without this pass they
	// accumulate forever and no cleanup can ever expire them.
	if err := cleanup.expireOrphanClaims(instances, activeClaims); err != nil {
		return nil, nil, err
	}
	return cleanup.stopped, cleanup.reclaim, nil
}

type registryCleanup struct {
	ctx              context.Context
	store            runtimeMaintenanceStore
	listenerSnapshot network.TCPListenerSnapshot
	pidRunning       func(int) bool
	claimsByInstance map[string][]scenarioruntime.PortClaim
	refsByInstance   map[string][]scenarioruntime.ProcessRef
	stopped          []control.ResultItem
	reclaim          []PortReclaimCandidate
}

func (c *registryCleanup) expireInstance(bootID string, instance scenarioruntime.Instance) error {
	claims := c.claimsByInstance[instance.InstanceID]
	refs := c.refsByInstance[instance.InstanceID]
	reconciled := scenarioruntime.ReconcileRuntime(scenarioruntime.ReconcileInput{Now: time.Now().UTC(), CurrentBootID: bootID, Instance: instance, Claims: claims, ProcessRefs: refs, Processes: scenarioruntime.ProcessEvidenceFromRefs(refs, c.pidRunning), Listeners: runtimeListenerEvidence(c.listenerSnapshot, claims, refs)})
	if reconciled.Classification == scenarioruntime.ReconcileUnverified {
		return nil
	}
	if reconciled.Authoritative {
		return c.expireNonAuthoritativeClaims(reconciled.Claims)
	}
	if _, err := c.store.ExpireInstance(c.ctx, instance.InstanceID, reconciled.Reason); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
		return err
	}
	c.stopped = append(c.stopped, control.Stopped(instance.Scenario+"/"+instance.InstanceID, "Expired non-authoritative registry instance: "+reconciled.Reason))
	for _, claim := range claims {
		if err := c.expireInstanceClaim(instance.Scenario, claim); err != nil {
			return err
		}
	}
	return nil
}

func (c *registryCleanup) expireNonAuthoritativeClaims(claims []scenarioruntime.ReconciledClaim) error {
	for _, claim := range claims {
		if claim.Authoritative || claim.Claim.Status != scenarioruntime.ClaimStatusBound {
			continue
		}
		if _, err := c.store.ExpirePortClaim(c.ctx, claim.Claim.ClaimID); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
			return err
		}
		c.stopped = append(c.stopped, control.Stopped(strconv.Itoa(claim.Claim.Port), "Expired non-authoritative registry claim "+claim.Claim.ClaimID))
	}
	return nil
}

func (c *registryCleanup) expireInstanceClaim(scenarioName string, claim scenarioruntime.PortClaim) error {
	if _, err := c.store.ExpirePortClaim(c.ctx, claim.ClaimID); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
		return err
	}
	c.stopped = append(c.stopped, control.Stopped(strconv.Itoa(claim.Port), "Expired non-authoritative registry claim "+claim.ClaimID))
	if claim.Port > 0 {
		for _, listener := range c.listenerSnapshot.Listening(claim.Port).Listeners {
			if listener.PID > 0 {
				c.reclaim = append(c.reclaim, PortReclaimCandidate{Scenario: scenarioName, Port: claim.Port, PID: listener.PID})
			}
		}
	}
	return nil
}

func (c *registryCleanup) expireOrphanClaims(instances []scenarioruntime.Instance, claims []scenarioruntime.PortClaim) error {
	activeIDs := make(map[string]struct{}, len(instances))
	for _, instance := range instances {
		activeIDs[instance.InstanceID] = struct{}{}
	}
	orphanIDs := orphanInstanceIDs(activeIDs, claims)
	if len(orphanIDs) == 0 {
		return nil
	}
	current, err := c.store.GetInstances(c.ctx, orphanIDs)
	if err != nil {
		return err
	}
	for _, claim := range claims {
		if !claimIsOrphaned(activeIDs, current, claim) {
			continue
		}
		if _, err := c.store.ExpirePortClaim(c.ctx, claim.ClaimID); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
			return err
		}
		c.stopped = append(c.stopped, control.Stopped(strconv.Itoa(claim.Port), "Expired orphaned registry claim "+claim.ClaimID+" (instance no longer active)"))
	}
	return nil
}

func orphanInstanceIDs(active map[string]struct{}, claims []scenarioruntime.PortClaim) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, claim := range claims {
		if _, ok := active[claim.InstanceID]; ok {
			continue
		}
		if _, ok := seen[claim.InstanceID]; !ok {
			seen[claim.InstanceID] = struct{}{}
			ids = append(ids, claim.InstanceID)
		}
	}
	return ids
}

func claimIsOrphaned(active map[string]struct{}, current map[string]scenarioruntime.Instance, claim scenarioruntime.PortClaim) bool {
	if _, ok := active[claim.InstanceID]; ok {
		return false
	}
	instance, ok := current[claim.InstanceID]
	if !ok {
		return true
	}
	return instance.Status == scenarioruntime.StatusStopped || instance.Status == scenarioruntime.StatusFailed || instance.Status == scenarioruntime.StatusExpired
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

	ctx := context.Background()
	instances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return nil, err
	}
	instanceIDs := make([]string, 0, len(instances))
	for _, instance := range instances {
		instanceIDs = append(instanceIDs, instance.InstanceID)
	}
	// One batched query instead of one ListProcessRefs round trip per instance.
	refsByInstance, err := store.ListProcessRefsForInstances(ctx, instanceIDs)
	if err != nil {
		return nil, err
	}
	out := make([]trackedProcessRef, 0)
	for _, instance := range instances {
		for _, ref := range refsByInstance[instance.InstanceID] {
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
