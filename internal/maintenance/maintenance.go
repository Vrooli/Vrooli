package maintenance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/portspec"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

const (
	maintenanceParameterA = 5
)

type Controller struct {
	Root   string
	Home   string
	DBPath string
}

const (
	runtimeInstanceEnv = "VROOLI_RUNTIME_INSTANCE_ID"
	runtimeScenarioEnv = "VROOLI_SCENARIO"
	runtimeVariantEnv  = "VROOLI_VARIANT"
)

type SystemProcess struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Command string `json:"command"`
}

type PortListener = network.PortListener

type RuntimeClaimInfo struct {
	ClaimID                   string                                  `json:"claim_id"`
	InstanceID                string                                  `json:"instance_id"`
	Scenario                  string                                  `json:"scenario"`
	Generation                int64                                   `json:"generation,omitempty"`
	PortName                  string                                  `json:"port_name"`
	EnvVar                    string                                  `json:"env_var,omitempty"`
	Port                      int                                     `json:"port"`
	BindHost                  string                                  `json:"bind_host"`
	URL                       string                                  `json:"url,omitempty"`
	ClaimStatus               string                                  `json:"claim_status"`
	InstanceStatus            string                                  `json:"instance_status,omitempty"`
	SupervisorID              string                                  `json:"supervisor_id,omitempty"`
	SupervisorStatus          string                                  `json:"supervisor_status,omitempty"`
	SupervisorFresh           *bool                                   `json:"supervisor_fresh,omitempty"`
	LeaseFresh                *bool                                   `json:"lease_fresh,omitempty"`
	HeartbeatDeadline         *time.Time                              `json:"heartbeat_deadline,omitempty"`
	SupervisorDeadline        *time.Time                              `json:"supervisor_heartbeat_deadline,omitempty"`
	HealthStatus              string                                  `json:"health_status,omitempty"`
	HealthReady               *bool                                   `json:"health_ready,omitempty"`
	Reconciliation            scenarioruntime.ReconcileClassification `json:"reconciliation,omitempty"`
	ReconcileReason           string                                  `json:"reconcile_reason,omitempty"`
	Authoritative             *bool                                   `json:"authoritative,omitempty"`
	CreatedAt                 time.Time                               `json:"created_at"`
	UpdatedAt                 time.Time                               `json:"updated_at"`
	ExpiresAt                 *time.Time                              `json:"expires_at,omitempty"`
	LastBoundAt               *time.Time                              `json:"last_bound_at,omitempty"`
	LastListenerCheckAt       *time.Time                              `json:"last_listener_check_at,omitempty"`
	LastListenerSeenAt        *time.Time                              `json:"last_listener_seen_at,omitempty"`
	FirstUnboundAt            *time.Time                              `json:"first_unbound_at,omitempty"`
	ConsecutiveListenerMisses int                                     `json:"consecutive_listener_misses"`
	ListenerStatus            string                                  `json:"listener_status,omitempty"`
	ListenerPID               *int                                    `json:"listener_pid,omitempty"`
	ListenerProcessLabel      string                                  `json:"listener_process_label,omitempty"`
	RecommendationCode        string                                  `json:"recommendation_code,omitempty"`
	RecommendationConfidence  string                                  `json:"recommendation_confidence,omitempty"`
	RecommendationRationale   string                                  `json:"recommendation_rationale,omitempty"`
}

type RuntimeProcessRefInfo struct {
	RefID          string `json:"ref_id"`
	InstanceID     string `json:"instance_id"`
	Scenario       string `json:"scenario,omitempty"`
	InstanceStatus string `json:"instance_status,omitempty"`
	PID            *int   `json:"pid,omitempty"`
	PGID           *int   `json:"pgid,omitempty"`
	ProcessID      string `json:"process_id,omitempty"`
	Step           string `json:"step,omitempty"`
	Command        string `json:"command,omitempty"`
	Status         string `json:"status,omitempty"`
	PIDRunning     *bool  `json:"pid_running,omitempty"`
}

type PortDiagnostic struct {
	Port               int                        `json:"port"`
	Scenario           string                     `json:"scenario,omitempty"`
	InUse              bool                       `json:"in_use"`
	Listeners          []PortListener             `json:"listeners,omitempty"`
	ListenerInspection network.ListenerInspection `json:"listener_inspection"`
	RegistryClaims     []RuntimeClaimInfo         `json:"registry_claims,omitempty"`
	RegistryProcesses  []RuntimeProcessRefInfo    `json:"registry_processes,omitempty"`
	HostOrphanCount    int                        `json:"host_orphan_count"`
	Recommendations    []string                   `json:"recommendations,omitempty"`

	// PortPolicy surfaces the ephemeral-range policy check so operators can
	// tell at a glance whether the conflict is a real orphan listener or a
	// kernel source-port steal. Populated on every diagnose call.
	PortPolicy PortPolicyReport `json:"port_policy"`
}

// PortPolicyReport captures whether the port sits inside the OS's live
// ephemeral window and which canonical Vrooli band (if any) it belongs to.
type PortPolicyReport struct {
	EphemeralMin         int    `json:"ephemeral_min"`
	EphemeralMax         int    `json:"ephemeral_max"`
	EphemeralSource      string `json:"ephemeral_source"`
	InsideEphemeralRange bool   `json:"inside_ephemeral_range"`
	CanonicalBand        string `json:"canonical_band"` // "api", "ui", "ws", "headroom", "" if outside
	AboveCanonicalMax    bool   `json:"above_canonical_max"`
}

// terminalPortClaimRetention is how long expired/released registry claim rows
// are kept for forensics before `vrooli cleanup locks` deletes them.
var terminalPortClaimRetention = tuning.TerminalClaimRetention()

var (
	inspectPortListenersFn    = network.InspectPortListeners
	captureListenerSnapshotFn = network.CaptureTCPListenerSnapshot
	pidIsRunningFn            = process.IsPIDRunning
	killProcessFn             = killProcess
	looksLikeVrooliProcessFn  = looksLikeVrooliProcess
	runProtoGenerateFn        = runProtoGenerate
	openRuntimeRegistryFn     = openRuntimeRegistryIfPresent
)

func NewController(root, home string) *Controller {
	return &Controller{
		Root: filepath.Clean(root),
		Home: filepath.Clean(home),
	}
}

