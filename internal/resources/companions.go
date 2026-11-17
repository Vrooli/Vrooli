package resources

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/process"
	activitynetwork "github.com/vrooli/vrooli/internal/resources/activitynetwork"
	manifestpkg "github.com/vrooli/vrooli/internal/resources/manifest"
	valuehelpers "github.com/vrooli/vrooli/internal/values"
	"github.com/vrooli/vrooli/packages/capacity/activityedge"
)

// startCompanions launches every declared host-side companion (idempotent,
// best-effort). A companion that fails to start is a logged warning, never fatal:
// the container is already up and the resource's health check surfaces a dead
// companion (e.g. whisper's edge owns the canonical port, so a down edge fails
// the health probe). When no companions are declared this is a no-op, keeping the
// driver byte-identical for every non-adopting managed-service resource.
func startCompanions(resourceName string, companions []ResourceCompanion, recoveryAttempts int, stderr io.Writer, parentPIDs ...int) {
	startCompanionsWithNetwork(resourceName, companions, recoveryAttempts, stderr, nil, parentPIDs...)
}

type activityEdgeNamespace struct {
	ChildPID     int
	EdgePort     int
	UpstreamPort int
	SocketPath   string
}

func startCompanionsWithNetwork(resourceName string, companions []ResourceCompanion, recoveryAttempts int, stderr io.Writer, network *activityEdgeNamespace, parentPIDs ...int) {
	parentPID := 0
	if len(parentPIDs) > 0 {
		parentPID = parentPIDs[0]
	}
	for _, c := range companions {
		if err := startCompanionWithNetwork(resourceName, c, recoveryAttempts, parentPID, network); err != nil {
			fmt.Fprintf(stderr, "warning: companion %q for %s did not start: %v\n", c.Name, resourceName, err)
		}
	}
}

