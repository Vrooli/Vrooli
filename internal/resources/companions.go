package resources

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/process"
)

// startCompanions launches every declared host-side companion (idempotent,
// best-effort). A companion that fails to start is a logged warning, never fatal:
// the container is already up and the resource's health check surfaces a dead
// companion (e.g. whisper's edge owns the canonical port, so a down edge fails
// the health probe). When no companions are declared this is a no-op, keeping the
// driver byte-identical for every non-adopting managed-service resource.
func startCompanions(resourceName string, companions []ResourceCompanion, recoveryAttempts int, stderr io.Writer, parentPIDs ...int) {
	parentPID := 0
	if len(parentPIDs) > 0 {
		parentPID = parentPIDs[0]
	}
	for _, c := range companions {
		if err := startCompanion(resourceName, c, recoveryAttempts, parentPID); err != nil {
			fmt.Fprintf(stderr, "warning: companion %q for %s did not start: %v\n", c.Name, resourceName, err)
		}
	}
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
	if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Command) == "" {
		return fmt.Errorf("companion declaration is missing name/command")
	}
	dir, err := resolveCompanionDir(resourceName)
	if err != nil {
		return err
	}
	pidPath := filepath.Join(dir, c.Name+".pid")
	statePath := filepath.Join(dir, c.Name+".state")
	previousPID, previousOK := readCompanionPID(statePath)
	pidfilePID, pidfileOK := readCompanionPID(pidPath)
	// Idempotent: a live companion is left running so a restart does not gap the
	// data path (the reverse proxy is stateless and dials the recreated container).
	if pid, ok := pidfilePID, pidfileOK; ok {
		if process.IsPIDRunning(pid) {
			clearCompanionCrashState(dir, c.Name)
			return nil
		}
		// A dead pidfile is stale state, not evidence that the companion is
		// still owned. Reap it before attempting recovery.
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
		if err := recordCompanionCrashAttempt(dir, c.Name, recoveryAttempts); err != nil {
			return err
		}
	}
	bin, err := exec.LookPath(c.Command)
	if err != nil {
		return fmt.Errorf("resolve %q on PATH: %w", c.Command, err)
	}
	logPath := filepath.Join(dir, c.Name+".log")
	boundCompanionLog(logPath, tuning.CompanionLogMaxBytes())
	logf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, tuning.PermFile)
	if err != nil {
		return err
	}
	defer logf.Close()

	cmd := shell.NewCommand(bin, c.Args...)
	if len(parentPIDs) > 0 && parentPIDs[0] > 1 {
		cmd = shell.NewCommand(bin, append(append([]string{}, c.Args...), "--parent-pid", strconv.Itoa(parentPIDs[0]))...)
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
	// Detach: the companion outlives this short-lived control process.
	_ = cmd.Process.Release()
	clearCompanionFailure(dir, c.Name)
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), tuning.PermFile); err != nil {
		return err
	}
	return os.WriteFile(statePath, []byte(strconv.Itoa(pid)), tuning.PermFile)
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