// NewControllerWithDBPath is used by long-lived control-plane services that
// already resolved a registry path (including test or sandbox databases). The
// ordinary constructor continues to derive the canonical path from Home.
func NewControllerWithDBPath(root, home, dbPath string) *Controller {
	c := NewController(root, home)
	if strings.TrimSpace(dbPath) != "" {
		c.DBPath = filepath.Clean(dbPath)
	}
	return c
}

func (c *Controller) ListRuntimeClaims() ([]RuntimeClaimInfo, error) {
	store, closeStore, err := openRuntimeRegistryFn(c.Home)
	if err != nil || store == nil {
		return nil, err
	}
	defer closeStore()
	return listRuntimeClaims(context.Background(), store, 0, "", false)
}

var errDemandStopBusy = errors.New("demand stop is owned by another controller")

// stopDemandManagedInstance tears down one instance that the registry has
// already atomically claimed for demand-based stopping. It performs a complete
// preflight over every live process reference before signaling any process,
// and requires the process to identify the exact instance, scenario, and
// variant. Foreign identity observed during preflight becomes an actionable
// failure. Stable handles opened before inspection prevent PID reuse from
// redirecting a later signal to a different process.
func (c *Controller) stopDemandManagedInstance(ctx context.Context, store runtimeMaintenanceStore, instance scenarioruntime.Instance) error {
	// Ownership has no expiry takeover: a paused controller must not overlap
	// another signaler. The kernel releases this lock when its process exits.
	lockCtx, cancelLock := context.WithTimeout(context.WithoutCancel(ctx), 250*time.Millisecond)
	defer cancelLock()
	unlock, err := store.AcquireDemandStopLock(lockCtx)
	if errors.Is(err, context.DeadlineExceeded) {
		return errDemandStopBusy
	}
	if err != nil {
		return fmt.Errorf("acquire demand stop ownership: %w", err)
	}
	defer unlock()
	started, err := store.DemandStopStarted(lockCtx, instance.InstanceID, instance.Generation)
	if err != nil {
		return err // Another owner may have completed this candidate already.
	}
	restore := func(cause error) error {
		if started {
			return cause // A partial teardown must remain unavailable to consumers.
		}
		// Cancellation must not strand an unstarted teardown in stopping.
		restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if _, restoreErr := store.RestoreDemandStopCandidate(restoreCtx, instance.InstanceID, instance.Generation, instance.Phase); restoreErr != nil {
			return errors.Join(cause, fmt.Errorf("restore demand candidate after failed preflight: %w", restoreErr))
		}
		return cause
	}
	refs, err := store.ListProcessRefs(ctx, instance.InstanceID)
	if err != nil {
		return restore(fmt.Errorf("list demand-managed process refs for %s: %w", instance.InstanceID, err))
	}
	host, err := (hostsession.DefaultProvider{}).Current(ctx, "")
	if err != nil {
		return restore(fmt.Errorf("resolve host session for demand-managed stop: %w", err))
	}
	variant := scenarioruntime.InstanceKey{Scenario: instance.Scenario, Variant: instance.Variant}.Normalize().Variant
	// A tracked launcher may have workers in its process group. Inspect every
	// member individually; a group-wide signal would also hit foreign members.
	groups := make(map[int]struct{})
	tracked := make(map[int]struct{})
	for _, ref := range refs {
		if ref.HostBootID != "" && ref.HostBootID != host.BootID {
			continue
		}
		if ref.PGID != nil && *ref.PGID > 0 {
			groups[*ref.PGID] = struct{}{}
		}
		if ref.Status == "running" && ref.PID != nil {
			tracked[*ref.PID] = struct{}{}
		}
	}
	if len(groups) > 0 {
		table, err := listProcessTableFn()
		if err != nil {
			return restore(fmt.Errorf("inspect demand process groups: %w", err))
		}
		for pid, entry := range table {
			if _, member := groups[entry.PGID]; !member || entry.State == "Z" {
				continue
			}
			if _, exists := tracked[pid]; exists {
				continue
			}
			// No synthetic registry record: these handles protect discovered
			// workers, while only authoritative refs receive status updates.
			refs = append(refs, scenarioruntime.ProcessRef{PID: &pid, Status: "running", HostBootID: host.BootID})
		}
	}
	if len(refs) > 1024 {
		return restore(fmt.Errorf("demand stop exceeds the 1024-process inspection bound"))
	}
	type target struct {
		ref    scenarioruntime.ProcessRef
		pid    int
		handle process.Handle
	}
	targets := make([]target, 0, len(refs))
	defer func() {
		for _, item := range targets {
			_ = item.handle.Close()
		}
	}()
	endedAt := time.Now().UTC()
	liveTargets := 0
	for _, ref := range refs {
		if ref.Status != "running" {
			continue
		}
		if ref.PID == nil || *ref.PID <= 0 {
			return restore(fmt.Errorf("running process ref %s has no valid pid; demand stop requires complete process identity", ref.RefID))
		}
		pid := *ref.PID
		if ref.HostBootID != "" && ref.HostBootID != host.BootID {
			return restore(fmt.Errorf("process ref %s belongs to host boot %q, not current boot %q", ref.RefID, ref.HostBootID, host.BootID))
		}
		handle, openErr := process.OpenHandle(pid)
		if openErr != nil {
			if !isMissingProcessError(openErr) {
				return restore(fmt.Errorf("open demand-managed process ref %s: %w", ref.RefID, openErr))
			}
			if ref.RefID != "" {
				if _, err := store.UpdateProcessRefStatus(ctx, ref.RefID, "exited", &endedAt); err != nil {
					return restore(err)
				}
			}
			continue
		}
		targets = append(targets, target{ref: ref, pid: pid, handle: handle})
		env, envErr := process.ReadEnvironment(pid)
		// If the original process exited while /proc was read, that read may
		// describe a reused PID. Do not use it to authorize any signal.
		alive, aliveErr := handle.Alive()
		if aliveErr != nil {
			return restore(fmt.Errorf("inspect process handle %s: %w", ref.RefID, aliveErr))
		}
		if !alive {
			continue
		}
		if envErr != nil {
			return restore(fmt.Errorf("inspect demand-managed process ref %s pid %d: %w", ref.RefID, pid, envErr))
		}
		if strings.TrimSpace(env[runtimeInstanceEnv]) != instance.InstanceID ||
			strings.TrimSpace(env[runtimeScenarioEnv]) != instance.Scenario ||
			strings.TrimSpace(env[runtimeVariantEnv]) != variant {
			return restore(fmt.Errorf("process ref %s pid %d does not identify runtime instance %s", ref.RefID, pid, instance.InstanceID))
		}
		liveTargets++
	}
	if !started && liveTargets == 0 && instance.OwnerPID != nil && *instance.OwnerPID > 0 && pidIsRunningFn(*instance.OwnerPID) {
		return restore(fmt.Errorf("demand-managed runtime %s still has a live lifecycle owner pid %d", instance.InstanceID, *instance.OwnerPID))
	}
	if err := ctx.Err(); err != nil {
		return restore(err)
	}
	// After preflight, cancellation cannot undo a delivered signal. Finish
	// bounded teardown bookkeeping even if the requesting caller goes away.
	teardownCtx, cancelTeardown := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancelTeardown()
	ctx = teardownCtx
	if err := store.BeginDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
		return restore(fmt.Errorf("persist demand signaling boundary: %w", err))
	}
	started = true
	for _, item := range targets {
		if err := item.handle.Signal(false); err != nil && !isMissingProcessError(err) {
			return fmt.Errorf("terminate demand-managed process ref %s: %w", item.ref.RefID, err)
		}
	}
	if len(targets) > 0 {
		time.Sleep(tuning.MaintenanceSettleDelay())
	}
	// Escalate all surviving members before waiting, so one slow exit does
	// not consume another worker's opportunity to terminate within the budget.
	for _, item := range targets {
		alive, err := item.handle.Alive()
		if err != nil {
			return fmt.Errorf("inspect demand-managed process ref %s: %w", item.ref.RefID, err)
		}
		if alive {
			if err := item.handle.Signal(true); err != nil && !isMissingProcessError(err) {
				return fmt.Errorf("force-terminate demand-managed process ref %s: %w", item.ref.RefID, err)
			}
		}
	}
	forceDeadline := time.Now().Add(2 * time.Second)
	for _, item := range targets {
		alive, err := item.handle.Alive()
		if err != nil {
			return fmt.Errorf("inspect demand-managed process ref %s: %w", item.ref.RefID, err)
		}
		// Signal delivery is asynchronous. Wait for this exact incarnation,
		// never for a later process that happens to reuse its PID. The entire
		// group shares one exit budget, independent of the number of workers.
		for alive && time.Now().Before(forceDeadline) {
			time.Sleep(10 * time.Millisecond)
			alive, err = item.handle.Alive()
			if err != nil {
				return fmt.Errorf("wait for demand-managed process ref %s: %w", item.ref.RefID, err)
			}
		}
		if alive {
			return fmt.Errorf("demand-managed process ref %s pid %d remained running", item.ref.RefID, item.pid)
		}
		if item.ref.RefID != "" {
			if _, err := store.UpdateProcessRefStatus(ctx, item.ref.RefID, "exited", &endedAt); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
				return fmt.Errorf("mark demand-managed process ref %s exited: %w", item.ref.RefID, err)
			}
		}
	}
	if len(groups) > 0 {
		table, err := listProcessTableFn()
		if err != nil {
			return fmt.Errorf("verify demand process groups after termination: %w", err)
		}
		for _, entry := range table {
			if _, member := groups[entry.PGID]; member && entry.State != "Z" {
				return fmt.Errorf("demand process group %d still contains pid %d; retaining port claims", entry.PGID, entry.PID)
			}
		}
	}
	if err := store.CompleteDemandStop(ctx, instance.InstanceID, instance.Generation); err != nil {
		return fmt.Errorf("finalize demand-managed runtime %s: %w", instance.InstanceID, err)
	}
	return nil
}