func activityEdgeNamespaceForManifest(manifest ResourceManifest) (*activityEdgeNamespace, error) {
	if manifest.ActivityNetwork == nil {
		return nil, nil
	}
	stateDir := filepath.Join(os.TempDir(), "vra-"+manifest.Name)
	childPID, err := activitynetwork.WaitForChildPID(stateDir, 30*time.Second, 100*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if !process.IsPIDRunning(childPID) {
		return nil, fmt.Errorf("RootlessKit network child %d is not running", childPID)
	}
	var upstreamPort int
	for _, port := range manifest.Ports {
		if port.Name == manifest.ActivityNetwork.UpstreamPortName {
			upstreamPort = manifestpkg.RuntimePortValue(manifest, port)
			break
		}
	}
	var edgePort int
	for _, port := range manifest.Ports {
		if port.Name == manifest.ActivityNetwork.EdgePortName {
			edgePort = port.Host
			break
		}
	}
	if upstreamPort < 1 || edgePort < 1 {
		return nil, fmt.Errorf("%s activity edge private ports are incomplete", manifest.Name)
	}
	socketDir := filepath.Join(os.TempDir(), "vru-"+manifest.Name)
	if err := os.MkdirAll(socketDir, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(socketDir, 0700); err != nil {
		return nil, err
	}
	return &activityEdgeNamespace{ChildPID: childPID, EdgePort: edgePort, UpstreamPort: upstreamPort, SocketPath: filepath.Join(socketDir, "upstream.sock")}, nil
}

func startManagedServiceCompanions(manifest ResourceManifest, recoveryAttempts int, stderr io.Writer, parentPID int) {
	if manifest.ActivityNetwork == nil {
		startCompanions(manifest.Name, manifest.Companions, recoveryAttempts, stderr, parentPID)
		return
	}
	if runtime.GOOS != "linux" {
		// Non-Linux owners keep the managed backend on its normal host loopback
		// port; no RootlessKit namespace or Unix bridge is needed there.
		companions := make([]ResourceCompanion, 0, len(manifest.Companions))
		for _, companion := range manifest.Companions {
			if companion.Name != "upstream-bridge" {
				companions = append(companions, companion)
			}
		}
		startCompanions(manifest.Name, companions, recoveryAttempts, stderr, parentPID)
		return
	}
	network, err := activityEdgeNamespaceForManifest(manifest)
	if err != nil {
		fmt.Fprintf(stderr, "warning: private activity namespace for %s companions is unavailable: %v\n", manifest.Name, err)
		return
	}
	// The stable host edge must outlive an unloaded backend. Start it first,
	// then its namespace bridge; other companions remain in the host owner path.
	ordered := append([]ResourceCompanion(nil), manifest.Companions...)
	sort.SliceStable(ordered, func(i, j int) bool {
		rank := func(n string) int {
			switch n {
			case manifestpkg.ActivityEdgeCompanionName:
				return 0
			case manifestpkg.UpstreamBridgeCompanionName:
				return 1
			default:
				return 2
			}
		}
		return rank(ordered[i].Name) < rank(ordered[j].Name)
	})
	startCompanionsWithNetwork(manifest.Name, ordered, recoveryAttempts, stderr, network, parentPID)
}

func sameNetworkNamespace(firstPID, secondPID int) bool {
	if firstPID <= 1 || secondPID <= 1 {
		return false
	}
	first, firstErr := os.Readlink(fmt.Sprintf("/proc/%d/ns/net", firstPID))
	second, secondErr := os.Readlink(fmt.Sprintf("/proc/%d/ns/net", secondPID))
	return firstErr == nil && secondErr == nil && first == second
}

// stopCompanions signals every declared companion to exit (idempotent,
// best-effort). No-op when none are declared.
func stopCompanions(resourceName string, companions []ResourceCompanion, stderr io.Writer) {
	for _, c := range companions {
		if err := stopCompanion(resourceName, c); err != nil {
			fmt.Fprintf(stderr, "warning: companion %q for %s did not stop cleanly: %v\n", c.Name, resourceName, err)
		}
	}
}

// resolveCompanionDir is the dir resolver seam (tests point it at a temp dir so
// the supervisor never touches the real runtime home).
var resolveCompanionDir = companionDir

var findExistingCompanion = existingResourceCompanion
var activityEdgeControlSocketReady = activityedge.ControlSocketReady

type CompanionStatus struct {
	Name     string `json:"name"`
	Port     int    `json:"port,omitempty"`
	Required bool   `json:"required"`
	PID      int    `json:"pid,omitempty"`
	Alive    bool   `json:"alive"`
	Failed   bool   `json:"failed,omitempty"`
	Failure  string `json:"failure,omitempty"`
}

// ResourceProcessRecord is the durable ownership projection written for a
// detached resource companion. The PID-file modification time bounds process
// creation because startCompanion writes the file immediately after Start.
type ResourceProcessRecord struct {
	Resource  string
	Name      string
	PID       int
	StartedAt time.Time
}

// ReadResourceProcessRecords reads the companion PID authority under the
// runtime-home processes tree. Malformed individual files are stale evidence
// and are skipped; an unreadable authority root is returned as an error.
func ReadResourceProcessRecords(home string) ([]ResourceProcessRecord, error) {
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyProcesses)
	if err != nil {
		return nil, err
	}
	resourceRoot := filepath.Join(root, "resources")
	resourcesEntries, err := os.ReadDir(resourceRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return []ResourceProcessRecord{}, nil
		}
		return nil, err
	}
	records := make([]ResourceProcessRecord, 0)
	for _, resourceEntry := range resourcesEntries {
		if !resourceEntry.IsDir() {
			continue
		}
		companionEntries, err := os.ReadDir(filepath.Join(resourceRoot, resourceEntry.Name()))
		if err != nil {
			continue
		}
		for _, companionEntry := range companionEntries {
			if companionEntry.IsDir() || filepath.Ext(companionEntry.Name()) != ".pid" {
				continue
			}
			pidPath := filepath.Join(resourceRoot, resourceEntry.Name(), companionEntry.Name())
			pid, ok := readCompanionPID(pidPath)
			if !ok {
				continue
			}
			info, err := companionEntry.Info()
			if err != nil {
				continue
			}
			records = append(records, ResourceProcessRecord{
				Resource:  resourceEntry.Name(),
				Name:      strings.TrimSuffix(companionEntry.Name(), ".pid"),
				PID:       pid,
				StartedAt: info.ModTime(),
			})
		}
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Resource == records[j].Resource {
			return records[i].Name < records[j].Name
		}
		return records[i].Resource < records[j].Resource
	})
	return records, nil
}

