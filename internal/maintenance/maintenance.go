package maintenance

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/control"
	"github.com/vrooli/vrooli/internal/network"
	"github.com/vrooli/vrooli/internal/portspec"
	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/templatevalidation"
)

type Controller struct {
	Root string
	Home string
}

type SystemProcess struct {
	PID     int    `json:"pid"`
	PPID    int    `json:"ppid"`
	Command string `json:"command"`
}

type (
	LockInfo     = network.LockInfo
	PortListener = network.PortListener
)

type RuntimeClaimInfo struct {
	ClaimID           string     `json:"claim_id"`
	InstanceID        string     `json:"instance_id"`
	Scenario          string     `json:"scenario"`
	Generation        int64      `json:"generation,omitempty"`
	PortName          string     `json:"port_name"`
	EnvVar            string     `json:"env_var,omitempty"`
	Port              int        `json:"port"`
	BindHost          string     `json:"bind_host"`
	URL               string     `json:"url,omitempty"`
	ClaimStatus       string     `json:"claim_status"`
	InstanceStatus    string     `json:"instance_status,omitempty"`
	LeaseFresh        *bool      `json:"lease_fresh,omitempty"`
	HeartbeatDeadline *time.Time `json:"heartbeat_deadline,omitempty"`
	HealthStatus      string     `json:"health_status,omitempty"`
	HealthReady       *bool      `json:"health_ready,omitempty"`
	Reconciliation    string     `json:"reconciliation,omitempty"`
	ReconcileReason   string     `json:"reconcile_reason,omitempty"`
	Authoritative     *bool      `json:"authoritative,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	LastBoundAt       *time.Time `json:"last_bound_at,omitempty"`
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
	Lock               *LockInfo                  `json:"lock,omitempty"`
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

var (
	listLocksFn              = network.ListLocks
	readLockFileFn           = network.ReadLockFile
	pruneStaleLocksFn        = network.PruneStaleLocks
	inspectPortListenersFn   = network.InspectPortListeners
	killProcessFn            = killProcess
	looksLikeVrooliProcessFn = looksLikeVrooliProcess
	runProtoGenerateFn       = runProtoGenerate
	openRuntimeRegistryFn    = openRuntimeRegistryIfPresent
)

func NewController(root, home string) *Controller {
	return &Controller{
		Root: filepath.Clean(root),
		Home: filepath.Clean(home),
	}
}

func (c *Controller) ListLocks() ([]LockInfo, error) {
	return listLocksFn(c.Home)
}

func (c *Controller) ListRuntimeClaims() ([]RuntimeClaimInfo, error) {
	store, closeStore, err := openRuntimeRegistryFn(c.Home)
	if err != nil || store == nil {
		return nil, err
	}
	defer closeStore()
	return listRuntimeClaims(context.Background(), store, 0, "")
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
		expiredLeases, err := store.ExpireStaleStartingLeases(ctx, now)
		if err != nil {
			return control.StopReport{}, err
		}
		for _, instance := range expiredLeases {
			stopped = append(stopped, control.Stopped(instance.Scenario+"/"+instance.InstanceID, "Expired stale starting runtime lease"))
		}
		expiredRuntime, err := expireNonAuthoritativeRegistryState(ctx, store)
		if err != nil {
			return control.StopReport{}, err
		}
		stopped = append(stopped, expiredRuntime...)

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
	}

	cleanedLocks, err := pruneStaleLocksFn(c.Home)
	if err != nil {
		return control.StopReport{}, err
	}
	for _, lock := range cleanedLocks {
		stopped = append(stopped, control.Stopped(fmt.Sprintf("%d", lock.Port), "Removed stale legacy lock"))
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
		time.Sleep(150 * time.Millisecond)
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
	cmd := exec.Command("make", "generate")
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
	root := filepath.Join(c.Home, ".vrooli", "processes", "scenarios")
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
	return looksLikeVrooliProcessFn(c.Root, c.Home, entry)
}

func (c *Controller) DiagnosePort(port int, scenarioName string) (PortDiagnostic, error) {
	inspection, err := inspectPortListenersFn(port)
	if err != nil {
		return PortDiagnostic{}, err
	}

	lockPath := network.LockPath(c.Home, port)
	var lock *LockInfo
	if info, err := os.Stat(lockPath); err == nil && !info.IsDir() {
		lockInfo, err := readLockFileFn(lockPath)
		if err != nil {
			return PortDiagnostic{}, err
		}
		lock = &lockInfo
	}

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
		Lock:               lock,
		HostOrphanCount:    snapshot.OrphanProcesses,
		PortPolicy:         describePortPolicy(port),
	}

	store, closeStore, err := openRuntimeRegistryFn(c.Home)
	if err != nil {
		return PortDiagnostic{}, err
	}
	if store != nil {
		defer closeStore()
		claims, err := listRuntimeClaims(context.Background(), store, port, diagnostic.Scenario)
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

func buildRecommendations(port int, diagnostic PortDiagnostic) []string {
	recommendations := make([]string, 0, 5)
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
	if diagnostic.Lock != nil && diagnostic.Lock.Stale {
		recommendations = append(recommendations, fmt.Sprintf("Clean stale lock file %s", diagnostic.Lock.Path))
	}
	for _, claim := range diagnostic.RegistryClaims {
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

// vrooliCLIExecutableBasenames are the `vrooli` CLI entrypoint basenames.
// These are transient user-initiated commands (e.g. `vrooli scenario restart`)
// that don't register a process record, so orphan detection must not
// classify them — otherwise `vrooli cleanup orphans` could SIGTERM a
// concurrent sibling invocation.
var vrooliCLIExecutableBasenames = map[string]struct{}{
	"vrooli":     {},
	"vrooli.exe": {},
}

func isVrooliCLIExecutable(exe string) bool {
	_, ok := vrooliCLIExecutableBasenames[processPathBase(exe)]
	return ok
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
			filepath.Join(root, "scenarios") + string(filepath.Separator),
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
		filepath.Join(home, ".vrooli") + sep,
		filepath.Join(root, "scenarios") + sep,
		filepath.Join(root, "resources") + sep,
		filepath.Join(root, "bin") + sep,
		filepath.Join(root, "packages") + sep,
		filepath.Join(root, "cmd") + sep,
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