func (c *Controller) reconcileDemandLeases(ctx context.Context, store runtimeMaintenanceStore, now time.Time) ([]control.ResultItem, []control.ResultItem, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	stopped := make([]control.ResultItem, 0)
	failed := make([]control.ResultItem, 0)
	activeInstances, err := store.ListInstances(ctx, scenarioruntime.InstanceFilter{Statuses: scenarioruntime.ActiveInstanceStatuses()})
	if err != nil {
		return nil, nil, err
	}
	activeConsumers := make(map[string]struct{}, len(activeInstances))
	for _, instance := range activeInstances {
		activeConsumers[scenarioruntime.DependencyConsumerID(instance.Scenario, instance.Variant)] = struct{}{}
	}
	dependencyLeases, err := store.ListDemandLeases(ctx, scenarioruntime.DemandLeaseFilter{Statuses: []string{scenarioruntime.DemandLeaseActive}})
	if err != nil {
		return nil, nil, err
	}
	for _, lease := range dependencyLeases {
		if lease.Kind != scenarioruntime.DemandLeaseDependency {
			continue
		}
		// Only lifecycle-owned dependency identities can be reconciled from
		// runtime rows. Transient CLI calls and watchers own their own lifetime;
		// absence from the instance registry does not mean they have stopped.
		key, lifecycleOwned := scenarioruntime.DependencyConsumerInstance(lease.ConsumerID)
		if !lifecycleOwned {
			continue
		}
		if _, ok := activeConsumers[lease.ConsumerID]; !ok {
			// Dependency bootstrap acquires its hold before the parent runtime
			// row exists. Keep that hold while the parent's durable start
			// operation is still running; otherwise a slow dependency start can
			// be reaped by the next supervisor tick.
			if op, opErr := store.GetLatestStartOperation(ctx, key.Scenario, key.Variant); opErr == nil && op.Status == scenarioruntime.StartOperationStatusRunning {
				activeConsumers[lease.ConsumerID] = struct{}{}
			}
		}
		if _, ok := activeConsumers[lease.ConsumerID]; !ok {
			if _, releaseErr := store.ReleaseDemandLease(ctx, lease.LeaseID, "dependency consumer stopped"); releaseErr != nil && !errors.Is(releaseErr, scenarioruntime.ErrDemandLeaseExpired) && !errors.Is(releaseErr, scenarioruntime.ErrNotFound) {
				failed = append(failed, control.Failed(lease.LeaseID, releaseErr))
			}
			continue
		}
		var renewErr error
		if lease.ExpiresAt.After(now) {
			_, renewErr = store.RenewDemandLease(ctx, lease.LeaseID, scenarioruntime.DefaultDemandLeaseTTL)
		} else {
			// A delayed supervisor tick may observe an active consumer after
			// the short crash-recovery deadline. Reacquire the same stable
			// identity before expiry archival instead of dropping a live
			// dependency hold.
			_, renewErr = store.AcquireDemandLease(ctx, lease, scenarioruntime.DefaultDemandLeaseTTL)
		}
		if renewErr != nil && !errors.Is(renewErr, scenarioruntime.ErrDemandLeaseExpired) {
			failed = append(failed, control.Failed(lease.LeaseID, renewErr))
		}
	}
	expired, err := store.ExpireDemandLeases(ctx, now)
	if err != nil {
		return nil, nil, err
	}
	if len(expired) > 0 {
		stopped = append(stopped, control.Stopped("demand-leases", fmt.Sprintf("Expired %d demand lease(s)", len(expired))))
	}
	if _, err := store.ClaimDemandStopCandidates(ctx, now, "demand lease expired"); err != nil {
		return nil, nil, err
	}
	candidates, err := store.PendingDemandStops(ctx)
	if err != nil {
		return nil, nil, err
	}
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return stopped, failed, err
		}
		if stopErr := c.stopDemandManagedInstance(ctx, store, candidate); stopErr != nil {
			if errors.Is(stopErr, errDemandStopBusy) {
				break // The registry-wide owner also excludes the remaining candidates.
			}
			if errors.Is(stopErr, scenarioruntime.ErrNotFound) {
				continue // Another owner already completed this intent.
			}
			failed = append(failed, control.Failed(candidate.Scenario+"/"+candidate.Variant, stopErr))
			continue
		}
		stopped = append(stopped, control.Stopped(candidate.Scenario+"/"+candidate.Variant, "Stopped demand-managed instance after demand expired"))
	}
	return stopped, failed, nil
}