var companionCrashWindow = tuning.CompanionCrashWindow()

// companionDir resolves <home>/.vrooli/processes/resources/<resource> (the
// runtime-home processes authority), creating it owned by the operator.
func companionDir(resourceName string) (string, error) {
	home, err := config.HomeDir()
	if err != nil {
		return "", err
	}
	root, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyProcesses)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, "resources", resourceName)
	if _, err := config.EnsureOwnedDir(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func startCompanion(resourceName string, c ResourceCompanion, recoveryAttempts int, parentPIDs ...int) error {
	parentPID := 0
	if len(parentPIDs) > 0 {
		parentPID = parentPIDs[0]
	}
	return startCompanionWithNetwork(resourceName, c, recoveryAttempts, parentPID, nil)
}

func startCompanionWithNetwork(resourceName string, c ResourceCompanion, recoveryAttempts, parentPID int, network *activityEdgeNamespace) error {
	if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Command) == "" {
		return fmt.Errorf("companion declaration is missing name/command")
	}
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return err
	}
	pidPath := filepath.Join(dir, c.Name+".pid")
	statePath := filepath.Join(dir, c.Name+".state")
	stableControlEdge := hasStableActivityEdge(resourceName) && c.Name == "activity-edge"
	previousPID, previousOK := readCompanionPID(statePath)
	pidfilePID, pidfileOK := readCompanionPID(pidPath)
	// Idempotent: a live companion is left running so a restart does not gap the
	// data path (the reverse proxy is stateless and dials the recreated container).
	if pid, ok := pidfilePID, pidfileOK; ok {
		if process.IsPIDRunning(pid) {
			joinNetwork := network != nil && c.Name == manifestpkg.UpstreamBridgeCompanionName
			wrongNamespace := joinNetwork && !sameNetworkNamespace(pid, network.ChildPID)
			if network != nil && stableControlEdge && !sameNetworkNamespace(pid, os.Getpid()) {
				wrongNamespace = true
			}
			if wrongNamespace {
				if stableControlEdge {
					bin, binErr := exec.LookPath(c.Command)
					if binErr != nil || !activityEdgeProcessOwned(pid, bin, filepath.Join(dir, "activity-edge.control.sock"), c.Args) {
						return fmt.Errorf("%s activity edge PID %d is outside the host network namespace and lacks exact ownership proof", resourceName, pid)
					}
				}
				if stopErr := stopCompanion(resourceName, c); stopErr != nil {
					return fmt.Errorf("move %s companion into its private network namespace: %w", resourceName, stopErr)
				}
				return startCompanionWithNetwork(resourceName, c, recoveryAttempts, parentPID, network)
			} else {
				if stableControlEdge {
					readyCtx, cancel := context.WithTimeout(context.Background(), time.Second)
					err := activityEdgeControlSocketReady(readyCtx, filepath.Join(dir, "activity-edge.control.sock"))
					cancel()
					if err != nil {
						bin, binErr := exec.LookPath(c.Command)
						if binErr != nil {
							return binErr
						}
						if !activityEdgeProcessOwned(pid, bin, filepath.Join(dir, "activity-edge.control.sock"), c.Args) {
							return fmt.Errorf("%s activity edge PID %d lacks exact executable and argv ownership proof", resourceName, pid)
						}
						if stopErr := stopCompanion(resourceName, c); stopErr != nil {
							return fmt.Errorf("retire %s activity edge without lifecycle control: %w", resourceName, stopErr)
						}
						pidfileOK = false
					} else {
						clearCompanionCrashState(dir, c.Name)
						return nil
					}
				} else {
					clearCompanionCrashState(dir, c.Name)
					return nil
				}
			}
		}
		// A dead pidfile is stale state, not evidence that the companion is
		// still owned. Reap it before attempting recovery.
		if stableControlEdge {
			if !previousOK || previousPID != pid {
				return fmt.Errorf("refuse to recover %s control socket without matching dead companion identity", resourceName)
			}
			if err := removeStaleActivityEdgeControlSocket(filepath.Join(dir, "activity-edge.control.sock"), pid); err != nil {
				return fmt.Errorf("recover stale %s activity-edge control socket: %w", resourceName, err)
			}
		}
		_ = os.Remove(pidPath)
		if !previousOK || previousPID != pid {
			if err := recordCompanionCrashAttempt(dir, c.Name, recoveryAttempts); err != nil {
				return err
			}
		}
	} else if previousOK && !process.IsPIDRunning(previousPID) {
		// Status intentionally reaps dead pidfiles. The state file preserves
		// the fact that the prior launch existed, so a later start still counts
		// the crash.
		if stableControlEdge {
			if err := removeStaleActivityEdgeControlSocket(filepath.Join(dir, "activity-edge.control.sock"), previousPID); err != nil {
				return fmt.Errorf("recover state-file-owned %s control socket: %w", resourceName, err)
			}
		}
		if err := recordCompanionCrashAttempt(dir, c.Name, recoveryAttempts); err != nil {
			return err
		}
	}
	bin, err := exec.LookPath(c.Command)
	if err != nil {
		return fmt.Errorf("resolve %q on PATH: %w", c.Command, err)
	}
	if pid, err := findExistingCompanion(resourceName, c, bin, parentPID); err != nil {
		return err
	} else if pid > 1 {
		if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), tuning.PermFile); err != nil {
			return err
		}
		if err := os.WriteFile(statePath, []byte(strconv.Itoa(pid)), tuning.PermFile); err != nil {
			return err
		}
		clearCompanionCrashState(dir, c.Name)
		return nil
	}
	logPath := filepath.Join(dir, c.Name+".log")
	boundCompanionLog(logPath, tuning.CompanionLogMaxBytes())
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, tuning.PermFile)
	if err != nil {
		return err
	}
	defer logf.Close()

	companionArgs := companionLaunchArgs(resourceName, c, dir, parentPID)
	companionBin := bin
	var cmd *exec.Cmd
	if network != nil && c.Name == manifestpkg.ActivityEdgeCompanionName {
		companionEnv := os.Environ()
		prefix := strings.ToUpper(strings.ReplaceAll(resourceName, "-", "_"))
		companionEnv = valuehelpers.SetEnv(companionEnv, prefix+"_ACTIVITY_EDGE_LISTEN", fmt.Sprintf("127.0.0.1:%d", network.EdgePort))
		privateEndpoint := "unix://" + network.SocketPath + "?host=" + url.QueryEscape(fmt.Sprintf("127.0.0.1:%d", network.UpstreamPort))
		companionEnv = valuehelpers.SetEnv(companionEnv, prefix+"_ACTIVITY_EDGE_UPSTREAM", privateEndpoint)
		cmd = shell.NewCommand(companionBin, companionArgs...)
		cmd.Env = detachedResourceEnvironment(companionEnv)
	} else if network != nil && c.Name == manifestpkg.UpstreamBridgeCompanionName {
		companionEnv := os.Environ()
		edgePID, ok := readCompanionPID(filepath.Join(dir, "activity-edge.pid"))
		if !ok || edgePID <= 1 {
			return fmt.Errorf("%s activity edge PID is unavailable for private bridge authorization", resourceName)
		}
		companionArgs = append(companionArgs, "--socket", network.SocketPath, "--upstream", fmt.Sprintf("127.0.0.1:%d", network.UpstreamPort), "--peer-pid", strconv.Itoa(edgePID))
		ownerArgs, err := activitynetwork.OwnerNamespaceArgs(network.ChildPID, append([]string{bin}, companionArgs...))
		if err != nil {
			return err
		}
		companionBin = "nsenter"
		companionArgs = ownerArgs
		cmd = shell.NewCommand(companionBin, companionArgs...)
		cmd.Env = detachedResourceEnvironment(companionEnv)
	} else {
		cmd = shell.NewCommand(companionBin, companionArgs...)
		cmd.Env = detachedResourceEnvironment(os.Environ())
	}
	cmd.Stdout = logf
	cmd.Stderr = logf
	if err := platform.ConfigureCommand(cmd, platform.ProcessOptions{Detached: true}); err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	if placeErr := placeResourceProcess(resourceName, "companion-"+c.Name, pid); placeErr != nil {
		fmt.Fprintf(logf, "warning: companion not placed in its own scope, staying in the caller's: %v\n", placeErr)
	}
	// Detach: the companion outlives this short-lived control process.
	_ = cmd.Process.Release()
	clearCompanionFailure(dir, c.Name)
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), tuning.PermFile); err != nil {
		return err
	}
	return os.WriteFile(statePath, []byte(strconv.Itoa(pid)), tuning.PermFile)
}

