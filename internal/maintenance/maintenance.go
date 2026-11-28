package maintenance

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
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
		if ref.Status == scenarioruntime.ProcessRefStatusRunning && ref.PID != nil {
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
			refs = append(refs, scenarioruntime.ProcessRef{PID: &pid, Status: scenarioruntime.ProcessRefStatusRunning, HostBootID: host.BootID})
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
		if ref.Status != scenarioruntime.ProcessRefStatusRunning {
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
				if _, err := store.UpdateProcessRefStatus(ctx, ref.RefID, scenarioruntime.ProcessRefStatusExited, &endedAt); err != nil {
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
			if _, err := store.UpdateProcessRefStatus(ctx, item.ref.RefID, scenarioruntime.ProcessRefStatusExited, &endedAt); err != nil && !errors.Is(err, scenarioruntime.ErrNotFound) {
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