// ReconcileDemandLeases runs only the demand-retention portion of maintenance.
// The runtime supervisor uses this bounded entry point on its normal tick so
// idle demand-managed scenarios do not depend on an unrelated operator start
// or explicit cleanup command to stop.
func (c *Controller) ReconcileDemandLeases(ctx context.Context) (control.StopReport, error) {
	return c.ReconcileDemandLeasesAt(ctx, time.Now().UTC())
}

// ReconcileDemandLeasesAt is the clock-injectable form used by the runtime
// supervisor and deterministic tests. Production callers should use
// ReconcileDemandLeases unless they already own a control-plane clock.
func (c *Controller) ReconcileDemandLeasesAt(ctx context.Context, at time.Time) (control.StopReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var store runtimeMaintenanceStore
	var closeStore func()
	var err error
	if strings.TrimSpace(c.DBPath) != "" {
		opened, openErr := scenarioruntime.NewSQLiteStore(ctx, scenarioruntime.Config{HomeDir: c.Home, DBPath: c.DBPath})
		if openErr != nil {
			return control.StopReport{}, openErr
		}
		store, closeStore = opened, func() { _ = opened.Close() }
	} else {
		store, closeStore, err = openRuntimeRegistryIfPresent(c.Home)
		if err != nil || store == nil {
			return control.StopReport{}, err
		}
	}
	defer closeStore()
	stopped, failed, err := c.reconcileDemandLeases(ctx, store, at)
	if err != nil {
		return control.StopReport{}, err
	}
	return control.StopReport{Stopped: stopped, Failed: failed, Message: control.StopSummary(len(stopped), len(failed))}, nil
}

func (c *Controller) CleanStaleLocks() (control.StopReport, error) {
	ctx := context.Background()
	stopped := make([]control.ResultItem, 0)
	failed := make([]control.ResultItem, 0)

	store, closeStore, err := openRuntimeRegistryFn(c.Home)
	if err != nil {
		return control.StopReport{}, err
	}
	if store != nil {
		defer closeStore()
		now := time.Now().UTC()
		// The guard is what keeps this sweep from reaping a live start. Setup
		// phases routinely outrun the heartbeat TTL, and every `--clean-stale`
		// start runs this sweep, so without owner-liveness the fleet reaps its
		// own in-flight starts whenever two of them overlap.
		guard := scenarioruntime.StartingLeaseGuard{PIDRunning: newPIDLivenessMemo(processIsRunning)}
		if host, hostErr := (hostsession.DefaultProvider{}).Current(ctx, ""); hostErr == nil {
			guard.CurrentBootID = host.BootID
		}
		demandStopped, demandFailed, err := c.reconcileDemandLeases(ctx, store, now)
		if err != nil {
			return control.StopReport{}, err
		}
		stopped = append(stopped, demandStopped...)
		failed = append(failed, demandFailed...)
		expiredLeases, err := store.ExpireStaleStartingLeases(ctx, now, guard)
		if err != nil {
			return control.StopReport{}, err
		}
		for _, instance := range expiredLeases {
			stopped = append(stopped, control.Stopped(instance.Scenario+"/"+instance.InstanceID, "Expired stale starting runtime lease"))
		}
		expiredRuntime, reclaimable, err := expireNonAuthoritativeRegistryState(ctx, store)
		if err != nil {
			return control.StopReport{}, err
		}
		stopped = append(stopped, expiredRuntime...)

		// Port reclamation: expiring the registry record frees the CLAIM, but a
		// leaked process from the dead instance may still hold the OS port, which
		// is exactly the orphan-squat that makes test-genie's stricter
		// resolveURLs time out. Evict each holder — but ONLY after re-validating
		// it is still a Vrooli process from this install (stillVrooliOrphan), so a
		// foreign service that legitimately reused the port is never killed.
		for _, candidate := range reclaimable {
			item, evicted := c.reclaimSquattedPort(candidate)
			switch {
			case evicted:
				stopped = append(stopped, item)
			case item.Error != "":
				failed = append(failed, item)
			}
		}

		finalized, err := finalizeStuckStoppingInstances(ctx, store)
		if err != nil {
			return control.StopReport{}, err
		}
		stopped = append(stopped, finalized...)

		expiredClaims, err := store.ListExpiredActivePortClaims(ctx, now)
		if err != nil {
			return control.StopReport{}, err
		}
		for _, claim := range expiredClaims {
			if claim.Status != scenarioruntime.ClaimStatusReserved {
				continue
			}
			expired, err := store.ExpirePortClaim(ctx, claim.ClaimID)
			if err != nil {
				failed = append(failed, control.Failed(claim.ClaimID, err))
				continue
			}
			stopped = append(stopped, control.Stopped(fmt.Sprintf("%d", expired.Port), "Expired abandoned registry port reservation"))
		}

		// Supervisor sessions only reached a terminal status through graceful
		// shutdown, so every SIGKILLed supervisor left a row claiming to be
		// running. Retire the ones this host can prove are dead — a live peer
		// is never touched, because the same guard demands positive evidence.
		expiredSessions, err := store.ExpireStaleSupervisorSessions(ctx, now, guard)
		if err != nil {
			failed = append(failed, control.Failed("supervisor-sessions", err))
		} else if len(expiredSessions) > 0 {
			stopped = append(stopped, control.Stopped("supervisor-sessions",
				fmt.Sprintf("Retired %d supervisor session(s) whose process is gone", len(expiredSessions))))
		}

		// Editor leases are agent sessions; the same proof-of-death guard
		// applies. A session that is slow to heartbeat is building, not gone.
		expiredEditors, err := store.ExpireStaleEditorLeases(ctx, now, guard)
		if err != nil {
			failed = append(failed, control.Failed("editor-leases", err))
		}
		for _, lease := range expiredEditors {
			stopped = append(stopped, control.Stopped("agent-session/"+lease.SessionID, "Expired editor lease: "+lease.StopReason))
		}
		prunedClaims, err := store.PruneTerminalPortClaims(ctx, now.Add(-terminalPortClaimRetention))
		if err != nil {
			failed = append(failed, control.Failed("claim-retention", err))
		} else if prunedClaims > 0 {
			stopped = append(stopped, control.Stopped("registry-claims", fmt.Sprintf("Pruned %d expired/released registry claim rows older than %s", prunedClaims, terminalPortClaimRetention)))
		}
	}

	// Always-on resident-claim sweep (capacity §8.6): refresh GPU claims still
	// observed on the host and expire dead ones, so resident model-server claims
	// no longer depend on a 6h ttl_seconds stopgap. Best-effort: a ledger that
	// does not exist yet, or transient sensing/DB trouble, never fails the lock
	// sweep this rides on.
	if capacityExpired, capErr := c.sweepCapacityClaims(ctx); capErr == nil {
		stopped = append(stopped, capacityExpired...)
	} else {
		failed = append(failed, control.Failed("capacity-sweep", capErr))
	}

	return control.StopReport{
		Stopped: stopped,
		Failed:  failed,
		Message: control.StopSummary(len(stopped), len(failed)),
	}, nil
}

