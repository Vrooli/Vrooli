package maintenance

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostsession"
	"github.com/vrooli/vrooli/internal/portspec"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

// stillVrooliOrphan performs a fresh /proc read for the given PID and returns
// true only if the current process still looks Vrooli-owned. It is the narrow
// re-validation step used by KillOrphans to guard against PID reuse between the
// snapshot and signal delivery.
func (c *Controller) stillVrooliOrphan(pid int) bool {
	entry, ok := readProcessEntryFn(pid)
	if !ok {
		return false
	}
	if !looksLikeVrooliProcessFn(c.Root, c.Home, entry) || insideLifecycleOwnedScope(entry) {
		return false
	}
	// A leaked desktop-bundle process is only eligible while it is still an
	// init orphan (ppid==1). If it was re-adopted by a live parent between the
	// snapshot and the signal, it is no longer a leak and must not be killed.
	if isLeakedDesktopBundleProcess(entry) && entry.PPID != 1 {
		return false
	}
	return true
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

// leakedDesktopBundleMarker is the path fragment unique to a scenario-to-desktop
// AppImage whose squashfs was extracted to /tmp. When the bundled UI exits, its
// private runtime (the bundled api process and its opencode children) can be
// left reparented to init instead of being reaped. The extracted-root prefix
// (/tmp/appimage_extracted_) plus this bundle fragment is specific enough that
// no live, registry-managed scenario ever matches it.
const (
	leakedDesktopExtractedPrefix = "/tmp/appimage_extracted_"
	leakedDesktopBundleMarker    = "/resources/bundle/"
)

// matchesLeakedDesktopBundlePath reports whether a resolved path (executable or
// the first argv token) points inside an extracted AppImage's private bundle
// runtime. Both the extracted-root prefix and the bundle marker must be present
// so an unrelated /tmp path can never match.
func matchesLeakedDesktopBundlePath(path string) bool {
	path = strings.TrimSpace(path)
	path = strings.TrimSuffix(path, " (deleted)")
	if path == "" {
		return false
	}
	return strings.HasPrefix(path, leakedDesktopExtractedPrefix) && strings.Contains(path, leakedDesktopBundleMarker)
}

// isLeakedDesktopBundleProcess recognizes a process left behind by a
// scenario-to-desktop bundle: its executable (or argv[0]) lives in an extracted
// AppImage's /resources/bundle tree. The ppid==1 (reparented-to-init) and
// not-registry-claimed guards are applied by the caller (collectOrphans); this
// predicate only identifies the executable class, so it can also be reused by
// looksLikeVrooliProcess for the kill-time re-validation.
func isLeakedDesktopBundleProcess(entry processTableEntry) bool {
	if matchesLeakedDesktopBundlePath(entry.Executable) {
		return true
	}
	if base, ok := commandExecutablePath(entry.Command); ok && matchesLeakedDesktopBundlePath(base) {
		return true
	}
	return false
}

// commandExecutablePath returns the first whitespace-separated token of a
// command line (argv[0]), which on Linux ps and /proc/<pid>/cmdline is the
// executable path. It differs from commandExecutableBase, which returns only
// the basename; the leaked-bundle match needs the full path to test the prefix.
func commandExecutablePath(command string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return "", false
	}
	return fields[0], true
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

	// A process left behind by a scenario-to-desktop bundle lives under
	// /tmp/appimage_extracted_*/resources/bundle, which is outside every
	// Vrooli-owned prefix below. Recognize it explicitly so the kill-time
	// re-validation agrees with collectOrphans that it is a reapable leak.
	if isLeakedDesktopBundleProcess(entry) {
		return true
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