func whisperEdgeProcessOwned(pid int, bin, socket string, declaredArgs []string) bool {
	return activityEdgeProcessOwned(pid, bin, socket, declaredArgs)
}

func activityEdgeProcessOwned(pid int, bin, socket string, declaredArgs []string) bool {
	expected := append(append([]string(nil), declaredArgs...), "--control-socket", socket)
	return activityEdgeProcessMatches(pid, bin, expected)
}

func activityEdgeProcessMatches(pid int, bin string, expected []string) bool {
	if pid <= 1 {
		return false
	}
	executable, executableErr := platform.ProcessExecutablePath(pid)
	command, commandErr := platform.ProcessCommandLine(pid)
	if executableErr != nil || commandErr != nil || filepath.Clean(strings.TrimSuffix(executable, " (deleted)")) != filepath.Clean(bin) {
		return false
	}
	for _, arg := range expected {
		if arg == "" || strings.ContainsAny(arg, " \t\r\n") {
			return false
		}
	}
	actual := strings.Fields(command)
	return len(actual) == len(expected)+1 && strings.Join(actual[1:], " ") == strings.Join(expected, " ")
}

// gracefullyRetireLegacyActivityEdge handles only the transition where the
// verified stable edge predates its control socket. SIGTERM asks its HTTP
// server to stop accepting and finish in-flight requests; we proceed only when
// the exact owned process exits and records a clean shutdown marker.
func gracefullyRetireLegacyActivityEdge(resourceName string, companion ResourceCompanion, servicePID int) error {
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return err
	}
	pidPath := filepath.Join(dir, companion.Name+".pid")
	pid, ok := readCompanionPID(pidPath)
	if !ok || pid <= 1 || !process.IsPIDRunning(pid) {
		return fmt.Errorf("%s activity edge is not a live tracked process; cannot prove graceful completion", resourceName)
	}
	incarnation, err := managedServiceProcessIncarnation(pid)
	if err != nil {
		return fmt.Errorf("%s activity edge incarnation is unavailable: %w", resourceName, err)
	}
	if incarnation == "" {
		return fmt.Errorf("%s activity edge incarnation is empty", resourceName)
	}
	socket := filepath.Join(dir, "activity-edge.control.sock")
	if _, err := os.Lstat(socket); err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("%s activity-edge control socket changed during legacy recovery", resourceName)
	}
	bin, err := exec.LookPath(companion.Command)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(bin) {
		bin, err = filepath.Abs(bin)
		if err != nil {
			return err
		}
	}
	legacyArgs := append(append([]string(nil), companion.Args...), "--parent-pid", strconv.Itoa(servicePID))
	if !activityEdgeProcessMatches(pid, bin, legacyArgs) {
		return fmt.Errorf("%s activity edge PID %d does not match the exact legacy executable and argv", resourceName, pid)
	}
	logPath := filepath.Join(dir, companion.Name+".log")
	before, err := os.Stat(logPath)
	if err != nil {
		return fmt.Errorf("read %s activity-edge log before graceful retirement: %w", resourceName, err)
	}
	currentIncarnation, err := managedServiceProcessIncarnation(pid)
	if err != nil || currentIncarnation != incarnation || !process.IsPIDRunning(pid) || !activityEdgeProcessMatches(pid, bin, legacyArgs) {
		return fmt.Errorf("%s activity edge owner changed before graceful signal", resourceName)
	}
	if err := terminateCompanion(pid); err != nil {
		return err
	}
	deadline := time.Now().Add(10 * time.Second)
	for process.IsPIDRunning(pid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if process.IsPIDRunning(pid) {
		return fmt.Errorf("%s activity edge PID %d did not exit after graceful signal", resourceName, pid)
	}
	if runtime.GOOS == "linux" {
		if err := verifyCompanionPIDAbsentFromScope(resourceName, companion.Name, servicePID, pid); err != nil {
			return fmt.Errorf("verify %s activity edge left its canonical scope: %w", resourceName, err)
		}
	}
	after, err := os.ReadFile(logPath)
	if err != nil || int64(len(after)) < before.Size() {
		return fmt.Errorf("%s activity-edge log changed during graceful retirement", resourceName)
	}
	newLog := after[before.Size():]
	if !cleanActivityEdgeShutdown(newLog) {
		return fmt.Errorf("%s activity edge exited without a clean graceful-shutdown receipt", resourceName)
	}
	clearCompanionCrashState(dir, companion.Name)
	_ = os.Remove(filepath.Join(dir, companion.Name+".state"))
	if err := os.Remove(pidPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func cleanActivityEdgeShutdown(log []byte) bool {
	return bytes.Contains(log, []byte("activity-edge exit: reason=signal sig=terminated")) &&
		!bytes.Contains(log, []byte("activity-edge exit: reason=signal sig=terminated shutdown_error="))
}

func resourceActivityEdgeCompanion(manifest ResourceManifest) (ResourceCompanion, bool) {
	for _, companion := range manifest.Companions {
		if companion.Name == manifestpkg.ActivityEdgeCompanionName {
			return companion, true
		}
	}
	return ResourceCompanion{}, false
}

func drainActivityEdgeForRestart(ctx context.Context, manifest ResourceManifest, servicePID int) error {
	return drainActivityEdgeForRestartWith(
		func() error { return controlResourceActivityEdge(ctx, manifest.Name, "drain") },
		func() error {
			companion, ok := resourceActivityEdgeCompanion(manifest)
			if !ok || !companion.Required {
				return fmt.Errorf("activity-edge control socket is missing and no required owner is declared")
			}
			return gracefullyRetireLegacyActivityEdge(manifest.Name, companion, servicePID)
		},
	)
}

func drainActivityEdgeForRestartWith(control, retireLegacy func() error) error {
	if control == nil {
		return fmt.Errorf("activity-edge drain owner is unavailable")
	}
	if err := control(); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("drain activity edge: %w", err)
	}
	if retireLegacy == nil {
		return fmt.Errorf("activity-edge control socket is missing and no legacy retirement owner is available")
	}
	if err := retireLegacy(); err != nil {
		return fmt.Errorf("recover missing activity-edge control socket: %w", err)
	}
	return nil
}

func companionLaunchArgs(resourceName string, c ResourceCompanion, dir string, parentPID int) []string {
	args := append([]string{}, c.Args...)
	if hasStableActivityEdge(resourceName) && c.Name == "activity-edge" {
		return append(args, "--control-socket", filepath.Join(dir, "activity-edge.control.sock"))
	}
	if parentPID > 1 {
		args = append(args, "--parent-pid", strconv.Itoa(parentPID))
	}
	return args
}

func hasStableActivityEdge(resourceName string) bool {
	switch resourceName {
	case "whisper", "reranker", "ollama", "kyutai-stt":
		return true
	default:
		return false
	}
}

func terminateCompanion(pid int) error {
	return platform.KillProcess(pid, false)
}

func stopCompanion(resourceName string, c ResourceCompanion) error {
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return err
	}
	pidPath := filepath.Join(dir, c.Name+".pid")
	pid, ok := readCompanionPID(pidPath)
	if !ok {
		clearCompanionCrashState(dir, c.Name)
		return nil // nothing tracked — already stopped
	}
	if process.IsPIDRunning(pid) {
		if err := terminateCompanion(pid); err != nil {
			return err
		}
		// Signal delivery is not completion. A proxy may hold the public port
		// while it drains, so retain its ownership record until it has exited.
		deadline := time.Now().Add(5 * time.Second)
		for process.IsPIDRunning(pid) {
			if time.Now().After(deadline) {
				return fmt.Errorf("companion %s pid %d did not exit within 5s; ownership retained", c.Name, pid)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	clearCompanionCrashState(dir, c.Name)
	_ = os.Remove(filepath.Join(dir, c.Name+".state"))
	return os.Remove(pidPath)
}

func companionStatuses(resourceName string, companions []ResourceCompanion) ([]CompanionStatus, error) {
	if len(companions) == 0 {
		return nil, nil
	}
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return nil, err
	}
	statuses := make([]CompanionStatus, 0, len(companions))
	for _, c := range companions {
		boundCompanionLog(filepath.Join(dir, c.Name+".log"), tuning.CompanionLogMaxBytes())
		pid, _ := readCompanionPID(filepath.Join(dir, c.Name+".pid"))
		failed, failure := readCompanionFailure(dir, c.Name)
		statuses = append(statuses, CompanionStatus{
			Name:     c.Name,
			Port:     c.Port,
			Required: c.Required,
			PID:      pid,
			Alive:    pid > 0 && process.IsPIDRunning(pid),
			Failed:   failed,
			Failure:  failure,
		})
		if pid > 0 && !process.IsPIDRunning(pid) {
			_ = os.Remove(filepath.Join(dir, c.Name+".pid"))
		}
	}
	return statuses, nil
}

func downCompanions(statuses []CompanionStatus) []CompanionStatus {
	down := make([]CompanionStatus, 0)
	for _, status := range statuses {
		if !status.Alive {
			down = append(down, status)
		}
	}
	return down
}

func requiredDownCompanions(statuses []CompanionStatus) []CompanionStatus {
	down := make([]CompanionStatus, 0)
	for _, status := range statuses {
		if status.Required && !status.Alive {
			down = append(down, status)
		}
	}
	return down
}

func companionDownMessage(resourceName string, down []CompanionStatus) string {
	if len(down) == 0 {
		return scenarioruntime.HealthStatusHealthy
	}
	parts := make([]string, 0, len(down))
	for _, status := range down {
		label := strings.TrimSpace(status.Name)
		if label == "" {
			label = "unnamed"
		}
		if status.Port > 0 {
			label = fmt.Sprintf("%s companion down (port %d)", label, status.Port)
		} else {
			label = fmt.Sprintf("%s companion down", label)
		}
		if status.Failed {
			if strings.TrimSpace(status.Failure) != "" {
				label += ": " + status.Failure
			} else {
				label += ": crash-loop cap reached"
			}
		}
		parts = append(parts, label)
	}
	message := "running; " + strings.Join(parts, "; ")
	if resourceName == "whisper" {
		message += " - STT unavailable, capacity reporting blind"
	}
	return message
}

func readCompanionPID(pidPath string) (int, bool) {
	data, err := os.ReadFile(pidPath)
	if err != nil {
		return 0, false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, false
	}
	return pid, true
}

func companionAttemptsPath(dir, name string) string {
	return filepath.Join(dir, name+".attempts")
}

func companionFailedPath(dir, name string) string {
	return filepath.Join(dir, name+".failed")
}

func recordCompanionCrashAttempt(dir, name string, recoveryAttempts int) error {
	if recoveryAttempts <= 0 {
		return nil
	}
	now := time.Now()
	attempts := recentCompanionAttempts(companionAttemptsPath(dir, name), now)
	if len(attempts) >= recoveryAttempts {
		message := fmt.Sprintf("crash-loop cap reached: %d respawn attempts within %s", recoveryAttempts, companionCrashWindow)
		if err := os.WriteFile(companionFailedPath(dir, name), []byte(message), tuning.PermFile); err != nil {
			return err
		}
		appendCompanionLog(dir, name, "companion supervisor: "+message+"\n")
		return fmt.Errorf("%s", message)
	}
	attempts = append(attempts, now)
	return writeCompanionAttempts(companionAttemptsPath(dir, name), attempts)
}

func recentCompanionAttempts(path string, now time.Time) []time.Time {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	cutoff := now.Add(-companionCrashWindow)
	lines := strings.Split(string(data), "\n")
	attempts := make([]time.Time, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		unixNano, err := strconv.ParseInt(strings.TrimSpace(line), 10, 64)
		if err != nil {
			continue
		}
		at := time.Unix(0, unixNano)
		if at.After(cutoff) {
			attempts = append(attempts, at)
		}
	}
	return attempts
}

func writeCompanionAttempts(path string, attempts []time.Time) error {
	lines := make([]string, 0, len(attempts))
	for _, at := range attempts {
		lines = append(lines, strconv.FormatInt(at.UnixNano(), 10))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), tuning.PermFile)
}

func readCompanionFailure(dir, name string) (bool, string) {
	data, err := os.ReadFile(companionFailedPath(dir, name))
	if err != nil {
		return false, ""
	}
	return true, strings.TrimSpace(string(data))
}

func clearCompanionCrashState(dir, name string) {
	clearCompanionFailure(dir, name)
	_ = os.Remove(companionAttemptsPath(dir, name))
}

func clearCompanionFailure(dir, name string) {
	_ = os.Remove(companionFailedPath(dir, name))
}

func appendCompanionLog(dir, name, line string) {
	logPath := filepath.Join(dir, name+".log")
	maxBytes := tuning.CompanionLogMaxBytes()
	if info, err := os.Stat(logPath); err == nil && info.Size() >= maxBytes {
		rotated := logPath + ".1"
		_ = os.Remove(rotated)
		if err := os.Rename(logPath, rotated); err == nil {
			boundCompanionLog(rotated, maxBytes)
		}
	}
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, tuning.PermFile)
	if err != nil {
		return
	}
	defer logf.Close()
	_, _ = logf.WriteString(line)
}

// boundCompanionLog keeps the newest complete lines in a retained log. This
// matters when the cap is introduced after an old companion has already
// produced an oversized file: rotation must bound the retained artifact too.
func boundCompanionLog(path string, maxBytes int64) {
	if maxBytes <= 0 {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil || int64(len(data)) <= maxBytes {
		return
	}
	data = data[len(data)-int(maxBytes):]
	if cut := strings.IndexByte(string(data), '\n'); cut >= 0 && cut+1 < len(data) {
		data = data[cut+1:]
	}
	_ = os.WriteFile(path, data, tuning.PermFile)
}

// Detached resources belong to their resource lifecycle, not to the agent that
// requested startup. Retaining that agent's credentials or run labels both
// leaks its authority and makes historical run drainage follow a live service.
func detachedResourceEnvironment(env []string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if key == cliutil.EnvIdentityToken || key == "VROOLI_RUN_ID" || strings.HasSuffix(key, "AGENT_TAG") {
			continue
		}
		out = append(out, entry)
	}
	return out
}