func (c *Controller) ListOrphans() ([]SystemProcess, error) {
	snapshot, err := c.Snapshot()
	if err != nil {
		return nil, err
	}
	return append([]SystemProcess(nil), snapshot.Orphans...), nil
}

func (c *Controller) KillOrphans() (control.StopReport, error) {
	orphans, err := c.ListOrphans()
	if err != nil {
		return control.StopReport{}, err
	}

	stopped := make([]control.ResultItem, 0, len(orphans))
	failed := make([]control.ResultItem, 0)
	for _, item := range orphans {
		// Re-validate right before signaling: if the PID has been recycled to
		// an unrelated process (or has already exited) between the snapshot and
		// now, skip the kill entirely. This closes the check-and-act race
		// between ListOrphans and kill.
		if !c.stillVrooliOrphan(item.PID) {
			stopped = append(stopped, control.Stopped(strconv.Itoa(item.PID), item.Command))
			continue
		}
		if err := killProcessFn(item.PID, false); err != nil && !isMissingProcessError(err) {
			failed = append(failed, control.Failed(strconv.Itoa(item.PID), err))
			continue
		}
		time.Sleep(tuning.MaintenanceSettleDelay())
		if process.IsPIDRunning(item.PID) {
			// Re-validate again: 150ms is long enough for the kernel to recycle
			// a PID on a busy box. Only escalate to SIGKILL while the PID still
			// resolves to a Vrooli process.
			if c.stillVrooliOrphan(item.PID) {
				if err := killProcessFn(item.PID, true); err != nil && !isMissingProcessError(err) {
					failed = append(failed, control.Failed(strconv.Itoa(item.PID), err))
					continue
				}
			}
		}
		stopped = append(stopped, control.Stopped(strconv.Itoa(item.PID), item.Command))
	}

	// Opportunistically prune scenario process records pointing at dead PIDs.
	// These accumulate whenever a scenario process exits outside the normal
	// stop path (crash, external kill, host reboot). Failures here are
	// swallowed so record hygiene never blocks the orphan sweep.
	if staleStopped, _ := c.cleanStaleScenarioRecords(); len(staleStopped) > 0 {
		stopped = append(stopped, staleStopped...)
	}

	return control.StopReport{
		Stopped: stopped,
		Failed:  failed,
		Message: control.StopSummary(len(stopped), len(failed)),
	}, nil
}

// reclaimSquattedPort evicts a process still holding a port whose owning
// instance was just expired as non-authoritative. It returns the result item
// and whether an eviction actually happened. The guard is the same one
// KillOrphans uses: the PID is signaled ONLY while it still resolves to a Vrooli
// process from this install, so a foreign service that reused the freed port —
// or a recycled PID — is never killed. A missing process (already gone) is a
// silent no-op: the port is already reclaimed.
func (c *Controller) reclaimSquattedPort(candidate PortReclaimCandidate) (control.ResultItem, bool) {
	name := fmt.Sprintf("%s:%d", candidate.Scenario, candidate.Port)
	// Re-validate right before signaling to close the check-and-act race: only a
	// live Vrooli orphan from this install is eligible for eviction.
	if !c.stillVrooliOrphan(candidate.PID) {
		return control.ResultItem{}, false
	}
	if err := killProcessFn(candidate.PID, false); err != nil && !isMissingProcessError(err) {
		return control.Failed(name, err), false
	}
	time.Sleep(tuning.MaintenanceSettleDelay())
	if pidIsRunningFn(candidate.PID) && c.stillVrooliOrphan(candidate.PID) {
		if err := killProcessFn(candidate.PID, true); err != nil && !isMissingProcessError(err) {
			return control.Failed(name, err), false
		}
	}
	return control.Stopped(name, fmt.Sprintf("Reclaimed port %d from orphaned pid %d", candidate.Port, candidate.PID)), true
}

// CleanStaleRecords removes scenario process records whose PID no longer
// resolves to a live process. Returns a StopReport describing each record
// that was pruned; intended for use by maintenance commands that want to
// surface hygiene cleanup alongside their primary action.
func (c *Controller) CleanStaleRecords() (control.StopReport, error) {
	stopped, err := c.cleanStaleScenarioRecords()
	if err != nil {
		return control.StopReport{}, err
	}
	return control.StopReport{
		Stopped: stopped,
		Message: control.StopSummary(len(stopped), 0),
	}, nil
}

func (c *Controller) ListTemplateValidationRuns(opts templatevalidation.CleanupOptions) (templatevalidation.CleanupResult, error) {
	opts.RepoRoot = c.Root
	opts.DryRun = true
	result := templatevalidation.ExecuteCleanup(templatevalidation.PlanCleanup(opts))
	return result, templatevalidation.ResultError(result)
}

func (c *Controller) CleanTemplateValidationRuns(opts templatevalidation.CleanupOptions) (templatevalidation.CleanupResult, error) {
	opts.RepoRoot = c.Root
	result := templatevalidation.ExecuteCleanup(templatevalidation.PlanCleanup(opts))
	if result.NeedsProtoGenerate && !result.DryRun && len(result.Failures) == 0 {
		if err := runProtoGenerateFn(c.Root); err != nil {
			return result, err
		}
		result.ProtoGenerateRan = true
	}
	return result, templatevalidation.ResultError(result)
}

func runProtoGenerate(repoRoot string) error {
	protoDir := filepath.Join(repoRoot, "packages", "proto")
	if _, err := os.Stat(filepath.Join(protoDir, "Makefile")); err != nil {
		return nil
	}
	cmd := shell.NewCommand("make", "generate")
	cmd.Dir = protoDir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// cleanStaleScenarioRecords walks $HOME/.vrooli/processes/scenarios/<name>/
// and removes every record whose PID is not running. It returns the set of
// pruned records as StopReport result items. The sweep is best-effort: an
// unreadable scenario directory is skipped, not reported as an error.
func (c *Controller) cleanStaleScenarioRecords() ([]control.ResultItem, error) {
	processesDir, err := repocontract.RuntimeHomeEntryPath(c.Home, repocontract.HomeKeyProcesses)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(processesDir, repocontractmeta.ScenarioDir)
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	pruned := make([]control.ResultItem, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenario := entry.Name()
		records, err := process.ReadScenarioRecords(c.Home, scenario)
		if err != nil {
			continue
		}
		for _, record := range records {
			if record.PID > 0 && process.IsPIDRunning(record.PID) {
				continue
			}
			step := record.Step
			if step == "" {
				continue
			}
			if err := process.RemoveScenarioRecord(c.Home, scenario, step); err != nil {
				continue
			}
			pruned = append(pruned, control.Stopped(scenario+"/"+step, "Removed stale process record (pid "+strconv.Itoa(record.PID)+")"))
		}
	}
	return pruned, nil
}

// stillVrooliOrphan performs a fresh /proc read for the given PID and returns
// true only if the current process still looks Vrooli-owned. It is the narrow
// re-validation step used by KillOrphans to guard against PID reuse between the
// snapshot and signal delivery.
func (c *Controller) stillVrooliOrphan(pid int) bool {
	entry, ok := readProcessEntryFn(pid)
	if !ok {
		return false
	}
	return looksLikeVrooliProcessFn(c.Root, c.Home, entry) && !insideAgentSessionScope(entry)
}

func (c *Controller) DiagnosePort(port int, scenarioName string) (PortDiagnostic, error) {
	inspection := inspectPortListenersFn(port)

	snapshot, err := c.Snapshot()
	if err != nil {
		return PortDiagnostic{}, err
	}

	diagnostic := PortDiagnostic{
		Port:               port,
		Scenario:           strings.TrimSpace(scenarioName),
		InUse:              len(inspection.Listeners) > 0,
		Listeners:          inspection.Listeners,
		ListenerInspection: inspection.Inspection,
		HostOrphanCount:    snapshot.OrphanProcesses,
		PortPolicy:         describePortPolicy(port),
	}

	store, closeStore, err := openRuntimeRegistryFn(c.Home)
	if err != nil {
		return PortDiagnostic{}, err
	}
	if store != nil {
		defer closeStore()
		claims, err := listRuntimeClaims(context.Background(), store, port, diagnostic.Scenario, diagnostic.InUse)
		if err != nil {
			return PortDiagnostic{}, err
		}
		diagnostic.RegistryClaims = claims
		processes, err := listRuntimeProcessRefs(context.Background(), store, claims)
		if err != nil {
			return PortDiagnostic{}, err
		}
		diagnostic.RegistryProcesses = processes
	}
	diagnostic.Recommendations = buildRecommendations(port, diagnostic)
	return diagnostic, nil
}

// describePortPolicy classifies a port against the live OS ephemeral window
// and the canonical Vrooli bands. It always returns a populated report so
// JSON consumers can rely on the field existing.
func describePortPolicy(port int) PortPolicyReport {
	eph := portspec.OSEphemeralRange(context.Background())
	band := ""
	if role, ok := portspec.CanonicalBand(port); ok {
		band = string(role)
	}
	return PortPolicyReport{
		EphemeralMin:         eph.Min,
		EphemeralMax:         eph.Max,
		EphemeralSource:      eph.Source,
		InsideEphemeralRange: eph.Contains(port),
		CanonicalBand:        band,
		AboveCanonicalMax:    portspec.IsAboveCanonicalMax(port),
	}
}

//nolint:gocyclo // maintenance recommendations enumerate independent listener, ownership, and remediation findings.
func buildRecommendations(port int, diagnostic PortDiagnostic) []string {
	recommendations := make([]string, 0, maintenanceParameterA)
	if diagnostic.PortPolicy.InsideEphemeralRange {
		recommendations = append(recommendations, fmt.Sprintf(
			"Port %d sits inside the OS ephemeral range %d-%d (source=%s); see docs/reference/port-allocation.md and run `go run ./cmd/vrooli-ports-migrate` to move it into a canonical band.",
			port, diagnostic.PortPolicy.EphemeralMin, diagnostic.PortPolicy.EphemeralMax, diagnostic.PortPolicy.EphemeralSource,
		))
	} else if diagnostic.PortPolicy.AboveCanonicalMax && !diagnostic.PortPolicy.InsideEphemeralRange {
		recommendations = append(recommendations, fmt.Sprintf(
			"Port %d is above the canonical safe zone (<=%d); consider moving it to keep parity with other OSes.",
			port, portspec.CanonicalMax,
		))
	}
	for _, claim := range diagnostic.RegistryClaims {
		switch claim.RecommendationCode {
		case scenarioruntime.PortRecommendationLikelyManifestDrift:
			recommendations = append(recommendations, fmt.Sprintf("Registry evidence for %s %s (%s) suggests manifest drift: %s", claim.Scenario, claim.PortName, claim.EnvVar, claim.RecommendationRationale))
		case scenarioruntime.PortRecommendationLikelyRuntimeFailure:
			recommendations = append(recommendations, fmt.Sprintf("Registry evidence for %s %s (%s) suggests a runtime listener failure: %s", claim.Scenario, claim.PortName, claim.EnvVar, claim.RecommendationRationale))
		case scenarioruntime.PortRecommendationUnboundWatch:
			recommendations = append(recommendations, fmt.Sprintf("Registry evidence for %s %s (%s) is inconclusive: %s", claim.Scenario, claim.PortName, claim.EnvVar, claim.RecommendationRationale))
		case scenarioruntime.PortRecommendationInspectionUnavailable:
			recommendations = append(recommendations, fmt.Sprintf("Registry listener evidence for %s %s (%s) is unavailable: %s", claim.Scenario, claim.PortName, claim.EnvVar, claim.RecommendationRationale))
		case scenarioruntime.PortRecommendationOrphanListenerInvestigate:
			recommendations = append(recommendations, fmt.Sprintf("Registry evidence for %s %s (%s) conflicts with current listener state: %s", claim.Scenario, claim.PortName, claim.EnvVar, claim.RecommendationRationale))
		}
		if claim.Authoritative != nil && !*claim.Authoritative {
			switch claim.Reconciliation {
			case scenarioruntime.ReconcileUnverified:
				recommendations = append(recommendations, fmt.Sprintf("Registry claim %s for %s is unverified (%s); restart the scenario under registry-enabled lifecycle before using strict registry discovery", claim.ClaimID, claim.Scenario, claim.ReconcileReason))
			case scenarioruntime.ReconcileStaleInstance, scenarioruntime.ReconcileStaleClaim:
				recommendations = append(recommendations, fmt.Sprintf("Run `vrooli cleanup locks` to expire non-authoritative registry claim %s for %s (%s)", claim.ClaimID, claim.Scenario, claim.ReconcileReason))
			default:
				recommendations = append(recommendations, fmt.Sprintf("Registry claim %s for %s is non-authoritative (%s)", claim.ClaimID, claim.Scenario, claim.ReconcileReason))
			}
		}
		if claim.ClaimStatus == scenarioruntime.ClaimStatusReserved && claim.ExpiresAt != nil && claim.ExpiresAt.Before(time.Now().UTC()) {
			recommendations = append(recommendations, fmt.Sprintf("Expire abandoned registry reservation %s for %s port %d", claim.ClaimID, claim.Scenario, claim.Port))
		}
		if claim.InstanceStatus == scenarioruntime.StatusExpired || claim.InstanceStatus == scenarioruntime.StatusFailed {
			recommendations = append(recommendations, fmt.Sprintf("Inspect registry claim %s for %s instance %s (%s)", claim.ClaimID, claim.Scenario, claim.InstanceID, claim.InstanceStatus))
		}
		if claim.LeaseFresh != nil && !*claim.LeaseFresh && claim.InstanceStatus == scenarioruntime.StatusStarting {
			recommendations = append(recommendations, fmt.Sprintf("Run `vrooli cleanup locks` to expire stale startup lease %s", claim.InstanceID))
		}
	}
	if diagnostic.InUse {
		recommendations = append(recommendations, fmt.Sprintf("Stop the process currently listening on port %d", port))
	}
	if !diagnostic.ListenerInspection.Available {
		recommendations = append(recommendations, fmt.Sprintf("Listener inspection unavailable: %s", diagnostic.ListenerInspection.Reason))
	}
	if diagnostic.HostOrphanCount > 0 {
		recommendations = append(recommendations, fmt.Sprintf("Host has %d orphaned Vrooli process(es); run `vrooli orphans` to inspect before `vrooli cleanup orphans`", diagnostic.HostOrphanCount))
	}
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No lock or listener conflict detected; inspect scenario logs for the failing service")
	}
	return recommendations
}

// systemDaemonExeBasenames are executables that must never be classified as
// Vrooli processes, even when invoked with Vrooli-looking arguments or cwd.
// Postgres workers inherit a connection string that mentions "vrooli"; fuse
// overlay mounts have Vrooli paths in their argv; the user's shell, SSH, IDE,
// and Claude Code subprocesses all frequently have a Vrooli cwd.
var systemDaemonExeBasenames = map[string]struct{}{
	"postgres":                {},
	"postmaster":              {},
	"fuse-overlayfs":          {},
	"fusermount":              {},
	"fusermount3":             {},
	"sshd":                    {},
	"ssh":                     {},
	"bash":                    {},
	"dash":                    {},
	"sh":                      {},
	"zsh":                     {},
	"fish":                    {},
	"opencode":                {},
	"code":                    {},
	"code-insiders":           {},
	"docker":                  {},
	"dockerd":                 {},
	"containerd":              {},
	"containerd-shim":         {},
	"containerd-shim-runc-v2": {},
	"runc":                    {},
	"git":                     {},
	"gpg":                     {},
	"gpg-agent":               {},
	"gnome-keyring-daemon":    {},
}

// interpreterExeBasenames identifies runtime interpreters that are themselves
// system binaries but may execute Vrooli code. For these, we require cwd to
// be under a Vrooli-owned directory (scenarios/ or resources/) to classify
// the process as Vrooli.
var interpreterExeBasenames = map[string]struct{}{
	"node":    {},
	"python":  {},
	"python3": {},
	"ruby":    {},
	"deno":    {},
	"bun":     {},
	"java":    {},
	"go":      {},
}

var protectedVrooliExecutableBasenames = map[string]struct{}{
	"vrooli-autoheal-loop":     {},
	"vrooli-autoheal-loop.exe": {},
}

var controlPlaneAPIExecutableBasenames = map[string]struct{}{
	"agent-manager-api":         {},
	"agent-manager-api.exe":     {},
	"swarm-manager-api":         {},
	"swarm-manager-api.exe":     {},
	"workspace-sandbox-api":     {},
	"workspace-sandbox-api.exe": {},
}

// bridgeAgentExecutableBasenames are OS-managed Bridge node agents. They are
// intentionally outside Vrooli's scenario runtime registry: launchd/systemd
// owns their lifetime and the Bridge control plane owns their health. On
// macOS /proc is unavailable, so ps gives us the full command line rather than
// a separate executable path; isBridgeAgentExecutable handles both forms.
var bridgeAgentExecutableBasenames = map[string]struct{}{
	"vrooli-bridge-agent":     {},
	"vrooli-bridge-agent.exe": {},
}

// vrooliCLIExecutableBasenames are the `vrooli` CLI entrypoint basenames.
// These are transient user-initiated commands (e.g. `vrooli scenario restart`)
// that don't register a process record, so orphan detection must not
// classify them — otherwise `vrooli cleanup orphans` could SIGTERM a
// concurrent sibling invocation.
var vrooliCLIExecutableBasenames = map[string]struct{}{
	"vrooli":     {},
	"vrooli.exe": {},
}

func isVrooliCLIExecutable(home, exe string) bool {
	if _, ok := vrooliCLIExecutableBasenames[processPathBase(exe)]; ok {
		return true
	}
	binDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBin)
	if err != nil {
		return false
	}
	exe = strings.TrimSuffix(strings.TrimSpace(exe), " (deleted)")
	rel, err := filepath.Rel(binDir, exe)
	if err != nil || rel == "." || filepath.IsAbs(rel) {
		return false
	}
	// Only direct children are installed command surfaces. Subdirectories may
	// contain a supervised workload and remain subject to ownership checks.
	return filepath.Dir(rel) == "."
}

func isControlPlaneAPIExecutable(entry processTableEntry) bool {
	if _, ok := controlPlaneAPIExecutableBasenames[processPathBase(entry.Executable)]; ok {
		return true
	}
	if _, ok := controlPlaneAPIExecutableBasenames[processPathBase(entry.Command)]; ok {
		return true
	}
	return false
}

func isBridgeAgentExecutable(entry processTableEntry) bool {
	for _, candidate := range []string{entry.Executable, entry.Command} {
		if base, ok := commandExecutableBase(candidate); ok {
			if _, known := bridgeAgentExecutableBasenames[base]; known {
				return true
			}
		}
	}
	return false
}

// commandExecutableBase returns the basename of the executable token in a
// process command line. It is separate from processPathBase because Darwin's
// ps reports `path/to/bin arg1 arg2` as one command field.
func commandExecutableBase(command string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return "", false
	}
	return processPathBase(fields[0]), true
}

func looksLikeVrooliProcess(root, home string, entry processTableEntry) bool {
	root = filepath.Clean(root)
	home = filepath.Clean(home)

	exe := strings.TrimSpace(entry.Executable)
	cwd := strings.TrimSpace(entry.Cwd)
	basename := processPathBase(exe)
	commandBasename := processPathBase(strings.TrimSpace(entry.Command))

	// Never classify known system daemons or user shells as Vrooli, even if
	// their cwd or argv happens to touch Vrooli paths.
	if _, ok := systemDaemonExeBasenames[basename]; ok {
		return false
	}
	if _, ok := protectedVrooliExecutableBasenames[basename]; ok {
		return false
	}
	if _, ok := protectedVrooliExecutableBasenames[commandBasename]; ok {
		return false
	}
	if isBridgeAgentExecutable(entry) {
		return false
	}

	vrooliOwnedPrefixes := vrooliOwnedPrefixes(root, home)

	// Primary signal: the executable itself lives under a Vrooli-owned path.
	if exe != "" {
		for _, prefix := range vrooliOwnedPrefixes {
			if hasPathPrefix(exe, prefix) {
				return true
			}
		}
		// The compiled vrooli binary sits at <root>/vrooli (repo checkout).
		if exe == filepath.Join(root, "vrooli") {
			return true
		}
	}

	// Secondary signal: a language interpreter whose working directory is
	// clearly inside a Vrooli scenario or resource tree (e.g. `node` running
	// vite inside scenarios/<name>/ui).
	if _, isInterpreter := interpreterExeBasenames[basename]; isInterpreter && cwd != "" {
		for _, prefix := range []string{
			filepath.Join(root, repocontractmeta.ScenarioDir) + string(filepath.Separator),
			filepath.Join(root, "resources") + string(filepath.Separator),
		} {
			if strings.HasPrefix(cwd, prefix) {
				return true
			}
		}
	}

	// Legacy fallback for environments where /proc is unavailable (e.g. some
	// container/host OS combinations): if the Executable field could not be
	// read, accept a conservative match on the command line — but only when
	// the command explicitly references a Vrooli-owned install path. This
	// avoids the false positives on postgres/fuse-overlayfs/shell cwds that
	// the previous substring-on-"vrooli" heuristic produced.
	if exe == "" && strings.TrimSpace(entry.Command) != "" {
		for _, prefix := range vrooliOwnedPrefixes {
			if strings.Contains(entry.Command, prefix) {
				return true
			}
		}
	}

	return false
}

func processPathBase(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, " (deleted)")
	if path == "" {
		return ""
	}
	return filepath.Base(path)
}

func vrooliOwnedPrefixes(root, home string) []string {
	sep := string(filepath.Separator)
	prefixes := []string{
		filepath.Join(root, repocontractmeta.ScenarioDir) + sep,
		filepath.Join(root, "resources") + sep,
		filepath.Join(root, "bin") + sep,
		filepath.Join(root, "packages") + sep,
		filepath.Join(root, "cmd") + sep,
	}
	if vrooliHome, err := repocontract.VrooliUserRoot(home); err == nil {
		prefixes = append(prefixes, vrooliHome+sep)
	}
	return prefixes
}

func hasPathPrefix(path, prefix string) bool {
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	if len(path) == len(prefix) {
		return true
	}
	return path[len(prefix)] == filepath.Separator || strings.HasSuffix(prefix, string(filepath.Separator))
}

func isMissingProcessError(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "process already finished") ||
		strings.Contains(text, "no such process")
}

// ListAgentSessions returns the live editor leases: every coding-agent
// session the launcher recorded whose process this host has not proven dead.
// A missing registry means no sessions, not an error.
func (c *Controller) ListAgentSessions() ([]scenarioruntime.EditorLease, error) {
	store, closeStore, err := openRuntimeRegistryFn(c.Home)
	if err != nil || store == nil {
		return nil, err
	}
	defer closeStore()
	ctx := context.Background()
	leases, err := store.ListEditorLeases(ctx, false)
	if err != nil {
		return nil, err
	}
	guard := scenarioruntime.StartingLeaseGuard{PIDRunning: newPIDLivenessMemo(processIsRunning)}
	if host, hostErr := (hostsession.DefaultProvider{}).Current(ctx, ""); hostErr == nil {
		guard.CurrentBootID = host.BootID
	}
	return scenarioruntime.LiveEditorLeases(leases, guard), nil
}
